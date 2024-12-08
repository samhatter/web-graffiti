package main

import (
	"fmt"
	"time"
)

func catchAndRelease(){

}

func monitor(){
	
}

func main() {
	fmt.Println("Web-Graffiti Initializing...")
	time.AfterFunc(3*24*time.Hour, catchAndRelease)
	time.AfterFunc()
}