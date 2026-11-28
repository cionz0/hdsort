package app_test

import (
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cionz0/hdsort/internal/adapters/filesystem"
	"github.com/cionz0/hdsort/internal/adapters/hash"
	"github.com/cionz0/hdsort/internal/adapters/similar"
	"github.com/cionz0/hdsort/internal/app"
	"github.com/cionz0/hdsort/internal/domain"
)

func TestAcceptance_ReviewListsOneRowPerSetUnderItsFolder(t *testing.T) {
	root := t.TempDir()
	photos := filepath.Join(root, "photos")
	other := filepath.Join(root, "other")
	if err := os.MkdirAll(photos, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(other, 0o755); err != nil {
		t.Fatal(err)
	}
	img := solidPNG(color.NRGBA{R: 20, G: 80, B: 40, A: 255})
	keeper := filepath.Join(photos, "photo.png")
	writePNG(t, keeper, img)
	writePNG(t, filepath.Join(photos, "photo (1).png"), img)
	copy := filepath.Join(other, "Copy of photo.png")
	writePNG(t, copy, img)

	var board app.FolderBoard
	result, err := app.Scan(root, "", filesystem.NewWalker(nil), filesystem.NewOpener(), hash.New(), similar.New(), true, func(ev app.ProgressEvent) {
		board.Apply(ev)
	})
	if err != nil {
		t.Fatal(err)
	}
	board.UseReview(app.Review(result.Session, nil))

	if board.Contains(keeper) != 1 {
		t.Fatalf("keeper rows = %d, want 1", board.Contains(keeper))
	}
	if board.Contains(copy) != 1 {
		t.Fatalf("copy rows = %d, want 1", board.Contains(copy))
	}
	for _, row := range board.Rows {
		for _, member := range row.Members {
			if member.Path != keeper {
				continue
			}
			if row.Folder != photos {
				t.Fatalf("folder = %s, want %s", row.Folder, photos)
			}
			if row.Kind != app.RowExact {
				t.Fatalf("kind = %s, want exact", row.Kind)
			}
			if len(row.Members) != 3 {
				t.Fatalf("members = %d, want 3", len(row.Members))
			}
		}
	}
}

func TestAcceptance_UniqueFilesLeaveTheList(t *testing.T) {
	root := t.TempDir()
	only := filepath.Join(root, "only.txt")
	writeFile(t, only, []byte("just-me"))
	keep := filepath.Join(root, "same.txt")
	writeFile(t, keep, []byte("same-bytes"))
	writeFile(t, filepath.Join(root, "same (1).txt"), []byte("same-bytes"))

	var live app.FolderBoard
	live.Apply(app.ProgressEvent{Kind: app.EventQueue, Path: only, Size: 7})
	live.Apply(app.ProgressEvent{Kind: app.EventProcessed, Path: only, Size: 7, Outcome: app.OutcomeUnique})
	if live.Contains(only) != 0 {
		t.Fatalf("unique rows = %d, want 0", live.Contains(only))
	}

	var board app.FolderBoard
	result, err := app.Scan(root, "", filesystem.NewWalker(nil), filesystem.NewOpener(), hash.New(), similar.New(), true, func(ev app.ProgressEvent) {
		board.Apply(ev)
	})
	if err != nil {
		t.Fatal(err)
	}
	board.UseReview(app.Review(result.Session, nil))
	if board.Contains(only) != 0 {
		t.Fatalf("reviewed unique rows = %d, want 0", board.Contains(only))
	}
	if board.Contains(keep) != 1 {
		t.Fatalf("duplicate rows = %d, want 1", board.Contains(keep))
	}
	for _, row := range board.Rows {
		if row.Kind == app.RowFile {
			t.Fatalf("file row remains: %+v", row)
		}
	}
}

func TestAcceptance_SizeIsHumanReadable(t *testing.T) {
	cases := []struct {
		n    int64
		want string
	}{
		{n: 0, want: "0 B"},
		{n: 1536, want: "1.5 KB"},
		{n: 39854, want: "39 KB"},
		{n: 5 << 20, want: "5.0 MB"},
	}
	for _, tc := range cases {
		if got := app.FormatSize(tc.n); got != tc.want {
			t.Fatalf("FormatSize(%d) = %q, want %q", tc.n, got, tc.want)
		}
	}
}

func TestAcceptance_PathIsRelativeToTheScanRoot(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "Users", "cionzo", "Desktop", "delme")
	inside := filepath.Join(root, ".activity_pages", "page_11.json")
	want := filepath.Join(".activity_pages", "page_11.json")
	if got := app.DisplayPath(root, inside); got != want {
		t.Fatalf("inside = %q, want %q", got, want)
	}
	outside := filepath.Join(string(filepath.Separator), "tmp", "other.txt")
	if got := app.DisplayPath(root, outside); got != outside {
		t.Fatalf("outside = %q, want %q", got, outside)
	}
}

func TestAcceptance_ScanStatusStaysOnTheSameRow(t *testing.T) {
	const path = "/scans/budget.pdf"
	var board app.FolderBoard
	board.Apply(app.ProgressEvent{Kind: app.EventQueue, Path: path, Size: 180})
	board.Apply(app.ProgressEvent{Kind: app.EventHashing, Path: path, Size: 180, Stage: app.StageFull})
	if len(board.Rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(board.Rows))
	}
	row := board.Rows[0]
	if row.Status != "reading" || row.Stage != app.StageFull || row.Size != 180 {
		t.Fatalf("row = %+v", row)
	}
}

func TestAcceptance_FolderOrderWalksOneDirectory(t *testing.T) {
	root := t.TempDir()
	session := domain.NewSession(root)
	session.Exact = []domain.ExactGroup{
		exactPair(t, filepath.Join(root, "beta", "three.txt"), filepath.Join(root, "beta", "three (1).txt"), "bbb"),
		exactPair(t, filepath.Join(root, "alpha", "two.txt"), filepath.Join(root, "alpha", "two (1).txt"), "aaa"),
		exactPair(t, filepath.Join(root, "alpha", "one.txt"), filepath.Join(root, "alpha", "one (1).txt"), "ccc"),
	}
	var board app.FolderBoard
	board.UseReview(app.Review(session, nil))
	if len(board.Rows) != 3 {
		t.Fatalf("rows = %d, want 3", len(board.Rows))
	}
	alpha := filepath.Join(root, "alpha")
	if board.Rows[0].Folder != alpha || board.Rows[1].Folder != alpha {
		t.Fatalf("first folders = %s, %s", board.Rows[0].Folder, board.Rows[1].Folder)
	}
	if board.Rows[2].Folder != filepath.Join(root, "beta") {
		t.Fatalf("last folder = %s", board.Rows[2].Folder)
	}
}

func TestAcceptance_SkipSetSkipsEveryMember(t *testing.T) {
	root := t.TempDir()
	photos := filepath.Join(root, "photos")
	paths := []string{
		filepath.Join(photos, "photo.png"),
		filepath.Join(photos, "photo (1).png"),
		filepath.Join(root, "other", "Copy of photo.png"),
	}
	session := domain.NewSession(root)
	next, err := app.SkipSet(session, paths)
	if err != nil {
		t.Fatal(err)
	}
	if len(next.Decisions) != len(paths) {
		t.Fatalf("decisions = %d, want %d", len(next.Decisions), len(paths))
	}
	for _, path := range paths {
		choice := decisionFor(next, path)
		if choice != domain.ChoiceSkip {
			t.Fatalf("%s choice = %q", path, choice)
		}
	}
}

func TestAcceptance_ApplyFolderKeepsThatFolder(t *testing.T) {
	root := t.TempDir()
	photos := filepath.Join(root, "photos")
	other := filepath.Join(root, "other")
	if err := os.MkdirAll(photos, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(other, 0o755); err != nil {
		t.Fatal(err)
	}
	img := solidPNG(color.NRGBA{R: 10, G: 20, B: 30, A: 255})
	keeper := filepath.Join(photos, "shot.png")
	inside := filepath.Join(photos, "shot (1).png")
	outside := filepath.Join(other, "Copy of shot.png")
	writePNG(t, keeper, img)
	writePNG(t, inside, img)
	writePNG(t, outside, img)
	writeFile(t, filepath.Join(photos, "note.txt"), []byte("same-note"))
	writeFile(t, filepath.Join(photos, "note (1).txt"), []byte("same-note"))
	near := solidPNG(color.NRGBA{R: 200, G: 30, B: 30, A: 255})
	changed := cloneNRGBA(near)
	changed.Set(3, 3, color.NRGBA{R: 201, G: 30, B: 30, A: 255})
	nearA := filepath.Join(photos, "near-a.png")
	nearB := filepath.Join(photos, "near-b.png")
	writePNG(t, nearA, near)
	writePNG(t, nearB, changed)

	result, err := app.Scan(root, "", filesystem.NewWalker(nil), filesystem.NewOpener(), hash.New(), similar.New(), false, nil)
	if err != nil {
		t.Fatal(err)
	}
	next, err := app.ApplySameTypeInFolder(result.Session, photos, keeper)
	if err != nil {
		t.Fatal(err)
	}
	if decisionFor(next, keeper) != domain.ChoiceKeep || decisionFor(next, inside) != domain.ChoiceKeep {
		t.Fatalf("photos copies = %q %q", decisionFor(next, keeper), decisionFor(next, inside))
	}
	if decisionFor(next, outside) != domain.ChoiceDelete {
		t.Fatalf("other copy = %q", decisionFor(next, outside))
	}
	for _, path := range []string{
		filepath.Join(photos, "note.txt"),
		filepath.Join(photos, "note (1).txt"),
		nearA,
		nearB,
	} {
		if choice := decisionFor(next, path); choice != "" {
			t.Fatalf("%s choice = %q, want none", path, choice)
		}
	}
}

func TestAcceptance_RunningActionDisablesItsControl(t *testing.T) {
	var view app.ScanView
	view.Apply(app.ProgressEvent{
		Kind:  app.EventHashing,
		Path:  "/docs/budget.pdf",
		Size:  180,
		Stage: app.StageFull,
	})
	reading := app.SlotFrom(view, "")
	if reading.Label == "" || !strings.Contains(reading.Label, "/docs/budget.pdf") || !strings.Contains(reading.Label, "full") {
		t.Fatalf("label = %q", reading.Label)
	}
	if !reading.StartDisabled || reading.DryDisabled || reading.ApplyDisabled {
		t.Fatalf("reading disables = start %v dry %v apply %v", reading.StartDisabled, reading.DryDisabled, reading.ApplyDisabled)
	}

	trashing := app.SlotFrom(app.ScanView{}, "/backup/Copy of photo.png")
	if !strings.Contains(trashing.Label, "/backup/Copy of photo.png") {
		t.Fatalf("trash label = %q", trashing.Label)
	}
	if trashing.StartDisabled || trashing.DryDisabled || !trashing.ApplyDisabled {
		t.Fatalf("trash disables = start %v dry %v apply %v", trashing.StartDisabled, trashing.DryDisabled, trashing.ApplyDisabled)
	}
}

func TestAcceptance_QuietFallbackIsNotBlackOnWhite(t *testing.T) {
	white := color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
	light := app.LightFallback
	if light.Background != white || light.Control != white || light.OnAccent != white {
		t.Fatalf("light solids = %+v", light)
	}
	if light.Surface != (color.NRGBA{R: 0xF6, G: 0xF6, B: 0xF6, A: 0xFF}) {
		t.Fatalf("light surface = %+v", light.Surface)
	}
	if light.Text.A != 217 || light.Secondary.A != 128 || light.Separator.A != 26 {
		t.Fatalf("light alpha text %d secondary %d separator %d", light.Text.A, light.Secondary.A, light.Separator.A)
	}
	if light.Accent != (color.NRGBA{R: 0x00, G: 0x7A, B: 0xFF, A: 0xFF}) {
		t.Fatalf("light accent = %+v", light.Accent)
	}
	dark := app.DarkFallback
	if dark.Background != (color.NRGBA{R: 0x1E, G: 0x1E, B: 0x1E, A: 0xFF}) {
		t.Fatalf("background = %+v", dark.Background)
	}
	if dark.Surface != (color.NRGBA{R: 0x28, G: 0x28, B: 0x28, A: 0xFF}) {
		t.Fatalf("surface = %+v", dark.Surface)
	}
	separator := color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 31}
	if dark.Separator != separator || dark.Separator == dark.Text {
		t.Fatalf("separator = %+v text = %+v", dark.Separator, dark.Text)
	}
	control := color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 64}
	if dark.Control != control {
		t.Fatalf("control = %+v", dark.Control)
	}
	if dark.Text.A == 0xFF || dark.Secondary.A == 0xFF {
		t.Fatalf("text alpha = %d secondary = %d", dark.Text.A, dark.Secondary.A)
	}
}

func TestAcceptance_MissingAccentKeepsTheQuietSurface(t *testing.T) {
	background := color.NRGBA{R: 1, G: 2, B: 3, A: 0xFF}
	got := app.ResolvePalette(app.AppearanceDark, app.SystemColors{
		Background: app.OptionalColor{Color: background, Set: true},
	})
	want := app.DarkFallback
	want.Background = background
	if got != want {
		t.Fatalf("palette = %+v, want %+v", got, want)
	}
}

func TestAcceptance_SystemPaletteFollowsAppearance(t *testing.T) {
	indigo := color.NRGBA{R: 0x3F, G: 0x51, B: 0xB5, A: 0xFF}
	separator := color.NRGBA{R: 9, G: 9, B: 9, A: 40}
	cases := []struct {
		name       string
		appearance app.Appearance
		system     app.SystemColors
		want       app.Palette
	}{
		{name: "light fallback", appearance: app.AppearanceLight, want: app.LightFallback},
		{name: "dark fallback", appearance: app.AppearanceDark, want: app.DarkFallback},
		{
			name:       "system background keeps the quiet roles",
			appearance: app.AppearanceDark,
			system: app.SystemColors{Background: app.OptionalColor{
				Color: color.NRGBA{R: 1, G: 2, B: 3, A: 0xFF}, Set: true,
			}},
			want: func() app.Palette {
				p := app.DarkFallback
				p.Background = color.NRGBA{R: 1, G: 2, B: 3, A: 0xFF}
				return p
			}(),
		},
		{
			name:       "supplied separator leaves the accent",
			appearance: app.AppearanceDark,
			system:     app.SystemColors{Separator: app.OptionalColor{Color: separator, Set: true}},
			want: func() app.Palette {
				p := app.DarkFallback
				p.Separator = separator
				return p
			}(),
		},
		{
			name:       "full system colors",
			appearance: app.AppearanceLight,
			system: app.SystemColors{
				Background: app.OptionalColor{Color: color.NRGBA{R: 0x11, G: 0x22, B: 0x33, A: 0xFF}, Set: true},
				Surface:    app.OptionalColor{Color: color.NRGBA{R: 0x10, G: 0x10, B: 0x10, A: 0xFF}, Set: true},
				Control:    app.OptionalColor{Color: color.NRGBA{R: 0x20, G: 0x20, B: 0x20, A: 0x40}, Set: true},
				Text:       app.OptionalColor{Color: color.NRGBA{R: 0x44, G: 0x55, B: 0x66, A: 0xD8}, Set: true},
				Secondary:  app.OptionalColor{Color: color.NRGBA{R: 0x44, G: 0x55, B: 0x66, A: 0x80}, Set: true},
				Separator:  app.OptionalColor{Color: separator, Set: true},
				Accent:     app.OptionalColor{Color: color.NRGBA{R: 0x77, G: 0x88, B: 0x99, A: 0xFF}, Set: true},
			},
			want: app.Palette{
				Background: color.NRGBA{R: 0x11, G: 0x22, B: 0x33, A: 0xFF},
				Surface:    color.NRGBA{R: 0x10, G: 0x10, B: 0x10, A: 0xFF},
				Control:    color.NRGBA{R: 0x20, G: 0x20, B: 0x20, A: 0x40},
				Text:       color.NRGBA{R: 0x44, G: 0x55, B: 0x66, A: 0xD8},
				Secondary:  color.NRGBA{R: 0x44, G: 0x55, B: 0x66, A: 0x80},
				Separator:  separator,
				Accent:     color.NRGBA{R: 0x77, G: 0x88, B: 0x99, A: 0xFF},
				OnAccent:   app.LightFallback.OnAccent,
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := app.ResolvePalette(tc.appearance, tc.system)
			if got != tc.want {
				t.Fatalf("palette = %+v, want %+v", got, tc.want)
			}
			if got.Accent == indigo {
				t.Fatalf("accent = indigo")
			}
		})
	}
}

func exactPair(t *testing.T, plain, generated, id string) domain.ExactGroup {
	t.Helper()
	now := time.Unix(1_700_000_000, 0)
	group, err := domain.NewExactGroup([]domain.FileRef{
		{Path: plain, Size: 4, ModTime: now, ContentID: domain.ContentID(id), Readable: true},
		{Path: generated, Size: 4, ModTime: now, ContentID: domain.ContentID(id), Readable: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	return group
}

func decisionFor(session domain.Session, path string) domain.Choice {
	for _, decision := range session.Decisions {
		if decision.Path == path {
			return decision.Choice
		}
	}
	return ""
}
