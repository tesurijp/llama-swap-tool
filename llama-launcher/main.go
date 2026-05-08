package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"fyne.io/systray"
)

var (
	targetUrl  string
	configPath string
)

func main() {
	chkflg := flag.NewFlagSet("llamaswap-launcher", flag.ContinueOnError)
	listenStr := chkflg.String("listen", "", "listen ip/port")
	certFile := chkflg.String("tls-cert-file", "", "TLS certificate file")
	keyFile := chkflg.String("tls-key-file", "", "TLS key file")
	configFile := chkflg.String("config", "", "config file path")
	chkflg.Parse(os.Args[1:])

	configPath = *configFile
	if configPath == "" {
		exePath, _ := os.Executable()
		configPath = filepath.Join(filepath.Dir(exePath), "config.yaml")
	}

	var useTLS = (*certFile != "" && *keyFile != "")
	// Set default ports.
	if *listenStr == "" {
		defaultPort := ":8080"
		if useTLS {
			defaultPort = ":8443"
		}
		listenStr = &defaultPort
	}
	targetUrl = fmt.Sprintf("http%s://localhost%s", map[bool]string{true: "s", false: ""}[useTLS], *listenStr)

	systray.Run(onReady, onExit)
}
