// Package watch polls for known game processes and fires verify (and,
// in a future iteration, monitor) when one launches or exits.
//
// The poller runs in a single background goroutine; transitions are
// detected by diffing the current snapshot against the previous one.
// Listeners get events on a buffered channel so the API layer can
// surface "iRacing just launched" without polling.
package watch

import (
	"context"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/hkbla/streamdeck-config/tray/config"
	"github.com/hkbla/streamdeck-config/tray/probe"
	"github.com/hkbla/streamdeck-config/tray/verify"
)

// Default poll interval. 2.5s feels right — fast enough to fire verify
// before the user finishes loading the menu, slow enough to be cheap.
const DefaultInterval = 2500 * time.Millisecond

// GameState — current snapshot for one game. Returned from /api/games.
type GameState struct {
	Name        string         `json:"name"`
	Running     bool           `json:"running"`
	LastLaunch  *time.Time     `json:"last_launch,omitempty"`
	LastExit    *time.Time     `json:"last_exit,omitempty"`
	LastVerify  *verify.Result `json:"last_verify,omitempty"`
}

// Event — emitted on game-launch / game-exit transitions.
type Event struct {
	Game string    `json:"game"`
	Kind string    `json:"kind"` // "launch" | "exit"
	Time time.Time `json:"time"`
}

// Watcher polls for known game exes and tracks per-game state.
type Watcher struct {
	cfg      *config.Config
	interval time.Duration
	monitor  MonitorRunner // optional; spawns the monitor script on launch

	mu        sync.RWMutex
	state     map[string]*GameState
	listeners []chan Event
}

// MonitorRunner — pluggable interface that the watcher uses to start the
// per-game monitor script on launch transitions. Decoupled from exec.Dispatcher
// to keep the watch package free of import cycles. Implemented by main.go.
type MonitorRunner interface {
	StartMonitor(scriptName string)
}

// New constructs a Watcher reading game definitions from cfg.
func New(cfg *config.Config) *Watcher {
	w := &Watcher{
		cfg:      cfg,
		interval: DefaultInterval,
		state:    map[string]*GameState{},
	}
	for name := range cfg.Games {
		w.state[name] = &GameState{Name: name}
	}
	return w
}

// SetMonitorRunner wires a runner. When nil (default), auto-monitor is skipped
// — the watcher still emits events and runs verify, just doesn't spawn scripts.
func (w *Watcher) SetMonitorRunner(r MonitorRunner) { w.monitor = r }

// Subscribe returns a channel that receives Events. Buffered so a slow
// reader doesn't block the watcher goroutine. Unsubscribe by closing
// the channel from the receiver side and dropping the reference.
func (w *Watcher) Subscribe() <-chan Event {
	ch := make(chan Event, 16)
	w.mu.Lock()
	w.listeners = append(w.listeners, ch)
	w.mu.Unlock()
	return ch
}

// States returns a snapshot of the current per-game state. Safe to call
// from any goroutine; doesn't block the watcher.
func (w *Watcher) States() map[string]GameState {
	w.mu.RLock()
	defer w.mu.RUnlock()
	out := make(map[string]GameState, len(w.state))
	for k, v := range w.state {
		out[k] = *v
	}
	return out
}

// Start launches the poller goroutine. Returns immediately; the poller
// runs until ctx is cancelled.
func (w *Watcher) Start(ctx context.Context) {
	go w.loop(ctx)
}

func (w *Watcher) loop(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	w.tick() // initial snapshot
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.tick()
		}
	}
}

func (w *Watcher) tick() {
	procs := probe.AllProcessNames()
	if procs == nil {
		return
	}
	procSet := make(map[string]struct{}, len(procs))
	for _, p := range procs {
		procSet[p] = struct{}{}
	}

	now := time.Now()
	w.mu.Lock()
	defer w.mu.Unlock()

	for name, gs := range w.state {
		def := w.cfg.Games[name]
		running := gameRunning(def, procSet)
		if running != gs.Running {
			// Transition!
			gs.Running = running
			ev := Event{Game: name, Time: now}
			if running {
				gs.LastLaunch = &now
				ev.Kind = "launch"
				log.Printf("[watch] %s LAUNCHED", name)
			} else {
				gs.LastExit = &now
				ev.Kind = "exit"
				log.Printf("[watch] %s EXITED", name)
			}
			w.broadcast(ev)
			// Fire verify async on launch (do not block the watcher loop)
			if running {
				go w.runVerify(name)
				// Auto-spawn monitor script if configured + runner wired
				if def.Monitoring != nil && def.Monitoring.Auto && def.Monitoring.Wrapper != "" && w.monitor != nil {
					script := def.Monitoring.Wrapper
					log.Printf("[watch] auto-monitor: spawning scripts/%s", script)
					go w.monitor.StartMonitor(script)
				}
			}
		}
	}
}

// runVerify runs a verify pass for the given game and stores the result
// on its GameState. Errors are logged but otherwise swallowed.
func (w *Watcher) runVerify(game string) {
	res := verify.Game(w.cfg, game)
	w.mu.Lock()
	if gs, ok := w.state[game]; ok {
		gs.LastVerify = &res
	}
	w.mu.Unlock()
	log.Printf("[watch] verify %s: %d ok, %d drift, %d missing",
		game, res.OkCount, res.DriftCount, res.MissingFiles)
}

func (w *Watcher) broadcast(ev Event) {
	for _, ch := range w.listeners {
		select {
		case ch <- ev:
		default:
			// Listener is slow; drop to avoid blocking the watcher.
		}
	}
}

// gameRunning checks if any of the game's declared exes appear in procSet.
// Handles cfg.Exe being a string OR []string OR nil — defensive because
// the JSON schema accepts both forms.
func gameRunning(def config.GameDef, procSet map[string]struct{}) bool {
	for _, name := range exeNames(def) {
		// Match case-insensitively, with or without .exe suffix
		key := strings.ToLower(strings.TrimSuffix(name, ".exe"))
		if _, ok := procSet[key]; ok {
			return true
		}
	}
	return false
}

// exeNames extracts the list of exe basenames to check for one game.
// Inputs accepted:
//   - cfg.Exe = "iRacingSim64DX11.exe" (single string)
//   - cfg.Exe = []string{"iRacingSim64DX11.exe", "iRacingUI.exe"}
//   - cfg.Sim / cfg.UI populated (path; we take basename)
//   - cfg.Exe = nil + cfg.UI = "" → returns nothing → game never marked running
func exeNames(def config.GameDef) []string {
	out := []string{}
	switch v := def.Exe.(type) {
	case string:
		if v != "" {
			out = append(out, basenameNoExt(v))
		}
	case []interface{}:
		for _, item := range v {
			if s, ok := item.(string); ok && s != "" {
				out = append(out, basenameNoExt(s))
			}
		}
	}
	if def.Sim != "" {
		out = append(out, basenameNoExt(def.Sim))
	}
	if def.UI != "" {
		out = append(out, basenameNoExt(def.UI))
	}
	return out
}

func basenameNoExt(path string) string {
	// strip dir
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			path = path[i+1:]
			break
		}
	}
	// strip extension
	if dot := strings.LastIndexByte(path, '.'); dot > 0 {
		path = path[:dot]
	}
	return path
}
