package domain

import "fmt"

// Choice is the owner's confirmation for one path.
type Choice string

const (
	ChoiceKeep   Choice = "keep"
	ChoiceDelete Choice = "delete"
	ChoiceSkip   Choice = "skip"
)

// Decision is an explicit keep, delete, or skip for one path.
type Decision struct {
	Path   string
	Choice Choice
}

// NewDecision rejects a choice outside keep, delete, and skip.
func NewDecision(path string, choice Choice) (Decision, error) {
	d := Decision{Path: path, Choice: choice}
	if err := d.validate(); err != nil {
		return Decision{}, err
	}
	return d, nil
}

func (d Decision) validate() error {
	if d.Path == "" {
		return fmt.Errorf("%w: decision", ErrEmptyPath)
	}
	switch d.Choice {
	case ChoiceKeep, ChoiceDelete, ChoiceSkip:
		return nil
	default:
		return fmt.Errorf("%w: %q", ErrDecisionChoice, d.Choice)
	}
}
