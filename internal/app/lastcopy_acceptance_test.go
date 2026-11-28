package app_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cionz0/hdsort/internal/adapters/filesystem"
	"github.com/cionz0/hdsort/internal/adapters/trash"
	"github.com/cionz0/hdsort/internal/app"
	"github.com/cionz0/hdsort/internal/domain"
)

func TestAcceptance_LastCopyStillNeedsConfirmation(t *testing.T) {
	_, trashDir, session, files := lastCopyPlan(t)
	store := openStore(t)
	_, err := app.Apply(session, files, store, trash.NewFreedesktop(trashDir), filesystem.NewOpener())
	if !errors.Is(err, domain.ErrLastCopy) {
		t.Fatalf("Apply error = %v", err)
	}
	for _, file := range files {
		if _, err := os.Stat(file.Path); err != nil {
			t.Fatal(err)
		}
	}
	if session.Status() != domain.StatusOpen {
		t.Fatalf("status = %s", session.Status())
	}
}

func TestAcceptance_ConfirmedLastCopyTrashesEveryCopy(t *testing.T) {
	root, trashDir, session, files := lastCopyPlan(t)
	store := openStore(t)
	got, err := app.ApplyAllowingLastCopy(session, files, store, trash.NewFreedesktop(trashDir), filesystem.NewOpener())
	if err != nil {
		t.Fatal(err)
	}
	if got.Status() != domain.StatusApplied {
		t.Fatalf("status = %s", got.Status())
	}
	for _, file := range files {
		if _, err := os.Stat(file.Path); !os.IsNotExist(err) {
			t.Fatalf("%s still present: %v", file.Path, err)
		}
		name := filepath.Base(file.Path)
		if _, err := os.Stat(filepath.Join(trashDir, "files", name)); err != nil {
			t.Fatal(err)
		}
	}
	if entries := listEntries(t, store); len(entries) != 0 {
		t.Fatalf("library = %+v", entries)
	}
	_ = root
}

func TestAcceptance_ConfirmationDoesNotSkipOtherChecks(t *testing.T) {
	root, trashDir := scanAndTrash(t)
	path := filepath.Join(root, "a.jpg")
	writeFile(t, path, []byte("photo"))
	file := refOf(t, path)
	group, err := domain.NewExactGroup([]domain.FileRef{file, file})
	if err != nil {
		t.Fatal(err)
	}
	rec, err := domain.NewRecommendation(file.Path, []string{file.Path}, domain.ReasonPlainName)
	if err != nil {
		t.Fatal(err)
	}
	group, err = group.WithRecommendation(rec)
	if err != nil {
		t.Fatal(err)
	}
	session := domain.NewSession(root)
	session.Exact = []domain.ExactGroup{group}
	_, err = app.ApplyAllowingLastCopy(session, []domain.FileRef{file}, openStore(t), trash.NewFreedesktop(trashDir), filesystem.NewOpener())
	if !errors.Is(err, domain.ErrUnconfirmedDelete) {
		t.Fatalf("Apply error = %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}

func TestAcceptance_LastCopyQuestionNamesThePaths(t *testing.T) {
	root, _, session, files := lastCopyPlan(t)
	paths, err := app.LastCopyPaths(session, files, openStore(t))
	if err != nil {
		t.Fatal(err)
	}
	question := app.LastCopyQuestion(root, paths)
	lines := strings.Split(question, "\n")
	for _, file := range files {
		name := filepath.Base(file.Path)
		if !lineIs(lines, name) {
			t.Fatalf("question = %q, missing a line %s", question, name)
		}
		if strings.Contains(question, string(file.ContentID)) {
			t.Fatalf("question contains the content ID %s", file.ContentID)
		}
	}
}

func lineIs(lines []string, want string) bool {
	for _, line := range lines {
		if line == want {
			return true
		}
	}
	return false
}

func lastCopyPlan(t *testing.T) (root, trashDir string, session domain.Session, files []domain.FileRef) {
	t.Helper()
	root, trashDir = scanAndTrash(t)
	keeper, duplicate, files := duplicatePair(t, root)
	session = domain.NewSession(root)
	session = decide(t, session, keeper, domain.ChoiceDelete)
	session = decide(t, session, duplicate, domain.ChoiceDelete)
	return root, trashDir, session, files
}
