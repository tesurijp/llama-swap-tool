package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	defaultUpstreamURL = "http://localhost:8080"
	defaultPort        = 11434
	heartbeatInterval  = 15 * time.Second
	dialTimeout        = 30 * time.Second
	keepAliveTimeout   = 60 * time.Second
	maxIdleConns       = 100
	idleConnTimeout    = 90 * time.Second
	serverReadTimeout  = 30 * time.Second
	serverIdleTimeout  = 120 * time.Second
	// Endpoint URLs
	upstreamChatCompletionsEndpoint = "/v1/chat/completions"
	upstreamModelsEndpoint          = "/v1/models"
	upstreamEmbeddingsEndpoint      = "/v1/embeddings"
)

// Config holds application configuration
type Config struct {
	DebugEnabled bool
	UpstreamURL  string
}

var (
	cfg        *Config
	httpClient *http.Client
)

func debugf(format string, v ...interface{}) {
	if !cfg.DebugEnabled {
		return
	}
	fmt.Fprintf(os.Stdout, "[DEBUG] "+format+"\n", v...)
}

// ===== Type Definitions =====

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
}

type GenerateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type StreamChunk struct {
	Choices []struct {
		Delta Message `json:"delta"`
	} `json:"choices"`
}

type APIResponse struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
}

type Model struct {
	Name       string `json:"name"`
	ModifiedAt string `json:"modified_at"`
	Size       int64  `json:"size"`
	Digest     string `json:"digest"`
}

type EmbeddingRequest struct {
	Model string   `json:"model"`
	Input interface{} `json:"input"`
}

type EmbeddingResponse struct {
	Model string `json:"model"`
	Data  []struct {
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
}

// Callback function types
type parserFunc func([]byte) (*ChatRequest, error)
type respBuilderFunc func(string) map[string]interface{}
type streamBuilderFunc func(content string, done bool) map[string]interface{}

// ===== Initialization =====

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

// ===== Helper Functions =====

// handleUpstreamError checks if the upstream response indicates an error and handles it
func handleUpstreamError(w http.ResponseWriter, resp *http.Response) bool {
	statusCode := resp.StatusCode
	if statusCode < 200 || statusCode >= 300 {
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			debugf("failed to read upstream error body: %v", err)
			respBody = []byte("")
		}
		debugf("upstream error body: %s", string(respBody))
		http.Error(w, fmt.Sprintf("upstream error: %d %s", resp.StatusCode, strings.TrimSpace(string(respBody))), resp.StatusCode)
		return true
	}
	return false
}

// ===== Streaming Common Processing =====

// buildChatStreamChunk builds a stream response chunk for chat API
func buildChatStreamChunk(content string, done bool) map[string]interface{} {
	return map[string]interface{}{
		"message": map[string]string{
			"role":    "assistant",
			"content": content,
		},
		"done": done,
	}
}

// buildGenerateStreamChunk builds a stream response chunk for generate API
func buildGenerateStreamChunk(content string, done bool) map[string]interface{} {
	return map[string]interface{}{
		"response": content,
		"done":     done,
	}
}

func streamLoop(w http.ResponseWriter, model string, resp *http.Response, builder streamBuilderFunc, ctx context.Context) {
	w.Header().Set("Content-Type", "application/x-ndjson")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", 500)
		return
	}

	reader := bufio.NewReader(resp.Body)
	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()

	done := make(chan struct{})
	var closeOnce sync.Once
	closeDone := func() {
		closeOnce.Do(func() { close(done) })
	}
	defer closeDone()

	go func() {
		for {
			select {
			case <-ticker.C:
				w.Write([]byte("\n"))
				flusher.Flush()
			case <-done:
				return
			case <-ctx.Done():
				debugf("stream heartbeat goroutine context done for model %s", model)
				return
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			debugf("stream context done for model %s", model)
			return
		default:
		}

		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		if strings.TrimSpace(data) == "[DONE]" {
			resp := builder("", true)
			resp["model"] = model
			json.NewEncoder(w).Encode(resp)
			flusher.Flush()
			debugf("stream %s [DONE]", model)
			break
		}

		var chunk StreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			debugf("stream parse error: %v", err)
			continue
		}

		if len(chunk.Choices) == 0 {
			continue
		}

		content := chunk.Choices[0].Delta.Content
		if content == "" {
			continue
		}

		debugf("stream chunk (%s) content: %s", model, content)
		resp := builder(content, false)
		resp["model"] = model
		json.NewEncoder(w).Encode(resp)
		flusher.Flush()
	}
	debugf("streamLoop exiting for model %s", model)
}

// ===== Common API Processing =====

// sendUpstreamRequest はアップストリームサーバーにリクエストを送信する共通関数
func sendUpstreamRequest(reqBody []byte) (*http.Response, error) {
	httpReq, err := http.NewRequest("POST", cfg.UpstreamURL+upstreamChatCompletionsEndpoint, bytes.NewBuffer(reqBody))
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

// ===== Request Parsers =====

// parseChatRequest converts ChatRequest bytes to ChatRequest
func parseChatRequest(data []byte) (*ChatRequest, error) {
	var req ChatRequest
	err := json.Unmarshal(data, &req)
	return &req, err
}

// parseGenerateRequest converts GenerateRequest bytes to ChatRequest
func parseGenerateRequest(data []byte) (*ChatRequest, error) {
	var req GenerateRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, err
	}
	return &ChatRequest{
		Model: req.Model,
		Messages: []Message{
			{Role: "user", Content: req.Prompt},
		},
		Stream: req.Stream,
	}, nil
}

// ===== Response Builders =====

// buildChatResponse builds chat API response
func buildChatResponse(content string) map[string]interface{} {
	return map[string]interface{}{
		"message": map[string]string{
			"role":    "assistant",
			"content": content,
		},
	}
}

// buildGenerateResponse builds generate API response
func buildGenerateResponse(content string) map[string]interface{} {
	return map[string]interface{}{
		"response": content,
	}
}

// ===== Common Handler Processing =====

// handleChatLikeRequest handles both /api/chat and /api/generate endpoints
func handleChatLikeRequest(
	w http.ResponseWriter,
	r *http.Request,
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

	req, err := parser(clientBody)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	body, err := json.Marshal(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp, err := sendUpstreamRequest(body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if handleUpstreamError(w, resp) {
		return
	}

	if req.Stream {
		streamLoop(w, req.Model, resp, streamBuilder, r.Context())
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
	clientResp["model"] = req.Model
	clientResp["done"] = true
	debugf("client response: %v", clientResp)
	json.NewEncoder(w).Encode(clientResp)
}

// ===== Endpoint Handlers =====

func chatHandler(w http.ResponseWriter, r *http.Request) {
	handleChatLikeRequest(w, r, parseChatRequest, buildChatResponse, buildChatStreamChunk)
}

func generateHandler(w http.ResponseWriter, r *http.Request) {
	handleChatLikeRequest(w, r, parseGenerateRequest, buildGenerateResponse, buildGenerateStreamChunk)
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

	// Prepare request for upstream /v1/embeddings
	// Ollama request format and OpenAI format for 'model' and 'input' are similar.
	// We can reuse clientBody or re-marshal if we want to be safe.
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
		http.Error(w, err.Error(), http.StatusBadGateway)
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

	// Convert OpenAI response to Ollama format: {"model": "...", "embeddings": [[...], [...]]}
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

// ===== Main =====

func main() {
	var port int
	cfg = &Config{}
	flag.BoolVar(&cfg.DebugEnabled, "d", false, "enable debug logging")
	flag.BoolVar(&cfg.DebugEnabled, "debug", false, "enable debug logging")
	flag.IntVar(&port, "port", defaultPort, "listen port for incoming connections")
	flag.StringVar(&cfg.UpstreamURL, "upstream", defaultUpstreamURL, "OpenAI-compatible server URL (include port)")
	flag.Parse()

	// apply flag values
	initHTTPClient()

	mux := http.NewServeMux()
	mux.HandleFunc("/", rootHandler)
	mux.HandleFunc("/api/chat", chatHandler)
	mux.HandleFunc("/api/generate", generateHandler)
	mux.HandleFunc("/api/tags", tagsHandler)
	mux.HandleFunc("/api/embed", embedHandler)

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      mux,
		ReadTimeout:  serverReadTimeout,
		WriteTimeout: 0,
		IdleTimeout:  serverIdleTimeout,
	}

	fmt.Printf("Ollama-compatible proxy running on :%d, upstream=%s\n", port, cfg.UpstreamURL)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}
