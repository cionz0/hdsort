package app_test

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/cionz0/hdsort/internal/adapters/filesystem"
	"github.com/cionz0/hdsort/internal/adapters/hash"
	"github.com/cionz0/hdsort/internal/app"
	"github.com/cionz0/hdsort/internal/domain"
)

const (
	partialWindow = 64 * 1024
	// emptyBLAKE3 is the BLAKE3-256 digest of an empty input.
	emptyBLAKE3 = "af1349b9f5f9a1a6a0404dea36dcc9499bcb25c9adc112b7cc9a93cae41f3262"
)

func TestAcceptance_IdenticalBytesShareContentID(t *testing.T) {
	tests := []struct {
		name string
		a, b string
		body []byte
		want domain.ContentID
	}{
		{name: "empty", a: "empty-a", b: "empty-b", want: emptyBLAKE3},
		{name: "same bytes", a: "a.txt", b: filepath.Join("nested", "b.txt"), body: []byte("same-bytes")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			pathA := filepath.Join(root, tt.a)
			pathB := filepath.Join(root, tt.b)
			writeFile(t, pathA, tt.body)
			writeFile(t, pathB, tt.body)

			result := scan(t, root, "", nil)
			idA := mustFile(t, result.Files, pathA).ContentID
			idB := mustFile(t, result.Files, pathB).ContentID
			if idA == "" || idA != idB {
				t.Fatalf("content IDs %q and %q", idA, idB)
			}
			if tt.want != "" && idA != tt.want {
				t.Fatalf("content ID = %s, want %s", idA, tt.want)
			}
		})
	}
}

func TestAcceptance_DifferentMiddleDifferentContentID(t *testing.T) {
	root := t.TempDir()
	pathA := filepath.Join(root, "a.bin")
	pathB := filepath.Join(root, "b.bin")
	writeFile(t, pathA, fileWithMiddle('A'))
	writeFile(t, pathB, fileWithMiddle('B'))

	result := scan(t, root, "", nil)
	idA := mustFile(t, result.Files, pathA).ContentID
	idB := mustFile(t, result.Files, pathB).ContentID
	if idA == "" || idB == "" || idA == idB {
		t.Fatalf("content IDs %q and %q", idA, idB)
	}
}

func TestAcceptance_SmallFilesSkipPartialFilter(t *testing.T) {
	root := t.TempDir()
	pathA := filepath.Join(root, "a.txt")
	pathB := filepath.Join(root, "b.txt")
	writeFile(t, pathA, []byte("alpha"))
	writeFile(t, pathB, []byte("beta"))

	info, err := os.Stat(pathA)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() >= 128*1024 {
		t.Fatalf("size = %d, want a file under 128 KiB", info.Size())
	}
	result := scan(t, root, "", nil)
	idA := mustFile(t, result.Files, pathA).ContentID
	idB := mustFile(t, result.Files, pathB).ContentID
	if idA == "" || idB == "" || idA == idB {
		t.Fatalf("content IDs %q and %q", idA, idB)
	}
}

func TestAcceptance_ExactGroupOfThree(t *testing.T) {
	root := t.TempDir()
	copies := []string{
		filepath.Join(root, "one.txt"),
		filepath.Join(root, "sub", "two.txt"),
		filepath.Join(root, "sub", "three.txt"),
	}
	for _, path := range copies {
		writeFile(t, path, []byte("duplicate"))
	}
	unique := filepath.Join(root, "unique.txt")
	writeFile(t, unique, []byte("just-once"))

	result := scan(t, root, "", nil)
	if result.Session.Status() != domain.StatusOpen {
		t.Fatalf("status = %q, want open", result.Session.Status())
	}
	if len(result.Session.Exact) != 1 {
		t.Fatalf("groups = %d, want 1", len(result.Session.Exact))
	}
	got := map[string]domain.ContentID{}
	for _, member := range result.Session.Exact[0].Members() {
		got[member.Path] = member.ContentID
	}
	if len(got) != 3 {
		t.Fatalf("members = %d, want 3", len(got))
	}
	var shared domain.ContentID
	for _, path := range copies {
		id, ok := got[path]
		if !ok || id == "" {
			t.Fatalf("missing member %s", path)
		}
		if shared == "" {
			shared = id
		}
		if id != shared {
			t.Fatalf("member %s content ID = %s, want %s", path, id, shared)
		}
	}
	if _, ok := got[unique]; ok {
		t.Fatal("unique file is in the exact group")
	}
	uniqueRef := mustFile(t, result.Files, unique)
	if uniqueRef.ContentID == "" || uniqueRef.ContentID == shared {
		t.Fatalf("unique content ID = %s", uniqueRef.ContentID)
	}
}

func TestAcceptance_SymlinkNotFollowed(t *testing.T) {
	t.Run("file", func(t *testing.T) {
		root := t.TempDir()
		real := filepath.Join(root, "real.txt")
		link := filepath.Join(root, "link.txt")
		writeFile(t, real, []byte("hello"))
		if err := os.Symlink("real.txt", link); err != nil {
			t.Fatalf("symlink: %v", err)
		}

		result := scan(t, root, "", nil)
		if _, ok := findFile(result.Files, link); ok {
			t.Fatal("symlink is a group candidate")
		}
		if len(result.Session.Exact) != 0 {
			t.Fatalf("groups = %d, want 0", len(result.Session.Exact))
		}
		if mustFile(t, result.Files, real).ContentID == "" {
			t.Fatal("real file has no content ID")
		}
		assertSymlink(t, result.Session.Skipped, link, "real.txt")
	})

	t.Run("directory", func(t *testing.T) {
		root := t.TempDir()
		inside := filepath.Join(root, "sub", "inside.txt")
		link := filepath.Join(root, "linkdir")
		writeFile(t, inside, []byte("only-through-the-real-directory"))
		if err := os.Symlink("sub", link); err != nil {
			t.Fatalf("symlink: %v", err)
		}

		result := scan(t, root, "", nil)
		if _, ok := findFile(result.Files, filepath.Join(link, "inside.txt")); ok {
			t.Fatal("symlink directory was followed")
		}
		if len(result.Session.Exact) != 0 {
			t.Fatalf("groups = %d, want 0", len(result.Session.Exact))
		}
		if mustFile(t, result.Files, inside).ContentID == "" {
			t.Fatal("real file has no content ID")
		}
		assertSymlink(t, result.Session.Skipped, link, "sub")
	})
}

func assertSymlink(t *testing.T, skipped []domain.Skipped, path, target string) {
	t.Helper()
	for _, item := range skipped {
		if item.Path != path {
			continue
		}
		if item.Reason != domain.SkipSymlink {
			t.Fatalf("reason = %q, want symlink", item.Reason)
		}
		if item.Target != target {
			t.Fatalf("target = %q, want %q", item.Target, target)
		}
		return
	}
	t.Fatalf("skipped = %+v, want %s marked as a symlink to %s", skipped, path, target)
}

func TestAcceptance_SkipGitAndNodeModules(t *testing.T) {
	root := t.TempDir()
	keep := filepath.Join(root, "keep.txt")
	skipped := []string{
		filepath.Join(root, ".git", "config"),
		filepath.Join(root, "node_modules", "pkg", "index.js"),
		filepath.Join(root, "sub", ".git", "objects", "ab"),
		filepath.Join(root, "src", "node_modules", "x.js"),
	}
	writeFile(t, keep, []byte("keep"))
	for _, path := range skipped {
		writeFile(t, path, []byte("skip"))
	}

	result := scan(t, root, "", nil)
	if _, ok := findFile(result.Files, keep); !ok {
		t.Fatal("kept file was not hashed")
	}
	for _, path := range skipped {
		if ref, ok := findFile(result.Files, path); ok {
			t.Fatalf("hashed %s with content ID %s", path, ref.ContentID)
		}
	}
}

func TestAcceptance_SkipDataDirectory(t *testing.T) {
	root := t.TempDir()
	dataDir := filepath.Join(root, "hdsort-data")
	keep := filepath.Join(root, "photo.jpg")
	library := filepath.Join(dataDir, "library.sqlite")
	writeFile(t, keep, []byte("photo"))
	writeFile(t, library, []byte("sqlite"))

	result := scan(t, root, dataDir, nil)
	if _, ok := findFile(result.Files, keep); !ok {
		t.Fatal("file outside the data directory was not hashed")
	}
	if _, ok := findFile(result.Files, library); ok {
		t.Fatal("data directory file was hashed")
	}
}

func TestAcceptance_DatalessFileSkipped(t *testing.T) {
	root := t.TempDir()
	present := filepath.Join(root, "present.txt")
	cloud := filepath.Join(root, "cloud.txt")
	writeFile(t, present, []byte("local"))
	writeFile(t, cloud, []byte("not-local"))

	result := scan(t, root, "", datalessProbe{path: cloud})
	if ref, ok := findFile(result.Files, cloud); ok {
		t.Fatalf("dataless file content ID = %s, want none", ref.ContentID)
	}
	if mustFile(t, result.Files, present).ContentID == "" {
		t.Fatal("present file has no content ID")
	}
	var found bool
	for _, skipped := range result.Session.Skipped {
		if skipped.Path == cloud && skipped.Reason == domain.SkipDataless {
			found = true
		}
	}
	if !found {
		t.Fatalf("skipped = %+v, want %s as dataless", result.Session.Skipped, cloud)
	}
}

func TestAcceptance_PartialHashDoesNotFullReadDifferentHeads(t *testing.T) {
	root := t.TempDir()
	pathA := filepath.Join(root, "a.bin")
	pathB := filepath.Join(root, "b.bin")
	bodyA := fileWithHead('A')
	bodyB := fileWithHead('B')
	writeFile(t, pathA, bodyA)
	writeFile(t, pathB, bodyB)

	counter := &countingSource{inner: filesystem.NewOpener()}
	result, err := app.Scan(root, "", filesystem.NewWalker(nil), counter, hash.New())
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{pathA, pathB} {
		n := counter.n[path]
		if n > 2*partialWindow {
			t.Fatalf("%s read %d bytes, past the 128 KiB partial window", path, n)
		}
		if n >= int64(len(bodyA)) {
			t.Fatalf("%s read %d bytes of a %d byte file", path, n, len(bodyA))
		}
		ref := mustFile(t, result.Files, path)
		if ref.ContentID != "" {
			t.Fatalf("%s content ID = %s, want none", path, ref.ContentID)
		}
	}
	if len(result.Session.Exact) != 0 {
		t.Fatalf("groups = %d, want 0", len(result.Session.Exact))
	}
}

func scan(t *testing.T, root, dataDir string, probe filesystem.Probe) app.Result {
	t.Helper()
	result, err := app.Scan(root, dataDir, filesystem.NewWalker(probe), filesystem.NewOpener(), hash.New())
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func writeFile(t *testing.T, path string, body []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatal(err)
	}
}

func mustFile(t *testing.T, files []domain.FileRef, path string) domain.FileRef {
	t.Helper()
	ref, ok := findFile(files, path)
	if !ok {
		t.Fatalf("missing file %s", path)
	}
	return ref
}

func findFile(files []domain.FileRef, path string) (domain.FileRef, bool) {
	for _, file := range files {
		if file.Path == path {
			return file, true
		}
	}
	return domain.FileRef{}, false
}

func fileWithMiddle(middle byte) []byte {
	buf := make([]byte, partialWindow*3)
	for i := 0; i < partialWindow; i++ {
		buf[i] = 'H'
		buf[partialWindow+i] = middle
		buf[partialWindow*2+i] = 'T'
	}
	return buf
}

func fileWithHead(head byte) []byte {
	buf := make([]byte, partialWindow*3)
	for i := range buf {
		buf[i] = 'Z'
	}
	for i := 0; i < partialWindow; i++ {
		buf[i] = head
	}
	return buf
}

type datalessProbe struct {
	path string
}

func (p datalessProbe) Dataless(path string, _ os.FileInfo) bool {
	return path == p.path
}

type countingSource struct {
	inner app.FileReader
	n     map[string]int64
}

func (c *countingSource) Open(path string) (io.ReadSeekCloser, error) {
	file, err := c.inner.Open(path)
	if err != nil {
		return nil, err
	}
	if c.n == nil {
		c.n = map[string]int64{}
	}
	return &countingFile{ReadSeekCloser: file, path: path, src: c}, nil
}

type countingFile struct {
	io.ReadSeekCloser
	path string
	src  *countingSource
}

func (f *countingFile) Read(p []byte) (int, error) {
	n, err := f.ReadSeekCloser.Read(p)
	f.src.n[f.path] += int64(n)
	return n, err
}
