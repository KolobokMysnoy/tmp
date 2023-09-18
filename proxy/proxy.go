package main

import (
	"bytes"
	"io"
	"log"
	"net"
	"net/http"
	"strings"

	rrs "github.com/KolobokMysnoy/tmp/general/requestResponseStruct"
)

type SaveFunc func(rrs.Response, rrs.Request) error

type Proxy interface {
	ServeHTTP(http.ResponseWriter, *http.Request)
	SaveReqAndResp(SaveFunc)
}

type ProxyHTTP struct {
	save SaveFunc
}

func (p *ProxyHTTP) SaveReqAndResp(addFunc SaveFunc) {
	p.save = addFunc
}

func (p ProxyHTTP) ServeHTTP(wr http.ResponseWriter, req *http.Request) {
	prepare := PreparationForHttp{}
	prepare.Prepare(wr, req)
	log.Print("Start serving HTTP")

	if clientIP, _, err := net.SplitHostPort(req.RemoteAddr); err == nil {
		appendHostToXForwardHeader(req.Header, clientIP)
	}

	postParams := make(map[string][]string)

	contentTypeValues := req.Header.Values("CONTENT-TYPE")
	for _, typeOfHead := range contentTypeValues {
		if typeOfHead == "application/x-www-form-urlencoded" {
			err := req.ParseForm()
			if err != nil {
				log.Println("Error parsing form data:", err)
				http.Error(wr, "Internal Server Error",
					http.StatusInternalServerError)
				return
			}

			for key, values := range req.Form {
				for _, value := range values {
					postParams[key] = append(postParams[key], value)
				}
			}
		}
	}
	log.Print("Get post params")

	getParams := make(map[string][]string)

	query := req.URL.Query()
	for key, values := range query {
		for _, value := range values {
			getParams[key] = append(getParams[key], value)
		}
	}
	log.Print("Get get params")

	var cookies []http.Cookie
	for _, cookie := range req.Cookies() {
		cookies = append(cookies, *cookie)
	}
	log.Print("Get cookies")

	requestToWork := rrs.Request{
		Scheme:     req.URL.Scheme,
		Method:     req.Method,
		Path:       req.URL.Path,
		Host:       req.Host,
		GetParams:  getParams,
		Headers:    req.Header,
		Cookies:    cookies,
		PostParams: postParams,
	}

	resp, err := p.getClientReply(req, wr)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	htmlBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("Error:", err)
		return
	}

	log.Print("Get server reply")

	response := rrs.Response{
		Code:    resp.StatusCode,
		Message: resp.Status,
		Headers: resp.Header,
		Body:    string(htmlBytes),
	}

	err = p.save(response, requestToWork)
	if err != nil {
		log.Print(err)
		return
	}
	log.Print("Save data to bd")

	log.Println(req.RemoteAddr, " ", resp.Status)

	copyHeader(wr.Header(), resp.Header)
	wr.WriteHeader(resp.StatusCode)
	io.Copy(wr, bytes.NewReader(htmlBytes))
}

func appendHostToXForwardHeader(header http.Header, host string) {
	if prior, ok := header["X-Forwarded-For"]; ok {
		host = strings.Join(prior, ", ") + ", " + host
	}
	header.Set("X-Forwarded-For", host)
}

func copyHeader(dest, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dest.Add(k, v)
		}
	}
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
