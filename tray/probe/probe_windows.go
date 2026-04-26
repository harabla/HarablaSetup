//go:build windows

// Real implementations using Windows APIs. TODO: replace stubs with real
// probes (registry queries, EnumDisplayDevices, perfcounter for CPU, etc).
package probe

import (
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

// AllProcessNames returns lowercased exe names of every running process
// using the ToolHelp32 snapshot API (cheap, supported on every Windows).
// Returns nil on error so callers can treat "no data" the same as "no
// matching games".
func AllProcessNames() []string {
	const TH32CS_SNAPPROCESS = 0x00000002
	snap, err := windows.CreateToolhelp32Snapshot(TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil
	}
	defer windows.CloseHandle(snap)

	var entry windows.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))
	if err := windows.Process32First(snap, &entry); err != nil {
		return nil
	}
	out := []string{}
	for {
		name := windows.UTF16ToString(entry.ExeFile[:])
		// Strip ".exe" suffix to match cfg.Games[].Exe entries which the
		// user may write either way; we lowercase for case-insensitive match.
		out = append(out, strings.ToLower(strings.TrimSuffix(name, ".exe")))
		if err := windows.Process32Next(snap, &entry); err != nil {
			break
		}
	}
	return out
}

func TopProcesses(n int) []Process {
	// TODO: golang.org/x/sys/windows + perfcounter for live CPU
	return []Process{
		{Name: "TODO: implement on Windows", CPUPercent: 0, RAMBytes: 0},
	}
}

func Displays() []Display {
	// TODO: shell out to MultiMonitorTool /scomma "" or call EnumDisplayDevices
	return []Display{
		{ID: `\\.\DISPLAY1`, Name: "TODO", Active: true},
	}
}

func VJoy() VJoyState {
	// TODO: read HKLM\SYSTEM\CurrentControlSet\services\vjoy\Parameters\Device0X\Enabled
	return VJoyState{Installed: false, Devices: nil}
}

func AllChecks() map[string]Check {
	// TODO: aggregate real probes
	return map[string]Check{
		"todo": {Name: "Windows probes not yet implemented", Status: "warn", Detail: "see probe_windows.go"},
	}
}
