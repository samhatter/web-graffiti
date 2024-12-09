package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"time"
	"github.com/google/uuid"
)

func catchAndRelease(){
	for {
		fmt.Println("Calling Catch and Release")

		size := getStorageSize()
		
		status, _ := sendSearch("flac")

		fmt.Printf("size: %d, results: %d\n", size, status)

		time.Sleep(5*time.Second)
	}
}

func getStorageSize()(int64){
	var size int64
	path := "/storage"
	filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("Couldn't Get Directory Size: %v\n", err)
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})

	fmt.Printf("Current Media Storage Size: %d\n",size)
	return size
}

func sendSearch(search string)(int, error) {
	url := "http://web-graffiti-gluetun:5554/api/v0/searches"
		
	type SearchRequest struct {
		Id string `json:"id"`
		SearchText string `json:"searchText"`
	}
	
	data := SearchRequest{
		Id: uuid.New().String(),
		SearchText: search,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		fmt.Printf("Error marshalling JSON: %v\n", err)
		return 0, err
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error making POST request: %v\n", err)
	}

	return resp.StatusCode, nil
}