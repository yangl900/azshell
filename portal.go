package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
)

var (
	resourceURI = "https://management.azure.com/providers/Microsoft.Portal/consoles/default?api-version=2018-10-01"
)

type consoleRequest struct {
	properties consoleRequestProperties `json:"properties"`
}

type consoleRequestProperties struct {
	osType string `json:"osType"`
}

type consoleResponse struct {
	properties consoleResponseProperties `json:"properties"`
}

type consoleResponseProperties struct {
	provisioningState string `json:"provisioningState"`
	uri               string `json:"uri"`
}

func RequestCloudShell() (string, error) {
	consoleReq := &consoleRequest{
		properties: consoleRequestProperties{
			osType: "linux",
		},
	}

	reqBody, err := json.Marshal(consoleReq)
	if err != nil {
		return "", errors.New("Failed to serialize: " + err.Error())
	}

	client := &http.Client{}
	req, _ := http.NewRequest("PUT", resourceURI, bytes.NewReader([]byte(reqBody)))

	token, err := acquireAuthTokenCurrentTenant()
	if err != nil {
		return "", errors.New("Failed to acquire auth token: " + err.Error())
	}

	req.Header.Set("Authorization", token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	response, err := client.Do(req)
	if err != nil {
		return "", errors.New("Request failed: " + err.Error())
	}

	defer response.Body.Close()
	buf, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", errors.New("Request failed: " + err.Error())
	}

	log.Printf("Request console response: %s", string(buf))

	resp := consoleResponse{}
	json.Unmarshal(buf, &resp)

	return resp.properties.uri, nil
}
