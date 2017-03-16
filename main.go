package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	password, set := os.LookupEnv("UPLOAD_SECRET")
	if !set {
		password = "p4ssw0rd"
	}

	http.Handle("/upload", withAuth(password, handler(upload)))
	http.Handle("/listen", handler(listen))
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	http.Handle("/", handler(index))
	log.Fatal(http.ListenAndServe(":8080", nil))
}
