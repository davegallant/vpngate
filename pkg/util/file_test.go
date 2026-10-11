package util

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteFileAtomic(t *testing.T) {
	t.Run("writes content with requested permissions", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "out.txt")

		err := WriteFileAtomic(path, []byte("hello"), 0o600)
		require.NoError(t, err)

		got, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Equal(t, "hello", string(got))

		info, err := os.Stat(path)
		require.NoError(t, err)
		if runtime.GOOS == "windows" {
			assert.Equal(t, os.FileMode(0o666), info.Mode().Perm())
		} else {
			assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
		}
	})

	t.Run("applies read-only permissions on Windows", func(t *testing.T) {
		if runtime.GOOS != "windows" {
			t.Skip("Windows uses the owner-writable bit to set the read-only attribute")
		}

		path := filepath.Join(t.TempDir(), "out.txt")
		require.NoError(t, WriteFileAtomic(path, []byte("read only"), 0o400))

		info, err := os.Stat(path)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o444), info.Mode().Perm())
	})

	t.Run("overwrites existing file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "out.txt")
		require.NoError(t, os.WriteFile(path, []byte("old"), 0o600))

		err := WriteFileAtomic(path, []byte("new"), 0o600)
		require.NoError(t, err)

		got, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Equal(t, "new", string(got))
	})

	t.Run("leaves no temp files behind on success", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "out.txt")

		require.NoError(t, WriteFileAtomic(path, []byte("x"), 0o600))

		entries, err := os.ReadDir(dir)
		require.NoError(t, err)
		assert.Len(t, entries, 1)
		assert.Equal(t, "out.txt", entries[0].Name())
	})

	t.Run("returns error when directory does not exist", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "nonexistent", "out.txt")
		err := WriteFileAtomic(path, []byte("x"), 0o600)
		assert.Error(t, err)
	})
}
