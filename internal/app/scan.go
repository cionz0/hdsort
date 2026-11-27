package app

import (
	"io"
	"time"

	"github.com/cionz0/hdsort/internal/domain"
)

// Availability is whether a walked file's bytes are local.
type Availability int

const (
	AvailabilityReadable Availability = iota
	AvailabilityDataless
	AvailabilitySymlink
)

// FileEntry is one path the walk found, before hashing.
// Target is set when Availability is a symlink.
type FileEntry struct {
	Path         string
	Size         int64
	ModTime      time.Time
	Availability Availability
	Target       string
}

// Walker lists the scan root. The dataDir path, when set, is not hashed.
type Walker interface {
	Walk(root, dataDir string) ([]FileEntry, error)
}

// FileReader opens a file for hashing. Tests wrap it to count bytes read.
type FileReader interface {
	Open(path string) (io.ReadSeekCloser, error)
}

// Identifier assigns content IDs. It is the hasher port.
type Identifier interface {
	Identify(files []FileEntry, source FileReader) ([]domain.FileRef, error)
}

// Similarizer attaches similar groups after exact grouping.
// The scan depends on this Strategy, not on a concrete fingerprint.
//
// Pattern: Strategy (Gamma, Helm, Johnson, Vlissides, 1994).
// See docs/patterns.md.
type Similarizer interface {
	Similar(files []domain.FileRef, source FileReader) ([]domain.SimilarGroup, error)
}

// Result is a finished scan: an open session plus every readable file ref,
// including files that are not in an exact group.
type Result struct {
	Session domain.Session
	Files   []domain.FileRef
}

// Scan walks root, hashes the readable files, groups exact duplicates, and
// then asks the similarity strategy for groups that are close but not
// byte-identical. Dataless files and symlinks are listed on the session and
// are not hashed. A symlink keeps its destination so the owner can see what
// the link is.
func Scan(root, dataDir string, walk Walker, files FileReader, id Identifier, sim Similarizer) (Result, error) {
	entries, err := walk.Walk(root, dataDir)
	if err != nil {
		return Result{}, err
	}
	hashing := make([]FileEntry, 0, len(entries))
	var skipped []domain.Skipped
	for _, entry := range entries {
		switch entry.Availability {
		case AvailabilityDataless:
			skipped = append(skipped, domain.Skipped{
				Path:   entry.Path,
				Reason: domain.SkipDataless,
			})
			continue
		case AvailabilitySymlink:
			skipped = append(skipped, domain.Skipped{
				Path:   entry.Path,
				Reason: domain.SkipSymlink,
				Target: entry.Target,
			})
			continue
		}
		hashing = append(hashing, entry)
	}
	refs, err := id.Identify(hashing, files)
	if err != nil {
		return Result{}, err
	}
	groups, err := domain.GroupExact(refs)
	if err != nil {
		return Result{}, err
	}
	similar, err := sim.Similar(refs, files)
	if err != nil {
		return Result{}, err
	}
	session := domain.NewSession(root)
	session.Exact = groups
	session.Similar = similar
	session.Skipped = skipped
	return Result{Session: session, Files: refs}, nil
}
