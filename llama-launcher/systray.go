package main

import (
	_ "embed"

	"fyne.io/systray"
)

var (
	//go:embed llamaswap.ico
	iconData []byte
)

func onReady() {
	if runTargetProgram() != nil {
		systray.Quit()
		return
	}

	systray.SetIcon(iconData)
	systray.SetTitle("llama-swap & proxy ")

	mOpenWeb := systray.AddMenuItem("Open Web UI", "Open llama-swap playground")
	mOpenLog := systray.AddMenuItem("Open log file", "Open ol-proxy log")

	mOpenConfig := systray.AddMenuItem("Open config file", "Open config file")
	if !launcherCfg.LlamaSwap.Enabled {
		mOpenWeb.Disable()
		mOpenConfig.Disable()
	}
	if !launcherCfg.OlProxy.Enabled {
		mOpenLog.Disable()
	}
	mRestart := systray.AddMenuItem("Restart", "Restart llama-swap & proxy")
	mTerminateChild := systray.AddMenuItem("Exit", "Ext")

	go func() {
		for {
			select {
			case <-mOpenWeb.ClickedCh:
				open(targetUrl)
			case <-mOpenLog.ClickedCh:
				if logFile != nil {
					open(logFile.Name())
				}
			case <-mOpenConfig.ClickedCh:
				open(configPath)
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
