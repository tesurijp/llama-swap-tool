package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// handleChatLikeRequest handles both /api/chat and /api/generate endpoints
func handleChatLikeRequest(
	w http.ResponseWriter,
	r *http.Request,
	endpoint string,
	parser parserFunc,
	respBuilder respBuilderFunc,
	streamBuilder streamBuilderFunc,
) {
	w.Header().Set("Content-Type", "application/json")

	clientBody, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	debugf("client POST body: %s", string(clientBody))

	openAIReq, model, stream, err := parser(clientBody)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	body, err := json.Marshal(openAIReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp, err := sendUpstreamRequest(endpoint, body)
	if err != nil {
		debugf("upstream request failed: %v", err)
		sendJSONError(w, "upstream server unavailable", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if handleUpstreamError(w, resp) {
		return
	}

	if stream {
		streamLoop(w, model, resp, streamBuilder, r.Context())
		return
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var openaiResp APIResponse
	if err := json.Unmarshal(respBody, &openaiResp); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if len(openaiResp.Choices) == 0 {
		http.Error(w, "no choices", http.StatusInternalServerError)
		return
	}

	clientResp := respBuilder(openaiResp.Choices[0].Message.Content)
	clientResp["model"] = model
	clientResp["done"] = true
	debugf("client response: %v", clientResp)
	json.NewEncoder(w).Encode(clientResp)
}

func chatHandler(w http.ResponseWriter, r *http.Request) {
	handleChatLikeRequest(w, r, upstreamChatCompletionsEndpoint, parseChatRequest, buildChatResponse, buildChatStreamChunk)
}

func generateHandler(w http.ResponseWriter, r *http.Request) {
	handleChatLikeRequest(w, r, upstreamChatCompletionsEndpoint, parseGenerateRequest, buildGenerateResponse, buildGenerateStreamChunk)
}

func versionHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"version": "0.1.48"})
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprint(w, "Ollama is running")
}

func tagsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	debugf("client GET /api/tags query: %s", r.URL.RawQuery)

	tagsURL := cfg.UpstreamURL + upstreamModelsEndpoint
	debugf("upstream request GET %s", tagsURL)

	resp, err := httpClient.Get(tagsURL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	debugf("upstream response status: %d", resp.StatusCode)
	if handleUpstreamError(w, resp) {
		return
	}

	var data struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := json.Unmarshal(body, &data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	models := make([]Model, 0, len(data.Data))
	for _, m := range data.Data {
		models = append(models, Model{
			Name:       m.ID,
			ModifiedAt: "",
			Size:       0,
			Digest:     "",
		})
	}

	clientResp := map[string]interface{}{
		"models": models,
	}
	debugf("client response /api/tags: %v", clientResp)
	json.NewEncoder(w).Encode(clientResp)
}

func showHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	clientBody, err := io.ReadAll(r.Body)
	if err != nil {
		sendJSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	var showReq ShowRequest
	if err := json.Unmarshal(clientBody, &showReq); err != nil {
		sendJSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// OpenAI /v1/models/{model}
	modelURL := cfg.UpstreamURL + upstreamModelsEndpoint + "/" + showReq.Name
	resp, err := httpClient.Get(modelURL)
	if err == nil {
		defer resp.Body.Close()
	}

	// Construct a basic response even if upstream model lookup fails
	showResp := ShowResponse{
		Modelfile: fmt.Sprintf("FROM %s", showReq.Name),
		Template:  "{{ .Prompt }}",
		Details: map[string]interface{}{
			"format":            "gguf",
			"family":            "llama",
			"parameter_size":    "unknown",
			"quantization_level": "unknown",
		},
	}

	json.NewEncoder(w).Encode(showResp)
}

func embedHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	clientBody, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	debugf("client POST /api/embed body: %s", string(clientBody))

	var embedReq EmbeddingRequest
	if err := json.Unmarshal(clientBody, &embedReq); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Basic validation of input
	if embedReq.Input == nil {
		sendJSONError(w, "input is required", http.StatusBadRequest)
		return
	}

	switch v := embedReq.Input.(type) {
	case string:
		if v == "" {
			sendJSONError(w, "input cannot be empty", http.StatusBadRequest)
			return
		}
	case []interface{}:
		if len(v) == 0 {
			sendJSONError(w, "input array cannot be empty", http.StatusBadRequest)
			return
		}
		for _, item := range v {
			if _, ok := item.(string); !ok {
				sendJSONError(w, "input array must contain only strings", http.StatusBadRequest)
				return
			}
		}
	default:
		sendJSONError(w, "input must be a string or an array of strings", http.StatusBadRequest)
		return
	}

	// Prepare request for upstream /v1/embeddings
	upstreamURL := cfg.UpstreamURL + upstreamEmbeddingsEndpoint
	httpReq, err := http.NewRequest("POST", upstreamURL, bytes.NewBuffer(clientBody))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")
	debugf("upstream request POST %s body: %s", httpReq.URL, string(clientBody))

	resp, err := httpClient.Do(httpReq)
	if err != nil {
		debugf("upstream request failed: %v", err)
		sendJSONError(w, "upstream server unavailable", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	debugf("upstream response status: %d", resp.StatusCode)
	if handleUpstreamError(w, resp) {
		return
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var openAIResp EmbeddingResponse
	if err := json.Unmarshal(respBody, &openAIResp); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert OpenAI response to Ollama format
	ollamaEmbeds := make([][]float32, len(openAIResp.Data))
	for _, item := range openAIResp.Data {
		if item.Index < len(ollamaEmbeds) {
			ollamaEmbeds[item.Index] = item.Embedding
		}
	}

	clientResp := map[string]interface{}{
		"model":      embedReq.Model,
		"embeddings": ollamaEmbeds,
	}

	debugf("client response /api/embed: %v", clientResp)
	json.NewEncoder(w).Encode(clientResp)
}
