package main

import (
	"bytes"
	"io"
	"log"
	"net"
	"net/http"

	rrs "github.com/KolobokMysnoy/tmp/general/requestResponseStruct"
)

type SaveFunc func(rrs.Response, rrs.Request) error

type Proxy interface {
	ServeHTTP(http.ResponseWriter, *http.Request)
	addSaveFunc(SaveFunc)
}

type ProxyHTTP struct {
	save SaveFunc
}

func (p *ProxyHTTP) addSaveFunc(addFunc SaveFunc) {
	p.save = addFunc
}

func (p ProxyHTTP) ServeHTTP(wr http.ResponseWriter, req *http.Request) {
	log.Print("start serving HTTP")

	PreparationForHttp{}.Prepare(wr, req)

	if clientIP, _, err := net.SplitHostPort(req.RemoteAddr); err == nil {
		appendHostToXForwardHeader(req.Header, clientIP)
	}

	log.Print("start getting data from request")
	request, err := translateFromHTTPtoRequest(req)
	if err != nil {
		log.Println("Error parsing form data:", err)
		http.Error(wr, "Internal Server Error",
			http.StatusInternalServerError)

		return
	}

	log.Print("start request to server")
	resp, err := p.getClientReply(req, wr)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	log.Print("start reading from response")
	htmlBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("Error:", err)
		return
	}

	response := rrs.Response{
		Code:    resp.StatusCode,
		Message: resp.Status,
		Headers: resp.Header,
		Body:    string(htmlBytes),
	}

	log.Print("start saving data to bd")
	err = p.save(response, request)
	if err != nil {
		log.Print(err)
		return
	}

	copyHeader(wr.Header(), resp.Header)
	wr.WriteHeader(resp.StatusCode)
	io.Copy(wr, bytes.NewReader(htmlBytes))
}

func (p *ProxyHTTP) getClientReply(req *http.Request, wr http.ResponseWriter) (*http.Response, error) {
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
