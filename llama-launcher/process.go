package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"fyne.io/systray"
	"golang.org/x/sys/windows"
)

var (
	swapProcess  *os.Process
	proxyProcess *os.Process
	logFile      *os.File
	launcherCfg  *LauncherConfig
)

type additionalFunc func(cmd *exec.Cmd)

func startProcess(program string, args []string, addFunc additionalFunc) (*os.Process, error) {
	programPath, err := filepath.Abs(program)
	if err != nil {
		// If Absolute path fails, try to use it as is (might be in PATH or relative to CWD)
		programPath = program
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
	launcherCfg, _ = loadLauncherConfig()

	if launcherCfg.LlamaSwap.Enabled {
		args := os.Args[1:]
		if launcherCfg.LlamaSwap.UseConfigArgs {
			args = launcherCfg.LlamaSwap.Args
		}
		swapProcess, err = startProcess(launcherCfg.LlamaSwap.Path, args, addSwapEnv)
		if err != nil {
			return err
		}
	}

	if launcherCfg.OlProxy.Enabled {
		args := os.Args[1:]
		if launcherCfg.OlProxy.UseConfigArgs {
			args = launcherCfg.OlProxy.Args
		}
		proxyProcess, err = startProcess(launcherCfg.OlProxy.Path, args, addProxyEnv)
		if err != nil {
			if swapProcess != nil {
				terminateProcess(&swapProcess)
			}
			return err
		}
	}
	return nil
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
