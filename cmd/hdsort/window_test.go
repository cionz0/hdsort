package main

import "testing"

func TestAcceptance_CommandStartsWindow(t *testing.T) {
	var ran bool
	previous := runWindow
	runWindow = func() error {
		ran = true
		return nil
	}
	t.Cleanup(func() { runWindow = previous })
	if err := run(); err != nil {
		t.Fatal(err)
	}
	if !ran {
		t.Fatal("window was not started")
	}
}
