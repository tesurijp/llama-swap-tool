package main

import (
	"encoding/json"
)

// convertToOpenAI converts Ollama-style options and parameters to OpenAI-style
func convertToOpenAI(model string, messages []Message, system string, format string, options Options, stream bool) *OpenAIChatRequest {
	var finalMessages []Message
	if system != "" {
		finalMessages = append(finalMessages, Message{Role: "system", Content: system})
	}
	finalMessages = append(finalMessages, messages...)

	req := &OpenAIChatRequest{
		Model:            model,
		Messages:         finalMessages,
		Stream:           stream,
		Temperature:      options.Temperature,
		Seed:             options.Seed,
		TopP:             options.TopP,
		MaxTokens:        options.NumPredict,
		Stop:             options.Stop,
		PresencePenalty:  options.PresencePenalty,
		FrequencyPenalty: options.FrequencyPenalty,
	}

	if format == "json" {
		req.ResponseFormat = &ResponseFormat{Type: "json_object"}
	}

	return req
}

// parseChatRequest converts ChatRequest bytes to OpenAIChatRequest
func parseChatRequest(data []byte) (*OpenAIChatRequest, string, bool, error) {
	var req ChatRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, "", false, err
	}
	return convertToOpenAI(req.Model, req.Messages, req.System, req.Format, req.Options, req.Stream), req.Model, req.Stream, nil
}

// parseGenerateRequest converts GenerateRequest bytes to OpenAIChatRequest
func parseGenerateRequest(data []byte) (*OpenAIChatRequest, string, bool, error) {
	var req GenerateRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, "", false, err
	}
	messages := []Message{
		{Role: "user", Content: req.Prompt},
	}
	return convertToOpenAI(req.Model, messages, req.System, req.Format, req.Options, req.Stream), req.Model, req.Stream, nil
}

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
