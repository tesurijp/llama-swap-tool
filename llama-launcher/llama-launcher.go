package main

import (
	_ "embed"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"fyne.io/systray"
	"golang.org/x/sys/windows"
)

const (
	SwapProgram  = "llama-swap.exe"
	ProxyProgram = "ol-proxy.exe"
)

var (
	swapProcess  *os.Process
	proxyProcess *os.Process
	targetUrl    string
	logFile      *os.File
	//go:embed llamaswap.ico
	iconData []byte
)

func main() {
	chkflg := flag.NewFlagSet("llamaswap-launcher", flag.ContinueOnError)
	listenStr := chkflg.String("listen", "", "listen ip/port")
	certFile := chkflg.String("tls-cert-file", "", "TLS certificate file")
	keyFile := chkflg.String("tls-key-file", "", "TLS key file")
	chkflg.Parse(os.Args[1:])

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

func onReady() {
	if runTargetProgram() != nil {
		systray.Quit()
		return
	}

	systray.SetIcon(iconData)
	systray.SetTitle("llama-swap & proxy ")

	mOpenWeb := systray.AddMenuItem("Open Web UI", "Open llama-swap playground")
	mOpenLog := systray.AddMenuItem("Open log file", "Open ol-proxy log")
	mRestart := systray.AddMenuItem("Restart", "Restart llama-swap & proxy")
	mTerminateChild := systray.AddMenuItem("Exit", "Ext")

	go func() {
		for {
			select {
			case <-mOpenWeb.ClickedCh:
				open(targetUrl)
			case <-mOpenLog.ClickedCh:
				open(logFile.Name())
			case <-mRestart.ClickedCh:
				restartChildProcess()
			case <-mTerminateChild.ClickedCh:
				systray.Quit()
			}
		}
	}()
}

func onExit() {
	terminateChildProcess()
}

type additionalFunc func(cmd *exec.Cmd)

func startProcess(program string, args []string, addFunc additionalFunc) (*os.Process, error) {
	programPath, err := filepath.Abs(program)
	if err != nil {
		return nil, err
	}

	cmd := exec.Command(programPath, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: windows.CREATE_NO_WINDOW,
	}
	addFunc(cmd)

	err = cmd.Start()
	if err != nil {
		return nil, err
	}

	go func() {
		cmd.Process.Wait()
		systray.Quit()
	}()

	return cmd.Process, nil
}

func addSwapEnv(cmd *exec.Cmd) {}
func addProxyEnv(cmd *exec.Cmd) {
	var err error
	logFile, err = os.CreateTemp("", "ol-proxy-*.log")
	if err == nil {
		(*cmd).Stdout = logFile
		(*cmd).Stderr = logFile
	}
}

func runTargetProgram() error {
	var err error
	swapProcess, err = startProcess(SwapProgram, os.Args[1:], addSwapEnv)
	if err == nil {
		proxyProcess, err = startProcess(ProxyProgram, []string{"-d"}, addProxyEnv)
		if err != nil {
			terminateProcess(&swapProcess)
		}
	}
	return err
}

func terminateProcess(p **os.Process) {
	if *p == nil {
		return
	}
	proc, err := os.FindProcess((*p).Pid)
	if err == nil {
		proc.Kill()
	}
	*p = nil
}

func open(path string) {
	exec.Command("rundll32", "url.dll,FileProtocolHandler", path).Start()
}

func restartChildProcess() {
	terminateChildProcess()
	go func() {
		programPath, _ := filepath.Abs(os.Args[0])
		exec.Command(programPath, os.Args[1:]...).Start()
	}()
}

func terminateChildProcess() {
	terminateProcess(&swapProcess)
	terminateProcess(&proxyProcess)
}
