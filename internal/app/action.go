package app

import "fmt"

// ActionSlot is the shared row of Start scan, the progress bar, Dry run,
// and Apply. Only the control whose action is running is disabled, and
// Label is the reason.
type ActionSlot struct {
	Label         string
	Fraction      float32
	Determinate   bool
	StartDisabled bool
	DryDisabled   bool
	ApplyDisabled bool
}

// SlotFrom builds the action row from the scan view and, when set, the
// path currently passed to Trash.
func SlotFrom(view ScanView, trashing string) ActionSlot {
	if trashing != "" {
		return ActionSlot{Label: "Trash " + trashing, ApplyDisabled: true}
	}
	if view.Listing && view.Total == 0 {
		return ActionSlot{Label: "Listing the folder…", StartDisabled: true}
	}
	if view.SimilarTotal > 0 && view.SimilarDone < view.SimilarTotal && view.Total > 0 && view.Done >= view.Total {
		return ActionSlot{
			Label:         "Comparing similar files",
			Fraction:      float32(view.SimilarDone) / float32(view.SimilarTotal),
			Determinate:   true,
			StartDisabled: true,
		}
	}
	running := view.Total == 0 || view.Done < view.Total
	if view.CurrentPath != "" && running {
		fraction, ok := view.Fraction()
		return ActionSlot{
			Label:         fmt.Sprintf("%s · %s", view.CurrentPath, view.Stage),
			Fraction:      fraction,
			Determinate:   ok,
			StartDisabled: true,
		}
	}
	if running && view.Total > 0 {
		fraction, ok := view.Fraction()
		return ActionSlot{
			Label:         "Scanning",
			Fraction:      fraction,
			Determinate:   ok,
			StartDisabled: true,
		}
	}
	fraction, ok := view.Fraction()
	return ActionSlot{Fraction: fraction, Determinate: ok}
}
