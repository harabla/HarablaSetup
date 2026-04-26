// Package exec dispatches whitelisted actions (toggle a monitor, run a known
// script). All actions are validated before dispatch — never accepts arbitrary
// commands or paths from the HTTP layer.
package exec

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/harabla/HarablaSetup/tray/config"
	"github.com/harabla/HarablaSetup/tray/probe"
)

// Result — outcome of a dispatched action. Status is "ok" or "error".
type Result struct {
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
	Output string `json:"output,omitempty"`
}

// Dispatcher knows the rig config and runs actions against it.
type Dispatcher struct {
	cfg *config.Config
}

func NewDispatcher(cfg *config.Config) *Dispatcher {
	return &Dispatcher{cfg: cfg}
}

// Run dispatches a named action with params. Whitelisted actions only.
func (d *Dispatcher) Run(action string, params map[string]string) Result {
	switch action {
	case "displays.toggle":
		return d.toggleDisplay(params["id"])
	case "displays.preset":
		return d.displayPreset(params["preset"])
	case "scripts.health":
		return d.runScript("health-check.ps1")
	case "scripts.monitor.iracing":
		return d.runScript("monitor-iracing.ps1")
	case "tray.refresh":
		return Result{Status: "ok", Detail: "noop — caller should re-fetch /api/state"}
	default:
		return Result{Status: "error", Detail: "unknown action: " + action}
	}
}

func (d *Dispatcher) toggleDisplay(id string) Result {
	if id == "" {
		return Result{Status: "error", Detail: "missing display id"}
	}
	// Validate id is one we know about (prevents arbitrary CLI args)
	known := false
	for _, mapped := range d.cfg.Displays {
		if mapped == id {
			known = true
			break
		}
	}
	if !known {
		return Result{Status: "error", Detail: "unknown display id: " + id}
	}
	if runtime.GOOS != "windows" {
		return Result{Status: "ok", Detail: fmt.Sprintf("[mock] would toggle %s", id)}
	}
	tool := d.cfg.Tools["multiMonitorTool"]
	if tool == "" {
		return Result{Status: "error", Detail: "multiMonitorTool path not in rig-config"}
	}
	out, err := exec.Command(tool, "/switch", id).CombinedOutput()
	if err != nil {
		return Result{Status: "error", Detail: err.Error(), Output: string(out)}
	}
	return Result{Status: "ok", Output: string(out)}
}

func (d *Dispatcher) displayPreset(preset string) Result {
	switch preset {
	case "all-on", "all-off", "vr-race", "work":
		// recognized
	default:
		return Result{Status: "error", Detail: "unknown preset: " + preset}
	}
	if runtime.GOOS != "windows" {
		return Result{Status: "ok", Detail: fmt.Sprintf("[mock] would apply preset %s", preset)}
	}
	tool := d.cfg.Tools["multiMonitorTool"]
	if tool == "" {
		return Result{Status: "error", Detail: "multiMonitorTool path not in rig-config"}
	}
	all := []string{}
	for _, id := range d.cfg.Displays {
		all = append(all, id)
	}
	var args []string
	switch preset {
	case "all-on", "work":
		args = append([]string{"/enable"}, all...)
	case "all-off":
		args = append([]string{"/disable"}, all...)
	case "vr-race":
		// Ultrawide on, top screens off
		args = []string{"/enable", d.cfg.Displays["ultrawide"]}
		// Then disable the top two
		out1, err := exec.Command(tool, args...).CombinedOutput()
		if err != nil {
			return Result{Status: "error", Detail: err.Error(), Output: string(out1)}
		}
		out2, err := exec.Command(tool, "/disable",
			d.cfg.Displays["topLeft"], d.cfg.Displays["topRight"]).CombinedOutput()
		if err != nil {
			return Result{Status: "error", Detail: err.Error(), Output: string(out1) + "\n" + string(out2)}
		}
		return Result{Status: "ok", Output: string(out1) + "\n" + string(out2)}
	}
	out, err := exec.Command(tool, args...).CombinedOutput()
	if err != nil {
		return Result{Status: "error", Detail: err.Error(), Output: string(out)}
	}
	return Result{Status: "ok", Output: string(out)}
}

// StartMonitor implements watch.MonitorRunner. Spawns the named script
// asynchronously (fire-and-forget) so the watcher doesn't block. On non-Windows
// hosts logs only — no actual spawn, since the PowerShell scripts won't run
// outside Windows.
func (d *Dispatcher) StartMonitor(scriptName string) {
	if runtime.GOOS != "windows" {
		fmt.Printf("[exec] mock: would spawn scripts/%s in background\n", scriptName)
		return
	}
	scriptPath := filepath.Join("scripts", scriptName)
	cmd := exec.Command("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", scriptPath)
	if err := cmd.Start(); err != nil {
		fmt.Printf("[exec] start scripts/%s failed: %v\n", scriptName, err)
		return
	}
	go cmd.Wait() // reap zombie when it exits, ignore result (script writes report itself)
	fmt.Printf("[exec] spawned scripts/%s pid=%d\n", scriptName, cmd.Process.Pid)
}

func (d *Dispatcher) runScript(name string) Result {
	// Whitelist enforced by switch in Run() — by the time we get here, name is safe
	if runtime.GOOS != "windows" {
		return Result{Status: "ok", Detail: fmt.Sprintf("[mock] would run scripts/%s", name)}
	}
	scriptPath := filepath.Join("scripts", name)
	out, err := exec.Command("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass",
		"-File", scriptPath).CombinedOutput()
	if err != nil {
		return Result{Status: "error", Detail: err.Error(), Output: string(out)}
	}
	return Result{Status: "ok", Output: string(out)}
}

// RunHealthCheck spawns scripts/health-check.ps1 in -Quiet mode and parses its
// JSON stdout into a map[string]probe.Check. Returns nil + error on failure.
// Only meaningful on Windows; elsewhere returns an error so the caller can
// fall back to the built-in probes.
func (d *Dispatcher) RunHealthCheck() (map[string]probe.Check, error) {
	if runtime.GOOS != "windows" {
		return nil, fmt.Errorf("health script only runs on windows")
	}
	out, err := exec.Command("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass",
		"-File", filepath.Join("scripts", "health-check.ps1"), "-Quiet").Output()
	if err != nil {
		return nil, fmt.Errorf("health-check.ps1: %w", err)
	}
	// Output may have a leading text line + JSON; isolate the JSON object.
	s := strings.TrimSpace(string(out))
	idx := strings.IndexByte(s, '{')
	if idx > 0 {
		s = s[idx:]
	}
	var parsed map[string]probe.Check
	if err := json.Unmarshal([]byte(s), &parsed); err != nil {
		return nil, fmt.Errorf("parse health JSON: %w", err)
	}
	return parsed, nil
}
