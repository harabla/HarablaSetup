//go:build !windows

package parse

import "fmt"

// ParseRegistry on non-Windows always returns an error so the verify layer
// can skip system-category checks during Mac dev. The mock probe layer
// covers any UI need for "what would this look like".
func ParseRegistry(path string) (map[string]string, error) {
	return nil, fmt.Errorf("registry: not supported on this OS (got %q)", path)
}
