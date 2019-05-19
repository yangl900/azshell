package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/docker/docker/pkg/term"
)

var (
	resourceURI = "https://management.azure.com/providers/Microsoft.Portal/consoles/default?api-version=2018-10-01"
	settingsURI = "https://management.azure.com/providers/Microsoft.Portal/userSettings/cloudconsole?api-version=2018-10-01"
	userAgent   = "github.com/yangl900/azshell"
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

// Terminal is the cloud shell terminal
type Terminal struct {
	SocketURI string `json:"socketUri"`
	ID        string `json:"id"`
	BaseURI   string
	TenantID  string
}

// CloudShellSettings is the cloud shell settings
type CloudShellSettings struct {
	Properties *CloudShellSettingProperties `json:"properties"`
}

// CloudShellSettingProperties is the properties of cloud shell setting
type CloudShellSettingProperties struct {
	PreferredLocation  string          `json:"preferredLoction"`
	StorageProfile     *StorageProfile `json:"storageProfile"`
	PreferredShellType string          `json:"preferredShellType"`
}

// StorageProfile is the user's storage profile
type StorageProfile struct {
	StorageAccountResourceID string `json:"storageAccountResourceId"`
	FileShareName            string `json:"fileShareName"`
	DiskSizeInGB             int    `json:"diskSizeInGB"`
}

// ReadCloudShellUserSettings read the user settings of cloud shell
func ReadCloudShellUserSettings(tenantID string) (*CloudShellSettings, error) {
	client := &http.Client{}
	req, _ := http.NewRequest("GET", settingsURI, nil)

	token, err := acquireAuthToken(tenantID)
	if err != nil {
		return nil, errors.New("Failed to acquire auth token: " + err.Error())
	}

	req.Header.Set("Authorization", token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", userAgent)

	response, err := client.Do(req)
	if err != nil {
		return nil, errors.New("Request failed. Failed to read user settings: " + err.Error())
	}

	defer response.Body.Close()
	buf, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, errors.New("Request failed. Failed to read user settings: " + err.Error())
	}

	resp := CloudShellSettings{}
	json.Unmarshal(buf, &resp)
	return &resp, nil
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
	req.Header.Set("User-Agent", userAgent)

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

// Resize resizes a terminal
func (t *Terminal) Resize(size *term.Winsize) error {
	requestURI := fmt.Sprintf("%s/terminals/%s/size?cols=%d&rows=%d&version=2019-01-01", t.BaseURI, t.ID, size.Width, size.Height)
	client := &http.Client{}
	req, _ := http.NewRequest("POST", requestURI, bytes.NewReader([]byte("")))

	token, err := acquireAuthToken(t.TenantID)
	if err != nil {
		return errors.New("Failed to acquire auth token: " + err.Error())
	}

	req.Header.Set("Authorization", token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", userAgent)

	_, err = client.Do(req)
	if err != nil {
		return errors.New("Request failed: " + err.Error())
	}

	return nil
}

// RequestTerminal request a terminal in cloud shell instance
func RequestTerminal(tenantID, URI, shellType string) (*Terminal, error) {
	requestURI := URI + "/terminals?cols=120&rows=80&version=2019-01-01&shell=" + shellType
	client := &http.Client{}
	req, _ := http.NewRequest("POST", requestURI, bytes.NewReader([]byte("")))

	token, err := acquireAuthToken(tenantID)
	if err != nil {
		return nil, errors.New("Failed to acquire auth token: " + err.Error())
	}

	log.Printf("Connecting terminal (%s)...", shellType)

	req.Header.Set("Authorization", token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", userAgent)

	response, err := client.Do(req)
	if err != nil {
		return nil, errors.New("Request failed: " + err.Error())
	}

	defer response.Body.Close()
	buf, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, errors.New("Request failed: " + err.Error())
	}

	t := &Terminal{BaseURI: URI, TenantID: tenantID}
	json.Unmarshal(buf, t)

	return t, nil
}
