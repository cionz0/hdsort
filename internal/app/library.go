package app

import (
	"errors"
	"time"

	"github.com/cionz0/hdsort/internal/domain"
)

// ErrSessionNotFound is returned when a session id is not in the repository.
var ErrSessionNotFound = errors.New("session not found")

// LibraryRepository is the survivor collection.
//
// Pattern: Repository (Evans, 2003). See docs/patterns.md.
type LibraryRepository interface {
	Put(entry domain.LibraryEntry) error
	List() ([]domain.LibraryEntry, error)
}

// SessionRepository stores and loads one scan session.
//
// Pattern: Repository (Evans, 2003). See docs/patterns.md.
type SessionRepository interface {
	Save(session domain.Session) (id string, err error)
	Load(id string) (domain.Session, error)
	Replace(id string, session domain.Session) error
}

// FileFacts is the metadata reconcile reads for one path.
// Exists is false when the path is missing or is not a regular file.
type FileFacts struct {
	Exists  bool
	Size    int64
	ModTime time.Time
}

// ReconcileDisk is the filesystem port reconcile uses.
type ReconcileDisk interface {
	FileReader
	Facts(path string) (FileFacts, error)
}

// ContentIDSource hashes one known file in full.
// Reconcile uses this instead of the staged scan hasher: a single large file
// would otherwise be skipped as a size-bucket of one.
type ContentIDSource interface {
	ContentID(path string, source FileReader) (domain.ContentID, error)
}

// Reconcile updates library entries whose path is inside root.
// A missing file or a different content ID marks the entry stale and leaves
// the stored content ID and path as they were. An unchanged file refreshes
// size, modification time, and indexed time. Entries outside root are not
// modified.
func Reconcile(root string, lib LibraryRepository, disk ReconcileDisk, ids ContentIDSource) error {
	entries, err := lib.List()
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !domain.PathInside(root, entry.Path) {
			continue
		}
		facts, err := disk.Facts(entry.Path)
		if err != nil {
			return err
		}
		if !facts.Exists {
			entry.Stale = true
			if err := lib.Put(entry); err != nil {
				return err
			}
			continue
		}
		id, err := ids.ContentID(entry.Path, disk)
		if err != nil {
			return err
		}
		if id != entry.ContentID {
			entry.Stale = true
			if err := lib.Put(entry); err != nil {
				return err
			}
			continue
		}
		entry.Stale = false
		entry.Size = facts.Size
		entry.ModTime = facts.ModTime
		entry.IndexedAt = time.Now()
		if err := lib.Put(entry); err != nil {
			return err
		}
	}
	return nil
}

// Match groups the scanned files and ranks each exact group with DefaultKeeper.
// A valid library entry for that content ID is passed to the policy. A stale
// entry is not. Library paths are not added as group members.
func Match(files []domain.FileRef, lib LibraryRepository) ([]domain.ExactGroup, error) {
	entries, err := lib.List()
	if err != nil {
		return nil, err
	}
	byID := make(map[domain.ContentID]domain.LibraryEntry, len(entries))
	for _, entry := range entries {
		if entry.Valid() {
			byID[entry.ContentID] = entry
		}
	}
	groups, err := domain.GroupExact(files)
	if err != nil {
		return nil, err
	}
	ranked := make([]domain.ExactGroup, 0, len(groups))
	for _, group := range groups {
		members := group.Members()
		var library *domain.LibraryEntry
		if entry, ok := byID[members[0].ContentID]; ok {
			library = &entry
		}
		choice, err := domain.DefaultKeeper(members, library)
		if err != nil {
			return nil, err
		}
		if choice.Present {
			group, err = group.WithRecommendation(choice.Recommendation)
			if err != nil {
				return nil, err
			}
		}
		ranked = append(ranked, group)
	}
	return ranked, nil
}
