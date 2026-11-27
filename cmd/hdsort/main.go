// Command hdsort is the composition root. It wires adapters and opens the
// native review window.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/cionz0/hdsort/internal/adapters/filesystem"
	"github.com/cionz0/hdsort/internal/adapters/gui"
	"github.com/cionz0/hdsort/internal/adapters/hash"
	"github.com/cionz0/hdsort/internal/adapters/similar"
	"github.com/cionz0/hdsort/internal/adapters/sqlite"
	"github.com/cionz0/hdsort/internal/adapters/trash"
)

// runWindow is the product entry. Tests replace it so they do not open a window.
var runWindow = defaultWindow

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "hdsort: %s\n", err)
		os.Exit(1)
	}
}

func run() error {
	return runWindow()
}

func defaultWindow() error {
	store, err := sqlite.OpenDefault()
	if err != nil {
		return err
	}
	defer store.Close()
	trasher, err := trash.System()
	if err != nil {
		return err
	}
	lib, err := sqlite.DefaultPath()
	if err != nil {
		return err
	}
	return gui.Run(gui.Deps{
		Pick:    filesystem.NewSystemPicker(),
		Walk:    filesystem.NewWalker(nil),
		Files:   filesystem.NewOpener(),
		ID:      hash.New(),
		Similar: similar.New(),
		Library: store,
		Trash:   trasher,
		Disk:    filesystem.NewOpener(),
		DataDir: filepath.Dir(lib),
	})
}
