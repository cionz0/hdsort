package app

import "image/color"

// Appearance is the operating system's light or dark choice.
type Appearance int

const (
	// AppearanceLight selects the light fallback palette.
	AppearanceLight Appearance = iota
	// AppearanceDark selects the dark fallback palette.
	AppearanceDark
)

// Palette is the four colors the window paints.
type Palette struct {
	Background color.NRGBA
	Text       color.NRGBA
	Primary    color.NRGBA
	OnPrimary  color.NRGBA
}

// OptionalColor is a system color. Set is false when the operating system
// does not provide that role.
type OptionalColor struct {
	Color color.NRGBA
	Set   bool
}

// SystemColors holds the roles the operating system actually provided.
type SystemColors struct {
	Background OptionalColor
	Text       OptionalColor
	Primary    OptionalColor
	OnPrimary  OptionalColor
}

// LightFallback is the palette used for every role the system does not
// provide while the appearance is light.
var LightFallback = Palette{
	Background: rgb(0xF5, 0xF5, 0xF7),
	Text:       rgb(0x1C, 0x1C, 0x1E),
	Primary:    rgb(0x1C, 0x1C, 0x1E),
	OnPrimary:  rgb(0xFF, 0xFF, 0xFF),
}

// DarkFallback is the palette used for every role the system does not
// provide while the appearance is dark.
var DarkFallback = Palette{
	Background: rgb(0x1C, 0x1C, 0x1E),
	Text:       rgb(0xF5, 0xF5, 0xF7),
	Primary:    rgb(0xF5, 0xF5, 0xF7),
	OnPrimary:  rgb(0x1C, 0x1C, 0x1E),
}

// ResolvePalette starts from the light or dark fallback, then replaces each
// role the system provided. A missing accent does not change the background.
func ResolvePalette(appearance Appearance, system SystemColors) Palette {
	palette := LightFallback
	if appearance == AppearanceDark {
		palette = DarkFallback
	}
	if system.Background.Set {
		palette.Background = system.Background.Color
	}
	if system.Text.Set {
		palette.Text = system.Text.Color
	}
	if system.Primary.Set {
		palette.Primary = system.Primary.Color
	}
	if system.OnPrimary.Set {
		palette.OnPrimary = system.OnPrimary.Color
	}
	return palette
}

func rgb(r, g, b uint8) color.NRGBA {
	return color.NRGBA{R: r, G: g, B: b, A: 0xFF}
}
