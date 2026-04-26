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

	"github.com/hkbla/streamdeck-config/tray/probe"
)

// Register attaches all /api/* routes onto mux.
func Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/state", handleState)
	mux.HandleFunc("/api/processes", handleProcesses)
	mux.HandleFunc("/api/displays", handleDisplays)
	mux.HandleFunc("/api/vjoy", handleVJoy)
	mux.HandleFunc("/api/health", handleHealth)
}

// stateResp aggregates the snapshot the docs UI cares about for a quick
// "everything live now" view. Heavier endpoints exist for detail panes.
type stateResp struct {
	OS        string                 `json:"os"`
	Processes []probe.Process        `json:"processes"`
	Displays  []probe.Display        `json:"displays"`
	VJoy      probe.VJoyState        `json:"vjoy"`
	Health    map[string]probe.Check `json:"health"`
}

func handleState(w http.ResponseWriter, r *http.Request) {
	resp := stateResp{
		OS:        runtime.GOOS,
		Processes: probe.TopProcesses(15),
		Displays:  probe.Displays(),
		VJoy:      probe.VJoy(),
		Health:    probe.AllChecks(),
	}
	writeJSON(w, resp)
}

func handleProcesses(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, probe.TopProcesses(50))
}

func handleDisplays(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, probe.Displays())
}

func handleVJoy(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, probe.VJoy())
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, probe.AllChecks())
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(v)
}
