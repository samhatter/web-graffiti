package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type UserDownload struct {
	Username string `json:"username"`
	Directories []DirectoryDownload `json:"directories"`
}

type DirectoryDownload struct {
	Directory string `json:"directory"`
	FileCount int `json:"fileCount"`
	Files []FileDownload `json:"files"`
}

type FileDownload struct {
	Id string `json:"id"`
	FileName string `json:"filename"`
	State string `json:"state"`
	Size int `json:"size"`
	RequestedAt string `json:"requestedAt"`
	UserName string `json:"username"`
}

func processDownloads() {
	waitTime, err := strconv.Atoi(os.Getenv("PROCESS_DOWNLOADS_TIMER"))
	if err != nil {
		fmt.Printf("Error Reading PROCESS_DOWNLOADS_TIMER%v\n", err)
		waitTime = 24
	}
	fmt.Printf("PROCESS_DOWNLOADS_TIMER: %d\n", waitTime)

	downloadTracker := make(map[string]time.Time)

	for {
		fmt.Println("Processing Downloads...")
		userDownloads, err := fetchActiveDownloads()
		if err != nil {
			fmt.Printf("Error fetching active downloads: %v\n", err)
		} else {
			processUserDownloads(userDownloads, downloadTracker)
		}

		scanShares()
		fmt.Println("Done Processing Downloads")
		fmt.Printf("Waiting on %d files\n", len(downloadTracker))
		time.Sleep(time.Duration(waitTime) * time.Minute)
	}
}

func fetchActiveDownloads() ([]UserDownload, error) {
	url := "http://web-graffiti-gluetun:5554/api/v0/transfers/downloads"

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

			var data []UserDownload
			err = json.Unmarshal(body, &data)
			if err != nil {
				return nil, fmt.Errorf("Error parsing response body: %v\n", err)
			} else {
				fmt.Printf("Found %d Downloads\n", len(data))
				return data, nil
			}
		}
	}

}

func processUserDownloads(userDownloads []UserDownload, downloadTracker map[string]time.Time) () {
	for _, userDownload := range userDownloads {
		for _, directoryDownload := range userDownload.Directories {
			allFilesDownloaded := true
			newDownloads := true
			for _, fileDownload := range directoryDownload.Files {
				if fileDownload.State == "Completed, Errored" || fileDownload.State == "Completed, Cancelled" {
					fmt.Printf("Retrying Download: %s\n", fileDownload.FileName)
					var err error
					newDownloads, err = retryDownload(fileDownload, directoryDownload, downloadTracker)
					if err != nil {
						fmt.Printf("Error retrying download: %v\n", err)
					}
					allFilesDownloaded = false
				} else if fileDownload.State != "Completed, Succeeded" {
					allFilesDownloaded = false
				}
			}

			if allFilesDownloaded && newDownloads {
				fmt.Printf("Folder Downloaded: %s\n", directoryDownload.Directory)

				for _, fileDownload := range directoryDownload.Files {
					addMetaData(fileDownload)
				}

				clearDownload(directoryDownload, downloadTracker)
			}
		}
	}
}

func retryDownload(fileDownload FileDownload, directoryDownload DirectoryDownload, downloadTracker map[string]time.Time) (bool, error) {
	maxDownloadTime, err := strconv.Atoi(os.Getenv("MAX_DOWNLOAD_TIME"))
	if err != nil {
		fmt.Printf("Error Reading MAX_DOWNLOAD_TIME%v\n", err)
		maxDownloadTime = 12
	}

	layout := "2006-01-02T15:04:05.9999999"
	parsedTime, err := time.Parse(layout, fileDownload.RequestedAt)

	if value, exists := downloadTracker[fileDownload.FileName]; exists && time.Duration(maxDownloadTime)*time.Hour < time.Now().Sub(value) {
		clearDownload(directoryDownload, downloadTracker)
		return false, nil
	} else {
		downloadTracker[fileDownload.FileName] = parsedTime
	}

	var requests []DownloadRequest
	requests = append(requests, DownloadRequest{
		FileName: fileDownload.FileName,
		Size: fileDownload.Size,
	})

	jsonData, err := json.Marshal(requests)
	if err != nil {
		return false, fmt.Errorf("Error marshalling JSON: %v\n", err)
	}

	url := fmt.Sprintf("http://web-graffiti-gluetun:5554/api/v0/transfers/downloads/%s", fileDownload.UserName)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return false, fmt.Errorf("Error creating request: %v\n", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("Error making POST request: %v\n", err)
	} else if resp.StatusCode != 201{
		return false, fmt.Errorf("Could not queue download status code: %d\n", resp.StatusCode)
	}

	return true, nil
}

func addMetaData(file FileDownload) {
	secretMessage := os.Getenv("SECRET_MESSAGE")
	downloadDir := os.Getenv("SLSKD_DOWNLOADS_DIR")

	cleanPath := filepath.Clean( strings.ReplaceAll(file.FileName, "\\", "/"))
	lastPart := filepath.Base(cleanPath)
	parentDir := filepath.Dir(cleanPath)
	path := filepath.Join(downloadDir, filepath.Base(parentDir), lastPart)

	extension := filepath.Ext(path)
	base := strings.TrimSuffix(path, extension)
	tmpFile := fmt.Sprintf("%s_tmp%s", base, extension)

	cmd := exec.Command("ffmpeg", "-i", path, "-metadata", fmt.Sprintf("secret=%s", secretMessage), "-c", "copy", "-y", tmpFile)
	if err := cmd.Run(); err != nil {
		fmt.Printf("ffmpeg failed: %v\n", err)
	} else {
		err := os.Rename(tmpFile, path)
		if err != nil {
			fmt.Printf("Failed to move file: %v\n", err)
		}
	}
}

func clearDownload(directoryDownload DirectoryDownload, downloadTracker map[string]time.Time)  {
	for _, fileDownload := range directoryDownload.Files {
		for {
			url := fmt.Sprintf("http://web-graffiti-gluetun:5554/api/v0/transfers/downloads/%s/%s?remove=true", fileDownload.UserName, fileDownload.Id)
		
			req, err := http.NewRequest(http.MethodDelete, url, nil)
			if err != nil {  
				fmt.Printf("Error creating request: %v\n", err)
			}

			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				fmt.Printf("Error making DELETE request: %v\n", err)
			} else if resp.StatusCode != 204{
				fmt.Printf("Could not remove download. Status code: %d\n", resp.StatusCode)
				break
			} else {
				break
			}
			time.Sleep(1)
		}
	}

	downloadDir := os.Getenv("SLSKD_DOWNLOADS_DIR")
	cleanPath := filepath.Clean(strings.ReplaceAll(directoryDownload.Directory, "\\", "/"))
	src := filepath.Join(downloadDir, filepath.Base(cleanPath))
	dst := filepath.Join("/storage",  filepath.Base(cleanPath))

	err := os.Rename(src, dst)
	if err != nil {
		fmt.Printf("Failed to move dir: %v\n", err)
	}
	
	for _, fileDownload := range directoryDownload.Files {
		delete(downloadTracker, fileDownload.FileName)
	}
}

func scanShares() {
	url := "http://web-graffiti-gluetun:5554/api/v0/shares"
		
	req, err := http.NewRequest(http.MethodPut, url, nil)
	if err != nil {  
		fmt.Printf("Error creating request: %v\n", err)
	}

	client := &http.Client{}
	
	_, err = client.Do(req)
	if err != nil {
		fmt.Printf("Error making PUT request: %v\n", err)
	}
}