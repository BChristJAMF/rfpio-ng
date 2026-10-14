package core

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	defaultTimeout = 30 * time.Second
	userAgent      = "rfpio-cli-go/1.0"
)

// sslContext creates an HTTP client with TLS configuration that uses system certificates.
func sslContext() *http.Client {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs:            nil, // use system certs
			InsecureSkipVerify: false,
		},
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
	}
	return &http.Client{
		Transport: transport,
		Timeout:   defaultTimeout,
	}
}

// httpGet executes an HTTP GET request and returns the response body as a string.
func httpGet(cfg *Config, endpoint string) (string, error) {
	base := strings.TrimRight(cfg.SiteURL, "/")
	ep := strings.TrimLeft(endpoint, "/")
	reqURL := base + "/" + ep

	client := sslContext()
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return "", &RFPIOAPIError{Message: fmt.Sprintf("failed to create request: %v", err)}
	}
	req.Header.Set("Authorization", "Bearer "+cfg.APIToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		if urlErr, ok := err.(*url.Error); ok {
			return "", &RFPIOAPIError{Message: fmt.Sprintf("Network error for GET %s: %v", reqURL, urlErr)}
		}
		return "", &RFPIOAPIError{Message: fmt.Sprintf("Request error for GET %s: %v", reqURL, err)}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", &RFPIOAPIError{Message: fmt.Sprintf("Failed to read response body: %v", err)}
	}

	if resp.StatusCode != http.StatusOK {
		msg := string(body)
		if msg == "" {
			msg = resp.Status
		}
		return "", &RFPIOAPIError{Code: resp.StatusCode, Message: fmt.Sprintf("HTTP %d for GET %s: %s", resp.StatusCode, reqURL, msg)}
	}

	return string(body), nil
}

// httpPostJSON executes an HTTP POST request with a JSON payload and returns the response body as a string.
func httpPostJSON(cfg *Config, endpoint string, payload any) (string, error) {
	base := strings.TrimRight(cfg.SiteURL, "/")
	ep := strings.TrimLeft(endpoint, "/")
	reqURL := base + "/" + ep

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return "", &RFPIOAPIError{Message: fmt.Sprintf("Failed to marshal request payload: %v", err)}
	}

	client := sslContext()
	req, err := http.NewRequest("POST", reqURL, nil)
	if err != nil {
		return "", &RFPIOAPIError{Message: fmt.Sprintf("failed to create request: %v", err)}
	}
	req.Header.Set("Authorization", "Bearer "+cfg.APIToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("User-Agent", userAgent)
	req.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	resp, err := client.Do(req)
	if err != nil {
		if urlErr, ok := err.(*url.Error); ok {
			return "", &RFPIOAPIError{Message: fmt.Sprintf("Network error for POST %s: %v", reqURL, urlErr)}
		}
		return "", &RFPIOAPIError{Message: fmt.Sprintf("Request error for POST %s: %v", reqURL, err)}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", &RFPIOAPIError{Message: fmt.Sprintf("Failed to read response body: %v", err)}
	}

	if resp.StatusCode != http.StatusOK {
		msg := string(respBody)
		if msg == "" {
			msg = resp.Status
		}
		return "", &RFPIOAPIError{Code: resp.StatusCode, Message: fmt.Sprintf("HTTP %d for POST %s: %s", resp.StatusCode, reqURL, msg)}
	}

	return string(respBody), nil
}

// parseResponse parses the raw JSON response string into a map.
func parseResponse(raw string) (any, error) {
	var result any
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return nil, &RFPIOAPIError{Message: fmt.Sprintf("Invalid JSON in response: %v", err)}
	}
	return result, nil
}
