package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
)

func randomizeKeys(inputMap map[string][]SearchFile) []string {
	keys := make([]string, 0, len(inputMap))
	for key := range inputMap {
		keys = append(keys, key)
	}
	
	rand.Shuffle(len(keys), func(i, j int) {
		keys[i], keys[j] = keys[j], keys[i]
	})

	return keys
}

func scanShares() {
	resp, err := send(http.MethodPut, "http://web-graffiti-gluetun:5554", "api/v0/shares", nil)
	if err != nil {
		fmt.Printf("Error Scanning Shares: %v\n", err)
	}
	resp.Body.Close()
}



func send(method string, base string, endpoint string, data any) (*http.Response, error){
	url := fmt.Sprintf("%s/%s", base, endpoint)

	var req *http.Request
	if data != nil {
		jsonData, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("Error marshalling JSON: %v\n", err)
		}
		req, err = http.NewRequest(method, url, bytes.NewBuffer(jsonData))
		if err != nil {  
			return nil, fmt.Errorf("Error creating request: %v\n", err)
		}
		req.Header.Set("Content-Type", "application/json")
	} else {
		var err error
		req, err = http.NewRequest(method, url, nil)
		if err != nil {  
			return nil, fmt.Errorf("Error creating request: %v\n", err)
		}
	}
		

	client := &http.Client{}
	
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Error making %s request: %v\n", method, err)
	} 

	return resp, nil
}