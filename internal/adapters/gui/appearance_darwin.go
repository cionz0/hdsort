//go:build darwin

package gui

/*
#cgo darwin CFLAGS: -x objective-c -fobjc-arc
#cgo darwin LDFLAGS: -framework AppKit -framework Foundation
#import <AppKit/AppKit.h>
#import <Foundation/Foundation.h>

static int hdsortIsDark(void) {
	NSAppearance *appearance = nil;
	if (NSApp != nil) {
		appearance = [NSApp effectiveAppearance];
	}
	if (appearance == nil) {
		appearance = [NSAppearance currentDrawingAppearance];
	}
	if (appearance == nil) {
		return 0;
	}
	NSAppearanceName match = [appearance bestMatchFromAppearancesWithNames:@[
		NSAppearanceNameAqua,
		NSAppearanceNameDarkAqua
	]];
	return [match isEqualToString:NSAppearanceNameDarkAqua] ? 1 : 0;
}

static int hdsortColor(int which, unsigned char *r, unsigned char *g, unsigned char *b, unsigned char *a) {
	NSColor *c = nil;
	switch (which) {
	case 0: c = [NSColor windowBackgroundColor]; break;
	case 1: c = [NSColor underPageBackgroundColor]; break;
	case 2: c = [NSColor controlColor]; break;
	case 3: c = [NSColor labelColor]; break;
	case 4: c = [NSColor secondaryLabelColor]; break;
	case 5: c = [NSColor separatorColor]; break;
	default: c = [NSColor controlAccentColor]; break;
	}
	NSAppearance *appearance = nil;
	if (NSApp != nil) {
		appearance = [NSApp effectiveAppearance];
	}
	if (appearance == nil) {
		appearance = [NSAppearance currentDrawingAppearance];
	}
	if (c == nil || appearance == nil) {
		return 0;
	}
	__block int ok = 0;
	__block unsigned char rr = 0, gg = 0, bb = 0, aa = 0;
	[appearance performAsCurrentDrawingAppearance:^{
		NSColor *rgb = [c colorUsingColorSpace:[NSColorSpace sRGBColorSpace]];
		if (rgb == nil) {
			return;
		}
		rr = (unsigned char)(rgb.redComponent * 255.0 + 0.5);
		gg = (unsigned char)(rgb.greenComponent * 255.0 + 0.5);
		bb = (unsigned char)(rgb.blueComponent * 255.0 + 0.5);
		aa = (unsigned char)(rgb.alphaComponent * 255.0 + 0.5);
		ok = 1;
	}];
	if (!ok) {
		return 0;
	}
	*r = rr;
	*g = gg;
	*b = bb;
	*a = aa;
	return 1;
}
*/
import "C"
import (
	"image/color"

	"github.com/cionz0/hdsort/internal/app"
)

func readSystem() (app.Appearance, app.SystemColors) {
	appearance := app.AppearanceLight
	if C.hdsortIsDark() == 1 {
		appearance = app.AppearanceDark
	}
	return appearance, app.SystemColors{
		Background: darwinColor(0),
		Surface:    darwinColor(1),
		Control:    darwinColor(2),
		Text:       darwinColor(3),
		Secondary:  darwinColor(4),
		Separator:  darwinColor(5),
		Accent:     darwinColor(6),
	}
}

func darwinColor(which C.int) app.OptionalColor {
	var r, g, b, a C.uchar
	if C.hdsortColor(which, &r, &g, &b, &a) == 0 {
		return app.OptionalColor{}
	}
	return app.OptionalColor{
		Color: color.NRGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: uint8(a)},
		Set:   true,
	}
}
