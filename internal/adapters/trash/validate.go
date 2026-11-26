package trash

import "strings"

// Validate rejects a path the Trash adapters must not send to the operating
// system. Each adapter calls it before the platform API or the Freedesktop move.
func Validate(path string) error {
	if strings.TrimSpace(path) == "" {
		return ErrEmptyPath
	}
	return nil
}
