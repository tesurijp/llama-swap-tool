package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

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
	mux.HandleFunc("/api/show", showHandler)
	mux.HandleFunc("/api/embed", embedHandler)
	mux.HandleFunc("/api/embeddings", embedHandler)
	mux.HandleFunc("/api/version", versionHandler)

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      mux,
		ReadTimeout:  serverReadTimeout,
		WriteTimeout: 0,
		IdleTimeout:  serverIdleTimeout,
	}

	// Graceful shutdown setup
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		fmt.Printf("Ollama-compatible proxy running on :%d, upstream=%s\n", port, cfg.UpstreamURL)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "server error: %v\n", err)
			os.Exit(1)
		}
	}()

	<-stop
	fmt.Println("\nShutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "shutdown error: %v\n", err)
	}
	fmt.Println("Server stopped")
}
