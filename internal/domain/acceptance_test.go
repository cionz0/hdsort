package domain_test

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/cionz0/hdsort/internal/domain"
)

func TestAcceptance_ExactGroupNeedsTwoMembers(t *testing.T) {
	one := domain.FileRef{Path: "a.jpg", ContentID: "same", Readable: true}
	tests := []struct {
		name    string
		members []domain.FileRef
		wantErr error
	}{
		{name: "none", wantErr: domain.ErrExactGroupMembers},
		{name: "one", members: []domain.FileRef{one}, wantErr: domain.ErrExactGroupMembers},
		{
			name: "two",
			members: []domain.FileRef{
				one,
				{Path: "b.jpg", ContentID: "same", Readable: true},
			},
		},
		{
			name: "different content IDs",
			members: []domain.FileRef{
				one,
				{Path: "b.jpg", ContentID: "other", Readable: true},
			},
			wantErr: domain.ErrExactGroupContent,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := domain.NewExactGroup(tt.members)
			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("NewExactGroup: %v", err)
				}
				return
			}
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("NewExactGroup error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestAcceptance_SimilarGroupHasNoRecommendation(t *testing.T) {
	group, err := domain.NewSimilarGroup([]domain.FileRef{
		{Path: "a.jpg", ContentID: "id-a", Readable: true},
		{Path: "b.jpg", ContentID: "id-b", Readable: true},
	})
	if err != nil {
		t.Fatalf("NewSimilarGroup: %v", err)
	}
	rec, err := domain.NewRecommendation("a.jpg", []string{"b.jpg"}, domain.ReasonPlainName)
	if err != nil {
		t.Fatalf("NewRecommendation: %v", err)
	}
	if err := group.SetRecommendation(rec); !errors.Is(err, domain.ErrSimilarRecommendation) {
		t.Fatalf("SetRecommendation error = %v, want rejection", err)
	}
	if _, ok := group.Recommendation(); ok {
		t.Fatal("similar group stored a delete recommendation")
	}
}

func TestAcceptance_AbsentRecommendationIsDistinct(t *testing.T) {
	group, err := domain.NewExactGroup([]domain.FileRef{
		{Path: "a.jpg", ContentID: "same", Readable: true},
		{Path: "b.jpg", ContentID: "same", Readable: true},
	})
	if err != nil {
		t.Fatalf("NewExactGroup: %v", err)
	}
	if _, ok := group.Recommendation(); ok {
		t.Fatal("new exact group returned a recommendation")
	}
	if _, err := domain.NewRecommendation("a.jpg", []string{"b.jpg"}, ""); !errors.Is(err, domain.ErrReasonCode) {
		t.Fatalf("empty reason error = %v, want unknown reason", err)
	}
	stored, err := group.WithRecommendation(domain.Recommendation{
		Keeper: "a.jpg",
		Trash:  []string{"b.jpg"},
		Reason: "",
	})
	if !errors.Is(err, domain.ErrReasonCode) {
		t.Fatalf("WithRecommendation error = %v, want unknown reason", err)
	}
	if _, ok := stored.Recommendation(); ok {
		t.Fatal("rejected recommendation was stored")
	}
}

func TestAcceptance_DecisionRequiresExplicitChoice(t *testing.T) {
	tests := []struct {
		name    string
		choice  domain.Choice
		wantErr bool
	}{
		{name: "keep", choice: domain.ChoiceKeep},
		{name: "delete", choice: domain.ChoiceDelete},
		{name: "skip", choice: domain.ChoiceSkip},
		{name: "empty", choice: "", wantErr: true},
		{name: "trash", choice: "trash", wantErr: true},
		{name: "other", choice: "drop", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := domain.NewDecision("a.jpg", tt.choice)
			if tt.wantErr {
				if !errors.Is(err, domain.ErrDecisionChoice) {
					t.Fatalf("NewDecision error = %v, want choice rejection", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("NewDecision: %v", err)
			}
		})
	}
}

func TestAcceptance_UnconfirmedDeleteRejected(t *testing.T) {
	files := []domain.FileRef{
		{Path: filepath.Join("scan", "a.jpg"), ContentID: "same", Readable: true},
		{Path: filepath.Join("scan", "b.jpg"), ContentID: "same", Readable: true},
	}
	trashPath := files[0].Path
	tests := []struct {
		name    string
		choice  domain.Choice
		present bool
		wantErr bool
	}{
		{name: "no decision", wantErr: true},
		{name: "keep", choice: domain.ChoiceKeep, present: true, wantErr: true},
		{name: "skip", choice: domain.ChoiceSkip, present: true, wantErr: true},
		{name: "delete", choice: domain.ChoiceDelete, present: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var decisions []domain.Decision
			if tt.present {
				d, err := domain.NewDecision(trashPath, tt.choice)
				if err != nil {
					t.Fatalf("NewDecision: %v", err)
				}
				decisions = []domain.Decision{d}
			}
			err := domain.CheckPlan(domain.Plan{
				Files:     files,
				Trash:     []string{trashPath},
				Decisions: decisions,
			})
			if tt.wantErr {
				if !errors.Is(err, domain.ErrUnconfirmedDelete) {
					t.Fatalf("CheckPlan error = %v, want unconfirmed delete", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("CheckPlan: %v", err)
			}
		})
	}
}

func TestAcceptance_LastCopyRejected(t *testing.T) {
	files := twoCopies()
	err := domain.CheckPlan(domain.Plan{
		Files:     files,
		Trash:     []string{files[0].Path, files[1].Path},
		Decisions: deleteBoth(t, files),
	})
	if !errors.Is(err, domain.ErrLastCopy) {
		t.Fatalf("CheckPlan error = %v, want last copy", err)
	}

	t.Run("one copy remains", func(t *testing.T) {
		err := domain.CheckPlan(domain.Plan{
			Files:     files,
			Trash:     []string{files[0].Path},
			Decisions: []domain.Decision{mustDecision(t, files[0].Path, domain.ChoiceDelete)},
		})
		if err != nil {
			t.Fatalf("CheckPlan: %v", err)
		}
	})
}

func TestAcceptance_LastCopyAllowedWhenLibraryKeepsIt(t *testing.T) {
	files := twoCopies()
	library := domain.LibraryEntry{
		ContentID: files[0].ContentID,
		Path:      filepath.Join("library", "a.jpg"),
		IndexedAt: time.Unix(1_700_000_000, 0),
	}
	base := func() domain.Plan {
		return domain.Plan{
			Files:     files,
			Trash:     []string{files[0].Path, files[1].Path},
			Decisions: deleteBoth(t, files),
			Library:   []domain.LibraryEntry{library},
		}
	}
	if err := domain.CheckPlan(base()); err != nil {
		t.Fatalf("CheckPlan: %v", err)
	}

	t.Run("library path is also trashed", func(t *testing.T) {
		plan := base()
		plan.Trash = append(plan.Trash, library.Path)
		plan.Decisions = append(plan.Decisions, mustDecision(t, library.Path, domain.ChoiceDelete))
		err := domain.CheckPlan(plan)
		if !errors.Is(err, domain.ErrLastCopy) {
			t.Fatalf("CheckPlan error = %v, want last copy", err)
		}
	})

	t.Run("invalid library entry", func(t *testing.T) {
		plan := base()
		plan.Library = []domain.LibraryEntry{{ContentID: files[0].ContentID}}
		err := domain.CheckPlan(plan)
		if !errors.Is(err, domain.ErrLastCopy) {
			t.Fatalf("CheckPlan error = %v, want last copy", err)
		}
	})
}

func TestAcceptance_MoveOutsideRootRejected(t *testing.T) {
	root := t.TempDir()
	from := filepath.Join(root, "a.jpg")
	tests := []struct {
		name    string
		dest    string
		wantErr bool
	}{
		{name: "inside the root", dest: filepath.Join(root, "photos", "a.jpg")},
		{name: "parent of the root", dest: filepath.Dir(root), wantErr: true},
		{name: "dotdot escapes the root", dest: filepath.Join(root, "..", "outside.jpg"), wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := domain.NewLayoutMove(root, from, tt.dest)
			if tt.wantErr {
				if !errors.Is(err, domain.ErrMoveOutsideRoot) {
					t.Fatalf("NewLayoutMove error = %v, want outside root", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("NewLayoutMove: %v", err)
			}
		})
	}
}

func TestAcceptance_SessionStatus(t *testing.T) {
	session := domain.NewSession(t.TempDir())
	if session.Status() != domain.StatusOpen {
		t.Fatalf("status = %q, want open", session.Status())
	}
	applied, err := domain.ApplyResult(session)
	if err != nil {
		t.Fatalf("ApplyResult: %v", err)
	}
	if applied.Status() != domain.StatusApplied {
		t.Fatalf("status = %q, want applied", applied.Status())
	}
	if session.Status() != domain.StatusOpen {
		t.Fatal("ApplyResult changed the caller's session")
	}
	if _, err := domain.ApplyResult(applied); !errors.Is(err, domain.ErrSessionNotOpen) {
		t.Fatalf("second ApplyResult error = %v, want not open", err)
	}
}

func twoCopies() []domain.FileRef {
	return []domain.FileRef{
		{Path: filepath.Join("scan", "a.jpg"), Size: 10, ContentID: "abc", Readable: true},
		{Path: filepath.Join("scan", "a (1).jpg"), Size: 10, ContentID: "abc", Readable: true},
	}
}

func deleteBoth(t *testing.T, files []domain.FileRef) []domain.Decision {
	t.Helper()
	return []domain.Decision{
		mustDecision(t, files[0].Path, domain.ChoiceDelete),
		mustDecision(t, files[1].Path, domain.ChoiceDelete),
	}
}

func mustDecision(t *testing.T, path string, choice domain.Choice) domain.Decision {
	t.Helper()
	d, err := domain.NewDecision(path, choice)
	if err != nil {
		t.Fatalf("NewDecision: %v", err)
	}
	return d
}
