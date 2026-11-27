package http_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cionz0/hdsort/internal/adapters/filesystem"
	httpadapter "github.com/cionz0/hdsort/internal/adapters/http"
	"github.com/cionz0/hdsort/internal/adapters/sqlite"
	"github.com/cionz0/hdsort/internal/adapters/trash"
	"github.com/cionz0/hdsort/internal/domain"
)

func TestAcceptance_ListenerIsLoopback(t *testing.T) {
	ln, err := httpadapter.Listen("")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	addr, ok := ln.Addr().(*net.TCPAddr)
	if !ok || addr.IP == nil || !addr.IP.IsLoopback() {
		t.Fatalf("address = %v, want a loopback IP", ln.Addr())
	}
}

func TestAcceptance_ListenerRejectsWildcard(t *testing.T) {
	if _, err := httpadapter.Listen("0.0.0.0:0"); err == nil {
		t.Fatal("binding 0.0.0.0 returned no error")
	}
}

func TestAcceptance_MissingTokenIs401(t *testing.T) {
	server, store := newServer(t)
	id := saveSession(t, store, domain.NewSession(t.TempDir()))
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/scans/"+id, nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestAcceptance_WrongTokenIs401(t *testing.T) {
	server, store := newServer(t)
	id := saveSession(t, store, domain.NewSession(t.TempDir()))
	req := httptest.NewRequest(http.MethodGet, "/scans/"+id, nil)
	req.Header.Set("Authorization", "Bearer not-the-token")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	if strings.Contains(rec.Header().Get("Set-Cookie"), "hdsort_token=") {
		t.Fatal("wrong token set a cookie")
	}
}

func TestAcceptance_BearerTokenReadsSession(t *testing.T) {
	server, store := newServer(t)
	root := t.TempDir()
	id := saveSession(t, store, domain.NewSession(root))
	rec := getBearer(t, server, "/scans/"+id)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Root string `json:"root"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Root != root {
		t.Fatalf("root = %s, want %s", body.Root, root)
	}
}

func TestAcceptance_ReviewPageShowsExactAndSimilar(t *testing.T) {
	server, store := newServer(t)
	root := t.TempDir()
	keeper := filepath.Join(root, "a.jpg")
	deleted := filepath.Join(root, "b.jpg")
	near := filepath.Join(root, "near.jpg")
	other := filepath.Join(root, "other.jpg")
	session := exactAndSimilar(t, root, keeper, deleted, near, other)
	id := saveSession(t, store, session)

	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/?token="+testToken+"&scan="+id, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body.String())
	}
	cookie := rec.Header().Get("Set-Cookie")
	for _, want := range []string{"hdsort_token=" + testToken, "Path=/", "HttpOnly", "Secure", "SameSite=Strict"} {
		if !strings.Contains(cookie, want) {
			t.Fatalf("cookie %q missing %q", cookie, want)
		}
	}
	exact, similar := splitSections(t, rec.Body.String())
	for _, want := range []string{keeper, deleted, "plain_name"} {
		if !strings.Contains(exact, want) {
			t.Fatalf("exact section missing %s\n%s", want, exact)
		}
	}
	if strings.Contains(exact, near) || strings.Contains(exact, other) {
		t.Fatalf("similar path leaked into the exact section\n%s", exact)
	}
	if !strings.Contains(similar, near) {
		t.Fatalf("similar section missing %s\n%s", near, similar)
	}
}

func TestAcceptance_ReviewPageShowsSyncBanner(t *testing.T) {
	body := reviewHTML(t, filepath.Join(t.TempDir(), "Dropbox", "photos"))
	if !strings.Contains(body, "Deleting the local file can delete the synced copy. This check is only a path-segment hint.") {
		t.Fatalf("banner missing\n%s", body)
	}
}

func TestAcceptance_ReviewPageOmitsSyncBanner(t *testing.T) {
	body := reviewHTML(t, t.TempDir())
	if strings.Contains(body, "synced copy") {
		t.Fatalf("banner present\n%s", body)
	}
}

func TestAcceptance_PostDecisionPersists(t *testing.T) {
	server, store := newServer(t)
	root := t.TempDir()
	path := filepath.Join(root, "a.jpg")
	id := saveSession(t, store, domain.NewSession(root))
	payload, err := json.Marshal(map[string]string{"path": path, "choice": "keep"})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/scans/"+id+"/decisions", bytes.NewReader(payload))
	req.Header.Set("Authorization", "Bearer "+testToken)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body.String())
	}

	got := getBearer(t, server, "/scans/"+id)
	var body struct {
		Decisions []struct {
			Path   string `json:"path"`
			Choice string `json:"choice"`
		} `json:"decisions"`
	}
	if err := json.Unmarshal(got.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Decisions) != 1 || body.Decisions[0].Path != path || body.Decisions[0].Choice != "keep" {
		t.Fatalf("decisions = %+v", body.Decisions)
	}
}

func TestAcceptance_ApplyRouteTrashes(t *testing.T) {
	root, trashDir := scanAndTrash(t)
	keeper := filepath.Join(root, "a.jpg")
	duplicate := filepath.Join(root, "b.jpg")
	writeFile(t, keeper, []byte("photo"))
	writeFile(t, duplicate, []byte("photo"))
	files := []domain.FileRef{fileRef(keeper, "same"), fileRef(duplicate, "same")}
	session := recommend(t, root, files, keeper, duplicate)
	session = decide(t, session, keeper, domain.ChoiceKeep)
	session = decide(t, session, duplicate, domain.ChoiceDelete)

	store := openStore(t)
	id := saveSession(t, store, session)
	server := httpadapter.New(testToken, store, store, trash.NewFreedesktop(trashDir), filesystem.NewOpener())
	req := httptest.NewRequest(http.MethodPost, "/scans/"+id+"/apply", nil)
	req.Header.Set("Authorization", "Bearer "+testToken)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body.String())
	}
	if _, err := os.Stat(duplicate); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("deleted file still at %s (%v)", duplicate, err)
	}
	body, err := os.ReadFile(filepath.Join(trashDir, "files", filepath.Base(duplicate)))
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "photo" {
		t.Fatalf("trashed bytes = %q", body)
	}
	entries, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Path != keeper {
		t.Fatalf("library = %+v", entries)
	}
}

func TestAcceptance_ReviewPageOffersDryRun(t *testing.T) {
	server, store := newServer(t)
	id := saveSession(t, store, domain.NewSession(t.TempDir()))
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/?token="+testToken+"&scan="+id, nil))
	body := rec.Body.String()
	dry := `<form method="post" action="/scans/` + id + `/dry-run"><button type="submit">Dry run</button></form>`
	apply := `<form method="post" action="/scans/` + id + `/apply"><button type="submit">Apply</button></form>`
	if !strings.Contains(body, dry) || !strings.Contains(body, apply) {
		t.Fatalf("controls missing\n%s", body)
	}
}

func TestAcceptance_DryRunPageWritesNothing(t *testing.T) {
	root := t.TempDir()
	keeper := filepath.Join(root, "a.jpg")
	duplicate := filepath.Join(root, "b.jpg")
	writeFile(t, keeper, []byte("photo"))
	writeFile(t, duplicate, []byte("photo"))
	files := []domain.FileRef{fileRef(keeper, "same"), fileRef(duplicate, "same")}
	session := recommend(t, root, files, keeper, duplicate)
	session = decide(t, session, keeper, domain.ChoiceKeep)
	session = decide(t, session, duplicate, domain.ChoiceDelete)

	store := openStore(t)
	if err := store.Put(domain.LibraryEntry{
		ContentID: "seed",
		Path:      filepath.Join(t.TempDir(), "kept.jpg"),
		IndexedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}
	before, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	id := saveSession(t, store, session)
	server := httpadapter.New(testToken, store, store, trash.NewFreedesktop(t.TempDir()), filesystem.NewOpener())

	htmlReq := httptest.NewRequest(http.MethodPost, "/scans/"+id+"/dry-run", nil)
	htmlReq.Header.Set("Authorization", "Bearer "+testToken)
	htmlRec := httptest.NewRecorder()
	server.ServeHTTP(htmlRec, htmlReq)
	if htmlRec.Code != http.StatusOK {
		t.Fatalf("html status = %d, body %s", htmlRec.Code, htmlRec.Body.String())
	}
	page := htmlRec.Body.String()
	if !strings.Contains(page, duplicate) || !strings.Contains(page, "Nothing was changed.") {
		t.Fatalf("dry-run page missing the report\n%s", page)
	}

	jsonReq := httptest.NewRequest(http.MethodPost, "/scans/"+id+"/dry-run", nil)
	jsonReq.Header.Set("Authorization", "Bearer "+testToken)
	jsonReq.Header.Set("Accept", "application/json")
	jsonRec := httptest.NewRecorder()
	server.ServeHTTP(jsonRec, jsonReq)
	if jsonRec.Code != http.StatusOK {
		t.Fatalf("json status = %d, body %s", jsonRec.Code, jsonRec.Body.String())
	}
	var report struct {
		Changed bool     `json:"changed"`
		Trash   []string `json:"trash"`
	}
	if err := json.Unmarshal(jsonRec.Body.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.Changed {
		t.Fatal("changed = true")
	}
	if !contains(report.Trash, duplicate) {
		t.Fatalf("trash = %v, want %s", report.Trash, duplicate)
	}
	if _, err := os.Stat(duplicate); err != nil {
		t.Fatalf("duplicate missing after dry-run: %v", err)
	}
	after, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != len(before) || after[0].ContentID != before[0].ContentID || after[0].Path != before[0].Path {
		t.Fatalf("library changed from %+v to %+v", before, after)
	}
	loaded, err := store.Load(id)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Status() != domain.StatusOpen {
		t.Fatalf("status = %s, want open", loaded.Status())
	}
}

const testToken = "0123456789abcdef0123456789abcdef"

func newServer(t *testing.T) (*httpadapter.Server, *sqlite.Store) {
	t.Helper()
	store := openStore(t)
	server := httpadapter.New(testToken, store, store, trash.NewFreedesktop(t.TempDir()), filesystem.NewOpener())
	return server, store
}

func openStore(t *testing.T) *sqlite.Store {
	t.Helper()
	store, err := sqlite.Open(filepath.Join(t.TempDir(), "library.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}

func saveSession(t *testing.T, store *sqlite.Store, session domain.Session) string {
	t.Helper()
	id, err := store.Save(session)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func getBearer(t *testing.T, server *httpadapter.Server, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("Authorization", "Bearer "+testToken)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	return rec
}

func reviewHTML(t *testing.T, root string) string {
	t.Helper()
	server, store := newServer(t)
	id := saveSession(t, store, domain.NewSession(root))
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/?token="+testToken+"&scan="+id, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body.String())
	}
	return rec.Body.String()
}

func splitSections(t *testing.T, body string) (exact, similar string) {
	t.Helper()
	exactAt := strings.Index(body, `id="exact"`)
	similarAt := strings.Index(body, `id="similar"`)
	if exactAt < 0 || similarAt < exactAt {
		t.Fatalf("sections missing\n%s", body)
	}
	return body[exactAt:similarAt], body[similarAt:]
}

func exactAndSimilar(t *testing.T, root, keeper, deleted, near, other string) domain.Session {
	t.Helper()
	exactFiles := []domain.FileRef{fileRef(keeper, "same"), fileRef(deleted, "same")}
	group, err := domain.NewExactGroup(exactFiles)
	if err != nil {
		t.Fatal(err)
	}
	rec, err := domain.NewRecommendation(keeper, []string{deleted}, domain.ReasonPlainName)
	if err != nil {
		t.Fatal(err)
	}
	group, err = group.WithRecommendation(rec)
	if err != nil {
		t.Fatal(err)
	}
	similar, err := domain.NewSimilarGroup([]domain.FileRef{fileRef(near, "near"), fileRef(other, "other")})
	if err != nil {
		t.Fatal(err)
	}
	session := domain.NewSession(root)
	session.Exact = []domain.ExactGroup{group}
	session.Similar = []domain.SimilarGroup{similar}
	return session
}

func recommend(t *testing.T, root string, files []domain.FileRef, keeper, duplicate string) domain.Session {
	t.Helper()
	group, err := domain.NewExactGroup(files)
	if err != nil {
		t.Fatal(err)
	}
	rec, err := domain.NewRecommendation(keeper, []string{duplicate}, domain.ReasonPlainName)
	if err != nil {
		t.Fatal(err)
	}
	group, err = group.WithRecommendation(rec)
	if err != nil {
		t.Fatal(err)
	}
	session := domain.NewSession(root)
	session.Exact = []domain.ExactGroup{group}
	return session
}

func decide(t *testing.T, session domain.Session, path string, choice domain.Choice) domain.Session {
	t.Helper()
	next, err := domain.RecordDecision(session, path, choice)
	if err != nil {
		t.Fatal(err)
	}
	return next
}

func fileRef(path, id string) domain.FileRef {
	return domain.FileRef{Path: path, ContentID: domain.ContentID(id), Size: 5, Readable: true}
}

func writeFile(t *testing.T, path string, body []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatal(err)
	}
}

func scanAndTrash(t *testing.T) (root, trashDir string) {
	t.Helper()
	parent := t.TempDir()
	root = filepath.Join(parent, "scan")
	trashDir = filepath.Join(parent, "Trash")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	return root, trashDir
}

func contains(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
