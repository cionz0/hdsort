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

// SessionFile is one readable file from a scan, addressed by an opaque id.
// Preview requests use ID. They do not send Path.
type SessionFile struct {
	ID   string
	File FileRef
}

// Size and ModTime are the file metadata last reconciled.
// Stale means the file is missing or its content ID changed; a stale entry
// is not a survivor.
type LibraryEntry struct {
	ContentID ContentID
	Path      string
	Size      int64
	ModTime   time.Time
	IndexedAt time.Time
	Stale     bool
}

// Valid reports whether the entry can count as the kept copy of its content ID.
// An entry needs a content ID and a path, and it must not be stale.
func (e LibraryEntry) Valid() bool {
	return e.ContentID != "" && e.Path != "" && !e.Stale
}
