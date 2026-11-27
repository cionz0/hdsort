//go:build !darwin && !linux && !windows

package gui

import "github.com/cionz0/hdsort/internal/app"

func readSystem() (app.Appearance, app.SystemColors) {
	return app.AppearanceLight, app.SystemColors{}
}
