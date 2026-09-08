package storage

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

var (
	ErrFileAlreadyExists = errors.New("file already exists")
)

// PartFilePath returns the temporary filename used while downloading a file.
func PartFilePath(finalPath string) string {
	return finalPath + ".part"
}

// FileExists checks whether a file exists and is not a directory.
func FileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// DirExists checks whether a directory exists.
func DirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// AtomicFinalize moves a completed part file to its final destination.
func AtomicFinalize(partPath, finalPath string, overwrite bool) error {
	if !FileExists(partPath) {
		return fmt.Errorf("part file %q does not exist", partPath)
	}

	if FileExists(finalPath) && !overwrite {
		return fmt.Errorf("%w: %q", ErrFileAlreadyExists, finalPath)
	}

	targetDir := filepath.Dir(finalPath)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory %q: %w", targetDir, err)
	}

	// Try atomic rename
	err := os.Rename(partPath, finalPath)
	if err == nil {
		return nil
	}

	// Fallback in case of cross-device link error
	if err := copyAndRemove(partPath, finalPath); err != nil {
		return fmt.Errorf("failed cross-device move from %q to %q: %w", partPath, finalPath, err)
	}

	return nil
}

// copyAndRemove copies data from src to dst and then removes src.
func copyAndRemove(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}

	in.Close()
	out.Close()

	return os.Remove(src)
}

// CleanupFile safely removes a file, ignoring if it doesn't exist.
func CleanupFile(path string) error {
	err := os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
