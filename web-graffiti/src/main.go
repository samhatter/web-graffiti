package main

import (
	"fmt"
)

func main() {
	fmt.Println("Web-Graffiti Initializing...")
	go catchAndRelease()
	go updatePort()

	select {}
}