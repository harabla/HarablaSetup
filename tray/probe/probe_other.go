//go:build !windows

// Mock implementations for development on Mac / Linux. Returns plausible data
// shaped like what the Windows probes would produce on the actual rig, so the
// frontend can be built and styled without the real PC connected.
package probe

import (
	"os"
	"strings"
	"sync"
)

// MockRunningGamesEnv lets dev override which game exes appear running.
// Set MOCK_GAMES=iRacingSim64DX11.exe,iRacingUI.exe to simulate iRacing
// launched. Comma-separated, case-insensitive. Used by AllProcessNames.
const MockRunningGamesEnv = "MOCK_GAMES"

var mockGameOnce sync.Once
var mockGames []string

func mockGameNames() []string {
	mockGameOnce.Do(func() {
		raw := os.Getenv(MockRunningGamesEnv)
		if raw == "" {
			return
		}
		for _, s := range strings.Split(raw, ",") {
			s = strings.TrimSpace(s)
			if s != "" {
				mockGames = append(mockGames, s)
			}
		}
	})
	return mockGames
}

// AllProcessNames returns lowercased process names of every running process
// the OS reports. Used by the game-launch watcher to detect known game exes.
// On Mac/Linux this returns the static base list plus anything injected via
// MOCK_GAMES env var so dev can simulate launch events.
func AllProcessNames() []string {
	base := []string{
		"iracingui",            // simulated as always running for dev convenience
		"obs64",
		"discord",
		"spotify",
		"joystickgremlin",
		"vjoyconf",
		"fanalab",
		"irffb",
		"crewchiefv4",
		"simhub",
		"virtualdesktop.streamer.service",
		"hwinfo64",
		"streamdeck",
	}
	for _, g := range mockGameNames() {
		base = append(base, strings.ToLower(g))
	}
	return base
}

func TopProcesses(n int) []Process {
	all := []Process{
		{PID: 4321, Name: "iRacingSim64DX11", CPUPercent: 38.2, RAMBytes: 6_800_000_000, Description: "iRacing Sim"},
		{PID: 4322, Name: "iRacingUI", CPUPercent: 1.4, RAMBytes: 850_000_000, Description: "iRacing UI"},
		{PID: 5012, Name: "obs64", CPUPercent: 12.7, RAMBytes: 1_400_000_000, Description: "OBS Studio"},
		{PID: 6001, Name: "Discord", CPUPercent: 8.3, RAMBytes: 540_000_000, Description: "Discord"},
		{PID: 6101, Name: "Spotify", CPUPercent: 2.1, RAMBytes: 320_000_000, Description: "Spotify"},
		{PID: 7000, Name: "JoystickGremlin", CPUPercent: 0.8, RAMBytes: 90_000_000, Description: "Joystick Gremlin"},
		{PID: 7100, Name: "vJoyConf", CPUPercent: 0.0, RAMBytes: 12_000_000, Description: "vJoy Configuration"},
		{PID: 7200, Name: "FanaLab", CPUPercent: 1.2, RAMBytes: 180_000_000, Description: "FanaLab"},
		{PID: 7300, Name: "iRFFB", CPUPercent: 4.5, RAMBytes: 65_000_000, Description: "iRFFB 2022"},
		{PID: 7400, Name: "CrewChiefV4", CPUPercent: 3.2, RAMBytes: 240_000_000, Description: "Crew Chief"},
		{PID: 7500, Name: "SimHub", CPUPercent: 5.8, RAMBytes: 410_000_000, Description: "SimHub"},
		{PID: 7600, Name: "VirtualDesktop.Streamer.Service", CPUPercent: 6.7, RAMBytes: 220_000_000, Description: "Virtual Desktop"},
		{PID: 7700, Name: "HWiNFO64", CPUPercent: 1.1, RAMBytes: 95_000_000, Description: "HWiNFO64"},
		{PID: 8000, Name: "StreamDeck", CPUPercent: 0.5, RAMBytes: 120_000_000, Description: "Stream Deck app"},
	}
	if n > len(all) {
		n = len(all)
	}
	return all[:n]
}

func Displays() []Display {
	return []Display{
		{ID: `\\.\DISPLAY1`, Name: "Ultrawide",  Active: true, Width: 3440, Height: 1440, Primary: true},
		{ID: `\\.\DISPLAY2`, Name: "Top-Left",   Active: true, Width: 1920, Height: 1080},
		{ID: `\\.\DISPLAY3`, Name: "Top-Right",  Active: false, Width: 1920, Height: 1080},
	}
}

func VJoy() VJoyState {
	return VJoyState{
		Installed: true,
		Devices: []VJoyDevice{
			{ID: 1, Enabled: true,  Buttons: 128, Axes: 0, FFB: false},
			{ID: 2, Enabled: true,  Buttons: 0,   Axes: 1, FFB: true},
		},
	}
}

func AllChecks() map[string]Check {
	return map[string]Check{
		"vjoy_device_1":    {Name: "vJoy Device 1 (buttons)",    Status: "ok",   Detail: "128 buttons, FFB off"},
		"vjoy_device_2":    {Name: "vJoy Device 2 (FFB)",        Status: "ok",   Detail: "1 axis, FFB on"},
		"hidhide":          {Name: "HidHide whitelist",          Status: "ok",   Detail: "JoystickGremlin.exe + iRacingSim64DX11.exe"},
		"gremlin_running":  {Name: "Joystick Gremlin running",   Status: "warn", Detail: "process found, profile not confirmed", FixHint: "open Gremlin and verify fanatec-iracing.xml is loaded"},
		"fanatec_driver":   {Name: "Fanatec driver",             Status: "ok",   Detail: "v452 (current)"},
		"iracing_controls": {Name: "iRacing controls.cfg",       Status: "warn", Detail: "diff from canonical: 3 lines", FixHint: "run scripts/health-check.ps1 -Fix"},
		"openxr_toolkit":   {Name: "OpenXR Toolkit",             Status: "ok",   Detail: "settings.cfg present"},
		"virtual_desktop":  {Name: "Virtual Desktop streamer",   Status: "ok",   Detail: "service running"},
		"display_topleft":  {Name: "Display: Top-Left",          Status: "ok",   Detail: "1920x1080 @ 60Hz"},
		"display_topright": {Name: "Display: Top-Right",         Status: "fail", Detail: "not detected", FixHint: "check DP cable; should be \\.\\DISPLAY3"},
		"display_ultra":    {Name: "Display: Ultrawide",         Status: "ok",   Detail: "3440x1440 @ 144Hz (primary)"},
	}
}
