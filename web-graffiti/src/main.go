package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

func catchAndRelease(){

}

func webhookHandler (w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}
	fmt.Printf("Received webhook payload: %s\n", body)
}

func main() {
	port := ":8080"
	fmt.Println("Web-Graffiti Initializing...")
	log.Fatal(http.ListenAndServe(port, nil))
	fmt.Printf("Server is running on port %s\n", port)
	time.AfterFunc(3*24*time.Hour, catchAndRelease)
	http.HandleFunc("/slskd_webhook", webhookHandler)
}