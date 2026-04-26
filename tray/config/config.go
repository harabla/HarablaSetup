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
	"runtime"
)

// Config — the full rig-config.json shape. All paths are absolute strings;
// %ENVVAR% expansion happens at load time on Windows.
type Config struct {
	Tools    map[string]string  `json:"tools"`
	Games    map[string]GameDef `json:"games"`
	VR       VRConfig           `json:"vr"`
	Displays map[string]string  `json:"displays"`
	Logs     string             `json:"logs"`
	Deck     DeckConfig         `json:"deck"`
	Audio    map[string]string  `json:"audio"`

	// Loaded path (informational), not in JSON
	loadedFrom string
}

type GameDef struct {
	UI         string   `json:"ui,omitempty"`
	Sim        string   `json:"sim,omitempty"`
	Exe        string   `json:"exe,omitempty"`
	Documents  string   `json:"documents,omitempty"`
	Settings   []string `json:"settings,omitempty"`
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

// expand walks string fields and replaces %ENVVAR% on Windows. No-op elsewhere.
func (c *Config) expand() {
	if runtime.GOOS != "windows" {
		return
	}
	for k, v := range c.Tools {
		c.Tools[k] = os.ExpandEnv(v)
	}
	for k, g := range c.Games {
		g.UI = os.ExpandEnv(g.UI)
		g.Sim = os.ExpandEnv(g.Sim)
		g.Exe = os.ExpandEnv(g.Exe)
		g.Documents = os.ExpandEnv(g.Documents)
		for i, s := range g.Settings {
			g.Settings[i] = os.ExpandEnv(s)
		}
		c.Games[k] = g
	}
	c.VR.OpenXRToolkit = os.ExpandEnv(c.VR.OpenXRToolkit)
	c.VR.VirtualDesktop = os.ExpandEnv(c.VR.VirtualDesktop)
	c.Logs = os.ExpandEnv(c.Logs)
	c.Deck.AppPath = os.ExpandEnv(c.Deck.AppPath)
	c.Deck.ProfilesDir = os.ExpandEnv(c.Deck.ProfilesDir)
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
				UI:        `C:\Program Files (x86)\iRacing\iRacingUI.exe`,
				Sim:       `C:\Program Files (x86)\iRacing\iRacingSim64DX11.exe`,
				Documents: `%USERPROFILE%\Documents\iRacing`,
				Settings:  []string{"app.ini", "rendererDX11.ini", "dxconfig.ini"},
			},
			"PUBG": {
				Exe:      `C:\Program Files (x86)\Steam\steamapps\common\PUBG\TslGame.exe`,
				Settings: []string{`%LOCALAPPDATA%\TslGame\Saved\Config\WindowsNoEditor\GameUserSettings.ini`},
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
