package parse

import (
	"bufio"
	"os"
	"strings"
)

// ParseINI reads a .ini file and returns a flat dotted-key map. Sections
// become the prefix; values within the section get joined with a dot. So
// the input
//
//   [Graphics]
//   MultiSamples=4
//
// produces "Graphics.MultiSamples" -> "4".
//
// Handles:
//   - Standard INI: [Section] / key=value / # or ; comments
//   - Unreal flavour: section names like [/Script/Engine.GameUserSettings],
//     dotted keys like sg.TextureQuality (we don't try to parse the dot
//     inside the key — it stays as part of the key name, so the result is
//     "/Script/Engine.GameUserSettings.sg.TextureQuality" → "100").
//   - Quoted values are trimmed of surrounding double-quotes.
//   - Trailing comments after a value (e.g. "key=val ; note") get stripped.
//   - Values trim surrounding whitespace.
//   - Duplicate keys: last write wins (some games legitimately repeat).
//   - Lines that aren't section headers / comments / key=value are
//     silently skipped.
func ParseINI(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	out := map[string]string{}
	section := ""
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024) // tolerate long lines

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if line[0] == ';' || line[0] == '#' {
			continue
		}
		if line[0] == '[' && strings.HasSuffix(line, "]") {
			section = strings.TrimSpace(line[1 : len(line)-1])
			continue
		}
		eq := strings.IndexByte(line, '=')
		if eq < 0 {
			continue
		}
		key := strings.TrimSpace(line[:eq])
		val := strings.TrimSpace(line[eq+1:])
		// Strip trailing inline comment (only when preceded by whitespace, to
		// avoid mangling values containing # or ;)
		val = stripInlineComment(val)
		val = strings.Trim(val, `"`)
		if key == "" {
			continue
		}
		fqk := key
		if section != "" {
			fqk = section + "." + key
		}
		out[fqk] = val
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// stripInlineComment removes a trailing "; comment" or "# comment" but only
// when the marker is preceded by whitespace, so legitimate values like
// "rgb(255;128;0)" or "C:\Path#Anchor" survive.
func stripInlineComment(s string) string {
	for i := 1; i < len(s); i++ {
		if (s[i] == ';' || s[i] == '#') && (s[i-1] == ' ' || s[i-1] == '\t') {
			return strings.TrimRight(s[:i], " \t")
		}
	}
	return s
}
