package trash

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/cionz0/hdsort/internal/app"
)

// Freedesktop moves a file into a Trash directory that follows the
// Freedesktop Trash specification: files/ holds the file, info/ holds the
// matching .trashinfo entry. The original is renamed, not removed.
//
// dir is the Trash directory itself. Tests pass a temporary directory.
// The home Trash ($XDG_DATA_HOME/Trash) is not opened here.
//
// Pattern: Ports and Adapters (Cockburn, 2005). See docs/patterns.md.
type Freedesktop struct {
	dir string
}

// NewFreedesktop returns an adapter for dir.
func NewFreedesktop(dir string) *Freedesktop {
	return &Freedesktop{dir: dir}
}

// Trash moves path into dir/files and writes dir/info/<name>.trashinfo.
func (f *Freedesktop) Trash(path string) error {
	if err := Validate(path); err != nil {
		return err
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	if _, err := os.Stat(abs); err != nil {
		return err
	}
	filesDir := filepath.Join(f.dir, "files")
	infoDir := filepath.Join(f.dir, "info")
	if err := os.MkdirAll(filesDir, 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(infoDir, 0o755); err != nil {
		return err
	}
	name, err := uniqueName(filesDir, infoDir, filepath.Base(abs))
	if err != nil {
		return err
	}
	if err := os.Rename(abs, filepath.Join(filesDir, name)); err != nil {
		return err
	}
	body := fmt.Sprintf("[Trash Info]\nPath=%s\nDeletionDate=%s\n",
		escapePath(abs),
		time.Now().Format("2006-01-02T15:04:05"),
	)
	return os.WriteFile(filepath.Join(infoDir, name+".trashinfo"), []byte(body), 0o644)
}

func uniqueName(filesDir, infoDir, base string) (string, error) {
	name := base
	for n := 0; n < 10000; n++ {
		if n > 0 {
			name = fmt.Sprintf("%s.%d", base, n)
		}
		_, fileErr := os.Lstat(filepath.Join(filesDir, name))
		_, infoErr := os.Lstat(filepath.Join(infoDir, name+".trashinfo"))
		if errors.Is(fileErr, os.ErrNotExist) && errors.Is(infoErr, os.ErrNotExist) {
			return name, nil
		}
		if fileErr != nil && !errors.Is(fileErr, os.ErrNotExist) {
			return "", fileErr
		}
		if infoErr != nil && !errors.Is(infoErr, os.ErrNotExist) {
			return "", infoErr
		}
	}
	return "", fmt.Errorf("trash: too many copies of %s", base)
}

// escapePath percent-encodes a Trash Info path, leaving "/" literal.
// The Trash specification encodes the absolute path the way URIs do.
func escapePath(path string) string {
	var b []byte
	for i := 0; i < len(path); i++ {
		c := path[i]
		if c == '/' || isUnreserved(c) {
			b = append(b, c)
			continue
		}
		b = append(b, '%', hexDigit(c>>4), hexDigit(c&0x0f))
	}
	return string(b)
}

var _ app.Trasher = (*Freedesktop)(nil)

func isUnreserved(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_' || c == '.' || c == '~'
}

func hexDigit(n byte) byte {
	if n < 10 {
		return '0' + n
	}
	return 'A' + n - 10
}
