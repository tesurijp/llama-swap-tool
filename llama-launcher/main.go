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
	launcherCfg, _ = loadLauncherConfig()

	chkflg := flag.NewFlagSet("llamaswap-launcher", flag.ContinueOnError)
	listenStr := chkflg.String("listen", "", "listen ip/port")
	certFile := chkflg.String("tls-cert-file", "", "TLS certificate file")
	keyFile := chkflg.String("tls-key-file", "", "TLS key file")
	configFile := chkflg.String("config", "", "config file path")

	effectiveArgs := os.Args[1:]
	if launcherCfg.LlamaSwap.Enabled && launcherCfg.LlamaSwap.UseConfigArgs {
		effectiveArgs = launcherCfg.LlamaSwap.Args
	}
	chkflg.Parse(effectiveArgs)

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
