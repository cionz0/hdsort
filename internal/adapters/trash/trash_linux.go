//go:build linux

package trash

import (
	"os"
	"path/filepath"
)

// NewPlatform returns the Freedesktop adapter for the home Trash directory.
// That directory is $XDG_DATA_HOME/Trash, or ~/.local/share/Trash when
// XDG_DATA_HOME is unset. Tests pass NewFreedesktop a temporary directory
// instead of calling NewPlatform.
func NewPlatform() (*Freedesktop, error) {
	base := os.Getenv("XDG_DATA_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		base = filepath.Join(home, ".local", "share")
	}
	return NewFreedesktop(filepath.Join(base, "Trash")), nil
}
