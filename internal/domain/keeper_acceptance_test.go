package domain_test

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cionz0/hdsort/internal/domain"
)

func TestAcceptance_KeeperRecommendation(t *testing.T) {
	same := domain.ContentID("same")
	older := time.Unix(1_700_000_000, 0)
	newer := older.Add(time.Hour)

	libraryOutside := domain.LibraryEntry{
		ContentID: same,
		Path:      filepath.Join("library", "a.jpg"),
		IndexedAt: older,
	}
	libraryInsidePath := filepath.Join("scan", "a (1).jpg")
	libraryInside := domain.LibraryEntry{
		ContentID: same,
		Path:      libraryInsidePath,
		IndexedAt: older,
	}

	tests := []struct {
		name       string
		members    []domain.FileRef
		library    *domain.LibraryEntry
		wantKeeper string
		wantReason domain.ReasonCode
		wantAbsent bool
	}{
		{
			name: "plain name beats generated copies",
			members: []domain.FileRef{
				file("a.jpg", same, older),
				file("a (1).jpg", same, newer),
				file("Copy of a.jpg", same, newer),
			},
			wantKeeper: "a.jpg",
			wantReason: domain.ReasonPlainName,
		},
		{
			name: "plain name beats conflicted copy",
			members: []domain.FileRef{
				file("photo.jpg", same, older),
				file("photo conflicted copy.jpg", same, newer),
			},
			wantKeeper: "photo.jpg",
			wantReason: domain.ReasonPlainName,
		},
		{
			name: "newer plain name breaks the tie",
			members: []domain.FileRef{
				file("old.jpg", same, older),
				file("new.jpg", same, newer),
			},
			wantKeeper: "new.jpg",
			wantReason: domain.ReasonNewerMtimeTiebreak,
		},
		{
			name: "equal modification times leave no recommendation",
			members: []domain.FileRef{
				file("a.jpg", same, older),
				file("b.jpg", same, older),
			},
			wantAbsent: true,
		},
		{
			name: "library survivor outside the scan",
			members: []domain.FileRef{
				file(filepath.Join("scan", "a.jpg"), same, newer),
				file(filepath.Join("scan", "a (1).jpg"), same, older),
			},
			library:    &libraryOutside,
			wantKeeper: libraryOutside.Path,
			wantReason: domain.ReasonLibrarySurvivor,
		},
		{
			name: "library path inside the scan",
			members: []domain.FileRef{
				file(filepath.Join("scan", "a.jpg"), same, newer),
				file(libraryInsidePath, same, older),
			},
			library:    &libraryInside,
			wantKeeper: libraryInsidePath,
			wantReason: domain.ReasonLibrarySurvivor,
		},
		{
			name: "version words do not pick the keeper",
			members: []domain.FileRef{
				file("report_v1.pdf", same, older),
				file("report_final.pdf", same, newer),
			},
			wantKeeper: "report_final.pdf",
			wantReason: domain.ReasonNewerMtimeTiebreak,
		},
	}

	var policy domain.KeeperPolicy = domain.DefaultKeeper
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := policy(tt.members, tt.library)
			if err != nil {
				t.Fatalf("DefaultKeeper: %v", err)
			}
			if tt.wantAbsent {
				if got.Present {
					t.Fatalf("recommendation present: %+v", got.Recommendation)
				}
				if got.Recommendation.Reason != "" {
					t.Fatalf("absent recommendation has reason %q", got.Recommendation.Reason)
				}
				return
			}
			if !got.Present {
				t.Fatal("recommendation absent")
			}
			if got.Recommendation.Keeper != tt.wantKeeper {
				t.Fatalf("keeper = %s, want %s", got.Recommendation.Keeper, tt.wantKeeper)
			}
			if got.Recommendation.Reason != tt.wantReason {
				t.Fatalf("reason = %s, want %s", got.Recommendation.Reason, tt.wantReason)
			}
		})
	}

	t.Run("different bytes are not a group", func(t *testing.T) {
		groups, err := domain.GroupExact([]domain.FileRef{
			file("report_v1.pdf", "v1", older),
			file("report_final.pdf", "final", newer),
		})
		if err != nil {
			t.Fatalf("GroupExact: %v", err)
		}
		if len(groups) != 0 {
			t.Fatalf("groups = %d, want 0", len(groups))
		}
	})
}

func TestAcceptance_LibraryFileIsNotAMove(t *testing.T) {
	same := domain.ContentID("same")
	when := time.Unix(1_700_000_000, 0)
	library := domain.LibraryEntry{
		ContentID: same,
		Path:      filepath.Join("library", "a.jpg"),
		IndexedAt: when,
	}
	members := []domain.FileRef{
		file(filepath.Join("scan", "a.jpg"), same, when),
		file(filepath.Join("scan", "copy.jpg"), same, when.Add(time.Minute)),
	}
	got, err := domain.DefaultKeeper(members, &library)
	if err != nil {
		t.Fatalf("DefaultKeeper: %v", err)
	}
	if !got.Present {
		t.Fatal("recommendation absent")
	}
	if got.Recommendation.Keeper != library.Path {
		t.Fatalf("keeper = %s, want %s", got.Recommendation.Keeper, library.Path)
	}
	if got.Recommendation.Reason != domain.ReasonLibrarySurvivor {
		t.Fatalf("reason = %s", got.Recommendation.Reason)
	}
	if len(got.Moves) != 0 {
		t.Fatalf("moves = %+v, want none", got.Moves)
	}
	for _, path := range got.Recommendation.Trash {
		if path == library.Path {
			t.Fatal("library path is in the delete list")
		}
	}
}

func TestAcceptance_ReasonTextNamesTheTieBreak(t *testing.T) {
	same := domain.ContentID("same")
	older := time.Unix(1_700_000_000, 0)
	got, err := domain.DefaultKeeper([]domain.FileRef{
		file("old.jpg", same, older),
		file("new.jpg", same, older.Add(time.Hour)),
	}, nil)
	if err != nil {
		t.Fatalf("DefaultKeeper: %v", err)
	}
	if got.Recommendation.Reason != domain.ReasonNewerMtimeTiebreak {
		t.Fatalf("reason = %s", got.Recommendation.Reason)
	}
	if !strings.Contains(got.Recommendation.Text, "tie-break") {
		t.Fatalf("reason text = %q, want it to say tie-break", got.Recommendation.Text)
	}
}

func file(path string, id domain.ContentID, mod time.Time) domain.FileRef {
	return domain.FileRef{
		Path:      path,
		ContentID: id,
		ModTime:   mod,
		Readable:  true,
	}
}
