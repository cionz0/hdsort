package similar

import (
	"bytes"
	"image"
	"io"
	"math/bits"
	"path/filepath"
	"strings"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"github.com/cionz0/hdsort/internal/app"
	"github.com/cionz0/hdsort/internal/domain"
)

const (
	// imageHamming is the average-hash cutoff from ADR-0010.
	imageHamming = 10
	// textHamming is the simhash cutoff from ADR-0010.
	textHamming = 3
)

const (
	kindImage = iota
	kindText
)

// Strategy is the similarity policy the scan calls after exact grouping.
type Strategy struct{}

// New returns the ADR-0010 strategy.
func New() Strategy { return Strategy{} }

type finger struct {
	file  domain.FileRef
	bits  uint64
	kind  int
	limit int
}

// Similar returns connected components of files that are close and not
// byte-identical. A file that shares its content ID with another file is
// omitted. The groups have no delete recommendation.
func (Strategy) Similar(files []domain.FileRef, source app.FileReader) ([]domain.SimilarGroup, error) {
	counts := map[domain.ContentID]int{}
	for _, file := range files {
		if file.Readable && file.ContentID != "" {
			counts[file.ContentID]++
		}
	}
	var prints []finger
	for _, file := range files {
		if !file.Readable || file.ContentID == "" || counts[file.ContentID] != 1 {
			continue
		}
		print, ok, err := fingerprint(file, source)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		prints = append(prints, print)
	}
	return components(prints)
}

func fingerprint(file domain.FileRef, source app.FileReader) (finger, bool, error) {
	switch strings.ToLower(filepath.Ext(file.Path)) {
	case ".jpg", ".jpeg", ".png", ".gif":
		body, err := readAll(source, file.Path)
		if err != nil {
			return finger{}, false, err
		}
		img, _, err := image.Decode(bytes.NewReader(body))
		if err != nil {
			return finger{}, false, nil
		}
		return finger{file: file, bits: averageHash(img), kind: kindImage, limit: imageHamming}, true, nil
	case ".txt", ".md":
		body, err := readAll(source, file.Path)
		if err != nil {
			return finger{}, false, err
		}
		return finger{file: file, bits: simhash(string(body)), kind: kindText, limit: textHamming}, true, nil
	default:
		return finger{}, false, nil
	}
}

func readAll(source app.FileReader, path string) ([]byte, error) {
	f, err := source.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}

func components(prints []finger) ([]domain.SimilarGroup, error) {
	n := len(prints)
	parent := make([]int, n)
	for i := range parent {
		parent[i] = i
	}
	var find func(int) int
	find = func(i int) int {
		if parent[i] != i {
			parent[i] = find(parent[i])
		}
		return parent[i]
	}
	union := func(a, b int) {
		ra, rb := find(a), find(b)
		if ra == rb {
			return
		}
		if ra < rb {
			parent[rb] = ra
		} else {
			parent[ra] = rb
		}
	}
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			if prints[i].kind != prints[j].kind {
				continue
			}
			if bits.OnesCount64(prints[i].bits^prints[j].bits) <= prints[i].limit {
				union(i, j)
			}
		}
	}
	buckets := map[int][]domain.FileRef{}
	var order []int
	for i, print := range prints {
		root := find(i)
		if _, ok := buckets[root]; !ok {
			order = append(order, root)
		}
		buckets[root] = append(buckets[root], print.file)
	}
	var groups []domain.SimilarGroup
	for _, root := range order {
		members := buckets[root]
		if len(members) < 2 {
			continue
		}
		group, err := domain.NewSimilarGroup(members)
		if err != nil {
			return nil, err
		}
		groups = append(groups, group)
	}
	return groups, nil
}

var _ app.Similarizer = Strategy{}
