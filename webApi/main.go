package main

import (
	"net/http"

	"github.com/go-chi/chi"
)

func main() {
	router := chi.NewRouter()
	router.Get("/requests", requests)
	router.Get("/repeat/{id}", repeat)
	router.Get("/requests/{id}", requestById)
	router.Get("/scan/{id}", repeat)

	http.Handle("/", router)
	http.ListenAndServe(":8081", nil)
}
