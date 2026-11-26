package domain

import (
	"fmt"
	"path/filepath"
)

// Plan is the input to the pure plan check. Trash lists paths the plan would
// move to Trash. Files are the known copies, grouped by content ID for the
// last-copy rule. Library holds survivors that already exist outside this plan.
type Plan struct {
	Files     []FileRef
	Trash     []string
	Decisions []Decision
	Library   []LibraryEntry
}

// CheckPlan rejects a plan that would trash a path without a delete decision,
// or that would trash every known copy of a content ID the library does not keep.
//
// A valid library entry counts as a kept copy when its path is not itself in
// the trash list. The check reads its arguments and does not touch the disk.
func CheckPlan(p Plan) error {
	byPath := make(map[string]Choice, len(p.Decisions))
	for _, d := range p.Decisions {
		if err := d.validate(); err != nil {
			return err
		}
		key := filepath.Clean(d.Path)
		if prev, ok := byPath[key]; ok && prev != d.Choice {
			return fmt.Errorf("%w: conflicting choices for %s", ErrDecisionChoice, d.Path)
		}
		byPath[key] = d.Choice
	}

	trash := make(map[string]struct{}, len(p.Trash))
	for _, path := range p.Trash {
		if path == "" {
			return fmt.Errorf("%w: empty trash path", ErrUnconfirmedDelete)
		}
		key := filepath.Clean(path)
		if byPath[key] != ChoiceDelete {
			return fmt.Errorf("%w: %s", ErrUnconfirmedDelete, path)
		}
		trash[key] = struct{}{}
	}

	return checkLastCopy(p.Files, p.Library, trash)
}

func checkLastCopy(files []FileRef, library []LibraryEntry, trash map[string]struct{}) error {
	byID := make(map[ContentID][]string)
	var order []ContentID
	for _, f := range files {
		if f.ContentID == "" || f.Path == "" {
			continue
		}
		key := filepath.Clean(f.Path)
		paths, ok := byID[f.ContentID]
		if !ok {
			order = append(order, f.ContentID)
		}
		byID[f.ContentID] = append(paths, key)
	}

	for _, id := range order {
		if copyRemains(byID[id], library, id, trash) {
			continue
		}
		return fmt.Errorf("%w: %s", ErrLastCopy, id)
	}
	return nil
}

// copyRemains reports whether at least one known copy of id stays out of Trash.
// Known copies are the file paths in the plan plus each valid library entry
// for that content ID.
func copyRemains(paths []string, library []LibraryEntry, id ContentID, trash map[string]struct{}) bool {
	for _, path := range paths {
		if _, trashed := trash[path]; !trashed {
			return true
		}
	}
	for _, entry := range library {
		if entry.ContentID != id || !entry.Valid() {
			continue
		}
		if _, trashed := trash[filepath.Clean(entry.Path)]; !trashed {
			return true
		}
	}
	return false
}
