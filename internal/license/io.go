package license

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Load reads a licence file.
func Load(path string) (File, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return File{}, err
	}
	return Parse(b)
}

// Parse reads a licence from JSON.
func Parse(b []byte) (File, error) {
	var f File
	if err := json.Unmarshal(b, &f); err != nil {
		return File{}, fmt.Errorf("license: %w", err)
	}
	return f, nil
}

// Save writes a licence file, replacing it in one step so a reader never
// sees half of it.
func Save(path string, f File) error {
	b, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	tmp := path + ".new"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

// InData is the licence path inside a data directory.
func InData(dir string) string { return filepath.Join(dir, FileName) }
