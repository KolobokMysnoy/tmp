package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	rrs "github.com/KolobokMysnoy/tmp/general/requestResponseStruct"

	BD "github.com/KolobokMysnoy/tmp/general/BD"

	"github.com/go-chi/chi"
)

func repeat(writer http.ResponseWriter, request *http.Request) {
	id := chi.URLParam(request, "id")
	bd := BD.MongoDB{}

	req, err := bd.GetRequestByID(id)
	if err != nil {
		errStr := "can't get request by id = " + id + ":"
		log.Print(errStr, err)
		http.Error(writer, errStr,
			http.StatusInternalServerError)
		return
	}

	resp, err := getServerReply(req)
	if err != nil {
		log.Print(err)
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

	err = bd.SaveResponseRequest(response, req)
	if err != nil {
		log.Print(err)
		return
	}
	log.Print("Save data to bd")

	copyHeader(writer.Header(), resp.Header)
	writer.WriteHeader(resp.StatusCode)
	io.Copy(writer, bytes.NewReader(htmlBytes))
}

func getServerReply(req rrs.Request) (*http.Response, error) {
	client := &http.Client{}

	postParamsString := url.Values(req.PostParams).Encode()
	url := &url.URL{
		Scheme:   req.Scheme,
		Host:     req.Host,
		Path:     req.Path,
		RawQuery: url.Values(req.GetParams).Encode(),
	}

	outgoingReq, err := http.NewRequest(
		req.Method,
		url.String(),
		strings.NewReader(postParamsString))

	if err != nil {
		return nil, err
	}

	outgoingReq.Header = req.Headers
	for _, cookie := range req.Cookies {
		outgoingReq.AddCookie(&cookie)
	}

	resp, err := client.Do(outgoingReq)
	if err != nil {
		return nil, err
	}

	delHopHeaders(resp.Header)

	return resp, nil
}

func requests(wr http.ResponseWriter, req *http.Request) {
	bd := BD.MongoDB{}

	allReq, err := bd.GetAllRequests()
	if err != nil {
		errStr := "can't get requests"
		log.Print(errStr, err)
		http.Error(wr, errStr,
			http.StatusInternalServerError)
		return
	}

	responseJSON, err := json.Marshal(allReq)
	if err != nil {
		http.Error(wr, err.Error(), http.StatusInternalServerError)
		return
	}

	wr.Header().Set("Content-Type", "application/json")
	_, err = wr.Write(responseJSON)
	if err != nil {
		http.Error(wr, err.Error(), http.StatusInternalServerError)
		return
	}

	wr.WriteHeader(http.StatusOK)
}

func requestById(wr http.ResponseWriter, req *http.Request) {
	id := chi.URLParam(req, "id")
	bd := BD.MongoDB{}

	reqById, err := bd.GetRequestByID(id)
	if err != nil {
		errStr := "can't get request by id = " + id + ":"
		log.Print(errStr, err)
		http.Error(wr, errStr,
			http.StatusInternalServerError)
		return
	}

	responseJSON, err := json.Marshal(reqById)
	if err != nil {
		http.Error(wr, err.Error(), http.StatusInternalServerError)
		return
	}

	wr.Header().Set("Content-Type", "application/json")
	_, err = wr.Write(responseJSON)
	if err != nil {
		http.Error(wr, err.Error(), http.StatusInternalServerError)
		return
	}

	wr.WriteHeader(http.StatusOK)
}

func scan(wr http.ResponseWriter, request *http.Request) {
	log.Print("run operation scan")
	id := chi.URLParam(request, "id")
	bd := BD.MongoDB{}

	log.Print("request by id =", id)
	req, err := bd.GetRequestByID(id)
	if err != nil {
		errStr := "can't get request by id = " + id + ":"
		log.Print(errStr, err)
		http.Error(wr, errStr,
			http.StatusInternalServerError)
		return
	}

	log.Print("getting urls from dictionary")
	urls, err := getAllUrls()
	if err != nil {
		log.Println("error at getting urls:", err)
		return
	}

	log.Print("start scanning")
	urlsFound, err := getErrorUrls(urls, req)
	if err != nil {
		log.Print(err)
		http.Error(wr, "Error with checking urls", http.StatusInternalServerError)
		return
	}

	log.Print("Get server reply")

	responseBody := []byte(strings.Join(urlsFound, "\n"))

	wr.WriteHeader(http.StatusOK)
	io.Copy(wr, bytes.NewReader(responseBody))
}
