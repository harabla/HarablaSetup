// Package notify wraps cross-platform desktop notifications. Used by the
// game-launch watcher to surface drift / launch events outside the docs
// page (toast on Windows, Notification Center on macOS, libnotify on Linux).
//
// Single function: Notify(title, body, level) — fire-and-forget.
package notify

import (
	"log"

	"github.com/gen2brain/beeep"
)

// Level — toast severity. "info" / "warn" / "fail".
type Level string

const (
	Info Level = "info"
	Warn Level = "warn"
	Fail Level = "fail"
)

// Notify shows a desktop notification. Errors logged but never returned —
// missing notification daemon shouldn't crash the tray.
func Notify(title, body string, level Level) {
	var err error
	switch level {
	case Fail:
		err = beeep.Alert(title, body, "")
	case Warn:
		err = beeep.Notify(title, body, "")
	default:
		err = beeep.Notify(title, body, "")
	}
	if err != nil {
		log.Printf("[notify] %s: %v", level, err)
	}
}
