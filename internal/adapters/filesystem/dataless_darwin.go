//go:build darwin

package filesystem

import (
	"io/fs"
	"syscall"
)

// sfDataless is SF_DATALESS from Apple's sys/stat.h: the file is a dataless object.
const sfDataless uint32 = 0x40000000

func (osProbe) Dataless(_ string, info fs.FileInfo) bool {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat == nil {
		return false
	}
	return stat.Flags&sfDataless != 0
}
