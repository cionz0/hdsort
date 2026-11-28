package gui

import (
	"gioui.org/widget/material"

	"github.com/cionz0/hdsort/internal/app"
)

func paintTheme(th *material.Theme, appearance app.Appearance, system app.SystemColors) app.Palette {
	palette := app.ResolvePalette(appearance, system)
	th.Palette.Bg = palette.Background
	th.Palette.Fg = palette.Text
	th.Palette.ContrastBg = palette.Control
	th.Palette.ContrastFg = palette.Text
	return palette
}
