BUILD_DIR := $(CURDIR)/build
GO := go

.PHONY: all clean llama-launcher ol-proxy

all: llama-launcher ol-proxy

llama-launcher: $(BUILD_DIR)/llama-launcher.exe

ol-proxy: $(BUILD_DIR)/ol-proxy.exe

$(BUILD_DIR)/llama-launcher.exe: $(wildcard llama-launcher/*.go) llama-launcher/llamaswap.ico
	@mkdir -p $(BUILD_DIR)
	cd llama-launcher && GOOS=windows GOARCH=amd64 $(GO) build -ldflags="-H=windowsgui" -o $(BUILD_DIR)/llama-launcher.exe .

$(BUILD_DIR)/ol-proxy.exe: $(wildcard ol-proxy/*.go)
	@mkdir -p $(BUILD_DIR)
	cd ol-proxy && GOOS=windows GOARCH=amd64 $(GO) build -o $(BUILD_DIR)/ol-proxy.exe .

clean:
	rm -rf $(BUILD_DIR)
