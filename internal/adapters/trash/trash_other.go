//go:build !darwin && !linux && !windows

package trash

import (
	"fmt"

	"github.com/cionz0/hdsort/internal/app"
)

// System reports that this OS has no Trash adapter.
func System() (app.Trasher, error) {
	return nil, fmt.Errorf("trash is not supported on this OS")
}
