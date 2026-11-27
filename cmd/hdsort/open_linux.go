//go:build linux

package main

import "os/exec"

func openBrowser(url string) {
	_ = exec.Command("xdg-open", url).Start()
}
