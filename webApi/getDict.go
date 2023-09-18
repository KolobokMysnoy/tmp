package main

import (
	"io"
	"log"
	"net/http"
	"strings"

	rrs "github.com/KolobokMysnoy/tmp/general/requestResponseStruct"
)

func getAllUrls() ([]string, error) {
	//http://raw.githubusercontent.com/maurosoria/dirsearch/master/db/dicc.txt
	client := &http.Client{}

	outgoingReq, err := http.NewRequest(
		"GET",
		"http://raw.githubusercontent.com/maurosoria/dirsearch/master/db/dicc.txt",
		strings.NewReader(""),
	)

	if err != nil {
		return nil, err
	}

	resp, err := client.Do(outgoingReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	htmlBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	parsedUrls := parseParams(htmlBytes)

	return parsedUrls, nil
}

func parseParams(htmlBytes []byte) []string {
	stringBytes := string(htmlBytes)

	stringArr := strings.Split(stringBytes, ",")
	for i, v := range stringArr {
		tmp := strings.TrimSpace(v)
		stringArr[i] = strings.Trim(tmp, "\"")
	}

	return stringArr
}

func getErrorUrls(urls []string, req rrs.Request) ([]string, error) {
	var results []string
	for _, v := range urls {
		req.Path = v

		var resp *http.Response
		resp, _ = getServerReply(req)

		defer resp.Body.Close()

		log.Print(resp.StatusCode)
		if resp.StatusCode != http.StatusNotFound {
			// TODO file output??
			results = append(results, v)
		}
	}

	return results, nil
}
