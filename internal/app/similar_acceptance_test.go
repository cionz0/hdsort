package app_test

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cionz0/hdsort/internal/domain"
)

func TestAcceptance_SimilarImagesGroupWithoutRecommendation(t *testing.T) {
	root := t.TempDir()
	solid := solidPNG(color.NRGBA{R: 180, G: 40, B: 40, A: 255})
	changed := cloneNRGBA(solid)
	changed.Set(3, 3, color.NRGBA{R: 181, G: 40, B: 40, A: 255})
	left := filepath.Join(root, "red.png")
	right := filepath.Join(root, "red-bit.png")
	writePNG(t, left, solid)
	writePNG(t, right, changed)

	result := scan(t, root, "", nil)
	if len(result.Session.Similar) != 1 {
		t.Fatalf("similar groups = %d, want 1", len(result.Session.Similar))
	}
	group := result.Session.Similar[0]
	members := group.Members()
	if len(members) != 2 {
		t.Fatalf("members = %d, want 2", len(members))
	}
	if members[0].ContentID == "" || members[0].ContentID == members[1].ContentID {
		t.Fatalf("content IDs %q and %q", members[0].ContentID, members[1].ContentID)
	}
	if _, ok := group.Recommendation(); ok {
		t.Fatal("similar group has a delete recommendation")
	}

	store := openStore(t)
	id, err := store.Save(result.Session)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Load(id)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Similar) != 1 {
		t.Fatalf("stored similar groups = %d, want 1", len(loaded.Similar))
	}
	loadedMembers := loaded.Similar[0].Members()
	if len(loadedMembers) != 2 || loadedMembers[0].Path != members[0].Path || loadedMembers[1].Path != members[1].Path {
		t.Fatalf("stored members = %+v", loadedMembers)
	}
	if _, ok := loaded.Similar[0].Recommendation(); ok {
		t.Fatal("stored similar group has a delete recommendation")
	}
}

func TestAcceptance_IdenticalImagesAreExactOnly(t *testing.T) {
	root := t.TempDir()
	body := encodePNG(t, solidPNG(color.NRGBA{R: 20, G: 80, B: 140, A: 255}))
	writeFile(t, filepath.Join(root, "a.png"), body)
	writeFile(t, filepath.Join(root, "b.png"), body)

	result := scan(t, root, "", nil)
	if len(result.Session.Exact) != 1 {
		t.Fatalf("exact groups = %d, want 1", len(result.Session.Exact))
	}
	if len(result.Session.Similar) != 0 {
		t.Fatalf("similar groups = %d, want 0", len(result.Session.Similar))
	}
}

func TestAcceptance_SimilarTextGroupsWithoutRecommendation(t *testing.T) {
	root := t.TempDir()
	base := strings.Repeat("the quick brown fox jumps over the lazy dog ", 40)
	writeFile(t, filepath.Join(root, "a.txt"), []byte(base))
	writeFile(t, filepath.Join(root, "b.txt"), []byte(strings.Replace(base, "lazy", "sleepy", 1)))

	result := scan(t, root, "", nil)
	if len(result.Session.Exact) != 0 {
		t.Fatalf("exact groups = %d, want 0", len(result.Session.Exact))
	}
	if len(result.Session.Similar) != 1 {
		t.Fatalf("similar groups = %d, want 1", len(result.Session.Similar))
	}
	members := result.Session.Similar[0].Members()
	if len(members) != 2 || members[0].ContentID == members[1].ContentID {
		t.Fatalf("members = %+v", members)
	}
	if _, ok := result.Session.Similar[0].Recommendation(); ok {
		t.Fatal("similar group has a delete recommendation")
	}
}

func TestAcceptance_DistantTextDoesNotGroup(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "a.txt"), []byte("alpha bravo charlie delta echo foxtrot golf hotel"))
	writeFile(t, filepath.Join(root, "b.txt"), []byte("wallet rocket marble violin copper zipper quartz lantern"))

	result := scan(t, root, "", nil)
	if len(result.Session.Similar) != 0 {
		t.Fatalf("similar groups = %d, want 0", len(result.Session.Similar))
	}
	if len(result.Session.Exact) != 0 {
		t.Fatalf("exact groups = %d, want 0", len(result.Session.Exact))
	}
}

func TestAcceptance_UserCanStillMarkSimilarForDelete(t *testing.T) {
	root := t.TempDir()
	base := strings.Repeat("the quick brown fox jumps over the lazy dog ", 40)
	writeFile(t, filepath.Join(root, "a.txt"), []byte(base))
	writeFile(t, filepath.Join(root, "b.txt"), []byte(strings.Replace(base, "lazy", "sleepy", 1)))

	result := scan(t, root, "", nil)
	if len(result.Session.Similar) != 1 {
		t.Fatalf("similar groups = %d, want 1", len(result.Session.Similar))
	}
	member := result.Session.Similar[0].Members()[0].Path
	next, err := domain.RecordDecision(result.Session, member, domain.ChoiceDelete)
	if err != nil {
		t.Fatal(err)
	}
	if len(next.Decisions) != 1 || next.Decisions[0].Path != member || next.Decisions[0].Choice != domain.ChoiceDelete {
		t.Fatalf("decisions = %+v", next.Decisions)
	}
	if _, ok := next.Similar[0].Recommendation(); ok {
		t.Fatal("similar group gained a delete recommendation")
	}
}

func cloneNRGBA(src *image.NRGBA) *image.NRGBA {
	dst := image.NewNRGBA(src.Bounds())
	copy(dst.Pix, src.Pix)
	return dst
}

func solidPNG(c color.Color) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			img.Set(x, y, c)
		}
	}
	return img
}

func writePNG(t *testing.T, path string, img image.Image) {
	t.Helper()
	writeFile(t, path, encodePNG(t, img))
}

func encodePNG(t *testing.T, img image.Image) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}
