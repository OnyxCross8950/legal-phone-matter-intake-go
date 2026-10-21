package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	client, err := newPhoneClient()
	if err != nil {
		log.Fatal(err)
	}
	s := &intake{pending: make(map[string]matter), client: client}
	mux := http.NewServeMux()
	mux.HandleFunc("/intake/start", s.start)
	mux.HandleFunc("/intake/confirm", s.confirm)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
