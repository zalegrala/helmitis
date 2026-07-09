package stamp

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
)

// Write writes all files under outDir, creating parent directories.
func Write(files []File, outDir string) error {
	for _, f := range files {
		full := filepath.Join(outDir, f.Path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(full, f.Content, 0o644); err != nil {
			return err
		}
	}
	return nil
}

// Check compares freshly built files against what is on disk under outDir.
// Returns drift=true and a list of human-readable differences if any file is
// missing or has different content.
func Check(files []File, outDir string) (bool, []string) {
	var diffs []string
	for _, f := range files {
		full := filepath.Join(outDir, f.Path)
		onDisk, err := os.ReadFile(full)
		if err != nil {
			diffs = append(diffs, fmt.Sprintf("%s: missing on disk", f.Path))
			continue
		}
		if !bytes.Equal(onDisk, f.Content) {
			diffs = append(diffs, fmt.Sprintf("%s: content differs", f.Path))
		}
	}
	return len(diffs) > 0, diffs
}
