package main

import (
	"net/http"
	"strings"

	rrs "github.com/KolobokMysnoy/tmp/general/requestResponseStruct"
)

var hopHeaders = []string{
	"Connection",
	"Keep-Alive",
	"Proxy-Connection",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"Te",
	"Trailers",
	"Transfer-Encoding",
	"Upgrade",
}

var deleteSpecificHeaders = []string{
	"Accept-encoding",
}

func delHopHeaders(header http.Header) {
	for _, h := range hopHeaders {
		header.Del(h)
	}
}

func deleteHopHeaders(w http.ResponseWriter, r *http.Request) {
	delHopHeaders(r.Header)
	r.RequestURI = ""
}

func removeEncoding(w http.ResponseWriter, r *http.Request) {
	header := r.Header
	for _, b := range deleteSpecificHeaders {
		header.Del(b)
	}
}

type Preparation interface {
	Prepare(http.Handler) http.Handler
}

type PreparationForHttp struct {
}

func (p PreparationForHttp) Prepare(w http.ResponseWriter, r *http.Request) {
	deleteHopHeaders(w, r)
	removeEncoding(w, r)
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

func getPostParams(req http.Request) (map[string][]string, error) {
	results := make(map[string][]string)
	contentTypeValues := req.Header.Values("CONTENT-TYPE")

	for _, typeOfHead := range contentTypeValues {
		if typeOfHead == "application/x-www-form-urlencoded" {
			err := req.ParseForm()
			if err != nil {
				return nil, err
			}

			for key, values := range req.Form {
				for _, value := range values {
					results[key] = append(results[key], value)
				}
			}
		}
	}
	return results, nil
}

func getCookies(src *http.Request) []http.Cookie {
	var results []http.Cookie
	for _, cookie := range src.Cookies() {
		results = append(results, *cookie)
	}

	return results
}

func translateFromHTTPtoRequest(src *http.Request) (rrs.Request, error) {
	postParams, err := getPostParams(*src)
	if err != nil {
		return rrs.Request{}, err
	}

	getParams := make(map[string][]string)

	query := src.URL.Query()
	for key, values := range query {
		for _, value := range values {
			getParams[key] = append(getParams[key], value)
		}
	}

	cookies := getCookies(src)

	return rrs.Request{
		Scheme:     src.URL.Scheme,
		Method:     src.Method,
		Path:       src.URL.Path,
		Host:       src.Host,
		GetParams:  getParams,
		Headers:    src.Header,
		Cookies:    cookies,
		PostParams: postParams,
	}, nil
}
