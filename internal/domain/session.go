package domain

import "path/filepath"

// SessionStatus is the lifecycle of one scan session.
type SessionStatus string

const (
	StatusOpen    SessionStatus = "open"
	StatusApplied SessionStatus = "applied"
)

// Session is one scan: root, groups, decisions, layout moves, and status.
// Status starts at open. ApplyResult is the only transition, from open to applied.
type Session struct {
	Root      string
	Exact     []ExactGroup
	Similar   []SimilarGroup
	Decisions []Decision
	Moves     []LayoutMove

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
