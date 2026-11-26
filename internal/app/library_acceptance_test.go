package app_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/cionz0/hdsort/internal/adapters/filesystem"
	"github.com/cionz0/hdsort/internal/adapters/hash"
	"github.com/cionz0/hdsort/internal/adapters/sqlite"
	"github.com/cionz0/hdsort/internal/app"
	"github.com/cionz0/hdsort/internal/domain"
)

func TestAcceptance_ReconcileMissingMarksStale(t *testing.T) {
	root := t.TempDir()
	missing := filepath.Join(root, "gone.jpg")
	store := openStore(t)
	const id = domain.ContentID("abc123")
	putEntry(t, store, domain.LibraryEntry{
		ContentID: id,
		Path:      missing,
		IndexedAt: time.Unix(1, 0),
	})

	if err := app.Reconcile(root, store, filesystem.NewOpener(), hash.New()); err != nil {
		t.Fatal(err)
	}
	got := mustEntry(t, store, missing)
	if !got.Stale {
		t.Fatal("entry is not stale")
	}
	if got.ContentID != id {
		t.Fatalf("content ID = %s, want %s", got.ContentID, id)
	}
}

func TestAcceptance_ReconcileChangedContentMarksStale(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "a.jpg")
	writeFile(t, path, []byte("one"))
	before := refOf(t, path)
	store := openStore(t)
	putEntry(t, store, domain.LibraryEntry{
		ContentID: before.ContentID,
		Path:      path,
		Size:      before.Size,
		ModTime:   before.ModTime,
		IndexedAt: time.Unix(1, 0),
	})
	writeFile(t, path, []byte("two"))

	if err := app.Reconcile(root, store, filesystem.NewOpener(), hash.New()); err != nil {
		t.Fatal(err)
	}
	got := mustEntry(t, store, path)
	if !got.Stale {
		t.Fatal("entry is not stale")
	}
	if got.ContentID != before.ContentID {
		t.Fatalf("content ID = %s, want the stored id %s", got.ContentID, before.ContentID)
	}
}

func TestAcceptance_ReconcileUnchangedRefreshesMetadata(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "a.jpg")
	writeFile(t, path, []byte("same"))
	file := refOf(t, path)
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	oldIndex := time.Unix(10, 0)
	store := openStore(t)
	putEntry(t, store, domain.LibraryEntry{
		ContentID: file.ContentID,
		Path:      path,
		Size:      file.Size + 9,
		ModTime:   time.Unix(1, 0),
		IndexedAt: oldIndex,
	})

	if err := app.Reconcile(root, store, filesystem.NewOpener(), hash.New()); err != nil {
		t.Fatal(err)
	}
	got := mustEntry(t, store, path)
	if got.Stale {
		t.Fatal("unchanged file was marked stale")
	}
	if got.ContentID != file.ContentID {
		t.Fatalf("content ID = %s, want %s", got.ContentID, file.ContentID)
	}
	if got.Size != info.Size() {
		t.Fatalf("size = %d, want %d", got.Size, info.Size())
	}
	if got.ModTime.UnixNano() != info.ModTime().UnixNano() {
		t.Fatalf("mod time = %s, want %s", got.ModTime, info.ModTime())
	}
	if !got.IndexedAt.After(oldIndex) {
		t.Fatalf("indexed at = %s, want after %s", got.IndexedAt, oldIndex)
	}
}

func TestAcceptance_ReconcileIgnoresEntriesOutsideRoot(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "keep.jpg")
	writeFile(t, outside, []byte("outside"))
	file := refOf(t, outside)
	indexed := time.Unix(50, 0)
	store := openStore(t)
	putEntry(t, store, domain.LibraryEntry{
		ContentID: file.ContentID,
		Path:      outside,
		Size:      file.Size,
		ModTime:   file.ModTime,
		IndexedAt: indexed,
	})
	missing := filepath.Join(root, "gone.jpg")
	putEntry(t, store, domain.LibraryEntry{
		ContentID: "inside-id",
		Path:      missing,
		IndexedAt: time.Unix(1, 0),
	})

	if err := app.Reconcile(root, store, filesystem.NewOpener(), hash.New()); err != nil {
		t.Fatal(err)
	}
	got := mustEntry(t, store, outside)
	if got.Stale || got.ContentID != file.ContentID || got.Path != outside || got.Size != file.Size {
		t.Fatalf("outside entry changed: %+v", got)
	}
	if got.ModTime.UnixNano() != file.ModTime.UnixNano() || got.IndexedAt.UnixNano() != indexed.UnixNano() {
		t.Fatalf("outside times changed: mod %s indexed %s", got.ModTime, got.IndexedAt)
	}
	if !mustEntry(t, store, missing).Stale {
		t.Fatal("inside missing entry is not stale")
	}
}

func TestAcceptance_StaleEntryDoesNotFormADuplicate(t *testing.T) {
	root := t.TempDir()
	remaining := filepath.Join(root, "a.jpg")
	writeFile(t, remaining, []byte("photo"))
	file := refOf(t, remaining)
	store := openStore(t)
	putEntry(t, store, domain.LibraryEntry{
		ContentID: file.ContentID,
		Path:      filepath.Join(root, "original.jpg"),
		IndexedAt: time.Unix(1, 0),
	})
	if err := app.Reconcile(root, store, filesystem.NewOpener(), hash.New()); err != nil {
		t.Fatal(err)
	}

	groups, err := app.Match([]domain.FileRef{file}, store)
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 0 {
		t.Fatalf("groups = %d, want 0", len(groups))
	}
}

func TestAcceptance_TwoCopiesRemainAGroupAfterStaleOriginal(t *testing.T) {
	root := t.TempDir()
	body := []byte("photo")
	plain := filepath.Join(root, "a.jpg")
	copy := filepath.Join(root, "a (1).jpg")
	original := filepath.Join(root, "original.jpg")
	writeFile(t, plain, body)
	writeFile(t, copy, body)
	writeFile(t, original, body)
	id := refOf(t, plain).ContentID
	if err := os.Remove(original); err != nil {
		t.Fatal(err)
	}
	store := openStore(t)
	putEntry(t, store, domain.LibraryEntry{
		ContentID: id,
		Path:      original,
		IndexedAt: time.Unix(1, 0),
	})
	if err := app.Reconcile(root, store, filesystem.NewOpener(), hash.New()); err != nil {
		t.Fatal(err)
	}

	groups, err := app.Match([]domain.FileRef{refOf(t, plain), refOf(t, copy)}, store)
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 1 {
		t.Fatalf("groups = %d, want 1", len(groups))
	}
	members := groups[0].Members()
	if len(members) != 2 {
		t.Fatalf("members = %d, want 2", len(members))
	}
	for _, member := range members {
		if member.Path == original {
			t.Fatalf("stale path %s is a group member", original)
		}
	}
	rec, ok := groups[0].Recommendation()
	if !ok {
		t.Fatal("recommendation is absent")
	}
	if rec.Keeper != plain {
		t.Fatalf("keeper = %s, want %s", rec.Keeper, plain)
	}
	if rec.Keeper == original {
		t.Fatalf("keeper is the stale path %s", original)
	}
	if rec.Reason != domain.ReasonPlainName {
		t.Fatalf("reason = %s, want plain_name", rec.Reason)
	}
}

func TestAcceptance_MatchOutsideLibrarySurvivor(t *testing.T) {
	root := t.TempDir()
	body := []byte("photo")
	plain := filepath.Join(root, "a.jpg")
	copy := filepath.Join(root, "a (1).jpg")
	writeFile(t, plain, body)
	writeFile(t, copy, body)
	outside := filepath.Join(t.TempDir(), "library.jpg")
	writeFile(t, outside, body)
	id := refOf(t, plain).ContentID
	store := openStore(t)
	putEntry(t, store, domain.LibraryEntry{
		ContentID: id,
		Path:      outside,
		Size:      int64(len(body)),
		IndexedAt: time.Unix(1, 0),
	})

	groups, err := app.Match([]domain.FileRef{refOf(t, plain), refOf(t, copy)}, store)
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 1 {
		t.Fatalf("groups = %d, want 1", len(groups))
	}
	rec, ok := groups[0].Recommendation()
	if !ok {
		t.Fatal("recommendation is absent")
	}
	if rec.Keeper != outside {
		t.Fatalf("keeper = %s, want %s", rec.Keeper, outside)
	}
	if rec.Reason != domain.ReasonLibrarySurvivor {
		t.Fatalf("reason = %s, want library_survivor", rec.Reason)
	}
}

func TestAcceptance_SessionRoundTrip(t *testing.T) {
	root := t.TempDir()
	plain := domain.FileRef{
		Path: filepath.Join(root, "a.jpg"), Size: 4, ModTime: time.Unix(100, 0),
		ContentID: "aaa", Readable: true,
	}
	copy := domain.FileRef{
		Path: filepath.Join(root, "a (1).jpg"), Size: 4, ModTime: time.Unix(90, 0),
		ContentID: "aaa", Readable: true,
	}
	otherA := domain.FileRef{
		Path: filepath.Join(root, "b.jpg"), Size: 2, ModTime: time.Unix(80, 0),
		ContentID: "bbb", Readable: true,
	}
	otherB := domain.FileRef{
		Path: filepath.Join(root, "b (1).jpg"), Size: 2, ModTime: time.Unix(70, 0),
		ContentID: "bbb", Readable: true,
	}
	ranked, err := domain.NewExactGroup([]domain.FileRef{plain, copy})
	if err != nil {
		t.Fatal(err)
	}
	rec, err := domain.NewRecommendation(plain.Path, []string{copy.Path}, domain.ReasonPlainName)
	if err != nil {
		t.Fatal(err)
	}
	rec.Text = "plain name"
	ranked, err = ranked.WithRecommendation(rec)
	if err != nil {
		t.Fatal(err)
	}
	bare, err := domain.NewExactGroup([]domain.FileRef{otherA, otherB})
	if err != nil {
		t.Fatal(err)
	}
	session := domain.NewSession(root)
	session.Exact = []domain.ExactGroup{ranked, bare}

	store := openStore(t)
	id, err := store.Save(session)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Load(id)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Root != session.Root {
		t.Fatalf("root = %s, want %s", loaded.Root, session.Root)
	}
	if loaded.Status() != domain.StatusOpen {
		t.Fatalf("status = %s, want open", loaded.Status())
	}
	if len(loaded.Exact) != 2 {
		t.Fatalf("groups = %d, want 2", len(loaded.Exact))
	}
	assertSameMembers(t, loaded.Exact[0].Members(), []domain.FileRef{plain, copy})
	got, ok := loaded.Exact[0].Recommendation()
	if !ok {
		t.Fatal("recommendation is absent")
	}
	if got.Keeper != rec.Keeper || got.Reason != rec.Reason || got.Text != rec.Text {
		t.Fatalf("recommendation = %+v, want %+v", got, rec)
	}
	if len(got.Trash) != 1 || got.Trash[0] != copy.Path {
		t.Fatalf("trash = %v, want [%s]", got.Trash, copy.Path)
	}
	assertSameMembers(t, loaded.Exact[1].Members(), []domain.FileRef{otherA, otherB})
	if _, ok := loaded.Exact[1].Recommendation(); ok {
		t.Fatal("second group has a recommendation")
	}
}

func TestAcceptance_DefaultLibraryPath(t *testing.T) {
	got, err := sqlite.DefaultPath()
	if err != nil {
		t.Fatal(err)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	var want string
	switch runtime.GOOS {
	case "darwin":
		want = filepath.Join(home, "Library", "Application Support", "hdsort", "library.sqlite")
	case "windows":
		base := os.Getenv("AppData")
		if base == "" {
			base = filepath.Join(home, "AppData", "Roaming")
		}
		want = filepath.Join(base, "hdsort", "library.sqlite")
	default:
		base := os.Getenv("XDG_DATA_HOME")
		if base == "" {
			base = filepath.Join(home, ".local", "share")
		}
		want = filepath.Join(base, "hdsort", "library.sqlite")
	}
	if got != want {
		t.Fatalf("path = %s, want %s", got, want)
	}
}

func openStore(t *testing.T) *sqlite.Store {
	t.Helper()
	store, err := sqlite.Open(filepath.Join(t.TempDir(), "library.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	mode, err := store.JournalMode()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.EqualFold(mode, "wal") {
		t.Fatalf("journal mode = %s, want wal", mode)
	}
	return store
}

func putEntry(t *testing.T, store *sqlite.Store, entry domain.LibraryEntry) {
	t.Helper()
	if err := store.Put(entry); err != nil {
		t.Fatal(err)
	}
}

func mustEntry(t *testing.T, store *sqlite.Store, path string) domain.LibraryEntry {
	t.Helper()
	entries, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.Path == path {
			return entry
		}
	}
	t.Fatalf("missing library entry %s", path)
	return domain.LibraryEntry{}
}

func refOf(t *testing.T, path string) domain.FileRef {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	id, err := hash.New().ContentID(path, filesystem.NewOpener())
	if err != nil {
		t.Fatal(err)
	}
	return domain.FileRef{
		Path:      path,
		Size:      info.Size(),
		ModTime:   info.ModTime(),
		ContentID: id,
		Readable:  true,
	}
}

func assertSameMembers(t *testing.T, got, want []domain.FileRef) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("members = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i].Path != want[i].Path || got[i].ContentID != want[i].ContentID {
			t.Fatalf("member %d = %s %s, want %s %s", i, got[i].Path, got[i].ContentID, want[i].Path, want[i].ContentID)
		}
		if got[i].Size != want[i].Size || got[i].Readable != want[i].Readable {
			t.Fatalf("member %d metadata = size %d readable %v", i, got[i].Size, got[i].Readable)
		}
		if got[i].ModTime.UnixNano() != want[i].ModTime.UnixNano() {
			t.Fatalf("member %d mod time = %s, want %s", i, got[i].ModTime, want[i].ModTime)
		}
	}
}
