//go:build linux

package gui

import (
	"image/color"

	"github.com/godbus/dbus/v5"

	"github.com/cionz0/hdsort/internal/app"
)

func readSystem() (app.Appearance, app.SystemColors) {
	appearance := app.AppearanceLight
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return appearance, app.SystemColors{}
	}
	defer conn.Close()
	obj := conn.Object("org.freedesktop.portal.Desktop", "/org/freedesktop/portal/desktop")
	if scheme, ok := portalValue(obj, "color-scheme"); ok {
		if number, ok := asUint(scheme); ok && number == 1 {
			appearance = app.AppearanceDark
		}
	}
	var colors app.SystemColors
	if accent, ok := portalValue(obj, "accent-color"); ok {
		if c, ok := accentRGB(accent); ok {
			colors.Primary = app.OptionalColor{Color: c, Set: true}
		}
	}
	return appearance, colors
}

func portalValue(obj dbus.BusObject, key string) (any, bool) {
	call := obj.Call("org.freedesktop.portal.Settings.Read", 0, "org.freedesktop.appearance", key)
	if call.Err != nil || len(call.Body) == 0 {
		return nil, false
	}
	value := call.Body[0]
	for {
		variant, ok := value.(dbus.Variant)
		if !ok {
			return value, true
		}
		value = variant.Value()
	}
}

func asUint(value any) (uint32, bool) {
	switch n := value.(type) {
	case uint32:
		return n, true
	case int32:
		if n < 0 {
			return 0, false
		}
		return uint32(n), true
	default:
		return 0, false
	}
}

func accentRGB(value any) (color.NRGBA, bool) {
	var numbers []float64
	switch parts := value.(type) {
	case []float64:
		numbers = parts
	case []any:
		for _, part := range parts {
			number, ok := part.(float64)
			if !ok {
				return color.NRGBA{}, false
			}
			numbers = append(numbers, number)
		}
	default:
		return color.NRGBA{}, false
	}
	if len(numbers) < 3 {
		return color.NRGBA{}, false
	}
	channels := [3]uint8{}
	for i := 0; i < 3; i++ {
		number := numbers[i]
		if number < 0 {
			number = 0
		}
		if number > 1 {
			number = 1
		}
		channels[i] = uint8(number*255 + 0.5)
	}
	return color.NRGBA{R: channels[0], G: channels[1], B: channels[2], A: 0xFF}, true
}
