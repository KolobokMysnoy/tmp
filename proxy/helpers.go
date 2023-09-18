package main

import (
	"log"
	"net/http"
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

	log.Println("Delete headers completed!")
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
	log.Print("Start prepare")
	deleteHopHeaders(w, r)
	removeEncoding(w, r)
}
