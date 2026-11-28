package app_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/cionz0/hdsort/internal/app"
)

func TestAcceptance_OutlineUsesOneNamePerLevel(t *testing.T) {
	root := t.TempDir()
	a := filepath.Join(root, "a")
	b := filepath.Join(a, "b")
	c := filepath.Join(a, "c")
	rows := []app.FolderRow{
		{Key: "b-set", Folder: b, Title: "photo.jpg", Kind: app.RowExact},
		{Key: "c-set", Folder: c, Title: "notes.txt", Kind: app.RowExact},
	}
	nodes := app.Outline(root, rows, []string{root, a, b, c})
	wantNames := []string{filepath.Base(root), "a", "b", "photo.jpg", "c", "notes.txt"}
	wantDepth := []int{0, 1, 2, 3, 2, 3}
	if len(nodes) != len(wantNames) {
		t.Fatalf("nodes = %v", outlineNames(nodes))
	}
	sep := string(filepath.Separator)
	for i, node := range nodes {
		if node.Name != wantNames[i] || node.Depth != wantDepth[i] {
			t.Fatalf("node %d = %s depth %d", i, node.Name, node.Depth)
		}
		if strings.Contains(node.Name, sep) {
			t.Fatalf("name %q contains a separator", node.Name)
		}
	}
}

func TestAcceptance_CollapsedDirectoryHidesItsContents(t *testing.T) {
	root := t.TempDir()
	a := filepath.Join(root, "a")
	b := filepath.Join(a, "b")
	c := filepath.Join(a, "c")
	rows := []app.FolderRow{
		{Key: "b-set", Folder: b, Title: "photo.jpg"},
		{Key: "c-set", Folder: c, Title: "notes.txt"},
	}
	closed := app.Outline(root, rows, []string{root})
	if got := outlineNames(closed); len(got) != 2 || got[0] != filepath.Base(root) || got[1] != "a" {
		t.Fatalf("collapsed = %v", got)
	}
	opened := app.Outline(root, rows, []string{root, a})
	if got := outlineNames(opened); len(got) != 4 || got[2] != "b" || got[3] != "c" {
		t.Fatalf("a expanded = %v", got)
	}
	for _, name := range outlineNames(opened) {
		if name == "photo.jpg" || name == "notes.txt" {
			t.Fatalf("set %s is visible", name)
		}
	}
}

func TestAcceptance_IntermediateFoldersStayInTheOutline(t *testing.T) {
	root := t.TempDir()
	a := filepath.Join(root, "a")
	b := filepath.Join(a, "b")
	c := filepath.Join(b, "c")
	rows := []app.FolderRow{{Key: "deep", Folder: c, Title: "photo.jpg"}}
	nodes := app.Outline(root, rows, []string{root, a, b, c})
	got := outlineNames(nodes)
	for _, name := range []string{"a", "b", "c", "photo.jpg"} {
		if !containsName(got, name) {
			t.Fatalf("names = %v, missing %s", got, name)
		}
	}
}

func TestAcceptance_FolderOutsideTheRootStaysOneNode(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	rows := []app.FolderRow{{Key: "lib", Folder: outside, Title: "lib.jpg"}}
	nodes := app.Outline(root, rows, []string{root, outside})
	var found bool
	for _, node := range nodes {
		if node.Directory && node.Name == filepath.Clean(outside) {
			found = true
		}
		parent := filepath.Dir(outside)
		if node.Path == parent || node.Name == parent {
			t.Fatalf("invented parent %s", parent)
		}
	}
	if !found {
		t.Fatalf("nodes = %v, want %s", outlineNames(nodes), outside)
	}
}

func outlineNames(nodes []app.OutlineNode) []string {
	names := make([]string, len(nodes))
	for i, node := range nodes {
		names[i] = node.Name
	}
	return names
}

func containsName(names []string, want string) bool {
	for _, name := range names {
		if name == want {
			return true
		}
	}
	return false
}
