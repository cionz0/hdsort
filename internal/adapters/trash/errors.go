package trash

import "errors"

// ErrEmptyPath is returned when a Trash adapter is given a blank path.
var ErrEmptyPath = errors.New("trash: empty path")
