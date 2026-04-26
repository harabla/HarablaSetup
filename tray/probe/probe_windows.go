//go:build windows

// Real implementations using Windows APIs. TODO: replace stubs with real
// probes (registry queries, EnumDisplayDevices, perfcounter for CPU, etc).
package probe

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
