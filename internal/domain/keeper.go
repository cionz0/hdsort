package domain

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// generatedCopy matches "name (1)" at the end of the base name.
var generatedCopy = regexp.MustCompile(` \(\d+\)$`)

// KeeperPolicy is the Strategy that ranks an exact group.
// DefaultKeeper is the only policy. A second policy needs an ADR.
//
// Pattern: Strategy (Gamma, Helm, Johnson, Vlissides, 1994).
// See docs/patterns.md.
type KeeperPolicy func(members []FileRef, library *LibraryEntry) (KeeperChoice, error)

// KeeperChoice is the policy result. Present is false when the signals tie,
// which is not a recommendation with an empty reason. Moves is empty: the
// default policy never proposes a layout move.
type KeeperChoice struct {
	Recommendation Recommendation
	Present        bool
	Moves          []LayoutMove
}

// DefaultKeeper ranks members that already share a content ID.
// library is optional; a nil or invalid entry is ignored. The library path
// wins when it records this content ID. Otherwise one plain name wins over
// generated copies. If those ranks tie, the newer modification time wins,
// and the reason text says that the time was only a tie-break. Equal times
// leave the recommendation absent.
func DefaultKeeper(members []FileRef, library *LibraryEntry) (KeeperChoice, error) {
	if len(members) < 2 {
		return KeeperChoice{}, fmt.Errorf("%w: got %d", ErrExactGroupMembers, len(members))
	}
	id := members[0].ContentID
	for _, member := range members[1:] {
		if member.ContentID != id {
			return KeeperChoice{}, ErrExactGroupContent
		}
	}
	if library != nil && library.Valid() && library.ContentID == id {
		return keeperChoice(library.Path, trashExcept(members, library.Path), ReasonLibrarySurvivor, "")
	}

	var plain []FileRef
	for _, member := range members {
		if !isGeneratedCopy(member.Path) {
			plain = append(plain, member)
		}
	}
	if len(plain) == 1 {
		return keeperChoice(plain[0].Path, trashExcept(members, plain[0].Path), ReasonPlainName, "")
	}

	candidates := members
	if len(plain) > 1 {
		candidates = plain
	}
	best, unique := uniqueNewest(candidates)
	if !unique {
		return KeeperChoice{}, nil
	}
	return keeperChoice(
		best.Path,
		trashExcept(members, best.Path),
		ReasonNewerMtimeTiebreak,
		"The newer modification time was only a tie-break.",
	)
}

func keeperChoice(keeper string, trash []string, reason ReasonCode, text string) (KeeperChoice, error) {
	rec, err := NewRecommendation(keeper, trash, reason)
	if err != nil {
		return KeeperChoice{}, err
	}
	rec.Text = text
	return KeeperChoice{Recommendation: rec, Present: true}, nil
}

func trashExcept(members []FileRef, keeper string) []string {
	keeper = filepath.Clean(keeper)
	trash := make([]string, 0, len(members))
	for _, member := range members {
		if filepath.Clean(member.Path) == keeper {
			continue
		}
		trash = append(trash, member.Path)
	}
	return trash
}

func isGeneratedCopy(path string) bool {
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	if strings.HasPrefix(base, "Copy of ") {
		return true
	}
	if strings.Contains(base, "conflicted copy") {
		return true
	}
	return generatedCopy.MatchString(base)
}

func uniqueNewest(files []FileRef) (FileRef, bool) {
	best := files[0]
	count := 1
	for _, file := range files[1:] {
		if file.ModTime.After(best.ModTime) {
			best = file
			count = 1
			continue
		}
		if file.ModTime.Equal(best.ModTime) {
			count++
		}
	}
	return best, count == 1
}
