package app_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cionz0/hdsort/internal/adapters/trash"
	"github.com/cionz0/hdsort/internal/app"
	"github.com/cionz0/hdsort/internal/domain"
)

func TestAcceptance_TrashFolderMovesTheDirectory(t *testing.T) {
	root := t.TempDir()
	trashDir := t.TempDir()
	writeFile(t, filepath.Join(root, "a", "b", "file.txt"), []byte("nested"))
	writeFile(t, filepath.Join(root, "a", "keep.txt"), []byte("keep"))
	writeFile(t, filepath.Join(root, "c", "other.txt"), []byte("stay"))

	_, err := app.TrashFolder(domain.NewSession(root), filepath.Join(root, "a"), trash.NewFreedesktop(trashDir))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "a")); !os.IsNotExist(err) {
		t.Fatalf("a still present: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "c", "other.txt")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(trashDir, "files", "a", "b", "file.txt")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(trashDir, "files", "a", "keep.txt")); err != nil {
		t.Fatal(err)
	}
}

func TestAcceptance_TrashFolderRejectsOutsideTheScanRoot(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	writeFile(t, filepath.Join(outside, "file.txt"), []byte("stay"))
	_, err := app.TrashFolder(domain.NewSession(root), outside, trash.NewFreedesktop(t.TempDir()))
	if err == nil {
		t.Fatal("expected an error")
	}
	if _, err := os.Stat(filepath.Join(outside, "file.txt")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(root); err != nil {
		t.Fatal(err)
	}
}

func TestAcceptance_TrashFolderDropsDecisionsInsideIt(t *testing.T) {
	root := t.TempDir()
	inside := filepath.Join(root, "a", "one.txt")
	outside := filepath.Join(root, "c", "two.txt")
	writeFile(t, inside, []byte("one"))
	writeFile(t, outside, []byte("two"))
	session := domain.NewSession(root)
	session = decide(t, session, inside, domain.ChoiceDelete)
	session = decide(t, session, outside, domain.ChoiceKeep)
	store := openStore(t)
	putEntry(t, store, domain.LibraryEntry{
		ContentID: "abc123",
		Path:      outside,
		IndexedAt: time.Unix(1, 0),
	})
	before := listEntries(t, store)

	next, err := app.TrashFolder(session, filepath.Join(root, "a"), trash.NewFreedesktop(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	if next.Status() != domain.StatusOpen {
		t.Fatalf("status = %s", next.Status())
	}
	if len(next.Decisions) != 1 || filepath.Clean(next.Decisions[0].Path) != outside {
		t.Fatalf("decisions = %+v", next.Decisions)
	}
	after := listEntries(t, store)
	if len(after) != len(before) || after[0].Path != before[0].Path || after[0].ContentID != before[0].ContentID {
		t.Fatalf("library = %+v, was %+v", after, before)
	}
}

func TestAcceptance_SetKeepsMembersOutsideTheTrashedFolder(t *testing.T) {
	root := t.TempDir()
	inside := filepath.Join(root, "a", "b", "one.txt")
	outside := filepath.Join(root, "c", "two.txt")
	rows := []app.FolderRow{{
		Key:     "set",
		Folder:  filepath.Dir(inside),
		Title:   "one.txt",
		Members: []app.FileSize{{Path: inside}, {Path: outside}},
	}}
	got := app.DropTrashed(rows, filepath.Join(root, "a"))
	if len(got) != 1 {
		t.Fatalf("rows = %d", len(got))
	}
	if len(got[0].Members) != 1 || filepath.Clean(got[0].Members[0].Path) != outside {
		t.Fatalf("members = %+v", got[0].Members)
	}
	if got[0].Folder != filepath.Dir(outside) {
		t.Fatalf("folder = %s", got[0].Folder)
	}
}
