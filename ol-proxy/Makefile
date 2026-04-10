BUILD_DIR = build

ol-proxy.exe:ol-proxy.go
	GOOS=windows GOARCH=amd64 go build -o $(BUILD_DIR)/ol-proxy.exe

$(BUILD_DIR):
	mkdir -p $(BUILD_DIR)
clean:
	rm -rf $(BUILD_DIR)
