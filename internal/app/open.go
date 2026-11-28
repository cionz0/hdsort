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

// ShownChoice is the radio selection for path. A recorded decision wins.
// Otherwise a recommendation selects keep for the keeper and delete for each
// path in deletes. No recommendation selects nothing, which is how a tie and
// every similar group stay clear.
func ShownChoice(session domain.Session, hasRecommendation bool, keeper string, deletes []FileSize, path string) domain.Choice {
	key := filepath.Clean(path)
	for _, decision := range session.Decisions {
		if filepath.Clean(decision.Path) == key {
			return decision.Choice
		}
	}
	if !hasRecommendation {
		return ""
	}
	if filepath.Clean(keeper) == key {
		return domain.ChoiceKeep
	}
	for _, item := range deletes {
		if filepath.Clean(item.Path) == key {
			return domain.ChoiceDelete
		}
	}
	return ""
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
