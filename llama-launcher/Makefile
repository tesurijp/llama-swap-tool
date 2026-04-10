BUILD_DIR = build

llama-launcher.exe:llama-launcher.go
	GOOS=windows GOARCH=amd64 go build -ldflags="-H=windowsgui" -o $(BUILD_DIR)/llama-launcher.exe

$(BUILD_DIR):
	mkdir -p $(BUILD_DIR)
clean:
	rm -rf $(BUILD_DIR)
