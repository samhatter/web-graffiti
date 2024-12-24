package main

import (
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

func queueDownloads() {
	waitTime, err := strconv.Atoi(os.Getenv("QUEUE_DOWNLOADS_TIMER"))
	if err != nil {
		fmt.Printf("Error Reading QUEUE_DOWNLOADS_TIMER%v\n", err)
		waitTime = 24
	}
	fmt.Printf("QUEUE_DOWNLOADS_TIMER: %d\n", waitTime)

	for {
		fmt.Println("Searching for Downloads")
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

		time.Sleep(time.Duration(waitTime) * time.Hour)
	}
}

func startDownloads(result []SearchUserFiles)(error) {
	storageSize := getStorageSize()
	fmt.Printf("Current Storage Size: %d\n", storageSize)
	
	GiB := 1024*1024*1024
	targetSize, err := strconv.Atoi(os.Getenv("TARGET_SIZE"))
	if err != nil {
		return fmt.Errorf("Error Reading TARGET_SIZE%v\n", err)
	}
	targetSize = targetSize*GiB
	fmt.Printf("Target Storage Size: %d\n", targetSize)

	chunkSize, err := strconv.Atoi(os.Getenv("DOWNLOAD_CHUNK_SIZE"))
	if err != nil {
		fmt.Printf("Error Reading DOWNLOAD_CHUNK_SIZE%v\n", err)
		chunkSize = 1
	}
	chunkSize = chunkSize*GiB
	fmt.Printf("Chunk Size: %d\n", chunkSize)

	maxFoldersPerUser, err := strconv.Atoi(os.Getenv("MAX_FOLDERS_PER_USER"))
	if err != nil {
		fmt.Printf("Error Reading MAX_FOLDERS_PER_USER%v\n", err)
		maxFoldersPerUser = 1
	}
	fmt.Printf("Max Folders Per User: %d\n", maxFoldersPerUser)

	downloadedSize := 0
	for _, userFiles := range result {
		files := userFiles.Files
		folderMap := groupSearchesByFolder(files)
		foldersDownloaded := 0
		keys := randomizeKeys(folderMap)
		for _, folderName := range keys {
			folder := folderMap[folderName]
			folderSize := 0
			for _, file := range folder {
					folderSize += file.Size
			}
			if len(folder) > 5 && (int64(folderSize) < (int64(chunkSize) - int64(downloadedSize)) && int64(folderSize) < (int64(targetSize) - int64(storageSize) - int64(downloadedSize)) && foldersDownloaded < maxFoldersPerUser) {
				foldersDownloaded += 1
				fmt.Printf("Queueing Folder, User:%s, Folder:%s\n", userFiles.UserName, folderName)
				var requests []DownloadRequest
				for _, file := range folder {
					requests = append(requests, DownloadRequest{
						FileName: file.FileName,
						Size: file.Size,
					})
				}
				resp, err := send(http.MethodPost, "http://web-graffiti-gluetun:5554", fmt.Sprintf("api/v0/transfers/downloads/%s", userFiles.UserName), requests)
				defer resp.Body.Close()
				if err != nil {
					fmt.Printf("Error Queueing Download: %v\n", err)
				} else if resp.StatusCode != 201{
					fmt.Printf("Could not queue download status code: %d\n", resp.StatusCode)
				} else {
					downloadedSize += folderSize
				}
			} 
		}
	}
	return nil
}

func groupSearchesByFolder(files []SearchFile)(map[string][]SearchFile) {
	output := make(map[string][]SearchFile)
	for _, file := range files {
		split := strings.Split(file.FileName, "\\")
		prefix := strings.Join(split[:len(split)-1], "\\")
		output[prefix] = append(output[prefix], file)
	}
	return output
}

func getSearch()([]SearchUserFiles, error){
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

func checkSearchResults(id string)([]SearchUserFiles, error){
	searchComplete := false
	for !searchComplete {
		resp, err := send(http.MethodGet, "http://web-graffiti-gluetun:5554", fmt.Sprintf("api/v0/searches/%s", id), nil)
		defer resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("Error  checking search result: %v\n", err)
		}

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

	var data []SearchUserFiles
	for data == nil {
		resp, err := send(http.MethodGet, "http://web-graffiti-gluetun:5554", fmt.Sprintf("api/v0/searches/%s/responses", id), nil)
		defer resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("Error making GET request: %v\n", err)
		}
		status := resp.StatusCode
		if status != 200 {
			fmt.Printf("Failed to Grab Search, StatusCode: %d", status)
			time.Sleep(1*time.Second)
		} else {
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, fmt.Errorf("Error Reading Response Body, %v\n", err)
			}
			var newData []SearchUserFiles
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
	data := SearchRequest{
		Id: uuid.New().String(),
		SearchText: search,
	}

	for {
		resp, err := send(http.MethodPost, "http://web-graffiti-gluetun:5554", "api/v0/searches", data)
		defer resp.Body.Close()
		if err != nil {
			fmt.Printf("Error sending search: %v\n", err)
			time.Sleep(1*time.Second)
		} else {
			return data.Id, resp.StatusCode, nil
		}
	}
}

