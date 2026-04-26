// Package probe reads live system state. Each function has a real
// implementation in *_windows.go and a mock in *_other.go for development on
// Mac/Linux. The mock data mirrors what we'd actually see on the rig so the
// frontend can be built without a Fanatec wheel attached.
package probe

// Process — single row from the top-N processes view.
type Process struct {
	PID         int     `json:"pid"`
	Name        string  `json:"name"`
	CPUPercent  float64 `json:"cpu_percent"`
	RAMBytes    uint64  `json:"ram_bytes"`
	Description string  `json:"description,omitempty"` // friendly name where available
}

// Display — one physical monitor as Windows sees it.
type Display struct {
	ID       string `json:"id"`        // e.g. "\\\\.\\DISPLAY1"
	Name     string `json:"name"`      // friendly: "Top-Left", "Ultrawide"
	Active   bool   `json:"active"`    // currently enabled
	Width    int    `json:"width,omitempty"`
	Height   int    `json:"height,omitempty"`
	Primary  bool   `json:"primary"`
}

// VJoyDevice — single virtual joystick exposed by the vJoy driver.
type VJoyDevice struct {
	ID        int  `json:"id"`         // 1..16
	Enabled   bool `json:"enabled"`
	Buttons   int  `json:"buttons"`
	Axes      int  `json:"axes"`
	FFB       bool `json:"ffb"`        // force feedback enabled
}

// VJoyState — overall vJoy status.
type VJoyState struct {
	Installed bool          `json:"installed"`
	Devices   []VJoyDevice  `json:"devices"`
}

// Check — one health-check entry. Status is "ok", "warn", or "fail".
type Check struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Detail  string `json:"detail,omitempty"`
	FixHint string `json:"fix_hint,omitempty"`
}
