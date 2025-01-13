package main

import (
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

func fetchActiveDownloads() ([]UserTransfer, error) {
	for {
		resp, err := send(http.MethodGet, "http://web-graffiti-gluetun:5554", "api/v0/transfers/downloads", nil)
		if err != nil {
			fmt.Printf("Error fetching downloads: %v\n", err)
			time.Sleep(1*time.Second)
		} else {
			body, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				return nil, fmt.Errorf("Error reading response body: %v\n", err)
			}

			var data []UserTransfer
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

func processUserDownloads (userDownloads []UserTransfer, downloadTracker map[string]time.Time) () {
	for _, userDownload := range userDownloads {
		for _, directoryDownload := range userDownload.Directories {
			allFilesDownloaded := true
			newDownloads := true
			for _, fileDownload := range directoryDownload.Files {
				if fileDownload.State == "Completed, Errored" || fileDownload.State == "Completed, Cancelled" {
					fmt.Printf("Retrying Download: %s\n", fileDownload.FileName)
					var err error
					newDownloads, err = retryDownload(fileDownload, downloadTracker)
					if err != nil {
						fmt.Printf("Error retrying download: %v\n", err)
					}
					allFilesDownloaded = false

					if !newDownloads {
						clearDownload(directoryDownload, downloadTracker)
						break
					}
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

func retryDownload(fileDownload FileTransfer, downloadTracker map[string]time.Time) (bool, error) {
	maxDownloadTime, err := strconv.Atoi(os.Getenv("MAX_DOWNLOAD_TIME"))
	if err != nil {
		fmt.Printf("Error Reading MAX_DOWNLOAD_TIME%v\n", err)
		maxDownloadTime = 12
	}

	layout := "2006-01-02T15:04:05.9999999"
	parsedTime, err := time.Parse(layout, fileDownload.RequestedAt)

	if value, exists := downloadTracker[fileDownload.FileName]; exists {
		if time.Duration(maxDownloadTime)*time.Hour < time.Now().Sub(value) {
			return false, nil
		}
	} else {
		downloadTracker[fileDownload.FileName] = parsedTime
	}

	var requests []DownloadRequest
	requests = append(requests, DownloadRequest{
		FileName: fileDownload.FileName,
		Size: fileDownload.Size,
	})

	resp, err := send(http.MethodGet, "http://web-graffiti-gluetun:5554", fmt.Sprintf("api/v0/transfers/downloads/%s", fileDownload.UserName), requests)

	if err != nil {
		return false, fmt.Errorf("Error fetching downloads: %v\n", err)
	} else {
		resp.Body.Close()
	}

	if resp.StatusCode != 201 && resp.StatusCode != 200 {
		return false, fmt.Errorf("Could not queue download status code: %d\n", resp.StatusCode)
	}

	return true, nil
}

func addMetaData(file FileTransfer) {
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

func clearDownload(directoryDownload DirectoryTransfer, downloadTracker map[string]time.Time)  {
	for _, fileDownload := range directoryDownload.Files {
		for {
			resp, err := send(http.MethodDelete, "http://web-graffiti-gluetun:5554", fmt.Sprintf("api/v0/transfers/downloads/%s/%s?remove=true", fileDownload.UserName, fileDownload.Id), nil)
			if err != nil {
				fmt.Printf("Error clearing download: %v\n", err)
			} else {
				resp.Body.Close()
			}

			if resp.StatusCode != 204{
				fmt.Printf("Could not remove download. Status code: %d\n", resp.StatusCode)
				break
			} else {
				break
			}
		}
	}

	downloadDir := os.Getenv("SLSKD_DOWNLOADS_DIR")
	cleanPath := filepath.Clean(strings.ReplaceAll(directoryDownload.Directory, "\\", "/"))
	src := filepath.Join(downloadDir, filepath.Base(cleanPath))
	dst := filepath.Join("/storage",  filepath.Base(cleanPath))

	err := os.Rename(src, dst)
	if err != nil {
		fmt.Printf("Failed to move dir: %v\n", err)
		err = os.RemoveAll(src)
		if err != nil {
			fmt.Printf("Failed to remove dir: %v\n", err)
		}
	}
	
	for _, fileDownload := range directoryDownload.Files {
		delete(downloadTracker, fileDownload.FileName)
	}
}