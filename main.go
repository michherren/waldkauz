package main

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/cloudhut/kowl/backend/pkg/api"
	"github.com/getlantern/systray"
	"github.com/michherren/waldkauz/icon"
	"github.com/pkg/browser"
	"go.uber.org/zap"
)

//go:embed waldkauz-data-template/frontend/*
//go:embed waldkauz-data-template/install/*
var dataDir embed.FS

//go:embed waldkauz-data-template/config_template.yaml
var configFile embed.FS

var dataDirPath = "waldkauz-data"

func main() {
	startupLogger := zap.NewExample()

	recreateDataDir()

	os.Setenv("CONFIG_FILEPATH", "waldkauz-data/config.yaml")
	cfg, err := api.LoadConfig(startupLogger)
	if err != nil {
		startupLogger.Fatal("failed to load config", zap.Error(err))
		notValidConfig()
	}
	err = cfg.Validate()
	if err != nil {
		startupLogger.Fatal("failed to validate config", zap.Error(err))
		notValidConfig()
	}

	a := api.New(&cfg)
	go a.Start()

	browser.OpenURL("http://localhost:9090")

	registerShutdownSignal()

	systray.Run(onReady, onExit)
}

func notValidConfig() {
	browser.OpenFile("waldkauz-data/install/instructions.html")
}

func recreateDataDir() {
	firstRun := false
	if _, err := os.Stat(dataDirPath); os.IsNotExist(err) {
		firstRun = true
	}

	fs.WalkDir(dataDir, ".", func(path string, info os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if path == "." {
			return nil
		}

		targetDir := filepath.Join(dataDirPath, strings.Join(strings.Split(filepath.Dir(path), string(os.PathSeparator))[1:], string(os.PathSeparator)))
		if _, err := os.Stat(targetDir); os.IsNotExist(err) {
			err := os.MkdirAll(targetDir, 0777)
			if err != nil {
				panic(fmt.Sprintf("datadir '%s' could not be created", dataDirPath))
			}
		}

		if stats, err := os.Stat(path); !os.IsNotExist(err) && !stats.IsDir() {
			fileContent, err := dataDir.ReadFile(path)
			if err != nil {
				panic(fmt.Sprintf("could not read file '%s' could not be created", dataDirPath))
			}

			targetPath := filepath.Join(targetDir, info.Name())
			err = os.WriteFile(targetPath, fileContent, 0644)
			if err != nil {
				panic(fmt.Sprintf("%s could not be created: %v", targetPath, err))
			}
		}

		return nil
	})

	if _, err := os.Stat(filepath.Join(dataDirPath, "config.yaml")); os.IsNotExist(err) {
		content, _ := configFile.ReadFile("waldkauz-data-template/config_template.yaml")
		targetPath := filepath.Join(dataDirPath, "config.yaml")
		err := os.WriteFile(targetPath, content, 0644)
		if err != nil {
			panic(fmt.Sprintf("%s could not be created: %v", targetPath, err))
		}
	}

	if firstRun {
		notValidConfig()
	}
}

func onReady() {
	systray.SetIcon(icon.Data)
	systray.SetTitle("Waldkauz")
	systray.SetTooltip("Waldkauz")

	mOpen := systray.AddMenuItem("Open Interface", "Open Interface")
	//mRestart := systray.AddMenuItem("Restart Server", "Restart Server")
	mQuit := systray.AddMenuItem("Quit", "Close Waldkauz")
	go func() {
		for {
			select {
			case <-mOpen.ClickedCh:
				browser.OpenURL("http://localhost:9090/topics")
			/*case <-mRestart.ClickedCh:
			browser.OpenURL("http://localhost:9090")*/
			case <-mQuit.ClickedCh:
				fmt.Println("Requesting quit")
				systray.Quit()
				fmt.Println("Finished quitting")
				return
			}
		}

	}()
}

func onExit() {
	// clean up here
}

func registerShutdownSignal() {
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit

		systray.Quit()
	}()
}
