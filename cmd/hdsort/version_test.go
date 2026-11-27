package main

import (
	"os"
	"strings"
	"testing"
)

func TestAcceptance_VersionDefault(t *testing.T) {
	if got := Version(); got != "v0.0.0" {
		t.Fatalf("Version() = %q, want v0.0.0", got)
	}
}

func TestAcceptance_VersionFromLdflags(t *testing.T) {
	previous := version
	t.Cleanup(func() { version = previous })
	version = "v1.2.3"
	if got := Version(); got != "v1.2.3" {
		t.Fatalf("Version() = %q, want the ldflags value", got)
	}
}

func TestAcceptance_WorkflowMatrix(t *testing.T) {
	body, err := os.ReadFile("../../.github/workflows/test.yml")
	if err != nil {
		t.Fatalf("read workflow: %v", err)
	}
	text := string(body)
	for _, needle := range []string{"ubuntu-latest", "macos-latest", "windows-latest", "1.27.4"} {
		if !strings.Contains(text, needle) {
			t.Errorf("workflow missing %q", needle)
		}
	}
}
