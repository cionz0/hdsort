package filesystem

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/cionz0/hdsort/internal/app"
)

// Probe reports the platform dataless flag for one file.
// A temporary file cannot raise that flag, so tests pass a probe.
// A nil probe uses the OS check. The probe is not a way to skip hashing
// in production.
type Probe interface {
	Dataless(path string, info fs.FileInfo) bool
}

type osProbe struct{}

// Walker walks a directory tree. It does not follow symlinks.
type Walker struct {
	probe Probe
}

// NewWalker returns a walker. A nil probe uses the OS dataless flag.
func NewWalker(probe Probe) *Walker {
	if probe == nil {
		probe = osProbe{}
	}
	return &Walker{probe: probe}
}

// Walk returns regular files under root, plus symlinks marked as symlinks.
// Directories named .git or node_modules are skipped, and so is dataDir
// when it is non-empty. Symlinks are not followed. Each one is returned
// with its destination so the scan can show it as a symlink.
func (w *Walker) Walk(root, dataDir string) ([]app.FileEntry, error) {
	root = filepath.Clean(root)
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("scan root is not a directory: %s", root)
	}
	if dataDir != "" {
		dataDir = filepath.Clean(dataDir)
	}

	var entries []app.FileEntry
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		path = filepath.Clean(path)
		if path != root && d.Type()&fs.ModeSymlink != 0 {
			if inside(dataDir, path) {
				return nil
			}
			target, err := os.Readlink(path)
			if err != nil {
				return err
			}
			entries = append(entries, app.FileEntry{
				Path:         path,
				Availability: app.AvailabilitySymlink,
				Target:       target,
			})
			return nil
		}
		if d.IsDir() {
			if path != root && (skipDirName(d.Name()) || inside(dataDir, path)) {
				return fs.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		if inside(dataDir, path) {
			return nil
		}
		fileInfo, err := d.Info()
		if err != nil {
			return err
		}
		availability := app.AvailabilityReadable
		if w.probe.Dataless(path, fileInfo) {
			availability = app.AvailabilityDataless
		}
		entries = append(entries, app.FileEntry{
			Path:         path,
			Size:         fileInfo.Size(),
			ModTime:      fileInfo.ModTime(),
			Availability: availability,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return entries, nil
}

func skipDirName(name string) bool {
	return name == ".git" || name == "node_modules"
}

func inside(dir, path string) bool {
	if dir == "" {
		return false
	}
	rel, err := filepath.Rel(dir, path)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// Opener opens files for the hasher.
type Opener struct{}

// NewOpener returns the OS file opener.
func NewOpener() Opener { return Opener{} }

// Open opens path for reading.
func (Opener) Open(path string) (io.ReadSeekCloser, error) {
	return os.Open(path)
}

// Facts reports size and modification time. A missing path or a path that
// is not a regular file is Exists false and not an error.
func (Opener) Facts(path string) (app.FileFacts, error) {
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return app.FileFacts{}, nil
		}
		return app.FileFacts{}, err
	}
	if !info.Mode().IsRegular() {
		return app.FileFacts{}, nil
	}
	return app.FileFacts{
		Exists:  true,
		Size:    info.Size(),
		ModTime: info.ModTime(),
	}, nil
}

var (
	_ app.Walker        = (*Walker)(nil)
	_ app.FileReader    = Opener{}
	_ app.ReconcileDisk = Opener{}
)
