package app

import (
	"errors"
	"path/filepath"

	"github.com/cionz0/hdsort/internal/domain"
)

// Launcher opens a path in the operating system's default application.
// The use case does not own that call.
type Launcher interface {
	Open(path string) error
}

// OpenCopy asks the launcher to open path. The path it passes is cleaned.
// The session's decisions are left as they were, whether the launch works or not.
func OpenCopy(session domain.Session, launch Launcher, path string) (domain.Session, error) {
	if launch == nil {
		return session, errors.New("no launcher")
	}
	if err := launch.Open(filepath.Clean(path)); err != nil {
		return session, err
	}
	return session, nil
}
