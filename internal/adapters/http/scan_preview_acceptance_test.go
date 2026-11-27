package http_test

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cionz0/hdsort/internal/adapters/filesystem"
	"github.com/cionz0/hdsort/internal/adapters/hash"
	httpadapter "github.com/cionz0/hdsort/internal/adapters/http"
	"github.com/cionz0/hdsort/internal/adapters/similar"
	"github.com/cionz0/hdsort/internal/adapters/trash"
	"github.com/cionz0/hdsort/internal/app"
)

func TestAcceptance_PostScanCreatesSession(t *testing.T) {
	root := t.TempDir()
	body := encodePNG(t, solidPNG(color.NRGBA{R: 10, G: 20, B: 30, A: 255}))
	writeBytes(t, filepath.Join(root, "a.png"), body)
	writeBytes(t, filepath.Join(root, "b.png"), body)

	server := scanServer(t, root)
	got := postScan(t, server, `{"path":"`+root+`"}`)
	if got.ID == "" {
		t.Fatal("missing session id")
	}
	if len(got.Exact) != 1 {
		t.Fatalf("exact groups = %d, want 1", len(got.Exact))
	}
}

func TestAcceptance_PostScanExactOnlyOmitsSimilar(t *testing.T) {
	root := nearPNGs(t)
	server := scanServer(t, root)

	home := httptest.NewRecorder()
	server.ServeHTTP(home, httptest.NewRequest(http.MethodGet, "/?token="+testToken, nil))
	if home.Code != http.StatusOK {
		t.Fatalf("home status = %d", home.Code)
	}
	if !strings.Contains(home.Body.String(), `name="exactOnly"`) {
		t.Fatalf("start control missing exact-only choice\n%s", home.Body.String())
	}

	limited := postScan(t, server, `{"path":"`+root+`","exactOnly":true}`)
	if !limited.ExactOnly {
		t.Fatal("response exactOnly = false")
	}
	if len(limited.Similar) != 0 {
		t.Fatalf("similar groups = %d, want 0", len(limited.Similar))
	}
	if len(limited.Exact) != 0 {
		t.Fatalf("exact groups = %d, want 0", len(limited.Exact))
	}

	full := postScan(t, server, `{"path":"`+root+`"}`)
	if full.ExactOnly {
		t.Fatal("omitted exactOnly was stored as true")
	}
	if len(full.Similar) != 1 {
		t.Fatalf("similar groups = %d, want 1", len(full.Similar))
	}
}

func TestAcceptance_EmptyPostUsesFolderPicker(t *testing.T) {
	root := t.TempDir()
	body := encodePNG(t, solidPNG(color.NRGBA{R: 1, G: 2, B: 3, A: 255}))
	writeBytes(t, filepath.Join(root, "a.png"), body)
	writeBytes(t, filepath.Join(root, "b.png"), body)

	server := scanServer(t, root)
	got := postScan(t, server, `{}`)
	if got.Root != root {
		t.Fatalf("root = %s, want %s", got.Root, root)
	}
	if len(got.Exact) != 1 {
		t.Fatalf("exact groups = %d, want 1", len(got.Exact))
	}
}

func TestAcceptance_PreviewByIDReturnsBytes(t *testing.T) {
	root := t.TempDir()
	body := encodePNG(t, solidPNG(color.NRGBA{R: 200, G: 10, B: 10, A: 255}))
	path := filepath.Join(root, "small.png")
	writeBytes(t, path, body)

	server := scanServer(t, root)
	got := postScan(t, server, `{"path":"`+root+`"}`)
	id := fileID(t, got, path)
	rec := getBearer(t, server, "/scans/"+got.ID+"/files/"+id+"/preview")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body.String())
	}
	if !bytes.Equal(rec.Body.Bytes(), body) {
		t.Fatalf("preview = %d bytes, file = %d bytes", rec.Body.Len(), len(body))
	}
}

func TestAcceptance_PreviewRejectsRawPath(t *testing.T) {
	root := t.TempDir()
	body := encodePNG(t, solidPNG(color.NRGBA{R: 9, G: 9, B: 9, A: 255}))
	path := filepath.Join(root, "small.png")
	writeBytes(t, path, body)
	server := scanServer(t, root)
	got := postScan(t, server, `{"path":"`+root+`"}`)
	id := fileID(t, got, path)

	query := getBearer(t, server, "/scans/"+got.ID+"/files/"+id+"/preview?path="+path)
	if query.Code != http.StatusBadRequest {
		t.Fatalf("query status = %d, want 400", query.Code)
	}
	if bytes.Equal(query.Body.Bytes(), body) {
		t.Fatal("query path returned the file")
	}

	req := httptest.NewRequest(http.MethodGet, "/scans/"+got.ID+"/files/"+id+"/preview", strings.NewReader(`{"path":"`+path+`"}`))
	req.Header.Set("Authorization", "Bearer "+testToken)
	req.Header.Set("Content-Type", "application/json")
	bodyRec := httptest.NewRecorder()
	server.ServeHTTP(bodyRec, req)
	if bodyRec.Code != http.StatusBadRequest {
		t.Fatalf("body status = %d, want 400", bodyRec.Code)
	}
	if bytes.Equal(bodyRec.Body.Bytes(), body) {
		t.Fatal("body path returned the file")
	}
}

func TestAcceptance_PreviewUnknownIDIs404(t *testing.T) {
	firstRoot := t.TempDir()
	secondRoot := t.TempDir()
	body := encodePNG(t, solidPNG(color.NRGBA{R: 4, G: 5, B: 6, A: 255}))
	firstPath := filepath.Join(firstRoot, "a.png")
	writeBytes(t, firstPath, body)
	writeBytes(t, filepath.Join(secondRoot, "b.png"), body)

	server := scanServer(t, firstRoot)
	first := postScan(t, server, `{"path":"`+firstRoot+`"}`)
	second := postScan(t, server, `{"path":"`+secondRoot+`"}`)
	id := fileID(t, first, firstPath)

	missing := getBearer(t, server, "/scans/"+second.ID+"/files/"+id+"/preview")
	if missing.Code != http.StatusNotFound {
		t.Fatalf("other session status = %d, want 404", missing.Code)
	}
	unknown := getBearer(t, server, "/scans/"+first.ID+"/files/0123456789abcdef0123456789abcdef/preview")
	if unknown.Code != http.StatusNotFound {
		t.Fatalf("unknown id status = %d, want 404", unknown.Code)
	}
}

func TestAcceptance_LargeImageIsThumbnail(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "wide.png")
	writeBytes(t, path, noisyPNG(t, 800, 800))
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() <= 1<<20 {
		t.Fatalf("fixture size = %d, want more than 1 MiB", info.Size())
	}

	server := scanServer(t, root)
	got := postScan(t, server, `{"path":"`+root+`"}`)
	rec := getBearer(t, server, "/scans/"+got.ID+"/files/"+fileID(t, got, path)+"/preview")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Content-Type") != "image/jpeg" {
		t.Fatalf("content type = %s, want image/jpeg", rec.Header().Get("Content-Type"))
	}
	img, err := jpeg.Decode(rec.Body)
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() > 320 {
		t.Fatalf("width = %d, want at most 320", img.Bounds().Dx())
	}
}

type scanJSON struct {
	ID        string `json:"id"`
	Root      string `json:"root"`
	ExactOnly bool   `json:"exactOnly"`
	Exact     []struct {
		Members []struct {
			Path string `json:"path"`
		} `json:"members"`
	} `json:"exact"`
	Similar []struct {
		Members []struct {
			Path string `json:"path"`
		} `json:"members"`
	} `json:"similar"`
	Files []struct {
		ID   string `json:"id"`
		Path string `json:"path"`
	} `json:"files"`
}

type pathPicker struct{ path string }

func (p pathPicker) Pick() (string, error) { return p.path, nil }

func scanServer(t *testing.T, pick string) *httpadapter.Server {
	t.Helper()
	store := openStore(t)
	return httpadapter.New(testToken, store, store, trash.NewFreedesktop(t.TempDir()), filesystem.NewOpener()).
		WithScan(filesystem.NewWalker(nil), filesystem.NewOpener(), hash.New(), similar.New(), pathPicker{path: pick})
}

func postScan(t *testing.T, server *httpadapter.Server, payload string) scanJSON {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/scans", strings.NewReader(payload))
	req.Header.Set("Authorization", "Bearer "+testToken)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body.String())
	}
	var got scanJSON
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	return got
}

func fileID(t *testing.T, got scanJSON, path string) string {
	t.Helper()
	for _, file := range got.Files {
		if file.Path == path {
			return file.ID
		}
	}
	t.Fatalf("no file id for %s in %+v", path, got.Files)
	return ""
}

func nearPNGs(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	solid := solidPNG(color.NRGBA{R: 180, G: 40, B: 40, A: 255})
	changed := cloneNRGBA(solid)
	changed.Set(3, 3, color.NRGBA{R: 181, G: 40, B: 40, A: 255})
	writeBytes(t, filepath.Join(root, "red.png"), encodePNG(t, solid))
	writeBytes(t, filepath.Join(root, "red-bit.png"), encodePNG(t, changed))
	return root
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

func cloneNRGBA(src *image.NRGBA) *image.NRGBA {
	dst := image.NewNRGBA(src.Bounds())
	copy(dst.Pix, src.Pix)
	return dst
}

func noisyPNG(t *testing.T, width, height int) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	rng := rand.New(rand.NewSource(1))
	for i := range img.Pix {
		img.Pix[i] = byte(rng.Intn(256))
	}
	return encodePNG(t, img)
}

func encodePNG(t *testing.T, img image.Image) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func writeBytes(t *testing.T, path string, body []byte) {
	t.Helper()
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatal(err)
	}
}

var _ app.FolderPicker = pathPicker{}
