package app

import (
	"path/filepath"
	"strings"

	"github.com/cionz0/hdsort/internal/domain"
)

// Row kinds in the one folder list.
const (
	RowFile    = "file"
	RowExact   = "exact"
	RowSimilar = "similar"
)

// FolderRow is one line in the folder list. A path is on at most one row.
type FolderRow struct {
	Key               string
	Folder            string
	Title             string
	Status            string
	Stage             HashStage
	Size              int64
	Kind              string
	Members           []FileSize
	HasRecommendation bool
	Keeper            string
	Deletes           []FileSize
	Text              string
	Freed             int64
}

// FolderBoard is the single list the window paints. Status changes update
// the row that already holds the path.
type FolderBoard struct {
	Rows []FolderRow
}

// Apply folds one progress event into the row for that path.
func (b *FolderBoard) Apply(ev ProgressEvent) {
	switch ev.Kind {
	case EventQueue:
		if b.find(ev.Path) >= 0 {
			return
		}
		b.Rows = append(b.Rows, fileRow(ev.Path, ev.Size, "queued", ""))
	case EventHashing:
		i := b.ensure(ev.Path, ev.Size)
		b.Rows[i].Status = "reading"
		b.Rows[i].Stage = ev.Stage
		if ev.Size != 0 {
			b.Rows[i].Size = ev.Size
		}
	case EventProcessed:
		if ev.Outcome == OutcomeUnique || ev.Outcome == OutcomeSymlink || ev.Outcome == OutcomeDataless {
			b.remove(ev.Path)
			break
		}
		i := b.ensure(ev.Path, ev.Size)
		if ev.Outcome != "" {
			b.Rows[i].Status = ev.Outcome
		}
		if ev.Size != 0 {
			b.Rows[i].Size = ev.Size
		}
	}
	b.order()
}

// UseReview replaces the list with one row per exact or similar group.
// A file that is not in a group is not listed. The group is filed under
// the suggested keeper's directory, or the first member's directory when
// the group has no recommendation.
func (b *FolderBoard) UseReview(model ReviewModel) {
	var groups []FolderRow
	for _, group := range model.Exact {
		groups = append(groups, exactGroupRow(group))
	}
	for _, group := range model.Similar {
		groups = append(groups, similarGroupRow(group))
	}
	b.Rows = groups
	b.order()
}

// Contains reports how many rows include path.
func (b FolderBoard) Contains(path string) int {
	key := filepath.Clean(path)
	n := 0
	for _, row := range b.Rows {
		for _, member := range row.Members {
			if filepath.Clean(member.Path) == key {
				n++
				break
			}
		}
	}
	return n
}

// SkipSet records skip for every path. It records no delete.
func SkipSet(session domain.Session, paths []string) (domain.Session, error) {
	next := session
	for _, path := range paths {
		var err error
		next, err = domain.RecordDecision(next, path, domain.ChoiceSkip)
		if err != nil {
			return session, err
		}
	}
	return next, nil
}

// ApplySameTypeInFolder keeps every member of a matching exact group that
// lives in folder and deletes members that live elsewhere. openPath decides
// the type: JPEG, PNG, and GIF are exact images, and every other extension
// is an exact file. Similar groups are left alone. A group with no member
// in folder is left alone.
func ApplySameTypeInFolder(session domain.Session, folder, openPath string) (domain.Session, error) {
	folder = filepath.Clean(folder)
	wantImage := isImage(openPath)
	next := session
	for _, group := range session.Exact {
		members := group.Members()
		if len(members) == 0 || isImage(members[0].Path) != wantImage {
			continue
		}
		inside := false
		for _, member := range members {
			if filepath.Clean(filepath.Dir(member.Path)) == folder {
				inside = true
				break
			}
		}
		if !inside {
			continue
		}
		for _, member := range members {
			choice := domain.ChoiceDelete
			if filepath.Clean(filepath.Dir(member.Path)) == folder {
				choice = domain.ChoiceKeep
			}
			var err error
			next, err = domain.RecordDecision(next, member.Path, choice)
			if err != nil {
				return session, err
			}
		}
	}
	return next, nil
}

func (b *FolderBoard) remove(path string) {
	key := filepath.Clean(path)
	kept := b.Rows[:0]
	for _, row := range b.Rows {
		if row.Kind == RowFile && len(row.Members) > 0 && filepath.Clean(row.Members[0].Path) == key {
			continue
		}
		kept = append(kept, row)
	}
	b.Rows = kept
}

func (b *FolderBoard) find(path string) int {
	key := filepath.Clean(path)
	for i, row := range b.Rows {
		for _, member := range row.Members {
			if filepath.Clean(member.Path) == key {
				return i
			}
		}
	}
	return -1
}

func (b *FolderBoard) ensure(path string, size int64) int {
	if i := b.find(path); i >= 0 {
		return i
	}
	b.Rows = append(b.Rows, fileRow(path, size, "queued", ""))
	return len(b.Rows) - 1
}

func (b *FolderBoard) order() {
	rows := b.Rows
	for i := 1; i < len(rows); i++ {
		row := rows[i]
		j := i
		for j > 0 && folderLess(row, rows[j-1]) {
			rows[j] = rows[j-1]
			j--
		}
		rows[j] = row
	}
}

func folderLess(a, b FolderRow) bool {
	if a.Folder != b.Folder {
		return a.Folder < b.Folder
	}
	if a.Title != b.Title {
		return a.Title < b.Title
	}
	return a.Key < b.Key
}

func fileRow(path string, size int64, status, stage string) FolderRow {
	path = filepath.Clean(path)
	return FolderRow{
		Key:     path,
		Folder:  filepath.Dir(path),
		Title:   filepath.Base(path),
		Status:  status,
		Stage:   HashStage(stage),
		Size:    size,
		Kind:    RowFile,
		Members: []FileSize{{Path: path, Size: size}},
	}
}

func exactGroupRow(group ExactView) FolderRow {
	anchor := group.Members[0].Path
	if group.HasRecommendation && group.Keeper != "" {
		anchor = group.Keeper
	}
	anchor = filepath.Clean(anchor)
	return FolderRow{
		Key:               strings.Join(memberPaths(group.Members), "\n"),
		Folder:            filepath.Dir(anchor),
		Title:             filepath.Base(anchor),
		Status:            RowExact,
		Kind:              RowExact,
		Size:              sizeOfMember(group.Members, anchor),
		Members:           group.Members,
		HasRecommendation: group.HasRecommendation,
		Keeper:            group.Keeper,
		Deletes:           group.Deletes,
		Text:              group.Text,
		Freed:             group.Freed,
	}
}

func similarGroupRow(group SimilarView) FolderRow {
	anchor := filepath.Clean(group.Members[0].Path)
	return FolderRow{
		Key:     strings.Join(memberPaths(group.Members), "\n"),
		Folder:  filepath.Dir(anchor),
		Title:   filepath.Base(anchor),
		Status:  RowSimilar,
		Kind:    RowSimilar,
		Size:    group.Members[0].Size,
		Members: group.Members,
	}
}

func memberPaths(members []FileSize) []string {
	paths := make([]string, len(members))
	for i, member := range members {
		paths[i] = member.Path
	}
	return paths
}

func sizeOfMember(members []FileSize, path string) int64 {
	key := filepath.Clean(path)
	for _, member := range members {
		if filepath.Clean(member.Path) == key {
			return member.Size
		}
	}
	return 0
}

func isImage(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".jpg", ".jpeg", ".png", ".gif":
		return true
	default:
		return false
	}
}
