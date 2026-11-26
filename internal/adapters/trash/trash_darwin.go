//go:build darwin

package trash

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/cionz0/hdsort/internal/app"
)

// Platform moves a file to the macOS Trash through Finder.
// go test does not call Trash; it checks Validate, which Trash calls first.
type Platform struct{}

// NewPlatform returns the macOS Trash adapter.
func NewPlatform() Platform { return Platform{} }

// Trash asks Finder to move path to the Trash. An empty path is rejected
// before Finder is started.
func (Platform) Trash(path string) error {
	if err := Validate(path); err != nil {
		return err
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	script := fmt.Sprintf("tell application \"Finder\" to delete POSIX file %s", strconv.Quote(abs))
	out, err := exec.Command("osascript", "-e", script).CombinedOutput()
	if err != nil {
		return fmt.Errorf("trash: finder: %w: %s", err, out)
	}
	return nil
}

var _ app.Trasher = Platform{}
