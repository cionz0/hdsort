package domain

import "time"

// ContentID is the BLAKE3 digest of a file's full bytes.
// Empty files still have a content ID. Hashing lives outside this package.
type ContentID string

// FileRef is one file a scan observed: path, size, modification time,
// content ID, and whether the bytes were readable.
type FileRef struct {
	Path      string
	Size      int64
	ModTime   time.Time
	ContentID ContentID
	Readable  bool
}

// LibraryEntry is a survivor already recorded for a content ID.
type LibraryEntry struct {
	ContentID ContentID
	Path      string
	IndexedAt time.Time
}

// Valid reports whether the entry can count as the kept copy of its content ID.
// An entry needs both a content ID and a path. A zero value is not a survivor.
func (e LibraryEntry) Valid() bool {
	return e.ContentID != "" && e.Path != ""
}
