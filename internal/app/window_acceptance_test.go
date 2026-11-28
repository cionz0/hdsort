package app_test

import (
	"bytes"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cionz0/hdsort/internal/adapters/filesystem"
	"github.com/cionz0/hdsort/internal/adapters/hash"
	"github.com/cionz0/hdsort/internal/adapters/similar"
	"github.com/cionz0/hdsort/internal/adapters/trash"
	"github.com/cionz0/hdsort/internal/app"
	"github.com/cionz0/hdsort/internal/domain"
)

func TestAcceptance_ScanReportsProgressBeforeReturn(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "a.txt")
	writeFile(t, path, []byte("hello"))

	got := make(chan app.ProgressEvent, 1)
	release := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, err := app.Scan(root, "", filesystem.NewWalker(nil), filesystem.NewOpener(), hash.New(), similar.New(), false, func(ev app.ProgressEvent) {
			if ev.Path == path && ev.Size == 5 {
				select {
				case got <- ev:
					<-release
				default:
				}
			}
		})
		if err != nil {
			t.Error(err)
		}
	}()

	select {
	case ev := <-got:
		if ev.Path != path || ev.Size != 5 {
			t.Fatalf("event = %+v", ev)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("no progress before the scan finished")
	}
	select {
	case <-done:
		t.Fatal("scan returned before the progress event was observed")
	default:
	}
	close(release)
	<-done
}

func TestAcceptance_ProgressShowsQueueAndProcessed(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "a.txt")
	link := filepath.Join(root, "link.txt")
	writeFile(t, path, []byte("hello"))
	if err := os.Symlink("a.txt", link); err != nil {
		t.Fatal(err)
	}
	var events []app.ProgressEvent
	_, err := app.Scan(root, "", filesystem.NewWalker(nil), filesystem.NewOpener(), hash.New(), similar.New(), true, func(ev app.ProgressEvent) {
		events = append(events, ev)
	})
	if err != nil {
		t.Fatal(err)
	}
	var view app.ScanView
	queued := false
	processed := false
	for _, ev := range events {
		view.Apply(ev)
		if ev.Kind == app.EventQueue && ev.Path == path && ev.Size == 5 {
			queued = true
		}
		if ev.Kind == app.EventProcessed && ev.Path == path && ev.Size == 5 {
			processed = true
		}
		if ev.Kind == app.EventQueue && ev.Path == link {
			t.Fatal("symlink was queued")
		}
	}
	if !queued || !processed {
		t.Fatalf("queued=%v processed=%v", queued, processed)
	}
	for _, row := range view.Queue {
		if row.Path == path {
			t.Fatal("hashed file is still queued")
		}
	}
	var sawLink bool
	for _, row := range view.Processed {
		if row.Path == link && row.Outcome == app.OutcomeSymlink {
			sawLink = true
		}
	}
	if !sawLink {
		t.Fatalf("processed = %+v, want the symlink", view.Processed)
	}
}

func TestAcceptance_ProgressNamesHasherStage(t *testing.T) {
	root := t.TempDir()
	small := filepath.Join(root, "small.txt")
	largeA := filepath.Join(root, "a.bin")
	largeB := filepath.Join(root, "b.bin")
	writeFile(t, small, []byte("tiny"))
	body := fileWithHead('A')
	writeFile(t, largeA, body)
	writeFile(t, largeB, body)

	stages := map[string][]app.HashStage{}
	result, err := app.Scan(root, "", filesystem.NewWalker(nil), filesystem.NewOpener(), hash.New(), similar.New(), true, func(ev app.ProgressEvent) {
		if ev.Kind == app.EventHashing {
			stages[ev.Path] = append(stages[ev.Path], ev.Stage)
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	if hasStage(stages[small], app.StagePartial) || !hasStage(stages[small], app.StageFull) {
		t.Fatalf("small stages = %v, want full and no partial", stages[small])
	}
	for _, path := range []string{largeA, largeB} {
		if !hasStage(stages[path], app.StagePartial) || !hasStage(stages[path], app.StageFull) {
			t.Fatalf("%s stages = %v, want partial and full", path, stages[path])
		}
		got := mustFile(t, result.Files, path).ContentID
		want, err := hash.New().ContentID(path, filesystem.NewOpener())
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Fatalf("%s content ID = %s, want %s", path, got, want)
		}
	}
}

func TestAcceptance_ExactGroupShowsKeeperAndSizes(t *testing.T) {
	keeper := domain.FileRef{Path: filepath.Join("photos", "a.jpg"), Size: 100, ContentID: "same", ModTime: time.Unix(2, 0)}
	dup := domain.FileRef{Path: filepath.Join("photos", "a (1).jpg"), Size: 40, ContentID: "same", ModTime: time.Unix(1, 0)}
	group, err := domain.NewExactGroup([]domain.FileRef{keeper, dup})
	if err != nil {
		t.Fatal(err)
	}
	session := domain.NewSession(t.TempDir())
	session.Exact = []domain.ExactGroup{group}
	model := app.Review(session, nil)
	if len(model.Exact) != 1 || !model.Exact[0].HasRecommendation {
		t.Fatalf("exact = %+v", model.Exact)
	}
	got := model.Exact[0]
	if got.Keeper != keeper.Path || got.Reason != string(domain.ReasonPlainName) {
		t.Fatalf("suggestion = %+v", got)
	}
	if got.KeeperSize != 100 || len(got.Deletes) != 1 || got.Deletes[0].Path != dup.Path || got.Deletes[0].Size != 40 || got.Freed != 40 {
		t.Fatalf("sizes = keeper %d deletes %+v freed %d", got.KeeperSize, got.Deletes, got.Freed)
	}
}

func TestAcceptance_TiedGroupShowsSizesWithoutRecommendation(t *testing.T) {
	when := time.Unix(5, 0)
	a := domain.FileRef{Path: "a.jpg", Size: 10, ContentID: "same", ModTime: when}
	b := domain.FileRef{Path: "b.jpg", Size: 25, ContentID: "same", ModTime: when}
	group, err := domain.NewExactGroup([]domain.FileRef{a, b})
	if err != nil {
		t.Fatal(err)
	}
	session := domain.NewSession(t.TempDir())
	session.Exact = []domain.ExactGroup{group}
	model := app.Review(session, nil)
	if model.Exact[0].HasRecommendation {
		t.Fatalf("recommendation = %+v, want none", model.Exact[0])
	}
	if len(model.Exact[0].Members) != 2 || model.Exact[0].Members[0].Size != 10 || model.Exact[0].Members[1].Size != 25 {
		t.Fatalf("members = %+v", model.Exact[0].Members)
	}
}

type stubPreview struct {
	body []byte
	ok   bool
	err  error
}

func (s stubPreview) Preview(string) ([]byte, bool, error) {
	return s.body, s.ok, s.err
}

func TestAcceptance_SystemPreviewUsedWhenPresent(t *testing.T) {
	want := []byte("system-preview")
	got, ok, err := app.LoadPreview("page_11.json", bytes.NewReader([]byte("not an image")), 12, stubPreview{body: want, ok: true})
	if err != nil || !ok || !bytes.Equal(got, want) {
		t.Fatalf("preview ok=%v err=%v body=%q", ok, err, got)
	}
}

func TestAcceptance_ImagePreviewWhenSystemHasNone(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	var buf bytes.Buffer
	if err := png.Encode(&buf, src); err != nil {
		t.Fatal(err)
	}
	got, ok, err := app.LoadPreview("shot.png", bytes.NewReader(buf.Bytes()), int64(buf.Len()), stubPreview{})
	if err != nil || !ok || len(got) == 0 {
		t.Fatalf("preview ok=%v err=%v len=%d", ok, err, len(got))
	}
}

func TestAcceptance_ImageGroupShowsThumbnail(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 400, 20))
	var buf bytes.Buffer
	if err := png.Encode(&buf, src); err != nil {
		t.Fatal(err)
	}
	thumb, ok, err := app.Thumbnail("wide.png", bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil || !ok {
		t.Fatalf("thumbnail ok=%v err=%v", ok, err)
	}
	img, err := jpeg.Decode(bytes.NewReader(thumb))
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() > 320 {
		t.Fatalf("width = %d, want <= 320", img.Bounds().Dx())
	}
	if _, ok, err := app.Thumbnail("notes.txt", bytes.NewReader([]byte("hello")), 5); err != nil || ok {
		t.Fatalf("text thumbnail ok=%v err=%v, want none", ok, err)
	}
}

func TestAcceptance_SimilarGroupHasNoDeleteSuggestion(t *testing.T) {
	group, err := domain.NewSimilarGroup([]domain.FileRef{
		{Path: "near.jpg", Size: 5, ContentID: "a"},
		{Path: "other.jpg", Size: 9, ContentID: "b"},
	})
	if err != nil {
		t.Fatal(err)
	}
	session := domain.NewSession(t.TempDir())
	session.Similar = []domain.SimilarGroup{group}
	model := app.Review(session, nil)
	if len(model.Similar) != 1 || model.Similar[0].HasRecommendation {
		t.Fatalf("similar = %+v", model.Similar)
	}
	if model.Similar[0].Members[0].Size != 5 || model.Similar[0].Members[1].Size != 9 {
		t.Fatalf("sizes = %+v", model.Similar[0].Members)
	}
}

func TestAcceptance_SyncedRootShowsBanner(t *testing.T) {
	synced := app.Review(domain.NewSession(filepath.Join(t.TempDir(), "Dropbox", "photos")), nil)
	if synced.Banner != app.BannerText {
		t.Fatalf("banner = %q", synced.Banner)
	}
	plain := app.Review(domain.NewSession(t.TempDir()), nil)
	if plain.Banner != "" {
		t.Fatalf("banner = %q, want empty", plain.Banner)
	}
}

func TestAcceptance_DryRunShowsNothingChanged(t *testing.T) {
	root := t.TempDir()
	keeper, duplicate, files := duplicatePair(t, root)
	session := recommend(t, root, files, keeper, duplicate)
	session = decide(t, session, keeper, domain.ChoiceKeep)
	session = decide(t, session, duplicate, domain.ChoiceDelete)
	store := openStore(t)
	report, err := app.DryRun(session, files, store, trash.NewFreedesktop(t.TempDir()), filesystem.NewOpener())
	if err != nil {
		t.Fatal(err)
	}
	text := app.FormatDryRun(report)
	if !bytes.Contains([]byte(text), []byte(duplicate)) || !bytes.Contains([]byte(text), []byte(app.NothingChanged)) {
		t.Fatalf("dry-run text = %q", text)
	}
	if _, err := os.Stat(duplicate); err != nil {
		t.Fatal(err)
	}
}

func TestAcceptance_ApplyNamesFileBeingTrashed(t *testing.T) {
	root, trashDir := scanAndTrash(t)
	keeper, duplicate, files := duplicatePair(t, root)
	session := recommend(t, root, files, keeper, duplicate)
	session = decide(t, session, keeper, domain.ChoiceKeep)
	session = decide(t, session, duplicate, domain.ChoiceDelete)
	store := openStore(t)
	saw := make(chan string, 1)
	release := make(chan struct{})
	done := make(chan struct{})
	inner := trash.NewFreedesktop(trashDir)
	go func() {
		defer close(done)
		_, err := app.Apply(session, files, store, gateTrash{inner: inner, saw: saw, release: release}, filesystem.NewOpener())
		if err != nil {
			t.Error(err)
		}
	}()
	select {
	case path := <-saw:
		if path != duplicate {
			t.Fatalf("trashing %s, want %s", path, duplicate)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("apply finished without naming the file")
	}
	select {
	case <-done:
		t.Fatal("apply returned before the trash path was observed")
	default:
	}
	close(release)
	<-done
	if _, err := os.Stat(duplicate); !os.IsNotExist(err) {
		t.Fatalf("duplicate still present: %v", err)
	}
	if _, err := os.Stat(filepath.Join(trashDir, "files", filepath.Base(duplicate))); err != nil {
		t.Fatal(err)
	}
}

type gateTrash struct {
	inner   app.Trasher
	saw     chan string
	release chan struct{}
}

func (g gateTrash) Trash(path string) error {
	select {
	case g.saw <- path:
		<-g.release
	default:
	}
	return g.inner.Trash(path)
}

func hasStage(stages []app.HashStage, want app.HashStage) bool {
	for _, stage := range stages {
		if stage == want {
			return true
		}
	}
	return false
}
