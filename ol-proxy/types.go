package main

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Options holds Ollama-style options
type Options struct {
	Temperature      *float32    `json:"temperature,omitempty"`
	Seed             *int        `json:"seed,omitempty"`
	TopP             *float32    `json:"top_p,omitempty"`
	TopK             *int        `json:"top_k,omitempty"`
	NumPredict       *int        `json:"num_predict,omitempty"`
	Stop             interface{} `json:"stop,omitempty"` // Can be string or []string
	PresencePenalty  *float32    `json:"presence_penalty,omitempty"`
	FrequencyPenalty *float32    `json:"frequency_penalty,omitempty"`
}

// ChatRequest represents the Ollama /api/chat request
type ChatRequest struct {
	Model     string    `json:"model"`
	Messages  []Message `json:"messages"`
	Stream    bool      `json:"stream"`
	System    string    `json:"system,omitempty"`
	Format    string    `json:"format,omitempty"`
	Options   Options   `json:"options,omitempty"`
	KeepAlive string    `json:"keep_alive,omitempty"`
}

// GenerateRequest represents the Ollama /api/generate request
type GenerateRequest struct {
	Model     string  `json:"model"`
	Prompt    string  `json:"prompt"`
	System    string  `json:"system,omitempty"`
	Template  string  `json:"template,omitempty"`
	Context   []int   `json:"context,omitempty"`
	Stream    bool    `json:"stream"`
	Raw       bool    `json:"raw,omitempty"`
	Format    string  `json:"format,omitempty"`
	Options   Options `json:"options,omitempty"`
	KeepAlive string  `json:"keep_alive,omitempty"`
}

// OpenAIChatRequest represents the OpenAI-compatible chat request
type OpenAIChatRequest struct {
	Model            string           `json:"model"`
	Messages         []Message        `json:"messages"`
	Stream           bool             `json:"stream"`
	Temperature      *float32         `json:"temperature,omitempty"`
	Seed             *int             `json:"seed,omitempty"`
	TopP             *float32         `json:"top_p,omitempty"`
	MaxTokens        *int             `json:"max_tokens,omitempty"`
	Stop             interface{}      `json:"stop,omitempty"`
	PresencePenalty  *float32         `json:"presence_penalty,omitempty"`
	FrequencyPenalty *float32         `json:"frequency_penalty,omitempty"`
	ResponseFormat   *ResponseFormat  `json:"response_format,omitempty"`
}

// ResponseFormat for OpenAI JSON mode
type ResponseFormat struct {
	Type string `json:"type"`
}

// StreamChunk for parsing OpenAI stream
type StreamChunk struct {
	Choices []struct {
		Delta Message `json:"delta"`
	} `json:"choices"`
}

// APIResponse for parsing OpenAI non-stream response
type APIResponse struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
}

// Model represents Ollama model metadata
type Model struct {
	Name       string `json:"name"`
	ModifiedAt string `json:"modified_at"`
	Size       int64  `json:"size"`
	Digest     string `json:"digest"`
}

// ShowRequest represents Ollama /api/show request
type ShowRequest struct {
	Name string `json:"name"`
}

// ShowResponse represents Ollama /api/show response
type ShowResponse struct {
	Modelfile string                 `json:"modelfile"`
	Template  string                 `json:"template"`
	System    string                 `json:"system"`
	Details   map[string]interface{} `json:"details"`
}

// EmbeddingRequest represents Ollama /api/embeddings request
type EmbeddingRequest struct {
	Model string      `json:"model"`
	Input interface{} `json:"input"`
}

// EmbeddingResponse represents OpenAI /v1/embeddings response
type EmbeddingResponse struct {
	Model string `json:"model"`
	Data  []struct {
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
}

// Callback function types
type parserFunc func([]byte) (*OpenAIChatRequest, string, bool, error)
type respBuilderFunc func(string) map[string]interface{}
type streamBuilderFunc func(content string, done bool) map[string]interface{}
