//go:build !darwin && !windows

package filesystem

import "io/fs"

func (osProbe) Dataless(string, fs.FileInfo) bool { return false }
