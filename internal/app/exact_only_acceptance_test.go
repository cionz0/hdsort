package app_test

import (
	"database/sql"
	"image/color"
	"path/filepath"
	"testing"

	"github.com/cionz0/hdsort/internal/adapters/filesystem"
	"github.com/cionz0/hdsort/internal/adapters/hash"
	"github.com/cionz0/hdsort/internal/adapters/similar"
	"github.com/cionz0/hdsort/internal/adapters/sqlite"
	"github.com/cionz0/hdsort/internal/app"
	"github.com/cionz0/hdsort/internal/domain"

	_ "modernc.org/sqlite"
)

func TestAcceptance_ExactOnlySkipsNearDuplicates(t *testing.T) {
	root := nearDuplicatePNGs(t)

	result := scanMode(t, root, failIfSimilar{t: t}, true)
	if len(result.Session.Similar) != 0 {
		t.Fatalf("similar groups = %d, want 0", len(result.Session.Similar))
	}
	if len(result.Session.Exact) != 0 {
		t.Fatalf("exact groups = %d, want 0", len(result.Session.Exact))
	}
	if !result.Session.ExactOnly {
		t.Fatal("session ExactOnly = false, want true")
	}
}

func TestAcceptance_DefaultScanStillGroupsSimilar(t *testing.T) {
	root := nearDuplicatePNGs(t)

	result := scanMode(t, root, similar.New(), false)
	if result.Session.ExactOnly {
		t.Fatal("default scan set ExactOnly")
	}
	if len(result.Session.Similar) != 1 {
		t.Fatalf("similar groups = %d, want 1", len(result.Session.Similar))
	}
	if len(result.Session.Exact) != 0 {
		t.Fatalf("exact groups = %d, want 0", len(result.Session.Exact))
	}
	if _, ok := result.Session.Similar[0].Recommendation(); ok {
		t.Fatal("similar group has a delete recommendation")
	}
}

func TestAcceptance_ExactOnlyKeepsByteIdentical(t *testing.T) {
	root := t.TempDir()
	body := encodePNG(t, solidPNG(color.NRGBA{R: 20, G: 80, B: 140, A: 255}))
	writeFile(t, filepath.Join(root, "a.png"), body)
	writeFile(t, filepath.Join(root, "b.png"), body)

	result := scanMode(t, root, failIfSimilar{t: t}, true)
	if len(result.Session.Exact) != 1 {
		t.Fatalf("exact groups = %d, want 1", len(result.Session.Exact))
	}
	if len(result.Session.Similar) != 0 {
		t.Fatalf("similar groups = %d, want 0", len(result.Session.Similar))
	}
}

func TestAcceptance_ExactOnlyRoundTrip(t *testing.T) {
	store := openStore(t)
	on := domain.NewSession(t.TempDir())
	on.ExactOnly = true
	onID := saveSession(t, store, on)
	loaded := loadSession(t, store, onID)
	if !loaded.ExactOnly {
		t.Fatal("stored exact-only session loaded as similar-files-included")
	}

	off := domain.NewSession(t.TempDir())
	offID := saveSession(t, store, off)
	loaded = loadSession(t, store, offID)
	if loaded.ExactOnly {
		t.Fatal("zero-value session loaded as exact-only")
	}

	legacy := openLegacySession(t)
	loaded = loadSession(t, legacy.store, legacy.id)
	if loaded.ExactOnly {
		t.Fatal("session stored without the flag loaded as exact-only")
	}
}

// failIfSimilar is the similarity port. A call fails the test: exact-only
// must not ask the strategy.
type failIfSimilar struct{ t *testing.T }

func (f failIfSimilar) Similar([]domain.FileRef, app.FileReader) ([]domain.SimilarGroup, error) {
	f.t.Helper()
	f.t.Fatal("similarity strategy was called")
	return nil, nil
}

func nearDuplicatePNGs(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	solid := solidPNG(color.NRGBA{R: 180, G: 40, B: 40, A: 255})
	changed := cloneNRGBA(solid)
	changed.Set(3, 3, color.NRGBA{R: 181, G: 40, B: 40, A: 255})
	writePNG(t, filepath.Join(root, "red.png"), solid)
	writePNG(t, filepath.Join(root, "red-bit.png"), changed)
	return root
}

func scanMode(t *testing.T, root string, sim app.Similarizer, exactOnly bool) app.Result {
	t.Helper()
	result, err := app.Scan(root, "", filesystem.NewWalker(nil), filesystem.NewOpener(), hash.New(), sim, exactOnly)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func saveSession(t *testing.T, store *sqlite.Store, session domain.Session) string {
	t.Helper()
	id, err := store.Save(session)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func loadSession(t *testing.T, store *sqlite.Store, id string) domain.Session {
	t.Helper()
	session, err := store.Load(id)
	if err != nil {
		t.Fatal(err)
	}
	return session
}

type legacySession struct {
	store *sqlite.Store
	id    string
}

// openLegacySession inserts a session row that has no exact_only value,
// the shape of a library written before ADR-0011, then opens it so the
// column is added with default 0.
func openLegacySession(t *testing.T) legacySession {
	t.Helper()
	path := filepath.Join(t.TempDir(), "legacy.sqlite")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
CREATE TABLE sessions (
  id TEXT PRIMARY KEY,
  root TEXT NOT NULL,
  status TEXT NOT NULL
)`); err != nil {
		t.Fatal(err)
	}
	const id = "legacy"
	if _, err := db.Exec(
		`INSERT INTO sessions (id, root, status) VALUES (?, ?, 'open')`,
		id, t.TempDir(),
	); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	store, err := sqlite.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	return legacySession{store: store, id: id}
}
