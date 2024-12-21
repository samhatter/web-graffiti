package main

import (
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