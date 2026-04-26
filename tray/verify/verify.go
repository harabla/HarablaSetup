// Package verify diffs game / system settings files against the expected
// values declared in rig-config.json. Reads each file via the parse package,
// compares each expected key to the live value, and returns a structured
// list of drifts.
//
// Lives next to (not inside) probe/ because verify is conceptually a
// distinct mode — driven by the user's expectations, not just system state.
package verify

import (
	"fmt"
	"os"

	"github.com/hkbla/streamdeck-config/tray/config"
	"github.com/hkbla/streamdeck-config/tray/parse"
)

// Result — outcome of one verify pass against one game (or "system").
type Result struct {
	Target     string  `json:"target"`             // game name or "system"
	Files      []FileResult `json:"files"`
	DriftCount int     `json:"drift_count"`
	OkCount    int     `json:"ok_count"`
	MissingFiles int   `json:"missing_files"`
}

// FileResult — outcome for one settings_files entry.
type FileResult struct {
	Path     string  `json:"path"`
	Category string  `json:"category"`
	Format   string  `json:"format"`
	Status   string  `json:"status"` // "ok" | "drift" | "missing" | "error"
	Error    string  `json:"error,omitempty"`
	Drifts   []Drift `json:"drifts,omitempty"`
}

// Drift — one expected key whose actual value didn't match.
type Drift struct {
	Key      string `json:"key"`
	Expected string `json:"expected"`
	Actual   string `json:"actual"` // empty if missing
	Reason   string `json:"reason"` // "value_mismatch" | "missing_key"
}

// Game runs a verify pass for one game.
func Game(cfg *config.Config, game string) Result {
	g, ok := cfg.Games[game]
	if !ok {
		return Result{Target: game, Files: nil, MissingFiles: 1}
	}
	return runFiles(game, g.SettingsFiles)
}

// System runs a verify pass for the OS-level settings_files.
func System(cfg *config.Config) Result {
	return runFiles("system", cfg.System.SettingsFiles)
}

// All runs verify for every game + system.
func All(cfg *config.Config) []Result {
	var out []Result
	for name := range cfg.Games {
		out = append(out, Game(cfg, name))
	}
	if len(cfg.System.SettingsFiles) > 0 {
		out = append(out, System(cfg))
	}
	return out
}

func runFiles(target string, files []config.SettingsFile) Result {
	res := Result{Target: target}
	for _, sf := range files {
		fr := FileResult{Path: sf.Path, Category: sf.Category, Format: sf.Format}

		// File missing? Note it; don't fail the whole pass.
		if _, err := os.Stat(sf.Path); os.IsNotExist(err) && sf.Format != "registry" {
			fr.Status = "missing"
			fr.Error = "file not found"
			res.Files = append(res.Files, fr)
			res.MissingFiles++
			continue
		}

		actual, err := parse.Parse(sf.Format, sf.Path)
		if err != nil {
			fr.Status = "error"
			fr.Error = err.Error()
			res.Files = append(res.Files, fr)
			continue
		}

		for key, expected := range sf.Expected {
			a, present := actual[key]
			if !present {
				fr.Drifts = append(fr.Drifts, Drift{
					Key:      key,
					Expected: expected,
					Actual:   "",
					Reason:   "missing_key",
				})
				continue
			}
			if a != expected {
				fr.Drifts = append(fr.Drifts, Drift{
					Key:      key,
					Expected: expected,
					Actual:   a,
					Reason:   "value_mismatch",
				})
			}
		}
		if len(fr.Drifts) == 0 {
			fr.Status = "ok"
			res.OkCount += len(sf.Expected)
		} else {
			fr.Status = "drift"
			res.DriftCount += len(fr.Drifts)
			res.OkCount += len(sf.Expected) - len(fr.Drifts)
		}
		res.Files = append(res.Files, fr)
	}
	return res
}

// SnapshotBaseline writes the actual current value into cfg.Games[game]'s
// SettingsFile.Expected for the matching (path, key). Used by the
// "Accept as new baseline" workflow. Returns an error if the (game, file,
// key) tuple isn't found in the config.
//
// Note: this only mutates the in-memory config; persistence to
// rig-config.json is a separate responsibility (see api/config.go).
func SnapshotBaseline(cfg *config.Config, game, filePath, key, actual string) error {
	g, ok := cfg.Games[game]
	if !ok {
		return fmt.Errorf("snapshot: unknown game %q", game)
	}
	for i, sf := range g.SettingsFiles {
		if sf.Path != filePath {
			continue
		}
		if sf.Expected == nil {
			sf.Expected = map[string]string{}
		}
		sf.Expected[key] = actual
		g.SettingsFiles[i] = sf
		cfg.Games[game] = g
		return nil
	}
	return fmt.Errorf("snapshot: file %q not found for game %q", filePath, game)
}
