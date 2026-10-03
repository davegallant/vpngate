package util

import (
	"os"
	"path/filepath"
)

// WriteFileAtomic writes data to path atomically: the content goes to a
// temp file in the same directory first, then is renamed over the
// destination, so a crash mid-write can never leave a half-written file
// behind for the next reader to parse.
func WriteFileAtomic(path string, data []byte, perm os.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	// Best effort: the rename below removes the temp file on success.
	defer func() { _ = os.Remove(tmpName) }()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}
