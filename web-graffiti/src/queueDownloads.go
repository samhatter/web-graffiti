package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

type File struct {
	Filename string `json:"filename"`
	Size int `json:"size"`
	IsLocked bool `json:"isLocked"`
}

type UserFiles struct {
	FileCount int `json:"fileCount"`
	Files []File `json:"files"`
	Username string `json:"username"`
}

type SearchStatus struct {
	IsComplete bool `json:"isComplete"`
}

type DownloadRequest struct {
	Filename string `json:"filename"`
	Size int `json:"size"`
}

func queueDownloads() {
	for {
		result, err := getSearch()
		if err != nil {
			fmt.Printf("Error Searching %v\n", err)
		}
		err = startDownloads(result)
		if err != nil {
			fmt.Printf("Error Queueing Downloads %v\n", err)
		} else {
			fmt.Println("Done Queueing Downloads")
		}

		time.Sleep(24*time.Hour)
	}
}

func startDownloads(result []UserFiles)(error) {
	size := getStorageSize()
	fmt.Printf("Current Storage Size: %d\n", size)
	targetSize, err := strconv.Atoi(os.Getenv("TARGET_SIZE"))
	if err != nil {
		return fmt.Errorf("Error Reading TARGET_SIZE%v\n", err)
	}
	targetSize = targetSize*1024*1024*1024
	fmt.Printf("Target Storage Size: %d\n", targetSize)
	for _, userFiles := range result {
		files := userFiles.Files
		folderMap := groupPathsByFolder(files)
		foldersDownloaded := 0
		for folderName, folder := range folderMap {
			folderSize := 0
			for _, file := range folder {
					folderSize += file.Size
			}
			if len(folder) > 5 && (int64(folderSize) < (int64(targetSize) - size) && foldersDownloaded < 5) {
				foldersDownloaded += 1
				fmt.Printf("Queueing Folder, User:%s, Folder:%s\n", userFiles.Username, folderName)
				var requests []DownloadRequest
				for _, file := range folder {
					requests = append(requests, DownloadRequest{
						Filename: file.Filename,
						Size: file.Size,
					})
				}
				jsonData, err := json.Marshal(requests)
				if err != nil {
					return fmt.Errorf("Error marshalling JSON: %v\n", err)
				}
				url := fmt.Sprintf("http://web-graffiti-gluetun:5554/api/v0/transfers/downloads/%s", userFiles.Username)
				req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonData))
				if err != nil {
					return fmt.Errorf("Error creating request: %v\n", err)
				}
				req.Header.Set("Content-Type", "application/json")
			
				client := &http.Client{}
				resp, err := client.Do(req)
				if err != nil {
					return fmt.Errorf("Error making POST request: %v\n", err)
				} else if resp.StatusCode != 201{
					fmt.Printf("Could not queue download status code: %d\n", resp.StatusCode)
				} else {
					size += int64(folderSize)
				}
			} 
		}
	}
	return nil
}

func groupPathsByFolder(files []File)(map[string][]File) {
	output := make(map[string][]File)
	for _, file := range files {
		split := strings.Split(file.Filename, "\\")
		prefix := strings.Join(split[:len(split)-1], "\\")
		output[prefix] = append(output[prefix], file)
	}
	return output
}

func getSearch()([]UserFiles, error){
	fmt.Println("Calling Catch and Release")
	
	id, status, err := sendSearch("flac")
	if err != nil {
		return nil, fmt.Errorf("Error Sending Search %v\n", err)
	} else if status != 200 {
		return nil, fmt.Errorf("Error Sending Search: Status %d\n", status)
	} else {
		results, err := checkSearchResults(id)
		if err != nil {
			return nil, fmt.Errorf("Error Checking Results %v\n", err)
		} else {
			return results, nil
		}
	}
}

func checkSearchResults(id string)([]UserFiles, error){
	searchComplete := false
	url := fmt.Sprintf("http://web-graffiti-gluetun:5554/api/v0/searches/%s", id)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("Error creating request: %v\n", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for !searchComplete {
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("Error making GET request: %v\n", err)
		}

		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("Error reading response body: %v\n", err)
		}
		
		var data SearchStatus
		err = json.Unmarshal(body, &data)
		if err != nil {
			return nil, fmt.Errorf("Error parsing response body: %v\n", err)
		}

		searchComplete = data.IsComplete
		fmt.Println("Waiting For Search")
		time.Sleep(1*time.Second)
	}

	var data []UserFiles
	url = fmt.Sprintf("http://web-graffiti-gluetun:5554/api/v0/searches/%s/responses", id)
	req, err = http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("Error creating request: %v\n", err)
	}
	req.Header.Set("Content-Type", "application/json")

	for data == nil {
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("Error making GET request: %v\n", err)
		}
		defer resp.Body.Close()
		status := resp.StatusCode
		if status != 200 {
			fmt.Printf("Failed to Grab Search, StatusCode: %d", status)
			time.Sleep(1*time.Second)
		} else {
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, fmt.Errorf("Error Reading Response Body, %v\n", err)
			}
			var newData []UserFiles
			err = json.Unmarshal(body, &data)
			if err != nil {
				return nil, fmt.Errorf("error parsing response body: %v\n", err)
			}
			if len(newData) != 0 {
				data = newData
			}
		}
	}
	fmt.Printf("Done Searching, %d responses\n", len(data))
	return data, nil
}

func getStorageSize()(int64){
	var size int64
	path := "/storage"
	filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("Couldn't Get Directory Size: %v\n", err)
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})

	return size
}

func sendSearch(search string)(string, int, error) {
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
		return "", 0, fmt.Errorf("Error marshalling JSON: %v\n", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", 0, fmt.Errorf("Error creating request: %v\n", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("Error making POST request: %v\n", err)
	}

	return data.Id, resp.StatusCode, nil
}