package app

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/cionz0/hdsort/internal/domain"
)

// ErrFolderOutsideRoot means the owner asked to trash a directory that is
// not the scan root and is not inside it.
var ErrFolderOutsideRoot = errors.New("folder is outside the scan root")

// TrashFolder moves folder to Trash in one call. It does not walk the tree,
// does not unlink, does not run the session plan, and does not write the
// library. A failed trash leaves the session unchanged. After a success,
// decisions whose path is inside folder are dropped. The session stays open.
func TrashFolder(session domain.Session, folder string, trash Trasher) (domain.Session, error) {
	if session.Status() != domain.StatusOpen {
		return session, domain.ErrSessionNotOpen
	}
	folder = filepath.Clean(folder)
	if !domain.PathInside(session.Root, folder) {
		return session, fmt.Errorf("%w: %s", ErrFolderOutsideRoot, folder)
	}
	if err := trash.Trash(folder); err != nil {
		return session, err
	}
	next := session
	next.Decisions = nil
	for _, decision := range session.Decisions {
		if domain.PathInside(folder, decision.Path) {
			continue
		}
		next.Decisions = append(next.Decisions, decision)
	}
	return next, nil
}

// DropTrashed removes members that lie inside folder. A set with none left
// is dropped. A set that still has a member is filed under the first
// remaining member's directory.
func DropTrashed(rows []FolderRow, folder string) []FolderRow {
	folder = filepath.Clean(folder)
	out := make([]FolderRow, 0, len(rows))
	for _, row := range rows {
		kept := make([]FileSize, 0, len(row.Members))
		for _, member := range row.Members {
			if domain.PathInside(folder, member.Path) {
				continue
			}
			kept = append(kept, member)
		}
		if len(kept) == 0 {
			continue
		}
		row.Members = kept
		row.Folder = filepath.Dir(filepath.Clean(kept[0].Path))
		if row.HasRecommendation && domain.PathInside(folder, row.Keeper) {
			row.HasRecommendation = false
			row.Keeper = ""
			row.Deletes = nil
		} else if len(row.Deletes) > 0 {
			deleted := make([]FileSize, 0, len(row.Deletes))
			for _, item := range row.Deletes {
				if !domain.PathInside(folder, item.Path) {
					deleted = append(deleted, item)
				}
			}
			row.Deletes = deleted
		}
		out = append(out, row)
	}
	return out
}
