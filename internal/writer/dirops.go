package writer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ValidatePath checks that relPath is within collectionRoot (prevents path traversal).
func ValidatePath(collectionRoot, relPath string) error {
	absRoot, err := filepath.Abs(collectionRoot)
	if err != nil {
		return fmt.Errorf("resolving root %q: %w", collectionRoot, err)
	}
	absPath, err := filepath.Abs(filepath.Join(collectionRoot, relPath))
	if err != nil {
		return fmt.Errorf("resolving path %q: %w", relPath, err)
	}
	rel, err := filepath.Rel(absRoot, absPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return fmt.Errorf("path %q traverses outside collection root", relPath)
	}
	return nil
}

// IsHTTPFile returns true if path has a .http extension with at least one character before it.
func IsHTTPFile(path string) bool {
	base := filepath.Base(path)
	return strings.HasSuffix(base, ".http") && len(base) > len(".http")
}

// CreateDirectory creates a new subdirectory named `name` inside collectionRoot.
func CreateDirectory(collectionRoot, name string) error {
	if err := ValidatePath(collectionRoot, name); err != nil {
		return err
	}
	return os.MkdirAll(filepath.Join(collectionRoot, name), 0755)
}

// CreateHTTPFile creates a new .http file at relPath (relative to collectionRoot).
// Creates any parent directories as needed. Writes a minimal comment header.
func CreateHTTPFile(collectionRoot, relPath string) error {
	if err := ValidatePath(collectionRoot, relPath); err != nil {
		return err
	}
	abs := filepath.Join(collectionRoot, relPath)
	if err := os.MkdirAll(filepath.Dir(abs), 0755); err != nil {
		return err
	}
	content := "# New Request Collection\n"
	return os.WriteFile(abs, []byte(content), 0644)
}

// RenameEntry renames a file or directory from oldRelPath to newRelPath.
// Both paths must be inside collectionRoot.
func RenameEntry(collectionRoot, oldRelPath, newRelPath string) error {
	if err := ValidatePath(collectionRoot, oldRelPath); err != nil {
		return err
	}
	if err := ValidatePath(collectionRoot, newRelPath); err != nil {
		return err
	}
	oldAbs := filepath.Join(collectionRoot, oldRelPath)
	newAbs := filepath.Join(collectionRoot, newRelPath)
	if err := os.MkdirAll(filepath.Dir(newAbs), 0755); err != nil {
		return err
	}
	return os.Rename(oldAbs, newAbs)
}

// MoveEntry moves a file or directory from srcRelPath to dstRelPath.
// Both must be inside collectionRoot.
func MoveEntry(collectionRoot, srcRelPath, dstRelPath string) error {
	return RenameEntry(collectionRoot, srcRelPath, dstRelPath)
}

// DeleteEntry removes a file or directory at relPath.
// For directories, uses recursive removal.
// Validates that path is inside collectionRoot before deletion.
func DeleteEntry(collectionRoot, relPath string) error {
	if err := ValidatePath(collectionRoot, relPath); err != nil {
		return err
	}
	return os.RemoveAll(filepath.Join(collectionRoot, relPath))
}
