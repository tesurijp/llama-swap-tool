package main

import (
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
	logFile      *os.File
)

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
