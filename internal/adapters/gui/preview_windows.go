//go:build windows

package gui

import (
	"bytes"
	"image"
	"image/png"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	shell32                         = windows.NewLazySystemDLL("shell32.dll")
	procSHCreateItemFromParsingName = shell32.NewProc("SHCreateItemFromParsingName")
	gdi32                           = windows.NewLazySystemDLL("gdi32.dll")
	procGetDIBits                   = gdi32.NewProc("GetDIBits")
	procGetObject                   = gdi32.NewProc("GetObjectW")
	procDeleteObject                = gdi32.NewProc("DeleteObject")
	procCreateCompatibleDC          = gdi32.NewProc("CreateCompatibleDC")
	procDeleteDC                    = gdi32.NewProc("DeleteDC")
	user32                          = windows.NewLazySystemDLL("user32.dll")
	procGetDC                       = user32.NewProc("GetDC")
	procReleaseDC                   = user32.NewProc("ReleaseDC")
)

// IShellItemImageFactory. https://learn.microsoft.com/en-us/windows/win32/api/shobjidl_core/nn-shobjidl_core-ishellitemimagefactory
var iidShellItemImageFactory = windows.GUID{
	Data1: 0xbcc18b79,
	Data2: 0xba16,
	Data3: 0x442f,
	Data4: [8]byte{0x80, 0xc4, 0x8a, 0x59, 0xc3, 0x0c, 0x46, 0x3b},
}

const (
	siigbfBiggerSizeOK  = 0x1
	siigbfThumbnailOnly = 0x8
)

type shellImageFactory struct {
	vtbl *shellImageFactoryVtbl
}

type shellImageFactoryVtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr
	GetImage       uintptr
}

type bitmapInfoHeader struct {
	Size          uint32
	Width         int32
	Height        int32
	Planes        uint16
	BitCount      uint16
	Compression   uint32
	SizeImage     uint32
	XPelsPerMeter int32
	YPelsPerMeter int32
	ClrUsed       uint32
	ClrImportant  uint32
}

type winBitmap struct {
	Type       int32
	Width      int32
	Height     int32
	WidthBytes int32
	Planes     uint16
	BitsPixel  uint16
	Bits       uintptr
}

func systemPreview(path string) ([]byte, bool, error) {
	_ = windows.CoInitializeEx(0, windows.COINIT_APARTMENTTHREADED)
	defer windows.CoUninitialize()
	pathPtr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil, false, nil
	}
	var factory *shellImageFactory
	hr, _, _ := procSHCreateItemFromParsingName.Call(
		uintptr(unsafe.Pointer(pathPtr)),
		0,
		uintptr(unsafe.Pointer(&iidShellItemImageFactory)),
		uintptr(unsafe.Pointer(&factory)),
	)
	if hr != 0 || factory == nil || factory.vtbl == nil {
		return nil, false, nil
	}
	defer syscallRelease(factory.vtbl.Release, unsafe.Pointer(factory))
	hbmp := factoryImage(factory, siigbfThumbnailOnly)
	if hbmp == 0 {
		hbmp = factoryImage(factory, siigbfBiggerSizeOK)
	}
	if hbmp == 0 {
		return nil, false, nil
	}
	defer procDeleteObject.Call(hbmp)
	body, ok := bitmapPNG(hbmp)
	return body, ok, nil
}

func factoryImage(factory *shellImageFactory, flags uintptr) uintptr {
	size := uintptr(320) | uintptr(240)<<32
	var hbmp uintptr
	hr, _, _ := syscall.SyscallN(factory.vtbl.GetImage, uintptr(unsafe.Pointer(factory)), size, flags, uintptr(unsafe.Pointer(&hbmp)))
	if hr != 0 {
		return 0
	}
	return hbmp
}

func bitmapPNG(hbmp uintptr) ([]byte, bool) {
	var info winBitmap
	if r, _, _ := procGetObject.Call(hbmp, unsafe.Sizeof(info), uintptr(unsafe.Pointer(&info))); r == 0 || info.Width <= 0 || info.Height <= 0 {
		return nil, false
	}
	hdc, _, _ := procGetDC.Call(0)
	if hdc == 0 {
		return nil, false
	}
	defer procReleaseDC.Call(0, hdc)
	mem, _, _ := procCreateCompatibleDC.Call(hdc)
	if mem == 0 {
		return nil, false
	}
	defer procDeleteDC.Call(mem)
	header := bitmapInfoHeader{
		Size:     uint32(unsafe.Sizeof(bitmapInfoHeader{})),
		Width:    info.Width,
		Height:   -info.Height,
		Planes:   1,
		BitCount: 32,
	}
	pixels := make([]byte, int(info.Width)*int(info.Height)*4)
	r, _, _ := procGetDIBits.Call(mem, hbmp, 0, uintptr(info.Height), uintptr(unsafe.Pointer(&pixels[0])), uintptr(unsafe.Pointer(&header)), 0)
	if r == 0 {
		return nil, false
	}
	img := image.NewNRGBA(image.Rect(0, 0, int(info.Width), int(info.Height)))
	for i := 0; i < len(pixels) && i+3 < len(pixels); i += 4 {
		img.Pix[i] = pixels[i+2]
		img.Pix[i+1] = pixels[i+1]
		img.Pix[i+2] = pixels[i]
		img.Pix[i+3] = 255
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, false
	}
	return buf.Bytes(), true
}

func syscallRelease(proc uintptr, this unsafe.Pointer) {
	_, _, _ = syscall.SyscallN(proc, uintptr(this))
}
