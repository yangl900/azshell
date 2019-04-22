package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"strings"

	"github.com/Azure/go-autorest/autorest/adal"
)

const (
	activeDirectoryEndpoint = "https://login.microsoftonline.com/"
	armResource             = "https://management.core.windows.net/"
	clientAppID             = "aebc6443-996d-45c2-90f0-388ff96faa56"
	commonTenant            = "common"
)

type responseJSON struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	Resource     string `json:"resource"`
	TokenType    string `json:"token_type"`
}

type tenant struct {
	ID          string `json:"id"`
	TenantID    string `json:"tenantId"`
	ContryCode  string `json:"countryCode"`
	DisplayName string `json:"displayName"`
}

type tenantList struct {
	Value []tenant `json:"value"`
}

func defaultTokenCachePath(tenant string) string {
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}

	return fmt.Sprintf("%s/.armclient/accessToken.%s.json", usr.HomeDir, strings.ToLower(tenant))
}

func acquireTokenDeviceCodeFlow(oauthConfig adal.OAuthConfig,
	applicationID string,
	resource string,
	callbacks ...adal.TokenRefreshCallback) (*adal.ServicePrincipalToken, error) {

	oauthClient := &http.Client{}
	deviceCode, err := adal.InitiateDeviceAuth(
		oauthClient,
		oauthConfig,
		applicationID,
		resource)
	if err != nil {
		return nil, fmt.Errorf("Failed to start device auth flow: %s", err)
	}

	fmt.Println(*deviceCode.Message)

	token, err := adal.WaitForUserCompletion(oauthClient, deviceCode)
	if err != nil {
		return nil, fmt.Errorf("Failed to finish device auth flow: %s", err)
	}

	spt, err := adal.NewServicePrincipalTokenFromManualToken(
		oauthConfig,
		applicationID,
		resource,
		*token,
		callbacks...)
	return spt, err
}

func refreshToken(oauthConfig adal.OAuthConfig,
	applicationID string,
	resource string,
	tokenCachePath string,
	callbacks ...adal.TokenRefreshCallback) (*adal.ServicePrincipalToken, error) {

	token, err := adal.LoadToken(tokenCachePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load token from cache: %v", err)
	}

	spt, err := adal.NewServicePrincipalTokenFromManualToken(
		oauthConfig,
		applicationID,
		resource,
		*token,
		callbacks...)
	if err != nil {
		return nil, err
	}
	return spt, spt.Refresh()
}

func saveToken(spt adal.Token, tenant string) error {
	err := adal.SaveToken(defaultTokenCachePath(tenant), 0600, spt)
	if err != nil {
		return err
	}

	return nil
}

func getTenants(commonTenantToken string) (ret []tenant, e error) {
	url, err := getRequestURL("/tenants?api-version=2018-01-01")
	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	req, _ := http.NewRequest(http.MethodGet, url, nil)

	req.Header.Set("Authorization", commonTenantToken)
	req.Header.Set("User-Agent", "yangl/prototype")
	req.Header.Set("Accept", "application/json")

	response, err := client.Do(req)
	if err != nil {
		return nil, errors.New("Failed to list tenants: " + err.Error())
	}

	defer response.Body.Close()
	buf, err := ioutil.ReadAll(response.Body)

	var tenants tenantList
	json.Unmarshal(buf, &tenants)

	return tenants.Value, nil
}

func acquireAuthTokenDeviceFlow(tenantID string) (string, error) {
	oauthConfig, err := adal.NewOAuthConfig(activeDirectoryEndpoint, tenantID)
	if err != nil {
		panic(err)
	}

	callback := func(token adal.Token) error {
		return saveToken(token, tenantID)
	}

	if _, err := os.Stat(defaultTokenCachePath(tenantID)); err == nil {
		token, err := adal.LoadToken(defaultTokenCachePath(tenantID))
		if err != nil {
			return "", err
		}

		var spt *adal.ServicePrincipalToken
		if token.IsExpired() {
			spt, err = refreshToken(*oauthConfig, clientAppID, armResource, defaultTokenCachePath(tenantID), callback)
			if err == nil {
				return fmt.Sprintf("%s %s", spt.Token().Type, spt.Token().AccessToken), nil
			}
		} else {
			return fmt.Sprintf("%s %s", token.Type, token.AccessToken), nil
		}
	}

	if tenantID != commonTenant {
		_, err := acquireAuthTokenDeviceFlow(commonTenant)
		if err != nil {
			return "", err
		}

		spt, err := refreshToken(*oauthConfig, clientAppID, armResource, defaultTokenCachePath(commonTenant), callback)
		if err != nil {
			return "", err
		}

		return fmt.Sprintf("%s %s", spt.Token().Type, spt.Token().AccessToken), nil
	}

	var spt *adal.ServicePrincipalToken
	spt, err = acquireTokenDeviceCodeFlow(
		*oauthConfig,
		clientAppID,
		armResource,
		callback)

	if err == nil {
		saveToken(spt.Token(), tenantID)
	}

	return fmt.Sprintf("%s %s", spt.Token().Type, spt.Token().AccessToken), nil
}

func acquireAuthTokenMSI(endpoint string) (string, error) {
	msiendpoint, _ := url.Parse(endpoint)

	parameters := url.Values{}
	parameters.Add("resource", armResource)

	msiendpoint.RawQuery = parameters.Encode()

	req, err := http.NewRequest("GET", msiendpoint.String(), nil)
	if err != nil {
		return "", err
	}

	req.Header.Add("Metadata", "true")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}

	responseBytes, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return "", err
	}

	var r responseJSON
	err = json.Unmarshal(responseBytes, &r)
	if err != nil {
		return "", err
	}

	return r.TokenType + " " + r.AccessToken, nil
}

func acquireBootstrapToken() (string, error) {
	endpoint, hasMsiEndpoint := os.LookupEnv("MSI_ENDPOINT")

	if hasMsiEndpoint {
		token, err := acquireAuthTokenMSI(endpoint)
		if err != nil {
			return "", err
		}

		return token, nil
	}

	return acquireAuthTokenDeviceFlow(commonTenant)
}

func acquireAuthTokenCurrentTenant() (string, error) {
	userSettings, err := readSettings()
	if err != nil {
		return "", fmt.Errorf("Failed to read current tennat: %v", err)
	}

	tenantID := userSettings.ActiveTenant
	if tenantID == "" {
		token, err := acquireBootstrapToken()
		if err != nil {
			return "", err
		}

		tenants, err := getTenants(token)
		if err != nil {
			return "", errors.New("Failed to list tenants: " + err.Error())
		}

		if len(tenants) == 0 {
			return "", fmt.Errorf("You don't have access to any tenants (directory)")
		}

		userSettings.ActiveTenant = tenants[0].TenantID
		tenantID = tenants[0].TenantID
		saveSettings(userSettings)
	}

	return acquireAuthToken(tenantID)
}

func acquireAuthToken(tenantID string) (string, error) {
	endpoint, hasMsiEndpoint := os.LookupEnv("MSI_ENDPOINT")

	if hasMsiEndpoint {
		token, err := acquireAuthTokenMSI(endpoint)
		if err != nil {
			return "", err
		}

		return token, nil
	}

	if tenantID == "" {
		panic(fmt.Errorf("Tenant ID required for acquire token"))
	}

	token, err := acquireAuthTokenDeviceFlow(tenantID)
	if err != nil {
		log.Println("Failed to login to tenant: ", tenantID)
	}

	return token, nil
}
