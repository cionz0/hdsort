package app

// Progress event kinds. The window applies them as they arrive.
const (
	EventListing   = "listing"
	EventQueue     = "queue"
	EventHashing   = "hashing"
	EventProcessed = "processed"
	EventSimilar   = "similar"
)

// Outcomes shown on a processed row.
const (
	OutcomeUnique   = "unique"
	OutcomeExact    = "exact"
	OutcomeWaiting  = "waiting"
	OutcomeSymlink  = "symlink"
	OutcomeDataless = "dataless"
)

// ProgressEvent is one step of a scan that is still running.
type ProgressEvent struct {
	Kind     string
	Found    int
	Path     string
	Size     int64
	Stage    HashStage
	Finished bool
	Outcome  string
	Done     int
	Total    int
}

// FileProgress is one path in the queue or the processed list.
type FileProgress struct {
	Path    string
	Size    int64
	Outcome string
}

// ScanView is the scan portion of the window, built only from progress events.
type ScanView struct {
	Listing      bool
	Found        int
	Queue        []FileProgress
	Processed    []FileProgress
	CurrentPath  string
	CurrentSize  int64
	Stage        HashStage
	Done         int
	Total        int
	SimilarDone  int
	SimilarTotal int
}

// Apply folds one progress event into the view.
func (v *ScanView) Apply(ev ProgressEvent) {
	switch ev.Kind {
	case EventListing:
		v.Listing = ev.Found == 0 || (v.Total == 0 && ev.Found >= 0 && len(v.Queue) == 0 && v.Done == 0)
		v.Found = ev.Found
		if ev.Found == 0 {
			v.Listing = true
		}
	case EventQueue:
		v.Listing = false
		v.Queue = append(v.Queue, FileProgress{Path: ev.Path, Size: ev.Size})
		v.Total++
	case EventHashing:
		v.Listing = false
		v.CurrentPath = ev.Path
		v.CurrentSize = ev.Size
		v.Stage = ev.Stage
		if ev.Finished {
			v.finish(ev.Path, ev.Size, "")
		}
	case EventProcessed:
		v.Listing = false
		v.finish(ev.Path, ev.Size, ev.Outcome)
	case EventSimilar:
		v.SimilarDone = ev.Done
		v.SimilarTotal = ev.Total
	}
}

func (v *ScanView) finish(path string, size int64, outcome string) {
	if outcome == "" {
		outcome = v.outcome(path)
	}
	queued := false
	for i, row := range v.Queue {
		if row.Path == path {
			v.Queue = append(v.Queue[:i], v.Queue[i+1:]...)
			queued = true
			break
		}
	}
	for i, row := range v.Processed {
		if row.Path == path {
			if outcome != "" {
				v.Processed[i].Outcome = outcome
			}
			if size != 0 {
				v.Processed[i].Size = size
			}
			return
		}
	}
	v.Processed = append(v.Processed, FileProgress{Path: path, Size: size, Outcome: outcome})
	if queued {
		v.Done++
	}
}

func (v *ScanView) outcome(path string) string {
	for _, row := range v.Processed {
		if row.Path == path {
			return row.Outcome
		}
	}
	return ""
}

// Fraction is the share of files finished. The bool is false while the
// folder is still being listed and the total is unknown.
func (v ScanView) Fraction() (float32, bool) {
	if v.Listing && v.Total == 0 {
		return 0, false
	}
	if v.Total == 0 {
		return 1, true
	}
	done := v.Done
	if done > v.Total {
		done = v.Total
	}
	return float32(done) / float32(v.Total), true
}

// Status is the line that names the file being read.
func (v ScanView) Status() string {
	if v.CurrentPath == "" {
		if v.Listing {
			return ""
		}
		return ""
	}
	return v.CurrentPath
}
