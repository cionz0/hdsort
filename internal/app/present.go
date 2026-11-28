package app

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

// SystemPreviewer is the operating-system preview. The use case does not
// own that call: it is the same kind of boundary as the folder dialog.
type SystemPreviewer interface {
	Preview(path string) ([]byte, bool, error)
}

// FormatSize renders a byte count the way a file list does: 39854 is "39 KB".
func FormatSize(n int64) string {
	if n < 0 {
		n = 0
	}
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div := int64(unit)
	exp := 0
	for n/div >= unit && exp < 3 {
		div *= unit
		exp++
	}
	value := float64(n) / float64(div)
	suffix := []string{"KB", "MB", "GB", "TB"}[exp]
	if value < 10 {
		return fmt.Sprintf("%.1f %s", value, suffix)
	}
	return fmt.Sprintf("%.0f %s", value, suffix)
}

// DisplayPath is path relative to the chosen scan root. A path outside
// that root stays unchanged.
func DisplayPath(root, path string) string {
	root = filepath.Clean(root)
	path = filepath.Clean(path)
	if root == "" || root == "." {
		return path
	}
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return path
	}
	return rel
}

// LoadPreview prefers a preview the operating system made. When that call
// returns nothing, a JPEG, PNG, or GIF still uses Thumbnail.
func LoadPreview(path string, r io.Reader, size int64, system SystemPreviewer) ([]byte, bool, error) {
	if system != nil {
		body, ok, err := system.Preview(path)
		if err == nil && ok && len(body) > 0 {
			return body, true, nil
		}
	}
	return Thumbnail(path, r, size)
}
