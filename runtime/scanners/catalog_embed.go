package scanners

import (
	"embed"
	"fmt"
	"io/fs"
	"strings"
)

//go:embed catalog/*.yaml
var catalogFS embed.FS

// BuiltIns returns the shipped scanner catalog.
func BuiltIns() ([]Definition, error) {
	entries, err := fs.ReadDir(catalogFS, "catalog")
	if err != nil {
		return nil, err
	}
	out := make([]Definition, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || (!strings.HasSuffix(e.Name(), ".yaml") && !strings.HasSuffix(e.Name(), ".yml")) {
			continue
		}
		data, err := catalogFS.ReadFile("catalog/" + e.Name())
		if err != nil {
			return nil, err
		}
		def, err := ParseDefinition(data)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", e.Name(), err)
		}
		out = append(out, def)
	}
	return out, nil
}

// LookupBuiltIn finds a built-in scanner by name.
func LookupBuiltIn(name string) (Definition, error) {
	all, err := BuiltIns()
	if err != nil {
		return Definition{}, err
	}
	for _, d := range all {
		if d.Name == name {
			return d, nil
		}
	}
	return Definition{}, fmt.Errorf("unknown built-in scanner %q", name)
}
