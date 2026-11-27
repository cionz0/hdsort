package sqlite

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cionz0/hdsort/internal/app"
	"github.com/cionz0/hdsort/internal/domain"

	_ "modernc.org/sqlite" // CGo-free database/sql driver. See ADR-0008.
)

const schema = `
CREATE TABLE IF NOT EXISTS library (
  content_id TEXT PRIMARY KEY,
  path TEXT NOT NULL,
  size INTEGER NOT NULL,
  mod_time INTEGER NOT NULL,
  indexed_at INTEGER NOT NULL,
  stale INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS sessions (
  id TEXT PRIMARY KEY,
  root TEXT NOT NULL,
  status TEXT NOT NULL,
  exact_only INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS exact_members (
  session_id TEXT NOT NULL,
  group_no INTEGER NOT NULL,
  member_no INTEGER NOT NULL,
  path TEXT NOT NULL,
  size INTEGER NOT NULL,
  mod_time INTEGER NOT NULL,
  content_id TEXT NOT NULL,
  readable INTEGER NOT NULL,
  PRIMARY KEY (session_id, group_no, member_no)
);

CREATE TABLE IF NOT EXISTS exact_recommendations (
  session_id TEXT NOT NULL,
  group_no INTEGER NOT NULL,
  keeper TEXT NOT NULL,
  reason TEXT NOT NULL,
  text TEXT NOT NULL,
  PRIMARY KEY (session_id, group_no)
);

CREATE TABLE IF NOT EXISTS exact_trash (
  session_id TEXT NOT NULL,
  group_no INTEGER NOT NULL,
  trash_no INTEGER NOT NULL,
  path TEXT NOT NULL,
  PRIMARY KEY (session_id, group_no, trash_no)
);

CREATE TABLE IF NOT EXISTS similar_members (
  session_id TEXT NOT NULL,
  group_no INTEGER NOT NULL,
  member_no INTEGER NOT NULL,
  path TEXT NOT NULL,
  size INTEGER NOT NULL,
  mod_time INTEGER NOT NULL,
  content_id TEXT NOT NULL,
  readable INTEGER NOT NULL,
  PRIMARY KEY (session_id, group_no, member_no)
);

CREATE TABLE IF NOT EXISTS decisions (
  session_id TEXT NOT NULL,
  decision_no INTEGER NOT NULL,
  path TEXT NOT NULL,
  choice TEXT NOT NULL,
  PRIMARY KEY (session_id, decision_no)
);

CREATE TABLE IF NOT EXISTS moves (
  session_id TEXT NOT NULL,
  move_no INTEGER NOT NULL,
  from_path TEXT NOT NULL,
  destination TEXT NOT NULL,
  confirmed INTEGER NOT NULL,
  PRIMARY KEY (session_id, move_no)
);

CREATE TABLE IF NOT EXISTS skipped (
  session_id TEXT NOT NULL,
  skip_no INTEGER NOT NULL,
  path TEXT NOT NULL,
  reason TEXT NOT NULL,
  target TEXT NOT NULL,
  PRIMARY KEY (session_id, skip_no)
);

CREATE TABLE IF NOT EXISTS session_files (
  session_id TEXT NOT NULL,
  file_id TEXT NOT NULL,
  path TEXT NOT NULL,
  size INTEGER NOT NULL,
  mod_time INTEGER NOT NULL,
  content_id TEXT NOT NULL,
  readable INTEGER NOT NULL,
  PRIMARY KEY (session_id, file_id)
);
`

// Store is the SQLite library and session repository.
// One file holds both. The process is the only writer.
type Store struct {
	db *sql.DB
}

// Open opens path, creates the parent directory, and requires WAL mode.
// Tests pass a temporary path. OpenDefault resolves the ADR-0008 path.
func Open(path string) (*Store, error) {
	if path == "" {
		return nil, fmt.Errorf("sqlite: empty path")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	store := &Store{db: db}
	if err := store.configure(); err != nil {
		db.Close()
		return nil, err
	}
	return store, nil
}

// OpenDefault opens the ADR-0008 library file. Tests do not call this.
func OpenDefault() (*Store, error) {
	path, err := DefaultPath()
	if err != nil {
		return nil, err
	}
	return Open(path)
}

// Close closes the database.
func (s *Store) Close() error {
	return s.db.Close()
}

// JournalMode returns the current SQLite journal mode.
func (s *Store) JournalMode() (string, error) {
	var mode string
	err := s.db.QueryRow("PRAGMA journal_mode").Scan(&mode)
	return mode, err
}

func (s *Store) configure() error {
	var mode string
	if err := s.db.QueryRow("PRAGMA journal_mode=WAL").Scan(&mode); err != nil {
		return err
	}
	if !strings.EqualFold(mode, "wal") {
		return fmt.Errorf("sqlite: journal mode %q, want wal", mode)
	}
	if _, err := s.db.Exec(schema); err != nil {
		return err
	}
	if err := ensureMoveConfirmed(s.db); err != nil {
		return err
	}
	return ensureExactOnly(s.db)
}

// ensureMoveConfirmed adds the confirmation column on a library created
// before moves recorded whether the owner confirmed them.
func ensureMoveConfirmed(db *sql.DB) error {
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('moves') WHERE name = 'confirmed'`).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	_, err := db.Exec(`ALTER TABLE moves ADD COLUMN confirmed INTEGER NOT NULL DEFAULT 0`)
	return err
}

// ensureExactOnly adds the scan-mode column on a library created before
// sessions recorded whether similarity was skipped (ADR-0011).
// The default 0 means similar files are included.
func ensureExactOnly(db *sql.DB) error {
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('sessions') WHERE name = 'exact_only'`).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	_, err := db.Exec(`ALTER TABLE sessions ADD COLUMN exact_only INTEGER NOT NULL DEFAULT 0`)
	return err
}

// Put inserts or replaces the library row for the entry's content ID.
func (s *Store) Put(entry domain.LibraryEntry) error {
	if entry.ContentID == "" || entry.Path == "" {
		return fmt.Errorf("sqlite: library entry needs a content ID and a path")
	}
	_, err := s.db.Exec(`
INSERT INTO library (content_id, path, size, mod_time, indexed_at, stale)
VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(content_id) DO UPDATE SET
  path = excluded.path,
  size = excluded.size,
  mod_time = excluded.mod_time,
  indexed_at = excluded.indexed_at,
  stale = excluded.stale`,
		string(entry.ContentID),
		entry.Path,
		entry.Size,
		entry.ModTime.UnixNano(),
		entry.IndexedAt.UnixNano(),
		boolInt(entry.Stale),
	)
	return err
}

// List returns every library row, stale entries included, ordered by path.
func (s *Store) List() ([]domain.LibraryEntry, error) {
	rows, err := s.db.Query(`
SELECT content_id, path, size, mod_time, indexed_at, stale
FROM library
ORDER BY path, content_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var entries []domain.LibraryEntry
	for rows.Next() {
		var (
			entry domain.LibraryEntry
			id    string
			mod   int64
			at    int64
			stale int
		)
		if err := rows.Scan(&id, &entry.Path, &entry.Size, &mod, &at, &stale); err != nil {
			return nil, err
		}
		entry.ContentID = domain.ContentID(id)
		entry.ModTime = time.Unix(0, mod).UTC()
		entry.IndexedAt = time.Unix(0, at).UTC()
		entry.Stale = stale != 0
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

var (
	_ app.LibraryRepository = (*Store)(nil)
	_ app.SessionRepository = (*Store)(nil)
)
