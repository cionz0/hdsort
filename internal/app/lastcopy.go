package app

import (
	"path/filepath"
	"sort"

	"github.com/cionz0/hdsort/internal/domain"
)

// LastCopyGroups lists each file whose every known copy would be trashed
// and that the library does not keep. Each group is that file's paths.
// The content ID is not part of the result.
func LastCopyGroups(session domain.Session, files []domain.FileRef, lib LibraryRepository) ([][]string, error) {
	if session.Status() != domain.StatusOpen {
		return nil, domain.ErrSessionNotOpen
	}
	library, err := lib.List()
	if err != nil {
		return nil, err
	}
	return domain.LostCopyGroups(domain.Plan{
		Files:     files,
		Trash:     trashPaths(session),
		Decisions: session.Decisions,
		Library:   library,
	}), nil
}

// ApplyIfLastCopiesConfirmed trashes only when every last-copy group is
// named in confirmed. A partial confirmation trashes nothing. A plan with
// no last-copy group uses Apply. Any other rejection still stops the plan.
func ApplyIfLastCopiesConfirmed(session domain.Session, files []domain.FileRef, lib LibraryRepository, trash Trasher, disk Relocator, confirmed [][]string) (domain.Session, error) {
	groups, err := LastCopyGroups(session, files, lib)
	if err != nil {
		return session, err
	}
	if len(groups) == 0 {
		return Apply(session, files, lib, trash, disk)
	}
	for _, group := range groups {
		if !pathsConfirmed(group, confirmed) {
			return session, domain.ErrLastCopy
		}
	}
	return apply(session, files, lib, trash, disk, true)
}

// UnconfirmedLastCopies returns the groups that confirmed does not cover.
func UnconfirmedLastCopies(groups, confirmed [][]string) [][]string {
	var pending [][]string
	for _, group := range groups {
		if !pathsConfirmed(group, confirmed) {
			pending = append(pending, group)
		}
	}
	return pending
}

// WithoutLastCopy drops every confirmed group that contains path.
// A later choice for that file must be confirmed again.
func WithoutLastCopy(confirmed [][]string, path string) [][]string {
	key := filepath.Clean(path)
	out := make([][]string, 0, len(confirmed))
	for _, group := range confirmed {
		drop := false
		for _, item := range group {
			if filepath.Clean(item) == key {
				drop = true
				break
			}
		}
		if !drop {
			out = append(out, group)
		}
	}
	return out
}

func pathsConfirmed(group []string, confirmed [][]string) bool {
	for _, item := range confirmed {
		if samePaths(group, item) {
			return true
		}
	}
	return false
}

func samePaths(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	aa := cleanedSorted(a)
	bb := cleanedSorted(b)
	for i := range aa {
		if aa[i] != bb[i] {
			return false
		}
	}
	return true
}

func cleanedSorted(paths []string) []string {
	out := make([]string, len(paths))
	for i, path := range paths {
		out[i] = filepath.Clean(path)
	}
	sort.Strings(out)
	return out
}
