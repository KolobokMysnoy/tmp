package main

import (
	"crypto/tls"
	"log"
	"net/http"
	"os"
	"path"

	db "github.com/KolobokMysnoy/tmp/general/BD"
)

var (
	hostname, _ = os.Hostname()

	dir      = path.Join(os.Getenv("HOME"), ".mitm")
	keyFile  = path.Join(dir, "ca-key.pem")
	certFile = path.Join(dir, "ca-cert.pem")
)

func main() {
	// load
	ca, err := loadCA()
	if err != nil {
		log.Fatal(err)
	}
	bd := db.MongoDB{}

	pr := ProxyHTTP{}
	pr.addSaveFunc(bd.SaveResponseRequest)

	p := &Proxy{
		CA: &ca,
		TLSServerConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
			//CipherSuites: []uint16{tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA},
		},
		Wrap: pr.ServeH,
	}
	log.Print("Proxy start at 8080")
	log.Fatal(http.ListenAndServe(":8080", p))
}
