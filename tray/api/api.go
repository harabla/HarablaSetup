// Package api wires the HTTP routes for the tray's local control surface.
//
// All endpoints are JSON. State endpoints are GET; action endpoints are POST.
// On non-Windows hosts (development on Mac), probe functions return mock data
// so the docs UI can be built and previewed without the real rig connected.
package api

import (
	"encoding/json"
	"net/http"
	"runtime"

	"strings"

	"github.com/hkbla/streamdeck-config/tray/config"
	"github.com/hkbla/streamdeck-config/tray/exec"
	"github.com/hkbla/streamdeck-config/tray/probe"
	"github.com/hkbla/streamdeck-config/tray/verify"
	"github.com/hkbla/streamdeck-config/tray/watch"
)

// Server holds the dependencies for handlers — config, action dispatcher,
// and the game-launch watcher.
type Server struct {
	cfg        *config.Config
	dispatcher *exec.Dispatcher
	watcher    *watch.Watcher
}

func NewServer(cfg *config.Config, watcher *watch.Watcher) *Server {
	return &Server{
		cfg:        cfg,
		dispatcher: exec.NewDispatcher(cfg),
		watcher:    watcher,
	}
}

// Register attaches all /api/* routes onto mux.
func (s *Server) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/state", s.handleState)
	mux.HandleFunc("/api/processes", s.handleProcesses)
	mux.HandleFunc("/api/displays", s.handleDisplays)
	mux.HandleFunc("/api/vjoy", s.handleVJoy)
	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/api/config", s.handleConfig)
	mux.HandleFunc("/api/action", s.handleAction)
	mux.HandleFunc("/api/verify", s.handleVerify)  // /api/verify (all)
	mux.HandleFunc("/api/verify/", s.handleVerify) // /api/verify/<game>[/baseline]
	mux.HandleFunc("/api/games", s.handleGames)    // game-launch watcher state
	mux.HandleFunc("/api/health/summary", s.handleHealthSummary) // compact for Stream Deck cells
}

type stateResp struct {
	OS         string                 `json:"os"`
	ConfigFrom string                 `json:"config_from"`
	Processes  []probe.Process        `json:"processes"`
	Displays   []probe.Display        `json:"displays"`
	VJoy       probe.VJoyState        `json:"vjoy"`
	Health     map[string]probe.Check `json:"health"`
}

func (s *Server) handleState(w http.ResponseWriter, r *http.Request) {
	resp := stateResp{
		OS:         runtime.GOOS,
		ConfigFrom: s.cfg.LoadedFrom(),
		Processes:  probe.TopProcesses(15),
		Displays:   probe.Displays(),
		VJoy:       probe.VJoy(),
		Health:     probe.AllChecks(),
	}
	writeJSON(w, resp)
}

func (s *Server) handleProcesses(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, probe.TopProcesses(50))
}

func (s *Server) handleDisplays(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, probe.Displays())
}

func (s *Server) handleVJoy(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, probe.VJoy())
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	// On Windows, prefer the real PowerShell health-check script so probes match
	// the canonical CLI tool. If it fails (script missing, error), fall back to
	// the built-in Go probes so the UI never breaks.
	if runtime.GOOS == "windows" {
		if checks, err := s.dispatcher.RunHealthCheck(); err == nil {
			writeJSON(w, checks)
			return
		}
	}
	writeJSON(w, probe.AllChecks())
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	// Expose config without rewrites (paths are read-only from frontend's POV)
	writeJSON(w, s.cfg)
}

type actionReq struct {
	Action string            `json:"action"`
	Params map[string]string `json:"params"`
}

func (s *Server) handleAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req actionReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.Action == "" {
		http.Error(w, "missing action", http.StatusBadRequest)
		return
	}
	res := s.dispatcher.Run(req.Action, req.Params)
	writeJSON(w, res)
}

// summaryResp — compact aggregated rig health, designed to be polled by a
// Stream Deck "Web Requests" cell which renders different colours/glyphs
// based on `status`.
type summaryResp struct {
	Status      string         `json:"status"` // "ok" | "warn" | "fail"
	Health      map[string]int `json:"health"` // {ok:N, warn:N, fail:N}
	Drift       summaryDrift   `json:"drift"`
	ActiveGame  string         `json:"active_game,omitempty"`
}

type summaryDrift struct {
	Total  int            `json:"total"`
	ByGame map[string]int `json:"by_game,omitempty"`
}

func (s *Server) handleHealthSummary(w http.ResponseWriter, r *http.Request) {
	resp := summaryResp{
		Status: "ok",
		Health: map[string]int{"ok": 0, "warn": 0, "fail": 0},
		Drift:  summaryDrift{ByGame: map[string]int{}},
	}

	// Aggregate health-check counts
	for _, c := range probe.AllChecks() {
		switch c.Status {
		case "ok":
			resp.Health["ok"]++
		case "warn":
			resp.Health["warn"]++
		case "fail":
			resp.Health["fail"]++
		}
	}

	// Aggregate drift counts across all games
	for _, gres := range verify.All(s.cfg) {
		if gres.DriftCount > 0 {
			resp.Drift.Total += gres.DriftCount
			resp.Drift.ByGame[gres.Target] = gres.DriftCount
		}
	}

	// Active game (first running)
	if s.watcher != nil {
		for name, st := range s.watcher.States() {
			if st.Running {
				resp.ActiveGame = name
				break
			}
		}
	}

	// Roll up overall status: fail wins, then warn, else ok
	if resp.Health["fail"] > 0 {
		resp.Status = "fail"
	} else if resp.Health["warn"] > 0 || resp.Drift.Total > 0 {
		resp.Status = "warn"
	}

	writeJSON(w, resp)
}

// handleGames returns the watcher's per-game state map: {running, last_launch,
// last_exit, last_verify} per game configured in rig-config.json. Refreshed
// at the watcher's poll interval (~2.5s).
func (s *Server) handleGames(w http.ResponseWriter, r *http.Request) {
	if s.watcher == nil {
		writeJSON(w, map[string]interface{}{})
		return
	}
	writeJSON(w, s.watcher.States())
}

// handleVerify routes:
//
//   GET  /api/verify                       — every game + system
//   GET  /api/verify/<game>                — one game (or "system")
//   POST /api/verify/<game>/baseline       — accept current actual as expected
//                                              body: {file, key, actual}
func (s *Server) handleVerify(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/verify"), "/"), "/")
	// parts[0] may be empty (the "every game" case)

	// POST .../baseline
	if r.Method == http.MethodPost && len(parts) >= 2 && parts[1] == "baseline" {
		s.handleVerifyBaseline(w, r, parts[0])
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if len(parts) == 0 || parts[0] == "" {
		writeJSON(w, verify.All(s.cfg))
		return
	}
	target := parts[0]
	if target == "system" {
		writeJSON(w, verify.System(s.cfg))
		return
	}
	writeJSON(w, verify.Game(s.cfg, target))
}

type baselineReq struct {
	File   string `json:"file"`
	Key    string `json:"key"`
	Actual string `json:"actual"`
}

func (s *Server) handleVerifyBaseline(w http.ResponseWriter, r *http.Request, game string) {
	var req baselineReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.File == "" || req.Key == "" {
		http.Error(w, "file and key required", http.StatusBadRequest)
		return
	}
	if err := verify.SnapshotBaseline(s.cfg, game, req.File, req.Key, req.Actual); err != nil {
		writeJSON(w, map[string]string{"status": "error", "detail": err.Error()})
		return
	}
	// Persist to rig-config.json on disk. If we loaded from dev defaults
	// (no real config file), Save() returns an error — surface it but the
	// in-memory change still applied for this session.
	if err := s.cfg.Save(); err != nil {
		writeJSON(w, map[string]string{
			"status": "ok",
			"detail": "in-memory only — " + err.Error(),
		})
		return
	}
	writeJSON(w, map[string]string{
		"status": "ok",
		"detail": "baseline saved to " + s.cfg.LoadedFrom(),
	})
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(v)
}
