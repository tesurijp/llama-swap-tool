package main

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type ProcessConfig struct {
	Enabled       bool     `yaml:"enabled"`
	Path          string   `yaml:"path"`
	UseConfigArgs bool     `yaml:"useConfigArgs"`
	Args          []string `yaml:"args"`
}

type LauncherConfig struct {
	LlamaSwap ProcessConfig `yaml:"llama_swap"`
	OlProxy   ProcessConfig `yaml:"ol_proxy"`
}

func getDefaultConfig() *LauncherConfig {
	return &LauncherConfig{
		LlamaSwap: ProcessConfig{
			Enabled:       true,
			Path:          "llama-swap.exe",
			UseConfigArgs: false,
			Args:          []string{},
		},
		OlProxy: ProcessConfig{
			Enabled:       true,
			Path:          "ol-proxy.exe",
			UseConfigArgs: true,
			Args:          []string{"-d"},
		},
	}
}

func loadLauncherConfig() (*LauncherConfig, error) {
	exePath, err := os.Executable()
	if err != nil {
		return getDefaultConfig(), err
	}
	configPath := filepath.Join(filepath.Dir(exePath), "llama-launcher.yaml")

	f, err := os.Open(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return getDefaultConfig(), nil
		}
		return getDefaultConfig(), err
	}
	defer f.Close()

	cfg := getDefaultConfig()
	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(cfg)
	if err != nil {
		return getDefaultConfig(), err
	}
	return cfg, nil
}
