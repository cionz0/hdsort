package trash_test

import (
	"errors"
	"os"
	"testing"

	"github.com/cionz0/hdsort/internal/adapters/trash"
)

func TestAcceptance_TrashRejectsEmptyPath(t *testing.T) {
	if err := trash.Validate(""); !errors.Is(err, trash.ErrEmptyPath) {
		t.Fatalf("Validate error = %v, want empty path", err)
	}
	if err := trash.Validate("   "); !errors.Is(err, trash.ErrEmptyPath) {
		t.Fatalf("Validate blank error = %v, want empty path", err)
	}
	dir := t.TempDir()
	if err := trash.NewFreedesktop(dir).Trash(""); !errors.Is(err, trash.ErrEmptyPath) {
		t.Fatalf("Trash error = %v, want empty path", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("empty path created %d trash entries", len(entries))
	}
}
