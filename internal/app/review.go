package app

import (
	"path/filepath"
	"strings"

	"github.com/cionz0/hdsort/internal/domain"
)

// BannerText is the synced-folder hint shown before a delete is confirmed.
const BannerText = "Deleting the local file can delete the synced copy. This check is only a path-segment hint."

// NothingChanged is the dry-run sentence. The plan writes nothing.
const NothingChanged = "Nothing was changed."

// FileSize is a path and its size in bytes.
type FileSize struct {
	Path string
	Size int64
}

// ExactView is one exact group as the window shows it.
type ExactView struct {
	HasRecommendation bool
	Keeper            string
	KeeperSize        int64
	Deletes           []FileSize
	Reason            string
	Text              string
	Freed             int64
	Members           []FileSize
}

// SimilarView is one similar group. It never carries a delete suggestion.
type SimilarView struct {
	Members           []FileSize
	HasRecommendation bool
}

// ReviewModel is the review portion of the window.
type ReviewModel struct {
	Banner  string
	Exact   []ExactView
	Similar []SimilarView
}

// SyncedBanner returns BannerText when a path segment of root is a known
// sync folder, and empty otherwise.
func SyncedBanner(root string) string {
	names := map[string]struct{}{
		"CloudStorage":     {},
		"Mobile Documents": {},
		"Dropbox":          {},
		"Google Drive":     {},
	}
	for _, part := range strings.Split(filepath.Clean(root), string(filepath.Separator)) {
		if _, ok := names[part]; ok {
			return BannerText
		}
	}
	return ""
}

// Review ranks exact groups with DefaultKeeper and leaves similar groups
// without a delete suggestion. library may be nil. File size is shown and
// is not a keeper reason.
func Review(session domain.Session, library []domain.LibraryEntry) ReviewModel {
	byID := map[domain.ContentID]domain.LibraryEntry{}
	for _, entry := range library {
		if entry.Valid() {
			byID[entry.ContentID] = entry
		}
	}
	model := ReviewModel{Banner: SyncedBanner(session.Root)}
	for _, group := range session.Exact {
		members := group.Members()
		var lib *domain.LibraryEntry
		if entry, ok := byID[members[0].ContentID]; ok {
			lib = &entry
		}
		item := ExactView{Members: sizesOf(members)}
		choice, err := domain.DefaultKeeper(members, lib)
		if err == nil && choice.Present {
			item.HasRecommendation = true
			item.Keeper = choice.Recommendation.Keeper
			item.Reason = string(choice.Recommendation.Reason)
			item.Text = choice.Recommendation.Text
			item.KeeperSize = sizeOf(members, choice.Recommendation.Keeper)
			for _, path := range choice.Recommendation.Trash {
				sz := sizeOf(members, path)
				item.Deletes = append(item.Deletes, FileSize{Path: path, Size: sz})
				item.Freed += sz
			}
		}
		model.Exact = append(model.Exact, item)
	}
	for _, group := range session.Similar {
		model.Similar = append(model.Similar, SimilarView{Members: sizesOf(group.Members())})
	}
	return model
}

// FormatDryRun renders the plan and states that nothing was written.
func FormatDryRun(report PlanReport) string {
	var b strings.Builder
	for _, path := range report.Trash {
		b.WriteString(path)
		b.WriteByte('\n')
	}
	b.WriteString(NothingChanged)
	return b.String()
}

func sizesOf(files []domain.FileRef) []FileSize {
	out := make([]FileSize, len(files))
	for i, file := range files {
		out[i] = FileSize{Path: file.Path, Size: file.Size}
	}
	return out
}

func sizeOf(files []domain.FileRef, path string) int64 {
	key := filepath.Clean(path)
	for _, file := range files {
		if filepath.Clean(file.Path) == key {
			return file.Size
		}
	}
	return 0
}
