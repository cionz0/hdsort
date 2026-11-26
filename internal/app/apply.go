package app

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/cionz0/hdsort/internal/domain"
)

// Trasher moves one confirmed delete to the operating system Trash.
// The use case does not unlink the file itself.
type Trasher interface {
	Trash(path string) error
}

// Relocator performs a confirmed layout move and reads the file metadata
// apply stores. The use case calls Rename only after the domain check that
// the destination stays inside the scan root.
type Relocator interface {
	Rename(from, to string) error
	Stat(path string) (size int64, modTime time.Time, err error)
}

// PlanReport is what apply would do. Dry-run returns it and writes nothing.
type PlanReport struct {
	Trash   []string
	Moves   []domain.LayoutMove
	Upserts []string
}

type survivor struct {
	contentID domain.ContentID
	path      string
}

type prepared struct {
	report    PlanReport
	survivors []survivor
}

// Apply checks the plan, moves confirmed deletes to Trash, renames confirmed
// moves, and only then upserts survivors at their final paths.
// A failed check or a failed trash call leaves the library unchanged and the
// returned session open.
func Apply(session domain.Session, files []domain.FileRef, lib LibraryRepository, trash Trasher, disk Relocator) (domain.Session, error) {
	plan, err := prepare(session, files, lib)
	if err != nil {
		return session, err
	}
	for _, path := range plan.report.Trash {
		if err := trash.Trash(path); err != nil {
			return session, err
		}
	}
	for _, move := range plan.report.Moves {
		if !domain.PathInside(session.Root, move.Destination) {
			return session, fmt.Errorf("%w: %s", domain.ErrMoveOutsideRoot, move.Destination)
		}
		if err := disk.Rename(move.From, move.Destination); err != nil {
			return session, err
		}
	}
	now := time.Now()
	for _, item := range plan.survivors {
		size, modTime, err := disk.Stat(item.path)
		if err != nil {
			return session, err
		}
		if err := lib.Put(domain.LibraryEntry{
			ContentID: item.contentID,
			Path:      item.path,
			Size:      size,
			ModTime:   modTime,
			IndexedAt: now,
		}); err != nil {
			return session, err
		}
	}
	return domain.ApplyResult(session)
}

// DryRun runs the same plan check as Apply and returns the Trash paths,
// the confirmed moves, and the library upsert paths. It does not call Trash,
// does not rename, and does not write the library. The session stays open.
// A failed check returns the same error Apply would return.
func DryRun(session domain.Session, files []domain.FileRef, lib LibraryRepository, trash Trasher, disk Relocator) (PlanReport, error) {
	plan, err := prepare(session, files, lib)
	if err != nil {
		return PlanReport{}, err
	}
	// trash and disk are part of the apply shell. Dry-run must not use them.
	_ = trash
	_ = disk
	return plan.report, nil
}

func prepare(session domain.Session, files []domain.FileRef, lib LibraryRepository) (prepared, error) {
	if session.Status() != domain.StatusOpen {
		return prepared{}, domain.ErrSessionNotOpen
	}
	library, err := lib.List()
	if err != nil {
		return prepared{}, err
	}
	trash := trashPaths(session)
	if err := domain.CheckPlan(domain.Plan{
		Files:     files,
		Trash:     trash,
		Decisions: session.Decisions,
		Library:   library,
	}); err != nil {
		return prepared{}, err
	}
	moves := confirmedMoves(session)
	survivors, err := survivorsOf(session, files, moves)
	if err != nil {
		return prepared{}, err
	}
	upserts := make([]string, len(survivors))
	for i, item := range survivors {
		upserts[i] = item.path
	}
	return prepared{
		report: PlanReport{
			Trash:   trash,
			Moves:   moves,
			Upserts: upserts,
		},
		survivors: survivors,
	}, nil
}

func trashPaths(session domain.Session) []string {
	var paths []string
	seen := map[string]struct{}{}
	add := func(path string) {
		key := filepath.Clean(path)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		paths = append(paths, path)
	}
	for _, group := range session.Exact {
		rec, ok := group.Recommendation()
		if !ok {
			continue
		}
		for _, path := range rec.Trash {
			add(path)
		}
	}
	for _, decision := range session.Decisions {
		if decision.Choice == domain.ChoiceDelete {
			add(decision.Path)
		}
	}
	return paths
}

func confirmedMoves(session domain.Session) []domain.LayoutMove {
	var moves []domain.LayoutMove
	for _, move := range session.Moves {
		if move.Confirmed {
			moves = append(moves, move)
		}
	}
	return moves
}

func survivorsOf(session domain.Session, files []domain.FileRef, moves []domain.LayoutMove) ([]survivor, error) {
	var paths []string
	seenPath := map[string]struct{}{}
	addPath := func(path string) {
		if choice, ok := choiceFor(session.Decisions, path); ok && (choice == domain.ChoiceDelete || choice == domain.ChoiceSkip) {
			return
		}
		key := filepath.Clean(path)
		if _, ok := seenPath[key]; ok {
			return
		}
		seenPath[key] = struct{}{}
		paths = append(paths, path)
	}
	for _, decision := range session.Decisions {
		if decision.Choice == domain.ChoiceKeep {
			addPath(decision.Path)
		}
	}
	for _, group := range session.Exact {
		rec, ok := group.Recommendation()
		if !ok {
			continue
		}
		addPath(rec.Keeper)
	}

	var survivors []survivor
	seenID := map[domain.ContentID]struct{}{}
	for _, path := range paths {
		id, ok := contentIDFor(session, files, path)
		if !ok {
			return nil, fmt.Errorf("survivor %s has no content ID", path)
		}
		if _, ok := seenID[id]; ok {
			continue
		}
		seenID[id] = struct{}{}
		survivors = append(survivors, survivor{
			contentID: id,
			path:      finalPath(path, moves),
		})
	}
	return survivors, nil
}

func finalPath(path string, moves []domain.LayoutMove) string {
	key := filepath.Clean(path)
	for _, move := range moves {
		if filepath.Clean(move.From) == key {
			return move.Destination
		}
	}
	return path
}

func choiceFor(decisions []domain.Decision, path string) (domain.Choice, bool) {
	key := filepath.Clean(path)
	var (
		choice domain.Choice
		found  bool
	)
	for _, decision := range decisions {
		if filepath.Clean(decision.Path) == key {
			choice = decision.Choice
			found = true
		}
	}
	return choice, found
}

func contentIDFor(session domain.Session, files []domain.FileRef, path string) (domain.ContentID, bool) {
	key := filepath.Clean(path)
	for _, file := range files {
		if filepath.Clean(file.Path) == key && file.ContentID != "" {
			return file.ContentID, true
		}
	}
	for _, group := range session.Exact {
		for _, member := range group.Members() {
			if filepath.Clean(member.Path) == key && member.ContentID != "" {
				return member.ContentID, true
			}
		}
		rec, ok := group.Recommendation()
		if ok && filepath.Clean(rec.Keeper) == key {
			members := group.Members()
			if len(members) > 0 && members[0].ContentID != "" {
				return members[0].ContentID, true
			}
		}
	}
	return "", false
}
