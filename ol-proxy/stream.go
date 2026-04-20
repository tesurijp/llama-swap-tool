package main

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
)

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

	var mu sync.Mutex
	reader := bufio.NewReader(resp.Body)

	for {
		select {
		case <-ctx.Done():
			debugf("stream context done for model %s", model)
			return
		default:
		}

		line, err := reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				debugf("stream read error: %v", err)
				mu.Lock()
				json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
				flusher.Flush()
				mu.Unlock()
			}
			break
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		if data == "[DONE]" {
			resp := builder("", true)
			resp["model"] = model
			mu.Lock()
			json.NewEncoder(w).Encode(resp)
			flusher.Flush()
			mu.Unlock()
			debugf("stream %s [DONE]", model)
			break
		}

		var chunk StreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			debugf("stream parse error: %v", err)
			mu.Lock()
			json.NewEncoder(w).Encode(map[string]string{"error": "parse error: " + err.Error()})
			flusher.Flush()
			mu.Unlock()
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
		mu.Lock()
		json.NewEncoder(w).Encode(resp)
		flusher.Flush()
		mu.Unlock()
	}
	debugf("streamLoop exiting for model %s", model)
}
