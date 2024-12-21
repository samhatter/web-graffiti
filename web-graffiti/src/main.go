package main

import (
	"fmt"
)

func main() {
	fmt.Println("Web-Graffiti Initializing...")
	go queueDownloads()
	go updatePort()
	go processDownloads()
	go processUploads()
	select {}
}