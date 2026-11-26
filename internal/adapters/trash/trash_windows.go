//go:build windows

package trash

import (
	"fmt"
	"path/filepath"
	"syscall"
	"unsafe"

	"github.com/cionz0/hdsort/internal/app"
)

const (
	foDelete          = 0x0003
	fofAllowUndo      = 0x0040
	fofNoConfirmation = 0x0010
	fofSilent         = 0x0004
	fofNoErrorUI      = 0x0400
)

// shFileOpStruct matches SHFILEOPSTRUCTW. Field order lets the compiler
// insert the padding the 32-bit and 64-bit Windows ABIs expect.
type shFileOpStruct struct {
	hwnd                  uintptr
	wFunc                 uint32
	pFrom                 *uint16
	pTo                   *uint16
	fFlags                uint16
	fAnyOperationsAborted int32
	hNameMappings         uintptr
	lpszProgressTitle     *uint16
}

var procSHFileOperation = syscall.NewLazyDLL("shell32.dll").NewProc("SHFileOperationW")

// Platform moves a file to the Windows Recycle Bin.
// go test does not call Trash; it checks Validate, which Trash calls first.
type Platform struct{}

// NewPlatform returns the Windows Trash adapter.
func NewPlatform() Platform { return Platform{} }

// Trash deletes path with FOF_ALLOWUNDO, which sends it to the Recycle Bin.
// An empty path is rejected before the API call. The file is not unlinked.
func (Platform) Trash(path string) error {
	if err := Validate(path); err != nil {
		return err
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	from, err := syscall.UTF16FromString(abs)
	if err != nil {
		return err
	}
	from = append(from, 0)
	op := shFileOpStruct{
		wFunc:  foDelete,
		pFrom:  &from[0],
		fFlags: fofAllowUndo | fofNoConfirmation | fofSilent | fofNoErrorUI,
	}
	r, _, _ := procSHFileOperation.Call(uintptr(unsafe.Pointer(&op)))
	if r != 0 {
		return fmt.Errorf("trash: recycle bin: code %d", r)
	}
	return nil
}

var _ app.Trasher = Platform{}
