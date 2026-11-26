package domain

import (
	"fmt"
	"path/filepath"
)

// SessionStatus is the lifecycle of one scan session.
type SessionStatus string

const (
	StatusOpen    SessionStatus = "open"
	StatusApplied SessionStatus = "applied"
)

// SkipReason says why a path was listed on the session and not hashed.
type SkipReason string

const (
	SkipDataless SkipReason = "dataless"
	SkipSymlink  SkipReason = "symlink"
)

// Skipped is a path the scan did not hash.
// Target is the symlink destination when Reason is symlink, and empty otherwise.
type Skipped struct {
	Path   string
	Reason SkipReason
	Target string
}

// Session is one scan: root, groups, decisions, layout moves, skipped paths,
// and status. Status starts at open. ApplyResult moves an open session to
// applied. RestoreSession reloads a status that was already stored.
type Session struct {
	Root      string
	Exact     []ExactGroup
	Similar   []SimilarGroup
	Decisions []Decision
	Moves     []LayoutMove
	Skipped   []Skipped

	status SessionStatus
}

// NewSession returns a session at status open. The root is cleaned and is not
// required to exist; this package does not touch the filesystem.
func NewSession(root string) Session {
	return Session{
		Root:   filepath.Clean(root),
		status: StatusOpen,
	}
}

// Status returns open or applied.
func (s Session) Status() SessionStatus {
	return s.status
}

// RestoreSession returns a copy of s with status set.
// Loading a saved session uses this. ApplyResult remains the only way an
// open session becomes applied during a scan.
func RestoreSession(s Session, status SessionStatus) (Session, error) {
	switch status {
	case StatusOpen, StatusApplied:
		s.status = status
		return s, nil
	default:
		return Session{}, fmt.Errorf("%w: %q", ErrSessionNotOpen, status)
	}
}

// ApplyResult records a successful apply by moving an open session to applied.
// A session that is already applied returns an error. The input session is
// left unchanged; the applied value is the returned copy.
func ApplyResult(s Session) (Session, error) {
	if s.status != StatusOpen {
		return s, ErrSessionNotOpen
	}
	s.status = StatusApplied
	return s, nil
}
