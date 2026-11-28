package app_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cionz0/hdsort/internal/adapters/filesystem"
	"github.com/cionz0/hdsort/internal/adapters/trash"
	"github.com/cionz0/hdsort/internal/app"
	"github.com/cionz0/hdsort/internal/domain"
)

func TestAcceptance_LastCopyWarningIsPerFile(t *testing.T) {
	root, trashDir, session, files := twoLastCopyFiles(t)
	store := openStore(t)
	groups, err := app.LastCopyGroups(session, files, store)
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 2 {
		t.Fatalf("groups = %d", len(groups))
	}
	_, err = app.ApplyIfLastCopiesConfirmed(session, files, store, trash.NewFreedesktop(trashDir), filesystem.NewOpener(), [][]string{groups[0]})
	if !errors.Is(err, domain.ErrLastCopy) {
		t.Fatalf("Apply error = %v", err)
	}
	for _, file := range files {
		if _, err := os.Stat(file.Path); err != nil {
			t.Fatalf("%s: %v", file.Path, err)
		}
	}
	if entries := listEntries(t, store); len(entries) != 0 {
		t.Fatalf("library = %+v", entries)
	}
	_ = root
}

func TestAcceptance_ApplyProceedsWhenEveryLastCopyIsConfirmed(t *testing.T) {
	_, trashDir, session, files := twoLastCopyFiles(t)
	store := openStore(t)
	groups, err := app.LastCopyGroups(session, files, store)
	if err != nil {
		t.Fatal(err)
	}
	got, err := app.ApplyIfLastCopiesConfirmed(session, files, store, trash.NewFreedesktop(trashDir), filesystem.NewOpener(), groups)
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
	}
	if entries := listEntries(t, store); len(entries) != 0 {
		t.Fatalf("library = %+v", entries)
	}
}

func TestAcceptance_ChangingAChoiceClearsThatConfirmation(t *testing.T) {
	_, trashDir, session, files := twoLastCopyFiles(t)
	store := openStore(t)
	groups, err := app.LastCopyGroups(session, files, store)
	if err != nil {
		t.Fatal(err)
	}
	confirmed := [][]string{groups[0], groups[1]}
	path := groups[0][0]
	session = decide(t, session, path, domain.ChoiceKeep)
	session = decide(t, session, path, domain.ChoiceDelete)
	confirmed = app.WithoutLastCopy(confirmed, path)
	_, err = app.ApplyIfLastCopiesConfirmed(session, files, store, trash.NewFreedesktop(trashDir), filesystem.NewOpener(), confirmed)
	if !errors.Is(err, domain.ErrLastCopy) {
		t.Fatalf("Apply error = %v", err)
	}
	for _, file := range files {
		if _, err := os.Stat(file.Path); err != nil {
			t.Fatal(err)
		}
	}
}

func TestAcceptance_LibraryCopyNeedsNoWarning(t *testing.T) {
	root, trashDir, session, files := lastCopyPlan(t)
	store := openStore(t)
	kept := filepath.Join(t.TempDir(), "kept.jpg")
	writeFile(t, kept, []byte("photo"))
	putEntry(t, store, domain.LibraryEntry{
		ContentID: files[0].ContentID,
		Path:      kept,
		IndexedAt: time.Unix(1, 0),
	})
	got, err := app.Apply(session, files, store, trash.NewFreedesktop(trashDir), filesystem.NewOpener())
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
	}
	_ = root
}

func twoLastCopyFiles(t *testing.T) (root, trashDir string, session domain.Session, files []domain.FileRef) {
	t.Helper()
	root, trashDir = scanAndTrash(t)
	for _, item := range []struct{ name, body string }{
		{"a.jpg", "one"},
		{"a-copy.jpg", "one"},
		{"b.jpg", "two"},
		{"b-copy.jpg", "two"},
	} {
		path := filepath.Join(root, item.name)
		writeFile(t, path, []byte(item.body))
		files = append(files, refOf(t, path))
	}
	session = domain.NewSession(root)
	for _, file := range files {
		session = decide(t, session, file.Path, domain.ChoiceDelete)
	}
	return root, trashDir, session, files
}
