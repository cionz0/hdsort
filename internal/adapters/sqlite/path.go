package sqlite

import (
	"os"
	"path/filepath"
	"runtime"
)

// DefaultPath is the ADR-0008 library file for this operating system.
// Sessions are stored in the same file, which lives in that directory.
func DefaultPath() (string, error) {
	switch runtime.GOOS {
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, "Library", "Application Support", "hdsort", "library.sqlite"), nil
	case "windows":
		base := os.Getenv("AppData")
		if base == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			base = filepath.Join(home, "AppData", "Roaming")
		}
		return filepath.Join(base, "hdsort", "library.sqlite"), nil
	default:
		base := os.Getenv("XDG_DATA_HOME")
		if base == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			base = filepath.Join(home, ".local", "share")
		}
		return filepath.Join(base, "hdsort", "library.sqlite"), nil
	}
}
