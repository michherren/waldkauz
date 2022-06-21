package main

import (
	"bytes"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/cloudhut/common/rest"
	"github.com/getlantern/systray"
	"github.com/go-chi/chi"
	"github.com/michherren/waldkauz/icon"
	"github.com/pkg/browser"
	"github.com/redpanda-data/console/backend/pkg/api"
	"go.uber.org/zap"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
	builtBy = "unknown"

	dataDirPath = "waldkauz-data"
	serverHost  = "http://localhost:8080"
)

//go:embed waldkauz-data-template/install
var dataDir embed.FS

//go:embed waldkauz-data-template/config_template.yaml
var configFile embed.FS

var BasePathCtxKey = &struct{ name string }{"ConsoleURLPrefix"}

//go:embed frontend
var FrontendFiles embed.FS

type routerHooks struct {
	logger zap.Logger
}

// Router Hooks
func (r *routerHooks) ConfigAPIRouter(c chi.Router) {}
func (r *routerHooks) ConfigWsRouter(c chi.Router)  {}
func (r *routerHooks) ConfigRouter(c chi.Router) {
	c.NotFound(rest.HandleNotFound(&r.logger))
	c.MethodNotAllowed(rest.HandleMethodNotAllowed(&r.logger))
	c.Group(func(cr chi.Router) {
		cr.Get("/*", handleFrontendResources(&r.logger))
	})
}

func main() {
	startupLogger := zap.NewExample()

	recreateDataDir()

	os.Setenv("CONFIG_FILEPATH", filepath.Join(dataDirPath, "config.yaml"))
	cfg, err := api.LoadConfig(startupLogger)
	if err != nil {
		notValidConfig()
		startupLogger.Fatal("failed to load config", zap.Error(err))
	}
	err = cfg.Validate()
	if err != nil {
		notValidConfig()
		startupLogger.Fatal("failed to validate config", zap.Error(err))
	}

	serverHost = fmt.Sprintf("http://localhost:%d", cfg.REST.HTTPListenPort)
	a := api.New(&cfg)

	a.Hooks.Route = &routerHooks{logger: *startupLogger}
	go a.Start()

	browser.OpenURL(serverHost)

	registerShutdownSignal()

	systray.Run(onReady, onExit)
}

func notValidConfig() {
	browser.OpenFile(filepath.Join(dataDirPath, "install", "instructions.html"))
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

		if !info.IsDir() {
			fileContent, err := dataDir.ReadFile(path)
			if err != nil {
				panic(fmt.Sprintf("could not read file '%s' could not be created", dataDirPath))
			}

			targetPath := filepath.Join(targetDir, info.Name())
			err = os.WriteFile(targetPath, fileContent, 0666)
			if err != nil {
				panic(fmt.Sprintf("%s could not be created: %v", targetPath, err))
			}
		}

		return nil
	})

	if _, err := os.Stat(filepath.Join(dataDirPath, "config.yaml")); os.IsNotExist(err) {
		content, _ := configFile.ReadFile("waldkauz-data-template/config_template.yaml")
		targetPath := filepath.Join(dataDirPath, "config.yaml")
		err := os.WriteFile(targetPath, content, 0666)
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
	systray.SetTooltip(fmt.Sprintf("Waldkauz - %s, commit %s, built at %s by %s", version, commit, date, builtBy))

	mOpen := systray.AddMenuItem("Open Interface", "Open Interface")
	//mRestart := systray.AddMenuItem("Restart Server", "Restart Server")
	mQuit := systray.AddMenuItem("Quit", "Close Waldkauz")
	go func() {
		for {
			select {
			case <-mOpen.ClickedCh:
				browser.OpenURL(fmt.Sprintf("%s/topics", serverHost))
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

func handleFrontendIndex(logger *zap.Logger) http.HandlerFunc {
	basePathMarker := []byte(`__BASE_PATH_REPLACE_MARKER__`)

	// Load index.html file
	indexFilepath := "frontend/index.html"
	fmt.Println(FrontendFiles)
	indexOriginal, err := FrontendFiles.ReadFile(indexFilepath)
	if err != nil {
		logger.Fatal("failed to load index.html from embedded filesystem",
			zap.String("index_filepath", indexFilepath), zap.Error(err))
	}

	return func(w http.ResponseWriter, r *http.Request) {
		index := indexOriginal
		// If there's an active URL rewrite we need to replace the marker in the index.html with the
		// used base path so that the frontend knows what base URL to use for all subsequent requests.
		if basePath, ok := r.Context().Value(BasePathCtxKey).(string); ok && len(basePath) > 0 {

			// prefix must end with slash! otherwise the last segment gets cut off: 'a/b/c' -> "can't find host/a/b/resouce"
			if !strings.HasSuffix(basePath, "/") {
				basePath = basePath + "/"
			}
			// If we're running under a prefix, we need to let the frontend know
			// https://github.com/cloudhut/kowl/issues/107
			index = bytes.ReplaceAll(indexOriginal, basePathMarker, []byte(basePath))
		}

		hash := hashData(index)
		// For index.html we always set cache-control and etag
		w.Header().Set("Cache-Control", "public, max-age=900, must-revalidate") // 900s = 15m
		w.Header().Set("ETag", hash)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		// Check if the client sent 'If-None-Match' potentially return "304" (not mofified / unchanged)
		clientEtag := r.Header.Get("If-None-Match")
		if len(clientEtag) > 0 && hash == clientEtag {
			// Client already has the latest version of the file
			w.WriteHeader(http.StatusNotModified)
			return
		}

		if _, err := w.Write(index); err != nil {
			fmt.Printf("failed to write index file to response writer", zap.Error(err))
		}
	}
}

func handleFrontendResources(logger *zap.Logger) http.HandlerFunc {
	handleIndex := handleFrontendIndex(logger)

	fsys, err := fs.Sub(FrontendFiles, "frontend")
	if err != nil {
		logger.Fatal("failed to build subtree from embedded frontend files", zap.Error(err))
	}

	httpFs := http.FS(fsys)
	fsHandler := http.StripPrefix("/", http.FileServer(httpFs))
	fileHashes, err := getHashes(fsys)
	if err != nil {
		logger.Fatal("failed to calculate file hashes", zap.Error(err))
	}

	return func(w http.ResponseWriter, r *http.Request) {
		f, err := httpFs.Open(r.URL.Path)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Printf("requested file not found", zap.String("requestURI", r.RequestURI), zap.String("path", r.URL.Path))
			}
			handleIndex(w, r) // everything else goes to index as well
			return
		}
		defer f.Close()

		// Set correct content-type
		switch filepath.Ext(r.URL.Path) {
		case ".css":
			w.Header().Set("Content-Type", "text/css; charset=utf-8")
		case ".js":
			w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		}

		// Set Cache-Control and ETag
		w.Header().Set("Cache-Control", "public, max-age=900, must-revalidate") // 900s = 15min
		hash, hashFound := fileHashes[r.URL.Path]
		if hashFound {
			w.Header().Set("ETag", hash)
			clientEtag := r.Header.Get("If-None-Match")
			if len(clientEtag) > 0 && hash == clientEtag {
				// Client already has the latest version of the file
				w.WriteHeader(http.StatusNotModified)
				return
			}
		}

		fsHandler.ServeHTTP(w, r)
	}
}

// getHashes takes a filesystem, goes through all files
// in it recursively and calculates a sha256 for each one.
// It returns a map from file path to sha256 (already pre formatted in hex).
func getHashes(fsys fs.FS) (map[string]string, error) {
	fileHashes := make(map[string]string)
	err := fs.WalkDir(fsys, ".", func(path string, info fs.DirEntry, err error) error {
		if !info.IsDir() {
			fileBytes, err := fs.ReadFile(fsys, path)
			if err != nil {
				return fmt.Errorf("failed to open file %q: %w", path, err)
			}

			fileHashes[path] = hashData(fileBytes)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("Could not construct eTagCache, error while scanning files in directory: %w", err)
	}
	return fileHashes, nil
}

// hashData takes a byte array, calculates its sha256, and returns the hash as a hex encoded string
func hashData(data []byte) string {
	hasher := sha256.New()
	hasher.Write(data)
	hash := hasher.Sum(nil)
	return hex.EncodeToString(hash)
}
