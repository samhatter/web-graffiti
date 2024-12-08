package main

import (
	"fmt"
	"io"
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
	fmt.Println("Web-Graffiti Initializing...")
	time.AfterFunc(3*24*time.Hour, catchAndRelease)
	http.HandleFunc("/slskd_webhook", webhookHandler)
}