package main

import (
	"os/exec"
)

func open(path string) {
	exec.Command("rundll32", "url.dll,FileProtocolHandler", path).Start()
}
