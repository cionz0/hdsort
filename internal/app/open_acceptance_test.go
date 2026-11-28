package app_test

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/cionz0/hdsort/internal/app"
	"github.com/cionz0/hdsort/internal/domain"
)

type recordingLauncher struct {
	path string
	err  error
}

func (r *recordingLauncher) Open(path string) error {
	r.path = path
	return r.err
}

func TestAcceptance_OpenCopyAsksTheSystemForThatFile(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "Parrocchia.m4a.mp4")
	other := filepath.Join(root, "keeper.m4a.mp4")
	session := domain.NewSession(root)
	session.Decisions = []domain.Decision{{Path: other, Choice: domain.ChoiceKeep}}
	launch := &recordingLauncher{}
	next, err := app.OpenCopy(session, launch, path+string(filepath.Separator))
	if err != nil {
		t.Fatal(err)
	}
	if launch.path != path {
		t.Fatalf("opened %q, want %q", launch.path, path)
	}
	if len(next.Decisions) != 1 || next.Decisions[0].Path != other || next.Decisions[0].Choice != domain.ChoiceKeep {
		t.Fatalf("decisions = %+v", next.Decisions)
	}
}

func TestAcceptance_OpenCopyReportsAFailure(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "copy.mp4")
	session := domain.NewSession(root)
	want := errors.New("open failed")
	next, err := app.OpenCopy(session, &recordingLauncher{err: want}, path)
	if !errors.Is(err, want) {
		t.Fatalf("err = %v", err)
	}
	if len(next.Decisions) != 0 {
		t.Fatalf("decisions = %+v", next.Decisions)
	}
}

func TestAcceptance_SuggestedChoiceIsTheDefault(t *testing.T) {
	root := t.TempDir()
	plain := filepath.Join(root, "photo.png")
	generated := filepath.Join(root, "photo (1).png")
	session := exactSession(t, root, plain, generated)
	model := app.Review(session, nil)
	if len(session.Decisions) != 0 {
		t.Fatalf("decisions = %+v", session.Decisions)
	}
	group := model.Exact[0]
	if app.ShownChoice(session, group.HasRecommendation, group.Keeper, group.Deletes, plain) != domain.ChoiceKeep {
		t.Fatalf("plain = %q", app.ShownChoice(session, group.HasRecommendation, group.Keeper, group.Deletes, plain))
	}
	if app.ShownChoice(session, group.HasRecommendation, group.Keeper, group.Deletes, generated) != domain.ChoiceDelete {
		t.Fatalf("copy = %q", app.ShownChoice(session, group.HasRecommendation, group.Keeper, group.Deletes, generated))
	}
}

func TestAcceptance_RecordedChoiceBeatsTheSuggestion(t *testing.T) {
	root := t.TempDir()
	plain := filepath.Join(root, "photo.png")
	generated := filepath.Join(root, "photo (1).png")
	session := exactSession(t, root, plain, generated)
	next, err := domain.RecordDecision(session, generated, domain.ChoiceKeep)
	if err != nil {
		t.Fatal(err)
	}
	group := app.Review(next, nil).Exact[0]
	if app.ShownChoice(next, group.HasRecommendation, group.Keeper, group.Deletes, generated) != domain.ChoiceKeep {
		t.Fatal("recorded keep did not win")
	}
}

func TestAcceptance_NoSuggestionLeavesTheRadiosClear(t *testing.T) {
	root := t.TempDir()
	when := time.Unix(1_700_000_000, 0)
	a := filepath.Join(root, "a.jpg")
	b := filepath.Join(root, "b.jpg")
	group, err := domain.NewExactGroup([]domain.FileRef{
		{Path: a, Size: 4, ContentID: "same", ModTime: when, Readable: true},
		{Path: b, Size: 4, ContentID: "same", ModTime: when, Readable: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	session := domain.NewSession(root)
	session.Exact = []domain.ExactGroup{group}
	exact := app.Review(session, nil).Exact[0]
	if exact.HasRecommendation {
		t.Fatal("tie has a recommendation")
	}
	if app.ShownChoice(session, exact.HasRecommendation, exact.Keeper, exact.Deletes, a) != "" {
		t.Fatal("tie selected a radio")
	}
	similar, err := domain.NewSimilarGroup([]domain.FileRef{
		{Path: filepath.Join(root, "near.jpg"), Size: 5, ContentID: "a"},
		{Path: filepath.Join(root, "other.jpg"), Size: 9, ContentID: "b"},
	})
	if err != nil {
		t.Fatal(err)
	}
	session.Similar = []domain.SimilarGroup{similar}
	model := app.Review(session, nil)
	for _, member := range model.Similar[0].Members {
		if app.ShownChoice(session, model.Similar[0].HasRecommendation, "", nil, member.Path) != "" {
			t.Fatalf("similar %s has a choice", member.Path)
		}
	}
}

func exactSession(t *testing.T, root, plain, generated string) domain.Session {
	t.Helper()
	when := time.Unix(1_700_000_000, 0)
	group, err := domain.NewExactGroup([]domain.FileRef{
		{Path: plain, Size: 4, ContentID: "same", ModTime: when, Readable: true},
		{Path: generated, Size: 4, ContentID: "same", ModTime: when, Readable: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	session := domain.NewSession(root)
	session.Exact = []domain.ExactGroup{group}
	return session
}
