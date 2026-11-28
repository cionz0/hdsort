package app_test

import (
	"errors"
	"path/filepath"
	"testing"

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
