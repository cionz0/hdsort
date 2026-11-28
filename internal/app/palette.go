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

// Palette is the window chrome. Accent fills only a committing action.
// Text, secondary text, the control fill, and the separator keep the alpha
// Apple uses for those semantic colors.
type Palette struct {
	Background color.NRGBA
	Surface    color.NRGBA
	Control    color.NRGBA
	Text       color.NRGBA
	Secondary  color.NRGBA
	Separator  color.NRGBA
	Accent     color.NRGBA
	OnAccent   color.NRGBA
}

// OptionalColor is a system color. Set is false when the operating system
// does not provide that role.
type OptionalColor struct {
	Color color.NRGBA
	Set   bool
}

// SystemColors holds the roles the operating system actually provided.
// On-accent is not one of them.
type SystemColors struct {
	Background OptionalColor
	Surface    OptionalColor
	Control    OptionalColor
	Text       OptionalColor
	Secondary  OptionalColor
	Separator  OptionalColor
	Accent     OptionalColor
}

// LightFallback is Aqua, resolved on this Mac. Apple documents the semantic
// names, not these numbers.
var LightFallback = Palette{
	Background: rgb(0xFF, 0xFF, 0xFF),
	Surface:    rgb(0xF6, 0xF6, 0xF6),
	Control:    rgb(0xFF, 0xFF, 0xFF),
	Text:       rgba(0x00, 0x00, 0x00, 217),
	Secondary:  rgba(0x00, 0x00, 0x00, 128),
	Separator:  rgba(0x00, 0x00, 0x00, 26),
	Accent:     rgb(0x00, 0x7A, 0xFF),
	OnAccent:   rgb(0xFF, 0xFF, 0xFF),
}

// DarkFallback is Dark Aqua, resolved on this Mac.
var DarkFallback = Palette{
	Background: rgb(0x1E, 0x1E, 0x1E),
	Surface:    rgb(0x28, 0x28, 0x28),
	Control:    rgba(0xFF, 0xFF, 0xFF, 64),
	Text:       rgba(0xFF, 0xFF, 0xFF, 217),
	Secondary:  rgba(0xFF, 0xFF, 0xFF, 140),
	Separator:  rgba(0xFF, 0xFF, 0xFF, 31),
	Accent:     rgb(0x00, 0x7A, 0xFF),
	OnAccent:   rgb(0xFF, 0xFF, 0xFF),
}

// ResolvePalette starts from the light or dark fallback, then replaces each
// role the system provided. A missing accent does not change the background,
// the surface, or the text. On-accent stays the fallback.
func ResolvePalette(appearance Appearance, system SystemColors) Palette {
	palette := LightFallback
	if appearance == AppearanceDark {
		palette = DarkFallback
	}
	if system.Background.Set {
		palette.Background = system.Background.Color
	}
	if system.Surface.Set {
		palette.Surface = system.Surface.Color
	}
	if system.Control.Set {
		palette.Control = system.Control.Color
	}
	if system.Text.Set {
		palette.Text = system.Text.Color
	}
	if system.Secondary.Set {
		palette.Secondary = system.Secondary.Color
	}
	if system.Separator.Set {
		palette.Separator = system.Separator.Color
	}
	if system.Accent.Set {
		palette.Accent = system.Accent.Color
	}
	return palette
}

func rgb(r, g, b uint8) color.NRGBA {
	return rgba(r, g, b, 0xFF)
}

func rgba(r, g, b, a uint8) color.NRGBA {
	return color.NRGBA{R: r, G: g, B: b, A: a}
}
