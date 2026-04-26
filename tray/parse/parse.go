// Package parse reads game config files in their various formats and produces
// a flat map[string]string of fully-qualified keys (e.g. "Graphics.MultiSamples"
// → "4"). Designed to be format-agnostic at the call site so the verify
// layer doesn't care whether the source was an INI, registry, or a custom
// game-specific format.
package parse

import "fmt"

// Parse reads a settings source and returns a flat dotted-key map.
//
// format: "ini" | "iracing-controls" | "registry" | "kv"
// path:   the file path or registry key (registry: prefix expected)
//
// On non-Windows hosts, registry parsing returns an error so callers can
// skip system-category checks during dev.
func Parse(format, path string) (map[string]string, error) {
	switch format {
	case "ini":
		return ParseINI(path)
	case "iracing-controls":
		return ParseIRacingControls(path)
	case "registry":
		return ParseRegistry(path)
	default:
		return nil, fmt.Errorf("parse: unknown format %q", format)
	}
}
