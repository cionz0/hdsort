//go:build !windows

package filesystem

import (
	"os/exec"
	"runtime"
)

func launch(path string) error {
	name := "xdg-open"
	if runtime.GOOS == "darwin" {
		name = "open"
	}
	return exec.Command(name, path).Run()
}
