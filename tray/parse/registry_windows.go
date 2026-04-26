//go:build windows

package parse

import (
	"fmt"
	"strings"

	"golang.org/x/sys/windows/registry"
)

// ParseRegistry reads a registry key and returns its values as a flat map.
//
// path format: "registry:HKEY_NAME\\Path\\To\\Key" (or "registry:HKCU\\..."
// using the standard HKxx prefixes).
//
// All value names under the key are returned; names get the basename of
// the key path as their prefix to keep the namespace tidy. For example,
// reading registry:HKCU\Control Panel\Mouse with values MouseSpeed=0 and
// SwapMouseButtons=0 returns:
//
//   "Mouse.MouseSpeed":      "0"
//   "Mouse.SwapMouseButtons": "0"
//
// Values are returned as strings regardless of underlying type (DWORD
// becomes its decimal string, REG_SZ as-is, REG_BINARY hex-encoded).
func ParseRegistry(path string) (map[string]string, error) {
	if !strings.HasPrefix(path, "registry:") {
		return nil, fmt.Errorf("registry: path must start with 'registry:' (got %q)", path)
	}
	rest := strings.TrimPrefix(path, "registry:")
	parts := strings.SplitN(rest, "\\", 2)
	if len(parts) < 2 {
		return nil, fmt.Errorf("registry: missing subkey in %q", path)
	}
	root, ok := rootKey(parts[0])
	if !ok {
		return nil, fmt.Errorf("registry: unknown root %q", parts[0])
	}
	subkey := parts[1]

	k, err := registry.OpenKey(root, subkey, registry.QUERY_VALUE)
	if err != nil {
		return nil, fmt.Errorf("registry: open %q: %w", path, err)
	}
	defer k.Close()

	names, err := k.ReadValueNames(-1)
	if err != nil {
		return nil, fmt.Errorf("registry: read names: %w", err)
	}

	prefix := lastSegment(subkey) + "."
	out := map[string]string{}
	for _, name := range names {
		val, err := readValueAsString(k, name)
		if err != nil {
			continue // skip unreadable values rather than failing the whole probe
		}
		out[prefix+name] = val
	}
	return out, nil
}

func rootKey(name string) (registry.Key, bool) {
	switch strings.ToUpper(name) {
	case "HKLM", "HKEY_LOCAL_MACHINE":
		return registry.LOCAL_MACHINE, true
	case "HKCU", "HKEY_CURRENT_USER":
		return registry.CURRENT_USER, true
	case "HKCR", "HKEY_CLASSES_ROOT":
		return registry.CLASSES_ROOT, true
	case "HKU", "HKEY_USERS":
		return registry.USERS, true
	}
	return 0, false
}

func lastSegment(p string) string {
	i := strings.LastIndexByte(p, '\\')
	if i < 0 {
		return p
	}
	return p[i+1:]
}

func readValueAsString(k registry.Key, name string) (string, error) {
	if s, _, err := k.GetStringValue(name); err == nil {
		return s, nil
	}
	if n, _, err := k.GetIntegerValue(name); err == nil {
		return fmt.Sprintf("%d", n), nil
	}
	if b, _, err := k.GetBinaryValue(name); err == nil {
		return fmt.Sprintf("%x", b), nil
	}
	return "", fmt.Errorf("registry: unsupported type for value %q", name)
}
