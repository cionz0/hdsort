package app_test

import (
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cionz0/hdsort/internal/adapters/filesystem"
	"github.com/cionz0/hdsort/internal/adapters/sqlite"
	"github.com/cionz0/hdsort/internal/adapters/trash"
	"github.com/cionz0/hdsort/internal/app"
	"github.com/cionz0/hdsort/internal/domain"
)

func TestAcceptance_ApplyTrashesThenIndexes(t *testing.T) {
	root, trashDir := scanAndTrash(t)
	keeper, duplicate, files := duplicatePair(t, root)
	session := recommend(t, root, files, keeper, duplicate)
	session = decide(t, session, keeper, domain.ChoiceKeep)
	session = decide(t, session, duplicate, domain.ChoiceDelete)
	store := openStore(t)

	got, err := app.Apply(session, files, store, trash.NewFreedesktop(trashDir), filesystem.NewOpener())
	if err != nil {
		t.Fatal(err)
	}
	if got.Status() != domain.StatusApplied {
		t.Fatalf("status = %s, want applied", got.Status())
	}
	if _, err := os.Stat(duplicate); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("deleted file still at %s (%v)", duplicate, err)
	}
	if _, err := os.Stat(keeper); err != nil {
		t.Fatal(err)
	}
	trashed := filepath.Join(trashDir, "files", filepath.Base(duplicate))
	body, err := os.ReadFile(trashed)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "photo" {
		t.Fatalf("trashed bytes = %q", body)
	}
	entry := mustEntry(t, store, keeper)
	if entry.ContentID != files[0].ContentID || entry.Stale {
		t.Fatalf("library entry = %+v", entry)
	}
}

func TestAcceptance_ApplyStopsWhenTrashFails(t *testing.T) {
	root := t.TempDir()
	keeper, duplicate, files := duplicatePair(t, root)
	session := recommend(t, root, files, keeper, duplicate)
	session = decide(t, session, keeper, domain.ChoiceKeep)
	session = decide(t, session, duplicate, domain.ChoiceDelete)
	store := openStore(t)
	putEntry(t, store, domain.LibraryEntry{
		ContentID: "seed",
		Path:      filepath.Join(t.TempDir(), "already-kept.jpg"),
	})
	before := listEntries(t, store)
	port := &failTrash{}

	got, err := app.Apply(session, files, store, port, filesystem.NewOpener())
	if err == nil {
		t.Fatal("apply succeeded")
	}
	if port.calls == 0 {
		t.Fatal("trash was not called")
	}
	if got.Status() != domain.StatusOpen {
		t.Fatalf("status = %s, want open", got.Status())
	}
	after := listEntries(t, store)
	if len(after) != len(before) {
		t.Fatalf("library rows = %d, want %d", len(after), len(before))
	}
	if after[0].ContentID != before[0].ContentID || after[0].Path != before[0].Path {
		t.Fatalf("library changed from %+v to %+v", before[0], after[0])
	}
	if _, err := os.Stat(duplicate); err != nil {
		t.Fatalf("duplicate missing after a failed trash: %v", err)
	}
}

func TestAcceptance_ApplyRejectsUnconfirmedDelete(t *testing.T) {
	root := t.TempDir()
	keeper, duplicate, files := duplicatePair(t, root)
	session := recommend(t, root, files, keeper, duplicate)
	session = decide(t, session, keeper, domain.ChoiceKeep)
	store := openStore(t)
	port := &failTrash{}

	got, err := app.Apply(session, files, store, port, filesystem.NewOpener())
	if !errors.Is(err, domain.ErrUnconfirmedDelete) {
		t.Fatalf("Apply error = %v, want unconfirmed delete", err)
	}
	if port.calls != 0 {
		t.Fatalf("trash calls = %d, want 0", port.calls)
	}
	if got.Status() != domain.StatusOpen {
		t.Fatalf("status = %s, want open", got.Status())
	}
}

func TestAcceptance_DryRunReportsPlan(t *testing.T) {
	root := t.TempDir()
	keeper, duplicate, files := duplicatePair(t, root)
	destination := filepath.Join(root, "sorted", "a.jpg")
	session := recommend(t, root, files, keeper, duplicate)
	session = decide(t, session, keeper, domain.ChoiceKeep)
	session = decide(t, session, duplicate, domain.ChoiceDelete)
	session = confirm(t, session, keeper, destination)
	store := openStore(t)

	report, err := app.DryRun(session, files, store, &failTrash{}, filesystem.NewOpener())
	if err != nil {
		t.Fatal(err)
	}
	if session.Status() != domain.StatusOpen {
		t.Fatalf("status = %s, want open", session.Status())
	}
	if !containsPath(report.Trash, duplicate) {
		t.Fatalf("trash = %v, want %s", report.Trash, duplicate)
	}
	if len(report.Moves) != 1 || report.Moves[0].Destination != destination || report.Moves[0].From != keeper {
		t.Fatalf("moves = %+v", report.Moves)
	}
	if !containsPath(report.Upserts, destination) {
		t.Fatalf("upserts = %v, want %s", report.Upserts, destination)
	}
}

func TestAcceptance_DryRunWritesNothing(t *testing.T) {
	root := t.TempDir()
	keeper, duplicate, files := duplicatePair(t, root)
	destination := filepath.Join(root, "sorted", "a.jpg")
	session := recommend(t, root, files, keeper, duplicate)
	session = decide(t, session, keeper, domain.ChoiceKeep)
	session = decide(t, session, duplicate, domain.ChoiceDelete)
	session = confirm(t, session, keeper, destination)
	store := openStore(t)
	putEntry(t, store, domain.LibraryEntry{
		ContentID: "seed",
		Path:      filepath.Join(t.TempDir(), "already-kept.jpg"),
	})
	before := len(listEntries(t, store))
	port := &failTrash{}

	if _, err := app.DryRun(session, files, store, port, filesystem.NewOpener()); err != nil {
		t.Fatal(err)
	}
	if port.calls != 0 {
		t.Fatalf("trash calls = %d, want 0", port.calls)
	}
	if _, err := os.Stat(duplicate); err != nil {
		t.Fatalf("delete target missing: %v", err)
	}
	if _, err := os.Stat(keeper); err != nil {
		t.Fatalf("keeper missing: %v", err)
	}
	if _, err := os.Stat(destination); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("dry-run created %s (%v)", destination, err)
	}
	if got := len(listEntries(t, store)); got != before {
		t.Fatalf("library rows = %d, want %d", got, before)
	}
	if session.Status() != domain.StatusOpen {
		t.Fatalf("status = %s, want open", session.Status())
	}
}

func TestAcceptance_DryRunRejectsUnconfirmedDelete(t *testing.T) {
	root := t.TempDir()
	keeper, duplicate, files := duplicatePair(t, root)
	session := recommend(t, root, files, keeper, duplicate)
	session = decide(t, session, keeper, domain.ChoiceKeep)
	store := openStore(t)
	port := &failTrash{}
	disk := filesystem.NewOpener()

	_, errApply := app.Apply(session, files, store, port, disk)
	_, errDry := app.DryRun(session, files, store, port, disk)
	if !errors.Is(errApply, domain.ErrUnconfirmedDelete) || !errors.Is(errDry, domain.ErrUnconfirmedDelete) {
		t.Fatalf("apply %v, dry-run %v", errApply, errDry)
	}
	if errApply.Error() != errDry.Error() {
		t.Fatalf("rejections differ: apply %v, dry-run %v", errApply, errDry)
	}
	if port.calls != 0 {
		t.Fatalf("trash calls = %d, want 0", port.calls)
	}
}

func TestAcceptance_DefaultLayoutIsEmpty(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "photos", "2020")
	docs := filepath.Join(root, "docs")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(docs, 0o755); err != nil {
		t.Fatal(err)
	}
	photo := filepath.Join(nested, "a.jpg")
	note := filepath.Join(docs, "notes.txt")
	writeFile(t, photo, []byte("photo"))
	writeFile(t, note, []byte("note"))
	info, err := os.Stat(nested)
	if err != nil || !info.IsDir() {
		t.Fatalf("nested folder: %v", err)
	}
	moves, err := domain.DefaultLayout(root, []domain.FileRef{refOf(t, photo), refOf(t, note)})
	if err != nil {
		t.Fatal(err)
	}
	if len(moves) != 0 {
		t.Fatalf("moves = %d, want 0", len(moves))
	}
}

func TestAcceptance_ConfirmedMoveStaysInsideRoot(t *testing.T) {
	root := t.TempDir()
	keeper := filepath.Join(root, "a.jpg")
	writeFile(t, keeper, []byte("photo"))
	file := refOf(t, keeper)
	destination := filepath.Join(root, "sorted", "a.jpg")
	session := domain.NewSession(root)
	session = decide(t, session, keeper, domain.ChoiceKeep)
	session = confirm(t, session, keeper, destination)
	store := openStore(t)

	got, err := app.Apply(session, []domain.FileRef{file}, store, trash.NewFreedesktop(t.TempDir()), filesystem.NewOpener())
	if err != nil {
		t.Fatal(err)
	}
	if got.Status() != domain.StatusApplied {
		t.Fatalf("status = %s, want applied", got.Status())
	}
	if _, err := os.Stat(keeper); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("source still present (%v)", err)
	}
	if _, err := os.Stat(destination); err != nil {
		t.Fatal(err)
	}
	entry := mustEntry(t, store, destination)
	if entry.ContentID != file.ContentID {
		t.Fatalf("library content ID = %s, want %s", entry.ContentID, file.ContentID)
	}
}

func TestAcceptance_UnconfirmedMoveDoesNotRun(t *testing.T) {
	root := t.TempDir()
	keeper := filepath.Join(root, "a.jpg")
	writeFile(t, keeper, []byte("photo"))
	file := refOf(t, keeper)
	destination := filepath.Join(root, "sorted", "a.jpg")
	move, err := domain.NewLayoutMove(root, keeper, destination)
	if err != nil {
		t.Fatal(err)
	}
	session := domain.NewSession(root)
	session.Moves = []domain.LayoutMove{move}
	session = decide(t, session, keeper, domain.ChoiceKeep)
	if move.Confirmed {
		t.Fatal("new move is confirmed")
	}
	store := openStore(t)

	got, err := app.Apply(session, []domain.FileRef{file}, store, trash.NewFreedesktop(t.TempDir()), filesystem.NewOpener())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(keeper); err != nil {
		t.Fatalf("source missing: %v", err)
	}
	if _, err := os.Stat(destination); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("unconfirmed move created %s (%v)", destination, err)
	}
	entry := mustEntry(t, store, keeper)
	if entry.Path != keeper {
		t.Fatalf("library path = %s, want %s", entry.Path, keeper)
	}
	if got.Status() != domain.StatusApplied {
		t.Fatalf("status = %s, want applied", got.Status())
	}
}

func TestAcceptance_FreedesktopTrashLayout(t *testing.T) {
	parent := t.TempDir()
	trashDir := filepath.Join(parent, "Trash")
	src := filepath.Join(parent, "docs", "note.txt")
	writeFile(t, src, []byte("hello"))
	abs, err := filepath.Abs(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := trash.NewFreedesktop(trashDir).Trash(src); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(src); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("original still present (%v)", err)
	}
	trashed, err := os.ReadFile(filepath.Join(trashDir, "files", "note.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(trashed) != "hello" {
		t.Fatalf("trashed bytes = %q", trashed)
	}
	info, err := os.ReadFile(filepath.Join(trashDir, "info", "note.txt.trashinfo"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(info), "[Trash Info]\n") || !strings.Contains(string(info), "DeletionDate=") {
		t.Fatalf("info = %q", info)
	}
	gotPath, ok := trashInfoPath(string(info))
	if !ok || gotPath != abs {
		t.Fatalf("info path = %q, want %s", gotPath, abs)
	}
}

type failTrash struct {
	calls int
}

func (f *failTrash) Trash(string) error {
	f.calls++
	return errors.New("trash failed")
}

func scanAndTrash(t *testing.T) (root, trashDir string) {
	t.Helper()
	parent := t.TempDir()
	root = filepath.Join(parent, "scan")
	trashDir = filepath.Join(parent, "Trash")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	return root, trashDir
}

func duplicatePair(t *testing.T, root string) (keeper, duplicate string, files []domain.FileRef) {
	t.Helper()
	keeper = filepath.Join(root, "a.jpg")
	duplicate = filepath.Join(root, "b.jpg")
	writeFile(t, keeper, []byte("photo"))
	writeFile(t, duplicate, []byte("photo"))
	files = []domain.FileRef{refOf(t, keeper), refOf(t, duplicate)}
	return keeper, duplicate, files
}

func recommend(t *testing.T, root string, files []domain.FileRef, keeper, duplicate string) domain.Session {
	t.Helper()
	group, err := domain.NewExactGroup(files)
	if err != nil {
		t.Fatal(err)
	}
	rec, err := domain.NewRecommendation(keeper, []string{duplicate}, domain.ReasonPlainName)
	if err != nil {
		t.Fatal(err)
	}
	group, err = group.WithRecommendation(rec)
	if err != nil {
		t.Fatal(err)
	}
	session := domain.NewSession(root)
	session.Exact = []domain.ExactGroup{group}
	return session
}

func decide(t *testing.T, session domain.Session, path string, choice domain.Choice) domain.Session {
	t.Helper()
	next, err := domain.RecordDecision(session, path, choice)
	if err != nil {
		t.Fatal(err)
	}
	return next
}

func confirm(t *testing.T, session domain.Session, from, destination string) domain.Session {
	t.Helper()
	move, err := domain.NewLayoutMove(session.Root, from, destination)
	if err != nil {
		t.Fatal(err)
	}
	session.Moves = append(session.Moves, move)
	next, err := domain.ConfirmMove(session, from)
	if err != nil {
		t.Fatal(err)
	}
	return next
}

func listEntries(t *testing.T, store *sqlite.Store) []domain.LibraryEntry {
	t.Helper()
	entries, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	return entries
}

func containsPath(paths []string, want string) bool {
	want = filepath.Clean(want)
	for _, path := range paths {
		if filepath.Clean(path) == want {
			return true
		}
	}
	return false
}

func trashInfoPath(body string) (string, bool) {
	for _, line := range strings.Split(body, "\n") {
		if !strings.HasPrefix(line, "Path=") {
			continue
		}
		decoded, err := url.PathUnescape(strings.TrimPrefix(line, "Path="))
		if err != nil {
			return "", false
		}
		return decoded, true
	}
	return "", false
}
