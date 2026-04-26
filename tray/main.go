// streamdeck-config tray — local server + tray icon for the gaming rig.
//
// Serves the docs/ static site at http://localhost:8765 and exposes /api/*
// endpoints for live state (vJoy, processes, monitors, audio sessions) and
// actions (toggle a monitor, run a script, refresh a config). The HTML page
// makes optional fetch() calls to enrich the static reference with live data;
// degrades cleanly if the tray isn't running.
package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/getlantern/systray"

	"github.com/hkbla/streamdeck-config/tray/api"
	"github.com/hkbla/streamdeck-config/tray/config"
	"github.com/hkbla/streamdeck-config/tray/exec"
	"github.com/hkbla/streamdeck-config/tray/watch"
)

const defaultAddr = "127.0.0.1:8765"

var (
	addr       = flag.String("addr", defaultAddr, "HTTP listen address")
	docsDir    = flag.String("docs", "", "Path to docs/ directory (default: auto-detect)")
	configPath = flag.String("config", "", "Path to rig-config.json (default: auto-detect)")
	noTray     = flag.Bool("no-tray", false, "Run without tray icon (CLI / headless mode)")
)

func main() {
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("[tray] config load failed: %v", err)
	}
	log.Printf("[tray] config:   %s", cfg.LoadedFrom())

	docs := resolveDocsDir(*docsDir)
	log.Printf("[tray] docs dir: %s", docs)
	log.Printf("[tray] listen:   http://%s", *addr)

	// Start the game-launch watcher with auto-monitor wired through the dispatcher
	dispatcher := exec.NewDispatcher(cfg)
	watcher := watch.New(cfg)
	watcher.SetMonitorRunner(dispatcher)
	watcher.Start(context.Background())

	srv := newServer(*addr, docs, cfg, watcher)

	// Start HTTP server in background
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[tray] server error: %v", err)
		}
	}()

	// Graceful shutdown on SIGINT/SIGTERM
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		<-sigCh
		log.Println("[tray] shutting down")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(ctx)
		systray.Quit()
	}()

	if *noTray {
		// Block forever; no tray UI
		select {}
	}
	systray.Run(onReady, onExit)
}

func newServer(addr, docs string, cfg *config.Config, watcher *watch.Watcher) *http.Server {
	mux := http.NewServeMux()

	// Static docs site
	fs := http.FileServer(http.Dir(docs))
	mux.Handle("/", fs)

	// Repo-root markdown files (VISION.md, README.md) — served as text/plain
	for _, name := range []string{"VISION.md", "README.md"} {
		path := filepath.Join(docs, "..", name)
		n := name // capture for closure
		mux.HandleFunc("/"+n, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			http.ServeFile(w, r, path)
		})
	}

	// API
	api.NewServer(cfg, watcher).Register(mux)

	return &http.Server{
		Addr:         addr,
		Handler:      logMiddleware(mux),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
}

// resolveDocsDir picks the docs directory: explicit flag > sibling of binary > cwd.
func resolveDocsDir(flagVal string) string {
	if flagVal != "" {
		return flagVal
	}
	exe, err := os.Executable()
	if err == nil {
		// If the binary lives in <repo>/tray/bin/, docs is at <repo>/docs/
		candidate := filepath.Join(filepath.Dir(exe), "..", "..", "docs")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			abs, _ := filepath.Abs(candidate)
			return abs
		}
	}
	// Fallback to ./docs from working dir
	if info, err := os.Stat("docs"); err == nil && info.IsDir() {
		abs, _ := filepath.Abs("docs")
		return abs
	}
	log.Fatalf("[tray] could not locate docs/ directory; pass -docs=<path>")
	return ""
}

func logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("[http] %s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

// --- tray icon callbacks --------------------------------------------------

func onReady() {
	systray.SetTitle("SDC")
	systray.SetTooltip("Stream Deck Config — http://" + *addr)

	mOpen := systray.AddMenuItem("Open in browser", "Open the docs page")
	mHealth := systray.AddMenuItem("Run health check", "Probe rig state")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "Stop the server")

	go func() {
		for {
			select {
			case <-mOpen.ClickedCh:
				openInBrowser("http://" + *addr)
			case <-mHealth.ClickedCh:
				log.Println("[tray] health check requested (TODO)")
			case <-mQuit.ClickedCh:
				systray.Quit()
				return
			}
		}
	}()
}

func onExit() {
	log.Println("[tray] exit")
}
