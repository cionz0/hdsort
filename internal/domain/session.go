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

// RecordDecision stores a keep, delete, or skip for path on an open session.
// A later decision for the same path replaces the earlier one. The input
// session is left unchanged.
func RecordDecision(s Session, path string, choice Choice) (Session, error) {
	if s.status != StatusOpen {
		return s, ErrSessionNotOpen
	}
	decision, err := NewDecision(path, choice)
	if err != nil {
		return s, err
	}
	next := s
	next.Decisions = append([]Decision(nil), s.Decisions...)
	key := filepath.Clean(decision.Path)
	for i, existing := range next.Decisions {
		if filepath.Clean(existing.Path) == key {
			next.Decisions[i] = decision
			return next, nil
		}
	}
	next.Decisions = append(next.Decisions, decision)
	return next, nil
}

// ConfirmMove marks the proposal whose source is from. The move still does
// not run until apply. The input session is left unchanged.
func ConfirmMove(s Session, from string) (Session, error) {
	if s.status != StatusOpen {
		return s, ErrSessionNotOpen
	}
	if from == "" {
		return s, fmt.Errorf("%w: move", ErrEmptyPath)
	}
	key := filepath.Clean(from)
	next := s
	next.Moves = append([]LayoutMove(nil), s.Moves...)
	for i, move := range next.Moves {
		if filepath.Clean(move.From) == key {
			next.Moves[i].Confirmed = true
			return next, nil
		}
	}
	return s, fmt.Errorf("session has no move from %s", from)
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
