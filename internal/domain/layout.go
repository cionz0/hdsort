package domain

// LayoutPolicy is the Strategy that proposes moves inside a scan root.
// DefaultLayout is the only policy. A second policy needs an ADR.
//
// Pattern: Strategy (Gamma, Helm, Johnson, Vlissides, 1994).
// See docs/patterns.md.
type LayoutPolicy func(root string, files []FileRef) ([]LayoutMove, error)

// DefaultLayout proposes no moves. A scan that only needed deduplication
// has an empty proposal list (ADR-0005).
func DefaultLayout(string, []FileRef) ([]LayoutMove, error) {
	return nil, nil
}
