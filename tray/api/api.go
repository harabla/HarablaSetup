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

	"github.com/hkbla/streamdeck-config/tray/config"
	"github.com/hkbla/streamdeck-config/tray/exec"
	"github.com/hkbla/streamdeck-config/tray/probe"
)

// Server holds the dependencies for handlers — config + action dispatcher.
type Server struct {
	cfg        *config.Config
	dispatcher *exec.Dispatcher
}

func NewServer(cfg *config.Config) *Server {
	return &Server{cfg: cfg, dispatcher: exec.NewDispatcher(cfg)}
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

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(v)
}
