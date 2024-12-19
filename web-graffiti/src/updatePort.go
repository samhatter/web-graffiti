package main

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

func updatePort() {
	waitTime, err := strconv.Atoi(os.Getenv("UPDATE_PORT_TIMER"))
	if err != nil {
		 fmt.Printf("Error Reading QUEUE_DOWNLOADS_TIMER%v\n", err)
		 return
	}

	for {
		port, err := fetchPort()
		if err != nil {
			fmt.Printf("Error fetching port: %v\n", err)
		} else {
			err = setPort(port)
		}
		time.Sleep(time.Duration(waitTime) * time.Minute)
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
		if strings.Contains(line, "listen_port") {
			newLine := fmt.Sprintf("  listen_port: %s", port)
			if line != newLine {
				line = newLine
				changed = true
			}
		}
		lines = append(lines, line)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("Error reading file: %v\n", err)
	}

	if !changed {
		fmt.Println("Port Unchanged")
		return nil
	}

	err = os.WriteFile(filePath, []byte(strings.Join(lines, "\n")+"\n"), 0644)
	if err != nil {
		return fmt.Errorf("Error writing to file: %v\n", err)
	}

	url := "http://web-graffiti-gluetun:5554/api/v0/application"
	req, err := http.NewRequest(http.MethodPut, url, nil)
	if err != nil {
		return fmt.Errorf("Error creating request: %v\n", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("Error making PUT request: %v\n", err)
	}
	defer resp.Body.Close()

	_, err = io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("Error reading response body: %v\n", err)
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("Error Restarting slskd: Status Code %d\n", resp.StatusCode)
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