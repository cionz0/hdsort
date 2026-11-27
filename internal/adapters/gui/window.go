package gui

import (
	"bytes"
	"fmt"
	"image"
	"os"
	"sync"
	"time"

	gioapp "gioui.org/app"
	"gioui.org/font/gofont"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/cionz0/hdsort/internal/app"
	"github.com/cionz0/hdsort/internal/domain"
)

// Deps is every port the window calls. The composition root fills it.
type Deps struct {
	Pick    interface{ Pick() (string, error) }
	Walk    app.Walker
	Files   app.FileReader
	ID      app.Identifier
	Similar app.Similarizer
	Library app.LibraryRepository
	Trash   app.Trasher
	Disk    app.Relocator
	DataDir string
}

// Run opens the review window and blocks until it closes.
// app.Main runs on the calling goroutine, which must be the process main.
func Run(d Deps) error {
	w := new(gioapp.Window)
	w.Option(gioapp.Title("hdsort"), gioapp.Size(unit.Dp(960), unit.Dp(720)))
	ui := newUI(d, w)
	go func() {
		err := ui.loop(w)
		if err != nil {
			fmt.Fprintf(os.Stderr, "hdsort: %s\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}()
	gioapp.Main()
	return nil
}

type choiceButtons struct {
	keep, drop, skip widget.Clickable
}

type ui struct {
	deps Deps
	win  *gioapp.Window
	th   *material.Theme

	pick, start, dry, apply widget.Clickable
	exactOnly               widget.Bool
	list                    widget.List
	clicks                  map[string]*choiceButtons

	mu       sync.Mutex
	root     string
	busy     bool
	applying string
	note     string
	view     app.ScanView
	review   app.ReviewModel
	session  domain.Session
	files    []domain.FileRef
	thumbs   map[string]image.Image
	ops      op.Ops
}

func newUI(d Deps, w *gioapp.Window) *ui {
	th := material.NewTheme()
	th.Shaper = text.NewShaper(text.WithCollection(gofont.Collection()))
	return &ui{
		deps:   d,
		win:    w,
		th:     th,
		list:   widget.List{List: layout.List{Axis: layout.Vertical}},
		clicks: map[string]*choiceButtons{},
		thumbs: map[string]image.Image{},
	}
}

func (u *ui) loop(w *gioapp.Window) error {
	for {
		switch e := w.Event().(type) {
		case gioapp.DestroyEvent:
			return e.Err
		case gioapp.FrameEvent:
			gtx := gioapp.NewContext(&u.ops, e)
			u.layout(gtx)
			e.Frame(gtx.Ops)
		}
	}
}

func (u *ui) buttons(path string) *choiceButtons {
	b, ok := u.clicks[path]
	if !ok {
		b = &choiceButtons{}
		u.clicks[path] = b
	}
	return b
}

func (u *ui) layout(gtx layout.Context) layout.Dimensions {
	u.mu.Lock()
	listing := u.view.Listing
	u.mu.Unlock()
	if listing {
		gtx.Execute(op.InvalidateCmd{At: gtx.Now.Add(100 * time.Millisecond)})
	}
	if u.pick.Clicked(gtx) {
		u.choose()
	}
	if u.start.Clicked(gtx) {
		u.begin()
	}
	if u.dry.Clicked(gtx) {
		u.dryRun()
	}
	if u.apply.Clicked(gtx) {
		u.applyPlan()
	}
	return layout.UniformInset(unit.Dp(12)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(material.H6(u.th, "hdsort").Layout),
			layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
			layout.Rigid(material.CheckBox(u.th, &u.exactOnly, "Exact duplicates only").Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(material.Button(u.th, &u.pick, "Choose folder").Layout),
					layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
					layout.Flexed(1, material.Body1(u.th, u.labelRoot()).Layout),
				)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
			layout.Rigid(material.Button(u.th, &u.start, "Start scan").Layout),
			layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
			layout.Rigid(u.bars),
			layout.Rigid(material.Body2(u.th, u.statusLine()).Layout),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				rows := u.rows()
				return material.List(u.th, &u.list).Layout(gtx, len(rows), func(gtx layout.Context, i int) layout.Dimensions {
					return rows[i](gtx)
				})
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{}.Layout(gtx,
					layout.Rigid(material.Button(u.th, &u.dry, "Dry run").Layout),
					layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
					layout.Rigid(material.Button(u.th, &u.apply, "Apply").Layout),
				)
			}),
		)
	})
}

func (u *ui) bars(gtx layout.Context) layout.Dimensions {
	u.mu.Lock()
	view := u.view
	u.mu.Unlock()
	frac, ok := view.Fraction()
	if !ok {
		frac = float32(time.Now().UnixMilli()%800) / 800
	}
	children := []layout.FlexChild{
		layout.Rigid(material.ProgressBar(u.th, frac).Layout),
	}
	if view.SimilarTotal > 0 {
		sim := float32(0)
		if view.SimilarTotal > 0 {
			sim = float32(view.SimilarDone) / float32(view.SimilarTotal)
		}
		children = append(children,
			layout.Rigid(layout.Spacer{Height: unit.Dp(4)}.Layout),
			layout.Rigid(material.ProgressBar(u.th, sim).Layout),
		)
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
}

func (u *ui) labelRoot() string {
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.root == "" {
		return "No folder chosen"
	}
	return u.root
}

func (u *ui) statusLine() string {
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.view.Listing {
		return fmt.Sprintf("Listing the folder… %d files found", u.view.Found)
	}
	if u.applying != "" {
		return "Trash " + u.applying
	}
	if u.view.CurrentPath == "" {
		return ""
	}
	return fmt.Sprintf("%s · %d bytes · %s", u.view.CurrentPath, u.view.CurrentSize, u.view.Stage)
}

func (u *ui) rows() []layout.Widget {
	u.mu.Lock()
	defer u.mu.Unlock()
	var rows []layout.Widget
	add := func(text string) {
		rows = append(rows, material.Body2(u.th, text).Layout)
	}
	if u.note != "" {
		add(u.note)
	}
	if u.review.Banner != "" {
		add(u.review.Banner)
	}
	if len(u.view.Queue) > 0 {
		add("Queue")
		for _, row := range u.view.Queue {
			add(fmt.Sprintf("%s · %d bytes", row.Path, row.Size))
		}
	}
	if len(u.view.Processed) > 0 {
		add("Processed")
		for _, row := range u.view.Processed {
			add(fmt.Sprintf("%s · %d bytes · %s", row.Path, row.Size, row.Outcome))
		}
	}
	for _, group := range u.review.Exact {
		if group.HasRecommendation {
			add(fmt.Sprintf("Keep %s (%d bytes). %s", group.Keeper, group.KeeperSize, group.Text))
			add(fmt.Sprintf("Suggested deletes free %d bytes", group.Freed))
		} else {
			add("No suggestion. The copies tie.")
		}
		for _, member := range group.Members {
			rows = append(rows, u.fileRow(member))
		}
	}
	for _, group := range u.review.Similar {
		add("Similar files. No delete suggestion.")
		for _, member := range group.Members {
			rows = append(rows, u.fileRow(member))
		}
	}
	return rows
}

func (u *ui) fileRow(member app.FileSize) layout.Widget {
	path := member.Path
	size := member.Size
	return func(gtx layout.Context) layout.Dimensions {
		b := u.buttons(path)
		if b.keep.Clicked(gtx) {
			u.decide(path, domain.ChoiceKeep)
		}
		if b.drop.Clicked(gtx) {
			u.decide(path, domain.ChoiceDelete)
		}
		if b.skip.Clicked(gtx) {
			u.decide(path, domain.ChoiceSkip)
		}
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(material.Body1(u.th, fmt.Sprintf("%s · %d bytes · %s", path, size, u.choice(path))).Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{}.Layout(gtx,
					layout.Rigid(material.Button(u.th, &b.keep, "Keep").Layout),
					layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
					layout.Rigid(material.Button(u.th, &b.drop, "Delete").Layout),
					layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
					layout.Rigid(material.Button(u.th, &b.skip, "Skip").Layout),
				)
			}),
			layout.Rigid(u.thumb(path)),
		)
	}
}

func (u *ui) thumb(path string) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		u.mu.Lock()
		img := u.thumbs[path]
		u.mu.Unlock()
		if img == nil {
			return layout.Dimensions{}
		}
		gtx.Constraints.Max.X = gtx.Dp(unit.Dp(160))
		gtx.Constraints.Max.Y = gtx.Dp(unit.Dp(120))
		return widget.Image{
			Src:   paint.NewImageOp(img),
			Fit:   widget.Contain,
			Scale: 1,
		}.Layout(gtx)
	}
}

func (u *ui) choice(path string) string {
	u.mu.Lock()
	defer u.mu.Unlock()
	for _, d := range u.session.Decisions {
		if d.Path == path {
			return string(d.Choice)
		}
	}
	return ""
}

func (u *ui) decide(path string, choice domain.Choice) {
	u.mu.Lock()
	next, err := domain.RecordDecision(u.session, path, choice)
	if err != nil {
		u.note = err.Error()
	} else {
		u.session = next
	}
	u.mu.Unlock()
	u.win.Invalidate()
}

func (u *ui) choose() {
	root, err := u.deps.Pick.Pick()
	u.mu.Lock()
	if err != nil {
		u.note = err.Error()
	} else {
		u.root = root
		u.note = ""
	}
	u.mu.Unlock()
	u.win.Invalidate()
}

func (u *ui) begin() {
	u.mu.Lock()
	if u.busy || u.root == "" {
		u.mu.Unlock()
		return
	}
	root := u.root
	exactOnly := u.exactOnly.Value
	u.busy = true
	u.view = app.ScanView{}
	u.review = app.ReviewModel{}
	u.session = domain.Session{}
	u.files = nil
	u.thumbs = map[string]image.Image{}
	u.note = ""
	u.mu.Unlock()
	go u.scan(root, exactOnly)
}

func (u *ui) scan(root string, exactOnly bool) {
	result, err := app.Scan(root, u.deps.DataDir, u.deps.Walk, u.deps.Files, u.deps.ID, u.deps.Similar, exactOnly, func(ev app.ProgressEvent) {
		u.mu.Lock()
		u.view.Apply(ev)
		u.mu.Unlock()
		u.win.Invalidate()
	})
	u.mu.Lock()
	u.busy = false
	if err != nil {
		u.note = err.Error()
		u.mu.Unlock()
		u.win.Invalidate()
		return
	}
	ranked, merr := app.Match(result.Files, u.deps.Library)
	if merr != nil {
		u.note = merr.Error()
	} else {
		result.Session.Exact = ranked
	}
	entries, lerr := u.deps.Library.List()
	if lerr != nil && u.note == "" {
		u.note = lerr.Error()
	}
	u.session = result.Session
	u.files = result.Files
	u.review = app.Review(result.Session, entries)
	u.mu.Unlock()
	u.win.Invalidate()
	u.loadThumbs()
}

func (u *ui) loadThumbs() {
	u.mu.Lock()
	var members []app.FileSize
	for _, group := range u.review.Exact {
		members = append(members, group.Members...)
	}
	for _, group := range u.review.Similar {
		members = append(members, group.Members...)
	}
	u.mu.Unlock()
	for _, member := range members {
		f, err := u.deps.Files.Open(member.Path)
		if err != nil {
			continue
		}
		body, ok, err := app.Thumbnail(member.Path, f, member.Size)
		f.Close()
		if err != nil || !ok {
			continue
		}
		img, _, err := image.Decode(bytes.NewReader(body))
		if err != nil {
			continue
		}
		u.mu.Lock()
		u.thumbs[member.Path] = img
		u.mu.Unlock()
		u.win.Invalidate()
	}
}

func (u *ui) dryRun() {
	u.mu.Lock()
	if u.busy {
		u.mu.Unlock()
		return
	}
	session := u.session
	files := append([]domain.FileRef(nil), u.files...)
	u.mu.Unlock()
	go func() {
		report, err := app.DryRun(session, files, u.deps.Library, u.deps.Trash, u.deps.Disk)
		u.mu.Lock()
		if err != nil {
			u.note = err.Error() + "\n" + app.NothingChanged
		} else {
			u.note = app.FormatDryRun(report)
		}
		u.mu.Unlock()
		u.win.Invalidate()
	}()
}

func (u *ui) applyPlan() {
	u.mu.Lock()
	if u.busy {
		u.mu.Unlock()
		return
	}
	u.busy = true
	session := u.session
	files := append([]domain.FileRef(nil), u.files...)
	u.mu.Unlock()
	go func() {
		next, err := app.Apply(session, files, u.deps.Library, progressTrash{inner: u.deps.Trash, ui: u}, u.deps.Disk)
		u.mu.Lock()
		u.busy = false
		u.applying = ""
		if err != nil {
			u.note = err.Error()
		} else {
			u.session = next
			u.note = "Applied."
		}
		u.mu.Unlock()
		u.win.Invalidate()
	}()
}

type progressTrash struct {
	inner app.Trasher
	ui    *ui
}

func (p progressTrash) Trash(path string) error {
	p.ui.mu.Lock()
	p.ui.applying = path
	p.ui.mu.Unlock()
	p.ui.win.Invalidate()
	return p.inner.Trash(path)
}
