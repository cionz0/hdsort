package gui

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"sync"
	"time"

	gioapp "gioui.org/app"
	"gioui.org/font/gofont"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
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
	Launch  app.Launcher
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
	launch widget.Clickable
	choice widget.Enum
}

type rowButtons struct {
	open, skip, apply widget.Clickable
}

type outlineButtons struct {
	disclosure, name widget.Clickable
}

type ui struct {
	deps Deps
	win  *gioapp.Window
	th   *material.Theme

	pick, start, dry, apply widget.Clickable
	exactOnly               widget.Bool
	list                    widget.List
	copies                  widget.List
	clicks                  map[string]*choiceButtons
	rows                    map[string]*rowButtons
	outline                 map[string]*outlineButtons
	folderTrash             widget.Clickable

	mu          sync.Mutex
	root        string
	scanning    bool
	applying    string
	applyBusy   bool
	drying      bool
	note        string
	view        app.ScanView
	board       app.FolderBoard
	review      app.ReviewModel
	session     domain.Session
	files       []domain.FileRef
	selected    string
	selectedDir string
	expanded    map[string]bool
	thumbs      map[string]image.Image
	palette     app.Palette
	themeAt     time.Time
	ops         op.Ops
}

func newUI(d Deps, w *gioapp.Window) *ui {
	th := material.NewTheme()
	th.Shaper = text.NewShaper(text.WithCollection(gofont.Collection()))
	appearance, colors := readSystem()
	ui := &ui{
		deps:     d,
		win:      w,
		th:       th,
		list:     widget.List{List: layout.List{Axis: layout.Vertical}},
		copies:   widget.List{List: layout.List{Axis: layout.Horizontal}},
		clicks:   map[string]*choiceButtons{},
		rows:     map[string]*rowButtons{},
		outline:  map[string]*outlineButtons{},
		expanded: map[string]bool{},
		thumbs:   map[string]image.Image{},
		palette:  paintTheme(th, appearance, colors),
	}
	return ui
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
	u.refreshTheme()
	u.mu.Lock()
	slot := app.SlotFrom(u.root, u.view, u.applying)
	if u.drying {
		slot.DryDisabled = true
	}
	if u.applyBusy && u.applying == "" {
		slot.ApplyDisabled = true
		if slot.Label == "" {
			slot.Label = "Apply"
		}
	}
	rows := append([]app.FolderRow(nil), u.board.Rows...)
	note, banner, selected, selectedDir, root := u.note, u.review.Banner, u.selected, u.selectedDir, u.root
	trashBusy := u.applying != ""
	scanned := u.view.Total > 0 || u.view.Done > 0
	u.mu.Unlock()
	gtx.Execute(op.InvalidateCmd{At: gtx.Now.Add(time.Second)})
	if !slot.Determinate && (slot.StartDisabled || slot.ApplyDisabled) {
		gtx.Execute(op.InvalidateCmd{At: gtx.Now.Add(100 * time.Millisecond)})
	}
	paint.Fill(gtx.Ops, u.palette.Background)
	if u.pick.Clicked(gtx) {
		u.choose()
	}
	if !slot.StartDisabled && u.start.Clicked(gtx) {
		u.begin()
	}
	if !slot.DryDisabled && u.dry.Clicked(gtx) {
		u.dryRun()
	}
	if !slot.ApplyDisabled && u.apply.Clicked(gtx) {
		u.applyPlan()
	}
	return layout.UniformInset(unit.Dp(12)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(material.H6(u.th, "hdsort").Layout),
			layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
			layout.Rigid(material.CheckBox(u.th, &u.exactOnly, "Exact duplicates only").Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(u.quietButton(&u.pick, "Choose folder")),
					layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
					layout.Flexed(1, u.hint(u.labelRoot())),
				)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return u.body(gtx, rows, note, banner, selected, selectedDir, root, scanned, trashBusy)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return u.actionRow(gtx, slot)
			}),
		)
	})
}

func (u *ui) refreshTheme() {
	if !u.themeAt.IsZero() && time.Since(u.themeAt) < time.Second {
		return
	}
	u.themeAt = time.Now()
	appearance, colors := readSystem()
	next := paintTheme(u.th, appearance, colors)
	if next != u.palette {
		u.palette = next
		u.win.Invalidate()
	}
}

func (u *ui) actionRow(gtx layout.Context, slot app.ActionSlot) layout.Dimensions {
	frac := slot.Fraction
	if !slot.Determinate && (slot.StartDisabled || slot.ApplyDisabled) {
		frac = float32(time.Now().UnixMilli()%800) / 800
	}
	return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(u.command(gtx, &u.start, "Start scan", slot.StartDisabled, false)),
		layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			bar := material.ProgressBar(u.th, frac)
			bar.Color = u.palette.Accent
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(u.hint(slot.Label)),
				layout.Rigid(bar.Layout),
			)
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
		layout.Rigid(u.command(gtx, &u.dry, "Dry run", slot.DryDisabled, false)),
		layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
		layout.Rigid(u.command(gtx, &u.apply, "Apply", slot.ApplyDisabled, true)),
	)
}

func (u *ui) command(gtx layout.Context, click *widget.Clickable, label string, disabled, accent bool) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		if disabled {
			gtx = gtx.Disabled()
		}
		if accent {
			return u.accentButton(click, label)(gtx)
		}
		return u.quietButton(click, label)(gtx)
	}
}

func (u *ui) quietButton(click *widget.Clickable, label string) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		btn := material.Button(u.th, click, label)
		btn.Background = u.palette.Control
		btn.Color = u.palette.Text
		btn.CornerRadius = unit.Dp(8)
		return widget.Border{
			Color:        u.palette.Separator,
			CornerRadius: unit.Dp(8),
			Width:        unit.Dp(1),
		}.Layout(gtx, btn.Layout)
	}
}

func (u *ui) accentButton(click *widget.Clickable, label string) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		btn := material.Button(u.th, click, label)
		btn.Background = u.palette.Accent
		btn.Color = u.palette.OnAccent
		btn.CornerRadius = unit.Dp(8)
		return btn.Layout(gtx)
	}
}

func (u *ui) hint(text string) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		label := material.Body2(u.th, text)
		label.Color = u.palette.Secondary
		return label.Layout(gtx)
	}
}

func (u *ui) fill(c color.NRGBA, w layout.Widget) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		macro := op.Record(gtx.Ops)
		dims := w(gtx)
		call := macro.Stop()
		defer clip.Rect(image.Rectangle{Max: dims.Size}).Push(gtx.Ops).Pop()
		paint.ColorOp{Color: c}.Add(gtx.Ops)
		paint.PaintOp{}.Add(gtx.Ops)
		call.Add(gtx.Ops)
		return dims
	}
}

func (u *ui) labelRoot() string {
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.root == "" {
		return "No folder chosen"
	}
	return u.root
}

func (u *ui) rowButtons(key string) *rowButtons {
	b, ok := u.rows[key]
	if !ok {
		b = &rowButtons{}
		u.rows[key] = b
	}
	return b
}

func (u *ui) body(gtx layout.Context, rows []app.FolderRow, note, banner, selected, selectedDir, root string, scanned, trashBusy bool) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if note == "" {
				return layout.Dimensions{}
			}
			return material.Body2(u.th, note).Layout(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if banner == "" {
				return layout.Dimensions{}
			}
			return material.Body2(u.th, banner).Layout(gtx)
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{}.Layout(gtx,
				layout.Flexed(0.8, func(gtx layout.Context) layout.Dimensions {
					return u.folderList(gtx, rows, selected, selectedDir, root)
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
				layout.Rigid(u.vline),
				layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
				layout.Flexed(1.2, func(gtx layout.Context) layout.Dimensions {
					return u.compare(gtx, rows, selected, selectedDir, root, scanned, trashBusy)
				}),
			)
		}),
	)
}

func (u *ui) vline(gtx layout.Context) layout.Dimensions {
	w := gtx.Dp(unit.Dp(1))
	h := gtx.Constraints.Max.Y
	if h < 1 {
		h = 1
	}
	defer clip.Rect(image.Rectangle{Max: image.Pt(w, h)}).Push(gtx.Ops).Pop()
	paint.ColorOp{Color: u.palette.Separator}.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	return layout.Dimensions{Size: image.Pt(w, h)}
}

func (u *ui) folderList(gtx layout.Context, rows []app.FolderRow, selected, selectedDir, root string) layout.Dimensions {
	if root == "" {
		return material.Body2(u.th, "Choose a folder and start a scan.").Layout(gtx)
	}
	u.mu.Lock()
	var expanded []string
	for path, on := range u.expanded {
		if on {
			expanded = append(expanded, path)
		}
	}
	u.mu.Unlock()
	nodes := app.Outline(root, rows, expanded)
	open := map[string]bool{}
	for _, path := range expanded {
		open[filepath.Clean(path)] = true
	}
	return material.List(u.th, &u.list).Layout(gtx, len(nodes), func(gtx layout.Context, i int) layout.Dimensions {
		node := nodes[i]
		inset := layout.Inset{Top: unit.Dp(2), Bottom: unit.Dp(2), Left: unit.Dp(16 * node.Depth)}
		if node.Directory {
			b := u.outlineButtons(node.Path)
			if b.disclosure.Clicked(gtx) {
				u.toggleDir(node.Path)
			}
			if b.name.Clicked(gtx) {
				u.mu.Lock()
				u.selectedDir = node.Path
				u.selected = ""
				u.mu.Unlock()
			}
			mark := "▸"
			if open[node.Path] {
				mark = "▾"
			}
			name := node.Name
			if node.Path == selectedDir {
				name = "● " + name
			}
			return inset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return b.disclosure.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							label := material.Body1(u.th, mark)
							label.Color = u.palette.Secondary
							return label.Layout(gtx)
						})
					}),
					layout.Rigid(layout.Spacer{Width: unit.Dp(4)}.Layout),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return b.name.Layout(gtx, material.Body1(u.th, name).Layout)
					}),
				)
			})
		}
		row := rowByKey(rows, node.RowKey)
		if row == nil {
			return layout.Dimensions{}
		}
		b := u.rowButtons(row.Key)
		if b.open.Clicked(gtx) {
			u.mu.Lock()
			u.selected = row.Key
			u.selectedDir = ""
			u.mu.Unlock()
		}
		title := row.Title
		if row.Key == selected {
			title = "● " + title
		}
		return inset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return b.open.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(material.Body1(u.th, title).Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if row.Status == "" {
							return layout.Dimensions{}
						}
						return u.hint(" · " + row.Status)(gtx)
					}),
				)
			})
		})
	})
}

func (u *ui) compare(gtx layout.Context, rows []app.FolderRow, selected, selectedDir, root string, scanned, trashBusy bool) layout.Dimensions {
	if selectedDir != "" {
		return u.folderPane(gtx, root, selectedDir, trashBusy)
	}
	var row *app.FolderRow
	for i := range rows {
		if rows[i].Key == selected {
			row = &rows[i]
			break
		}
	}
	if row == nil && len(rows) > 0 {
		row = &rows[0]
	}
	if row == nil {
		if scanned {
			return material.Body2(u.th, "No duplicate files.").Layout(gtx)
		}
		return material.Body2(u.th, "Choose a folder and start a scan.").Layout(gtx)
	}
	current := *row
	controls := u.rowButtons(current.Key)
	if controls.skip.Clicked(gtx) {
		u.skipSet(memberPaths(current))
	}
	if current.Kind == app.RowExact && controls.apply.Clicked(gtx) {
		u.applyFolder(current)
	}
	hint := current.Text
	if current.Kind == app.RowExact && !current.HasRecommendation {
		hint = "No suggestion. The copies tie."
	}
	if current.Kind == app.RowSimilar {
		hint = "Similar files. No delete suggestion."
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(material.H6(u.th, current.Title).Layout),
		layout.Rigid(u.hint(hint)),
		layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			children := []layout.FlexChild{
				layout.Rigid(u.quietButton(&controls.skip, "Skip set")),
			}
			if current.Kind == app.RowExact {
				label := "Apply to all files of this type in " + filepath.Base(current.Folder)
				children = append(children,
					layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
					layout.Rigid(u.accentButton(&controls.apply, label)),
				)
			}
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return material.List(u.th, &u.copies).Layout(gtx, len(current.Members), func(gtx layout.Context, i int) layout.Dimensions {
				return u.copyColumn(gtx, root, current, current.Members[i])
			})
		}),
	)
}

func (u *ui) copyColumn(gtx layout.Context, root string, row app.FolderRow, member app.FileSize) layout.Dimensions {
	b := u.buttons(member.Path)
	if b.launch.Clicked(gtx) {
		u.openCopy(member.Path)
	}
	shown := u.shownChoice(row, member.Path)
	if b.choice.Update(gtx) {
		u.decide(member.Path, domain.Choice(b.choice.Value))
	} else if b.choice.Value != string(shown) {
		b.choice.Value = string(shown)
	}
	gtx.Constraints.Min.X = gtx.Dp(unit.Dp(200))
	gtx.Constraints.Max.X = gtx.Dp(unit.Dp(240))
	return layout.Inset{Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return widget.Border{
			Color:        u.palette.Separator,
			CornerRadius: unit.Dp(8),
			Width:        unit.Dp(1),
		}.Layout(gtx, u.fill(u.palette.Surface, func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				lines := []layout.FlexChild{
					layout.Rigid(u.thumb(member.Path)),
					layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
					layout.Rigid(u.hint(app.DisplayPath(root, member.Path))),
					layout.Rigid(layout.Spacer{Height: unit.Dp(4)}.Layout),
					layout.Rigid(u.hint(app.FormatSize(member.Size))),
					layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
					layout.Rigid(u.quietButton(&b.launch, "Open")),
					layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
					layout.Rigid(u.radio(&b.choice, string(domain.ChoiceKeep), "Keep")),
					layout.Rigid(u.radio(&b.choice, string(domain.ChoiceDelete), "Delete")),
					layout.Rigid(u.radio(&b.choice, string(domain.ChoiceSkip), "Skip")),
				}
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx, lines...)
			})
		}))
	})
}

func (u *ui) radio(group *widget.Enum, key, label string) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		btn := material.RadioButton(u.th, group, key, label)
		btn.Color = u.palette.Text
		btn.IconColor = u.palette.Text
		return btn.Layout(gtx)
	}
}

func (u *ui) shownChoice(row app.FolderRow, path string) domain.Choice {
	u.mu.Lock()
	defer u.mu.Unlock()
	return app.ShownChoice(u.session, row.HasRecommendation, row.Keeper, row.Deletes, path)
}

func memberPaths(row app.FolderRow) []string {
	paths := make([]string, len(row.Members))
	for i, member := range row.Members {
		paths[i] = member.Path
	}
	return paths
}

func (u *ui) thumb(path string) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		width := gtx.Constraints.Max.X
		height := gtx.Dp(unit.Dp(140))
		size := image.Pt(width, height)
		return widget.Border{
			Color:        u.palette.Separator,
			CornerRadius: unit.Dp(4),
			Width:        unit.Dp(1),
		}.Layout(gtx, u.fill(u.palette.Surface, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min = size
			gtx.Constraints.Max = size
			u.mu.Lock()
			img := u.thumbs[path]
			u.mu.Unlock()
			if img == nil {
				caption := material.Caption(u.th, "No preview")
				caption.Color = u.palette.Secondary
				return layout.Center.Layout(gtx, caption.Layout)
			}
			return widget.Image{
				Src:   paint.NewImageOp(img),
				Fit:   widget.Contain,
				Scale: 1,
			}.Layout(gtx)
		}))
	}
}

func (u *ui) openCopy(path string) {
	go func() {
		u.mu.Lock()
		session := u.session
		u.mu.Unlock()
		_, err := app.OpenCopy(session, u.deps.Launch, path)
		u.mu.Lock()
		if err != nil {
			u.note = err.Error()
		}
		u.mu.Unlock()
		u.win.Invalidate()
	}()
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
	if u.scanning || u.root == "" {
		u.mu.Unlock()
		return
	}
	root := u.root
	exactOnly := u.exactOnly.Value
	u.scanning = true
	u.view = app.ScanView{}
	u.board = app.FolderBoard{}
	u.review = app.ReviewModel{}
	u.session = domain.Session{}
	u.files = nil
	u.selected = ""
	u.selectedDir = ""
	u.expanded = map[string]bool{filepath.Clean(root): true}
	u.thumbs = map[string]image.Image{}
	u.note = ""
	u.mu.Unlock()
	go u.scan(root, exactOnly)
}

func (u *ui) scan(root string, exactOnly bool) {
	result, err := app.Scan(root, u.deps.DataDir, u.deps.Walk, u.deps.Files, u.deps.ID, u.deps.Similar, exactOnly, func(ev app.ProgressEvent) {
		u.mu.Lock()
		u.view.Apply(ev)
		u.board.Apply(ev)
		u.mu.Unlock()
		u.win.Invalidate()
	})
	u.mu.Lock()
	u.scanning = false
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
	u.board.UseReview(u.review)
	if len(u.board.Rows) > 0 {
		u.selected = u.board.Rows[0].Key
	}
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
		body, ok, err := app.LoadPreview(member.Path, f, member.Size, osPreview{})
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
	if u.drying {
		u.mu.Unlock()
		return
	}
	u.drying = true
	session := u.session
	files := append([]domain.FileRef(nil), u.files...)
	u.mu.Unlock()
	go func() {
		report, err := app.DryRun(session, files, u.deps.Library, u.deps.Trash, u.deps.Disk)
		u.mu.Lock()
		u.drying = false
		if err != nil {
			u.note = err.Error() + "\n" + app.NothingChanged
		} else {
			u.note = app.FormatDryRun(report)
		}
		u.mu.Unlock()
		u.win.Invalidate()
	}()
}

func (u *ui) folderPane(gtx layout.Context, root, folder string, trashBusy bool) layout.Dimensions {
	name := app.DirectoryName(root, folder)
	if u.folderTrash.Clicked(gtx) && !trashBusy {
		u.trashFolder(folder)
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(material.H6(u.th, name).Layout),
		layout.Rigid(u.hint("Trash this folder and everything inside it.")),
		layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if trashBusy {
				gtx = gtx.Disabled()
			}
			return u.accentButton(&u.folderTrash, app.FolderTrashLabel(name))(gtx)
		}),
	)
}

func (u *ui) trashFolder(folder string) {
	u.mu.Lock()
	if u.applying != "" || u.applyBusy {
		u.mu.Unlock()
		return
	}
	u.applying = folder
	session := u.session
	u.mu.Unlock()
	u.win.Invalidate()
	go func() {
		next, err := app.TrashFolder(session, folder, u.deps.Trash)
		u.mu.Lock()
		u.applying = ""
		if err != nil {
			u.note = err.Error()
		} else {
			u.session = next
			u.board.Rows = app.DropTrashed(u.board.Rows, folder)
			if domain.PathInside(folder, u.selectedDir) {
				u.selectedDir = ""
			}
			if rowByKey(u.board.Rows, u.selected) == nil {
				u.selected = ""
			}
			u.note = ""
		}
		u.mu.Unlock()
		u.win.Invalidate()
	}()
}

func (u *ui) outlineButtons(key string) *outlineButtons {
	b := u.outline[key]
	if b == nil {
		b = &outlineButtons{}
		u.outline[key] = b
	}
	return b
}

func (u *ui) toggleDir(path string) {
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.expanded == nil {
		u.expanded = map[string]bool{}
	}
	u.expanded[path] = !u.expanded[path]
}

func rowByKey(rows []app.FolderRow, key string) *app.FolderRow {
	for i := range rows {
		if rows[i].Key == key {
			return &rows[i]
		}
	}
	return nil
}

func (u *ui) skipSet(paths []string) {
	u.mu.Lock()
	next, err := app.SkipSet(u.session, paths)
	if err != nil {
		u.note = err.Error()
	} else {
		u.session = next
		u.note = ""
	}
	u.mu.Unlock()
	u.win.Invalidate()
}

func (u *ui) applyFolder(row app.FolderRow) {
	open := ""
	if len(row.Members) > 0 {
		open = row.Members[0].Path
	}
	if row.Keeper != "" {
		open = row.Keeper
	}
	u.mu.Lock()
	next, err := app.ApplySameTypeInFolder(u.session, row.Folder, open)
	if err != nil {
		u.note = err.Error()
	} else {
		u.session = next
		u.note = ""
	}
	u.mu.Unlock()
	u.win.Invalidate()
}

func (u *ui) applyPlan() {
	u.mu.Lock()
	if u.applyBusy {
		u.mu.Unlock()
		return
	}
	u.applyBusy = true
	session := u.session
	files := append([]domain.FileRef(nil), u.files...)
	u.mu.Unlock()
	go func() {
		next, err := app.Apply(session, files, u.deps.Library, progressTrash{inner: u.deps.Trash, ui: u}, u.deps.Disk)
		u.mu.Lock()
		u.applyBusy = false
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
