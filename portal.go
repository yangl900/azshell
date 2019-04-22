package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

var (
	resourceURI = "https://management.azure.com/providers/Microsoft.Portal/consoles/default?api-version=2018-10-01"
)

type consoleRequest struct {
	Properties consoleRequestProperties `json:"properties"`
}

type consoleRequestProperties struct {
	OsType string `json:"osType"`
}

type consoleResponse struct {
	Properties consoleResponseProperties `json:"properties"`
}

type consoleResponseProperties struct {
	ProvisioningState string `json:"provisioningState"`
	URI               string `json:"uri"`
}

type terminalResponse struct {
	SocketURI string `json:"socketUri"`
}

// RequestCloudShell requests a cloud shell instance
func RequestCloudShell(tenantID string) (string, error) {
	consoleReq := &consoleRequest{
		Properties: consoleRequestProperties{
			OsType: "linux",
		},
	}

	reqBody, err := json.Marshal(consoleReq)
	if err != nil {
		return "", errors.New("Failed to serialize: " + err.Error())
	}

	client := &http.Client{}
	req, _ := http.NewRequest("PUT", resourceURI, bytes.NewReader([]byte(reqBody)))

	token, err := acquireAuthToken(tenantID)
	if err != nil {
		return "", errors.New("Failed to acquire auth token: " + err.Error())
	}

	log.Printf("Requesting Cloud Shell...")

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

	resp := consoleResponse{}
	json.Unmarshal(buf, &resp)

	if strings.EqualFold(resp.Properties.ProvisioningState, "Succeeded") {
		log.Printf("Succeeded.")
	}

	return resp.Properties.URI, nil
}

// RequestTerminal request a terminal in cloud shell instance
func RequestTerminal(tenantID, URI string) (string, error) {
	requestURI := URI + "/terminals?cols=120&rows=80&version=2019-01-01&shell=bash"
	client := &http.Client{}
	req, _ := http.NewRequest("POST", requestURI, bytes.NewReader([]byte("")))

	token, err := acquireAuthToken(tenantID)
	if err != nil {
		return "", errors.New("Failed to acquire auth token: " + err.Error())
	}

	log.Printf("Connecting terminal...")

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

	resp := terminalResponse{}
	json.Unmarshal(buf, &resp)

	return resp.SocketURI, nil
}
