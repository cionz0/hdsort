package app

import (
	"fmt"
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

// FolderPicker asks the operating system for a directory.
// The scan calls it when the request does not name a path.
//
// Pattern: Ports and Adapters (Cockburn, 2005). See docs/patterns.md.
type FolderPicker interface {
	Pick() (string, error)
}

// FileReader opens a file for hashing. Tests wrap it to count bytes read.
type FileReader interface {
	Open(path string) (io.ReadSeekCloser, error)
}

// HashStage is one filter from ADR-0007.
type HashStage string

const (
	StageSize    HashStage = "size"
	StagePartial HashStage = "partial"
	StageFull    HashStage = "full"
)

// Identifier assigns content IDs. It is the hasher port.
type Identifier interface {
	Identify(files []FileEntry, source FileReader) ([]domain.FileRef, error)
}

// StageReporter is an Identifier that names the filter it is running.
// A file reports finished when that filter was the last one for the file.
type StageReporter interface {
	IdentifyReporting(files []FileEntry, source FileReader, report func(path string, size int64, stage HashStage, finished bool)) ([]domain.FileRef, error)
}

// ReportingSimilarizer reports how many candidate files it has compared.
type ReportingSimilarizer interface {
	SimilarReporting(files []domain.FileRef, source FileReader, report func(done, total int)) ([]domain.SimilarGroup, error)
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

// Scan walks root, hashes the readable files, and groups exact duplicates.
// exactOnly is the ADR-0011 mode. The zero value runs the similarity strategy
// after exact grouping. True skips that call and leaves similar groups empty.
// sim is required either way: a nil strategy is not the exact-only switch.
// Dataless files and symlinks are listed on the session and are not hashed.
// A symlink keeps its destination so the owner can see what the link is.
func Scan(root, dataDir string, walk Walker, files FileReader, id Identifier, sim Similarizer, exactOnly bool, progress func(ProgressEvent)) (Result, error) {
	emit := func(ev ProgressEvent) {
		if progress != nil {
			progress(ev)
		}
	}
	emit(ProgressEvent{Kind: EventListing})
	entries, err := walk.Walk(root, dataDir)
	if err != nil {
		return Result{}, err
	}
	emit(ProgressEvent{Kind: EventListing, Found: len(entries)})
	hashing := make([]FileEntry, 0, len(entries))
	var skipped []domain.Skipped
	for _, entry := range entries {
		switch entry.Availability {
		case AvailabilityDataless:
			skipped = append(skipped, domain.Skipped{
				Path:   entry.Path,
				Reason: domain.SkipDataless,
			})
			emit(ProgressEvent{Kind: EventProcessed, Path: entry.Path, Size: entry.Size, Outcome: OutcomeDataless})
			continue
		case AvailabilitySymlink:
			skipped = append(skipped, domain.Skipped{
				Path:   entry.Path,
				Reason: domain.SkipSymlink,
				Target: entry.Target,
			})
			emit(ProgressEvent{Kind: EventProcessed, Path: entry.Path, Size: entry.Size, Outcome: OutcomeSymlink})
			continue
		}
		hashing = append(hashing, entry)
		emit(ProgressEvent{Kind: EventQueue, Path: entry.Path, Size: entry.Size})
	}
	var refs []domain.FileRef
	if staged, ok := id.(StageReporter); ok {
		refs, err = staged.IdentifyReporting(hashing, files, func(path string, size int64, stage HashStage, finished bool) {
			emit(ProgressEvent{Kind: EventHashing, Path: path, Size: size, Stage: stage, Finished: finished})
		})
	} else {
		refs, err = id.Identify(hashing, files)
	}
	if err != nil {
		return Result{}, err
	}
	counts := map[domain.ContentID]int{}
	for _, ref := range refs {
		if ref.ContentID != "" {
			counts[ref.ContentID]++
		}
	}
	for _, ref := range refs {
		outcome := OutcomeUnique
		if ref.ContentID != "" && counts[ref.ContentID] >= 2 {
			outcome = OutcomeExact
		} else if !exactOnly {
			outcome = OutcomeWaiting
		}
		emit(ProgressEvent{Kind: EventProcessed, Path: ref.Path, Size: ref.Size, Outcome: outcome})
	}
	groups, err := domain.GroupExact(refs)
	if err != nil {
		return Result{}, err
	}
	if sim == nil {
		return Result{}, fmt.Errorf("scan: similarity strategy is required")
	}
	var similar []domain.SimilarGroup
	if !exactOnly {
		total := 0
		for _, ref := range refs {
			if ref.Readable && ref.ContentID != "" && counts[ref.ContentID] == 1 {
				total++
			}
		}
		emit(ProgressEvent{Kind: EventSimilar, Done: 0, Total: total})
		if reporting, ok := sim.(ReportingSimilarizer); ok {
			similar, err = reporting.SimilarReporting(refs, files, func(done, total int) {
				emit(ProgressEvent{Kind: EventSimilar, Done: done, Total: total})
			})
		} else {
			similar, err = sim.Similar(refs, files)
			emit(ProgressEvent{Kind: EventSimilar, Done: total, Total: total})
		}
		if err != nil {
			return Result{}, err
		}
	}
	session := domain.NewSession(root)
	session.Exact = groups
	session.Similar = similar
	session.Skipped = skipped
	session.ExactOnly = exactOnly
	return Result{Session: session, Files: refs}, nil
}
