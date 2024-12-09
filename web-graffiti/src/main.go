package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
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

func catchAndRelease(){
	for {
		fmt.Println("Calling Catch and Release")

		size := getStorageSize()
		
		status, _ := sendSearch("flac")

		fmt.Printf("size: %d, results: %d\n", size, status)

		time.Sleep(5*time.Second)
	}
}

func updatePort() {
	for {
		port, err := fetchPort()
		if err != nil {
			fmt.Printf("Error fetching port: %v\n", err)
		} else {
			err = setPort(port)
		}
		time.Sleep(30 * time.Minute)
	}
}

func setPort(port string) (error){
	filePath := "/slskd/slskd.yml"
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		return err
	}
	defer file.Close()

	var lines []string

	scanner := bufio.NewScanner(file)
	changed := false
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "listenPort") {
			newLine := fmt.Sprintf("  listenPort: %s", port)
			if line != newLine {
				line = newLine
				changed = true
			}
		}
		lines = append(lines, line)
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		return err
	}

	if !changed {
		fmt.Println("Port Unchanged")
		return nil
	}

	err = os.WriteFile(filePath, []byte(strings.Join(lines, "\n")+"\n"), 0644)
	if err != nil {
		fmt.Printf("Error writing to file: %v\n", err)
		return err
	}

	url := "http://web-graffiti-gluetun:5554/api/v0/application"
	req, err := http.NewRequest(http.MethodPut, url, nil)
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		return err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error making PUT request: %v\n", err)
		return err
	}
	defer resp.Body.Close()

	_, err = io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response body: %v\n", err)
		return err
	}

	if resp.StatusCode != 200 {
		err = fmt.Errorf("Error Restarting slskd: Status Code %d\n", resp.StatusCode)
		fmt.Printf("Error Restarting slskd: Status Code %d\n", resp.StatusCode)
		return err
	}

	fmt.Printf("Port Successfully Changed to %s\n", port)
	return nil

}

func fetchPort() (string, error) {
	url := "http://web-graffiti-gluetun:8000/v1/openvpn/portforwarded"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("error creating request: %v\n", err)
	}

	username := os.Getenv("GLUETUN_USERNAME")
	password :=  os.Getenv("GLUETUN_PASSWORD")
	auth := username + ":" + password
	encodedAuth := base64.StdEncoding.EncodeToString([]byte(auth))
	req.Header.Add("Authorization", "Basic "+encodedAuth)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error sending request: %v\n", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %v\n", err)
	}

	type PortResponse struct {
		Port int `json:"port"`
	}

	var data PortResponse

	err = json.Unmarshal(body, &data)
	if err != nil {
		return "", fmt.Errorf("error parsing response body: %v\n", err)
	}

	return strconv.Itoa(data.Port), nil
}

func main() {
	fmt.Println("Web-Graffiti Initializing...")
	go catchAndRelease()
	go updatePort()

	select {}
}