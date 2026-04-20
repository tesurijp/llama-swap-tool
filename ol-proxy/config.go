package main

import (
	"fmt"
	"os"
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

var cfg *Config

func debugf(format string, v ...interface{}) {
	if cfg == nil || !cfg.DebugEnabled {
		return
	}
	fmt.Fprintf(os.Stdout, "[DEBUG] "+format+"\n", v...)
}
