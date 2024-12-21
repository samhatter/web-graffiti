package main

import (
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"os"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"
)

func processUploads() {
	waitTime, err := strconv.Atoi(os.Getenv("PROCESS_UPLOADS_TIMER"))
	if err != nil {
		fmt.Printf("Error Reading PROCESS_UPLOADS_TIMER%v\n", err)
		waitTime = 24
	}
	fmt.Printf("PROCESS_UPLOADS_TIMER: %d\n", waitTime)

	GiB := 1024*1024*1024
	removeChunkSize, err := strconv.Atoi(os.Getenv("REMOVE_CHUNK_SIZE"))
	if err != nil {
		fmt.Printf("Error Reading REMOVE_CHUNK_SIZE%v\n", err)
		removeChunkSize = 10
	}
	removeChunkSize = removeChunkSize*GiB
	fmt.Printf("REMOVE_CHUNK_SIZE: %d\n", removeChunkSize)

	for {
		fmt.Println("Processing Uploads...")
		directoryFrequency, err := fetchUploads()
		if err != nil {
			fmt.Printf("Could not fetch uploads. Error: %v\n", err)
		} else {
			directorySize, err := fetchShares()
			if err != nil {
				fmt.Printf("Could not fetch shares. Error: %v\n", err)
			} else {
				removeDirectories(directoryFrequency, directorySize, removeChunkSize)
			}
		}
		fetchShares()
		
		scanShares()
		fmt.Println("Done Processing Uploads")
		time.Sleep(time.Duration(waitTime) * time.Hour)
	}
}

func fetchUploads() (map[string]int, error) {
	url := "http://web-graffiti-gluetun:5554/api/v0/transfers/uploads?includeRemoved=true"

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("Error creating request: %v\n", err)
	}

	for {
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("Error making GET request: %v\n", err)
			time.Sleep(1*time.Second)
		} else {
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, fmt.Errorf("Error reading response body: %v\n", err)
			}

			var userUploads []UserTransfer
			err = json.Unmarshal(body, &userUploads)
			directoryFrequency := make(map[string]int)
			if err != nil {
				return nil, fmt.Errorf("Error parsing response body: %v\n", err)
			} else {
				fmt.Printf("Found %d Uploads\n", len(userUploads))
				for _, userUpload := range userUploads {
					for _, directoryUpload := range userUpload.Directories {
						directoryFrequency[directoryUpload.Directory] += directoryUpload.FileCount
					}
				}
				return directoryFrequency, nil
			}
		}
	}

}

func fetchShares() (map[string]int, error) {
	url := "http://web-graffiti-gluetun:5554/api/v0/shares/contents"

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("Error creating request: %v\n", err)
	}

	for {
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("Error making GET request: %v\n", err)
			time.Sleep(1*time.Second)
		} else {
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, fmt.Errorf("Error reading response body: %v\n", err)
			}

			var directoryShares []DirectoryShare
			err = json.Unmarshal(body, &directoryShares)
			directorySize := make(map[string]int)
			if err != nil {
				return nil, fmt.Errorf("Error parsing response body: %v\n", err)
			} else {
				fmt.Printf("Found %d Directories\n", len(directoryShares))
				for _, directoryShare := range directoryShares {
					correctedDirectoryName := fmt.Sprintf("/%v", strings.ReplaceAll(directoryShare.Name, "\\", "/"))
					for _, fileShare := range directoryShare.Files {
						directorySize[correctedDirectoryName] += fileShare.Size
					}
				}
				return directorySize, nil
			}
		}
	}

}

func removeDirectories(directoryFrequency map[string]int, directorySize map[string]int, removeChunkSize int) {
	directories := slices.Collect(maps.Keys(directorySize))
	sort.Slice(directories, func(i, j int) bool {
		return directoryFrequency[directories[i]] < directoryFrequency[directories[j]]
	})

	numDirs := len(directories)
	fmt.Println("Most Popular Uploads:")
	for i := 1; i < 6 && i <= numDirs; i++ {
		currentDir := directories[numDirs - i]
		fmt.Printf("- %s: %d uploads", currentDir, directoryFrequency[currentDir])
	}

	removedSize := 0
	for _, directory := range directories {
		if directory[:12] != "/storage/tv/"{
			os.Remove(directory)
			removedSize += directorySize[directory]
			fmt.Printf("Removing %s with %d uploads", directory, directoryFrequency[directory])
			if removedSize > removeChunkSize {
				return
			}
		}
	}
}