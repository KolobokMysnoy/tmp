package main

import (
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strings"

	rrs "github.com/KolobokMysnoy/tmp/general/requestResponseStruct"
)

type SaveFunc func(rrs.Response, rrs.Request) error

type ProxyHTTP struct {
	save SaveFunc
}

func (p *ProxyHTTP) addSaveFunc(addFunc SaveFunc) {
	p.save = addFunc
}

func (p *ProxyHTTP) ServeH(upstream http.Handler, isSecureCon bool) http.Handler {
	return http.HandlerFunc(func(wr http.ResponseWriter, req *http.Request) {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			log.Print("err read body: ", err)
			return
		}

		PreparationForHttp{}.Prepare(wr, req)

		if clientIP, _, err := net.SplitHostPort(req.RemoteAddr); err == nil {
			appendHostToXForwardHeader(req.Header, clientIP)
		}

		recorder := &customRecorder{ResponseWriter: wr}
		recorder.Header().Set("Content-Encoding", "identity")

		req.Body = ioutil.NopCloser(bytes.NewBuffer(body))

		log.Print("start getting data from request")
		request, err := translateFromHTTPtoRequest(req)
		if err != nil {
			log.Println("Error parsing form data:", err)
			http.Error(wr, "Internal Server Error",
				http.StatusInternalServerError)

			return
		}

		var protocol string
		if isSecureCon {
			protocol = "https"
		} else {
			protocol = "http"
		}

		request.Scheme = protocol

		req.Body = ioutil.NopCloser(bytes.NewBuffer(body))
		var resp rrs.Response
		if isSecureCon {
			log.Print("serve https")
			upstream.ServeHTTP(recorder, req)

			reqHeaders := parseReqHeaders(req)
			var resTextBody string
			// TODO:
			if strings.Contains(reqHeaders["Content-Type"], "text") ||
				(strings.Contains(reqHeaders["Content-Type"], "application") && !strings.Contains(reqHeaders["Content-Type"], "application/octet-stream")) {
				resTextBody = string(recorder.response)
			}

			resp = rrs.Response{
				Code:    recorder.code,
				Message: string(recorder.response),
				Headers: recorder.Header(),
				Body:    resTextBody,
			}
		} else {
			log.Print("serve http")

			resp, err = getHttpResponse(req, recorder)
			if err != nil {
				log.Println("Error getting  data:", err)
				http.Error(wr, "Internal Server Error",
					http.StatusInternalServerError)
			}
		}
		log.Print("Req", req)

		log.Print("start saving data to bd")
		go func() {
			if err := p.save(resp, request); err != nil {
				log.Print(err)
				return
			}
		}()

	})
}

func parseReqHeaders(r *http.Request) map[string]string {
	data := make(map[string]string)
	for name, values := range r.Header {
		if name != "Cookie" {
			data[name] = values[0]
		}
	}
	return data
}

// codeRecorder - не реализация интерфейса http.Hijacker
// даже если http.ResponseWriter внутри него является.
type customRecorder struct {
	http.ResponseWriter

	response []byte
	code     int
}

func (w *customRecorder) WriteHeader(code int) {
	w.code = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *customRecorder) Write(b []byte) (int, error) {
	w.response = append(w.response, b...)
	return w.ResponseWriter.Write(b)
}

func getHttpResponse(req *http.Request, wr http.ResponseWriter) (rrs.Response, error) {
	resp, err := getClientReply(req, wr)
	if err != nil {
		return rrs.Response{}, nil
	}
	defer resp.Body.Close()

	log.Print("start reading from response")
	htmlBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("Error:", err)
		return rrs.Response{}, nil
	}

	response := rrs.Response{
		Code:    resp.StatusCode,
		Message: resp.Status,
		Headers: resp.Header,
		Body:    string(htmlBytes),
	}

	copyHeader(wr.Header(), resp.Header)
	wr.WriteHeader(resp.StatusCode)
	io.Copy(wr, bytes.NewReader(htmlBytes))

	return response, nil
}

func getClientReply(req *http.Request, wr http.ResponseWriter) (*http.Response, error) {
	client := &http.Client{}

	// Read the request body
	requestBody, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(wr, "Server Error", http.StatusInternalServerError)
		log.Print("getClientReply: Error reading request body:", err)
		return nil, err
	}

	// Create a new request with the same method, URL, headers, and body
	outgoingReq, err := http.NewRequest(req.Method, req.URL.String(), bytes.NewReader(requestBody))
	if err != nil {
		http.Error(wr, "Server Error", http.StatusInternalServerError)
		log.Print("getClientReply: Error creating outgoing request:", err)
		return nil, err
	}

	// Copy headers from the original request to the outgoing request
	for key, values := range req.Header {
		for _, value := range values {
			outgoingReq.Header.Add(key, value)
		}
	}

	// Perform the outgoing request
	resp, err := client.Do(outgoingReq)
	if err != nil {
		http.Error(wr, "Server Error", http.StatusInternalServerError)
		log.Print("getClientReply: Error performing outgoing request:", err)
		return nil, err
	}

	delHopHeaders(resp.Header)

	return resp, nil
}
