package hash

import (
	"encoding/hex"
	"fmt"
	"io"

	"github.com/cionz0/hdsort/internal/app"
	"github.com/cionz0/hdsort/internal/domain"
	"lukechampine.com/blake3"
)

const (
	// partialWindow is the head and the tail covered by the partial filter.
	partialWindow int64 = 64 << 10
	// partialMinSize is the smallest file that takes the partial filter.
	// Smaller files skip it and are hashed in full.
	partialMinSize int64 = 128 << 10
)

// Hasher is the staged content-identity adapter.
// Filters run in order: size, then partial BLAKE3, then full BLAKE3.
// A later filter runs only for files the earlier filter could not separate.
type Hasher struct{}

// New returns the staged hasher.
func New() Hasher { return Hasher{} }

// Identify assigns a content ID to each file the filters could not separate.
// The content ID is the hex BLAKE3 digest of the full bytes. Files separated
// by size or by the partial filter are not fully read and keep an empty content ID.
func (Hasher) Identify(files []app.FileEntry, source app.FileReader) ([]domain.FileRef, error) {
	refs := make([]domain.FileRef, len(files))
	var small []int
	buckets := make(map[int64][]int)
	var sizes []int64
	for i, file := range files {
		refs[i] = domain.FileRef{
			Path:     file.Path,
			Size:     file.Size,
			ModTime:  file.ModTime,
			Readable: true,
		}
		if file.Size < partialMinSize {
			small = append(small, i)
			continue
		}
		if _, ok := buckets[file.Size]; !ok {
			sizes = append(sizes, file.Size)
		}
		buckets[file.Size] = append(buckets[file.Size], i)
	}

	for _, i := range small {
		id, err := fullID(source, files[i].Path)
		if err != nil {
			return nil, err
		}
		refs[i].ContentID = id
	}
	for _, size := range sizes {
		idxs := buckets[size]
		if len(idxs) < 2 {
			continue
		}
		if err := partialThenFull(files, idxs, refs, source); err != nil {
			return nil, err
		}
	}
	return refs, nil
}

// partialThenFull hashes the head, and the tail only when the head matches
// another file in the size bucket. The full digest runs only when the partial
// hash also matches.
func partialThenFull(files []app.FileEntry, idxs []int, refs []domain.FileRef, source app.FileReader) error {
	headGroups, headOrder, err := groupByRange(files, idxs, source, 0)
	if err != nil {
		return err
	}
	for _, key := range headOrder {
		members := headGroups[key]
		if len(members) < 2 {
			continue
		}
		offset := files[members[0]].Size - partialWindow
		tailGroups, tailOrder, err := groupByRange(files, members, source, offset)
		if err != nil {
			return err
		}
		for _, tailKey := range tailOrder {
			same := tailGroups[tailKey]
			if len(same) < 2 {
				continue
			}
			for _, idx := range same {
				id, err := fullID(source, files[idx].Path)
				if err != nil {
					return err
				}
				refs[idx].ContentID = id
			}
		}
	}
	return nil
}

func groupByRange(files []app.FileEntry, idxs []int, source app.FileReader, offset int64) (map[string][]int, []string, error) {
	groups := make(map[string][]int)
	var order []string
	for _, idx := range idxs {
		sum, err := hashRange(source, files[idx].Path, offset, partialWindow)
		if err != nil {
			return nil, nil, err
		}
		key := string(sum[:])
		if _, ok := groups[key]; !ok {
			order = append(order, key)
		}
		groups[key] = append(groups[key], idx)
	}
	return groups, order, nil
}

func hashRange(source app.FileReader, path string, offset, n int64) ([32]byte, error) {
	f, err := source.Open(path)
	if err != nil {
		return [32]byte{}, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()
	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		return [32]byte{}, fmt.Errorf("seek %s: %w", path, err)
	}
	buf := make([]byte, n)
	if _, err := io.ReadFull(f, buf); err != nil {
		return [32]byte{}, fmt.Errorf("read %s: %w", path, err)
	}
	return blake3.Sum256(buf), nil
}

// ContentID hashes one known file in full. Reconcile calls this so a
// singleton large file still gets a digest. The staged Identify path is
// unchanged.
func (Hasher) ContentID(path string, source app.FileReader) (domain.ContentID, error) {
	return fullID(source, path)
}

func fullID(source app.FileReader, path string) (domain.ContentID, error) {
	f, err := source.Open(path)
	if err != nil {
		return "", fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()
	h := blake3.New(32, nil)
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("hash %s: %w", path, err)
	}
	return domain.ContentID(hex.EncodeToString(h.Sum(nil))), nil
}

var (
	_ app.Identifier      = Hasher{}
	_ app.ContentIDSource = Hasher{}
)
