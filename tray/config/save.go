package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// Save writes the current Config back to its loaded path. Used by the
// "Accept as new baseline" workflow to persist updates to expected values.
//
// Trade-off: re-marshals the JSON, which loses any comments / specific
// formatting the user might have hand-edited. Acceptable because
// rig-config.json is gitignored per-PC config; users who want to preserve
// formatting can edit the file directly instead of clicking baseline.
//
// Returns an error if the config was loaded from dev defaults
// (no real path to write back to).
func (c *Config) Save() error {
	if c.loadedFrom == "" || c.loadedFrom == "<dev defaults>" {
		return fmt.Errorf("config: no source path to save to (loaded from %q)", c.loadedFrom)
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("config: marshal: %w", err)
	}
	tmp := c.loadedFrom + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return fmt.Errorf("config: write temp: %w", err)
	}
	if err := os.Rename(tmp, c.loadedFrom); err != nil {
		return fmt.Errorf("config: rename: %w", err)
	}
	return nil
}
