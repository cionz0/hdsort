//go:build darwin

package gui

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestAcceptance_QuickLookPreviewsAFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "shot.png")
	img := image.NewNRGBA(image.Rect(0, 0, 32, 24))
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	body, ok, err := systemPreview(path)
	if err != nil || !ok {
		t.Fatalf("preview ok=%v err=%v", ok, err)
	}
	if _, _, err := image.Decode(bytes.NewReader(body)); err != nil {
		t.Fatal(err)
	}
}
