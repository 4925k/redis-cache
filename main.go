package main

import (
	"log"
	"net/http"
)

func main() {
	log.Println("starting server")

	http.HandleFunc("/api", Handler)

	http.ListenAndServe(":8080", nil)
}

func Handler(w http.ResponseWriter, r *http.Request) {
	log.Println("in the handler")
}
