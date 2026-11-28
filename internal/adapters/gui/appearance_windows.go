//go:build windows

package gui

import (
	"image/color"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"

	"github.com/cionz0/hdsort/internal/app"
)

func readSystem() (app.Appearance, app.SystemColors) {
	appearance := app.AppearanceLight
	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`, registry.QUERY_VALUE)
	if err == nil {
		if value, _, err := key.GetIntegerValue("AppsUseLightTheme"); err == nil && value == 0 {
			appearance = app.AppearanceDark
		}
		key.Close()
	}
	var colors app.SystemColors
	if accent, ok := windowsAccent(); ok {
		colors.Accent = app.OptionalColor{Color: accent, Set: true}
	}
	return appearance, colors
}

func windowsAccent() (color.NRGBA, bool) {
	proc := windows.NewLazySystemDLL("dwmapi.dll").NewProc("DwmGetColorizationColor")
	var colorization uint32
	var opaque uint32
	result, _, _ := proc.Call(uintptr(unsafe.Pointer(&colorization)), uintptr(unsafe.Pointer(&opaque)))
	if result != 0 {
		return color.NRGBA{}, false
	}
	return color.NRGBA{
		R: uint8(colorization >> 16),
		G: uint8(colorization >> 8),
		B: uint8(colorization),
		A: 0xFF,
	}, true
}
