// Package config loads and validates rig-config.json — the per-PC source of
// truth for paths to tools, games, settings files, and monitor IDs. Read by
// the tray (for probes and action dispatch) and by the PowerShell scripts
// (for monitoring and health checks). Generated once during setup.ps1; not
// committed to git.
//
// Loading order:
//   1. -config flag if provided
//   2. ./rig-config.json next to the binary
//   3. <repo>/rig-config.json (sibling of docs/)
//   4. Built-in dev defaults (Mac/Linux only — pretends paths exist)
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
)

// Config — the full rig-config.json shape. All paths are absolute strings;
// %ENVVAR% expansion happens at load time on Windows.
type Config struct {
	Tools    map[string]string  `json:"tools"`
	Games    map[string]GameDef `json:"games"`
	System   SystemDef          `json:"system,omitempty"`
	VR       VRConfig           `json:"vr"`
	Displays map[string]string  `json:"displays"`
	Logs     string             `json:"logs"`
	Deck     DeckConfig         `json:"deck"`
	Audio    map[string]string  `json:"audio"`

	// Loaded path (informational), not in JSON
	loadedFrom string
}

type GameDef struct {
	UI            string         `json:"ui,omitempty"`
	Sim           string         `json:"sim,omitempty"`
	Exe           interface{}    `json:"exe,omitempty"` // string OR []string accepted
	Documents     string         `json:"documents,omitempty"`
	Settings      []string       `json:"settings,omitempty"`        // legacy: bare filenames under Documents
	SettingsFiles []SettingsFile `json:"settings_files,omitempty"`  // new: categorised + parseable
	Monitoring    *MonitoringDef `json:"monitoring,omitempty"`      // auto-spawn telemetry on launch
}

// MonitoringDef — declares whether the watcher should auto-fire a monitor
// script on launch, and which script to use. If omitted, no auto-monitor.
type MonitoringDef struct {
	Auto    bool   `json:"auto"`
	Wrapper string `json:"wrapper"` // e.g. "monitor-iracing.ps1"
}

// SettingsFile describes one parseable source (file path or registry key)
// belonging to a game or to the system block. Read by the verify layer.
type SettingsFile struct {
	Category string            `json:"category"` // graphics | controls | mouse | system | audio
	Path     string            `json:"path"`
	Format   string            `json:"format"`   // ini | iracing-controls | registry | kv
	Expected map[string]string `json:"expected,omitempty"`
}

// SystemDef holds OS-level settings_files (Windows registry mouse settings, etc.)
// — used for things that aren't owned by any one game.
type SystemDef struct {
	SettingsFiles []SettingsFile `json:"settings_files,omitempty"`
}

type VRConfig struct {
	OpenXRToolkit  string `json:"openXRToolkit,omitempty"`
	VirtualDesktop string `json:"virtualDesktop,omitempty"`
}

type DeckConfig struct {
	AppPath     string `json:"appPath,omitempty"`
	ProfilesDir string `json:"profilesDir,omitempty"`
}

// Load reads rig-config.json from the first path that exists, in priority
// order. Returns dev defaults on non-Windows when no file found, so Mac
// development works out of the box.
func Load(explicit string) (*Config, error) {
	candidates := []string{}
	if explicit != "" {
		candidates = append(candidates, explicit)
	}
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates,
			filepath.Join(filepath.Dir(exe), "rig-config.json"),
			filepath.Join(filepath.Dir(exe), "..", "..", "rig-config.json"),
		)
	}
	candidates = append(candidates, "rig-config.json")

	for _, p := range candidates {
		if data, err := os.ReadFile(p); err == nil {
			cfg := &Config{}
			if err := json.Unmarshal(data, cfg); err != nil {
				return nil, fmt.Errorf("parse %s: %w", p, err)
			}
			cfg.loadedFrom = p
			cfg.expand()
			return cfg, nil
		}
	}

	if runtime.GOOS != "windows" {
		// Dev defaults so Mac development works without manual setup
		cfg := devDefaults()
		cfg.loadedFrom = "<dev defaults>"
		return cfg, nil
	}
	return nil, fmt.Errorf("rig-config.json not found in any of: %v", candidates)
}

// LoadedFrom returns the path the config was read from (or "<dev defaults>").
func (c *Config) LoadedFrom() string { return c.loadedFrom }

// expandEnv expands both $VAR / ${VAR} (Unix-style) and %VAR% (Windows-style)
// environment variable references.
var winEnvRe = regexp.MustCompile(`%([^%]+)%`)

func expandEnv(s string) string {
	// First expand Windows-style %VAR%
	s = winEnvRe.ReplaceAllStringFunc(s, func(m string) string {
		name := m[1 : len(m)-1]
		if v, ok := os.LookupEnv(name); ok {
			return v
		}
		return m // leave unresolved vars as-is
	})
	// Then expand Unix-style $VAR / ${VAR}
	return os.ExpandEnv(s)
}

// expand walks string fields and replaces %ENVVAR% on Windows. No-op elsewhere.
func (c *Config) expand() {
	if runtime.GOOS != "windows" {
		return
	}
	for k, v := range c.Tools {
		c.Tools[k] = expandEnv(v)
	}
	for k, g := range c.Games {
		g.UI = expandEnv(g.UI)
		g.Sim = expandEnv(g.Sim)
		// Exe is interface{} (string or []string) — leave as-is; the watcher
		// normalises when reading.
		g.Documents = expandEnv(g.Documents)
		for i, s := range g.Settings {
			g.Settings[i] = expandEnv(s)
		}
		for i := range g.SettingsFiles {
			g.SettingsFiles[i].Path = expandEnv(g.SettingsFiles[i].Path)
		}
		c.Games[k] = g
	}
	for i := range c.System.SettingsFiles {
		c.System.SettingsFiles[i].Path = expandEnv(c.System.SettingsFiles[i].Path)
	}
	c.VR.OpenXRToolkit = expandEnv(c.VR.OpenXRToolkit)
	c.VR.VirtualDesktop = expandEnv(c.VR.VirtualDesktop)
	c.Logs = expandEnv(c.Logs)
	c.Deck.AppPath = expandEnv(c.Deck.AppPath)
	c.Deck.ProfilesDir = expandEnv(c.Deck.ProfilesDir)
}

func devDefaults() *Config {
	return &Config{
		Tools: map[string]string{
			"soundVolumeView":  `C:\Tools\SoundVolumeView\SoundVolumeView.exe`,
			"multiMonitorTool": `C:\Tools\MultiMonitorTool\MultiMonitorTool.exe`,
			"presentMon":       `C:\Tools\PresentMon\PresentMon-1.10.0-x64.exe`,
			"hwinfo":           `C:\Program Files\HWiNFO64\HWiNFO64.exe`,
			"joystickGremlin":  `C:\Program Files (x86)\Joystick Gremlin\joystick_gremlin.exe`,
			"iRFFB":            `C:\Program Files\iRFFB2022\iRFFB.exe`,
		},
		Games: map[string]GameDef{
			"iRacing": {
				UI:         `C:\Program Files (x86)\iRacing\iRacingUI.exe`,
				Sim:        `C:\Program Files (x86)\iRacing\iRacingSim64DX11.exe`,
				Documents:  `%USERPROFILE%\Documents\iRacing`,
				Settings:   []string{"app.ini", "rendererDX11.ini", "dxconfig.ini"},
				Monitoring: &MonitoringDef{Auto: true, Wrapper: "monitor-iracing.ps1"},
				// Dev mode points at the test fixtures so verify works on Mac
				SettingsFiles: []SettingsFile{
					{
						Category: "graphics", Format: "ini",
						Path: "tray/parse/testdata/iracing-app.ini",
						Expected: map[string]string{
							"Graphics.MultiSamples":  "4",
							"Graphics.MaxQuality":    "1",
							"Graphics.MirrorHigh":    "1",
							"Graphics.TextureQuality": "2",
						},
					},
					{
						Category: "controls", Format: "iracing-controls",
						Path: "tray/parse/testdata/iracing-controls.cfg",
						Expected: map[string]string{
							"Controls.Brake Bias increase": "vJoy_71",
							"Controls.Brake Bias decrease": "vJoy_72",
							"Controls.Black Box click":     "vJoy_56",
						},
					},
				},
			},
			"PUBG": {
				Exe:      `C:\Program Files (x86)\Steam\steamapps\common\PUBG\TslGame.exe`,
				Settings: []string{`%LOCALAPPDATA%\TslGame\Saved\Config\WindowsNoEditor\GameUserSettings.ini`},
				SettingsFiles: []SettingsFile{
					{
						Category: "graphics", Format: "ini",
						Path: "tray/parse/testdata/pubg-gameusersettings.ini",
						Expected: map[string]string{
							"ScalabilityGroups.sg.TextureQuality":      "3",
							"ScalabilityGroups.sg.AntiAliasingQuality": "2",
							"ScalabilityGroups.sg.ResolutionQuality":   "100.000000",
						},
					},
					{
						Category: "mouse", Format: "ini",
						Path: "tray/parse/testdata/pubg-gameusersettings.ini",
						Expected: map[string]string{
							"/Script/TslGame.TslGameUserSettings.AimSensitivity":    "32.000000",
							"/Script/TslGame.TslGameUserSettings.ScopeSensitivity": "20.000000",
						},
					},
				},
			},
		},
		VR: VRConfig{
			OpenXRToolkit:  `%APPDATA%\OpenXR-Toolkit`,
			VirtualDesktop: `C:\Program Files\Virtual Desktop Streamer\VirtualDesktop.Streamer.Service.exe`,
		},
		Displays: map[string]string{
			"topLeft":   `\\.\DISPLAY2`,
			"topRight":  `\\.\DISPLAY3`,
			"ultrawide": `\\.\DISPLAY1`,
		},
		Logs: `C:\Logs`,
		Deck: DeckConfig{
			AppPath:     `C:\Program Files\Elgato\StreamDeck\StreamDeck.exe`,
			ProfilesDir: `%APPDATA%\Elgato\StreamDeck\ProfilesV2`,
		},
		Audio: map[string]string{
			"discord": "Discord",
			"spotify": "Spotify",
			"iRacing": "iRacingSim64DX11",
		},
	}
}
