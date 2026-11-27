// Command hdsort is the composition root. It wires adapters and starts the
// loopback review UI.
package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/cionz0/hdsort/internal/adapters/filesystem"
	"github.com/cionz0/hdsort/internal/adapters/hash"
	httpadapter "github.com/cionz0/hdsort/internal/adapters/http"
	"github.com/cionz0/hdsort/internal/adapters/similar"
	"github.com/cionz0/hdsort/internal/adapters/sqlite"
	"github.com/cionz0/hdsort/internal/adapters/trash"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "hdsort: %s\n", err)
		os.Exit(1)
	}
}

func run() error {
	token, err := httpadapter.NewToken()
	if err != nil {
		return err
	}
	ln, err := httpadapter.Listen("127.0.0.1:0")
	if err != nil {
		return err
	}
	store, err := sqlite.OpenDefault()
	if err != nil {
		return err
	}
	defer store.Close()
	trasher, err := trash.System()
	if err != nil {
		return err
	}
	server := httpadapter.New(token, store, store, trasher, filesystem.NewOpener()).
		WithScan(filesystem.NewWalker(nil), filesystem.NewOpener(), hash.New(), similar.New(), filesystem.NewSystemPicker())
	url := fmt.Sprintf("http://%s/?token=%s", ln.Addr().String(), token)
	fmt.Println(url)
	go openBrowser(url)
	return http.Serve(ln, server)
}
