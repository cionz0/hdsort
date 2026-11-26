package domain

import (
	"fmt"
	"path/filepath"
	"strings"
)

// LayoutMove is a proposed rename or relocation. It stays inactive until the
// owner confirms it. NewLayoutMove rejects a destination outside the scan root.
type LayoutMove struct {
	From        string
	Destination string
}

// NewLayoutMove cleans root and destination and rejects a destination that
// escapes the scan root. filepath.Rel must not be ".." and must not start
// with ".." plus a separator (ADR-0005).
func NewLayoutMove(root, from, destination string) (LayoutMove, error) {
	if err := destinationInsideRoot(root, destination); err != nil {
		return LayoutMove{}, err
	}
	return LayoutMove{
		From:        filepath.Clean(from),
		Destination: filepath.Clean(destination),
	}, nil
}

func destinationInsideRoot(root, destination string) error {
	if root == "" || destination == "" {
		return fmt.Errorf("%w: empty path", ErrMoveOutsideRoot)
	}
	root = filepath.Clean(root)
	destination = filepath.Clean(destination)
	rel, err := filepath.Rel(root, destination)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrMoveOutsideRoot, destination)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("%w: %s", ErrMoveOutsideRoot, destination)
	}
	return nil
}
