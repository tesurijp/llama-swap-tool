package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strings"
)

var httpClient *http.Client

func initHTTPClient() {
	httpClient = &http.Client{
		// No global timeout; individual operations have their own timeouts
		Timeout: 0,
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   dialTimeout,
				KeepAlive: keepAliveTimeout,
			}).DialContext,
			ForceAttemptHTTP2:     false,
			MaxIdleConns:          maxIdleConns,
			IdleConnTimeout:       idleConnTimeout,
			ResponseHeaderTimeout: 0,
		},
	}
}

// sendJSONError sends an Ollama-compatible JSON error message
func sendJSONError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// handleUpstreamError checks if the upstream response indicates an error and handles it
func handleUpstreamError(w http.ResponseWriter, resp *http.Response) bool {
	statusCode := resp.StatusCode
	if statusCode < 200 || statusCode >= 300 {
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			debugf("failed to read upstream error body: %v", err)
			respBody = []byte("")
		}
		resp.Body.Close()
		debugf("upstream error body: %s", string(respBody))

		// Try to extract message from OpenAI error JSON
		var openAIError struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		errorMessage := strings.TrimSpace(string(respBody))
		if err := json.Unmarshal(respBody, &openAIError); err == nil && openAIError.Error.Message != "" {
			errorMessage = openAIError.Error.Message
		}

		sendJSONError(w, errorMessage, resp.StatusCode)
		return true
	}
	return false
}

// sendUpstreamRequest sends a request to the upstream server
func sendUpstreamRequest(endpoint string, reqBody []byte) (*http.Response, error) {
	httpReq, err := http.NewRequest("POST", cfg.UpstreamURL+endpoint, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	debugf("upstream request POST %s body: %s", httpReq.URL, string(reqBody))

	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	debugf("upstream response status: %d", resp.StatusCode)
	return resp, nil
}
