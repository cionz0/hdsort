//go:build windows

package filesystem

import (
	"io/fs"
	"syscall"
)

const (
	fileAttributeRecallOnDataAccess uint32 = 0x00400000
	fileAttributeRecallOnOpen       uint32 = 0x00040000
)

func (osProbe) Dataless(_ string, info fs.FileInfo) bool {
	stat, ok := info.Sys().(*syscall.Win32FileAttributeData)
	if !ok || stat == nil {
		return false
	}
	const mask = fileAttributeRecallOnDataAccess | fileAttributeRecallOnOpen
	return stat.FileAttributes&mask != 0
}
