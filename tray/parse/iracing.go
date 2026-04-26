package parse

// iRacing's controls.cfg is INI-shaped — a [Section] header, then key=value
// pairs. The current ParseINI handles it correctly. This file exists to
// give the format a stable name in the dispatcher and to leave room for
// per-game post-processing if iRacing adds quirks (e.g. multi-line entries
// for chord bindings) that need normalising.
//
// Today: simple delegation to ParseINI.

func ParseIRacingControls(path string) (map[string]string, error) {
	return ParseINI(path)
}
