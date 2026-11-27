//go:build ignore

// Render writes the hdsort icon at every size the desktop ports need.
// Run from the repository root: go run assets/icon/render.go
package main

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
)

const canvas = 1024

var (
	ink   = color.NRGBA{R: 18, G: 72, B: 64, A: 255}
	paper = color.NRGBA{R: 244, G: 241, B: 230, A: 255}
	back  = color.NRGBA{R: 168, G: 196, B: 184, A: 255}
	line  = color.NRGBA{R: 18, G: 72, B: 64, A: 255}
)

func main() {
	full := draw(canvas)
	dir := filepath.Dir(os.Args[0])
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	if err := writePNG(filepath.Join(dir, "icon.png"), full); err != nil {
		panic(err)
	}
	set := filepath.Join(dir, "icon.iconset")
	if err := os.RemoveAll(set); err != nil {
		panic(err)
	}
	if err := os.MkdirAll(set, 0o755); err != nil {
		panic(err)
	}
	sizes := []struct {
		name string
		px   int
	}{
		{"icon_16x16.png", 16},
		{"icon_16x16@2x.png", 32},
		{"icon_32x32.png", 32},
		{"icon_32x32@2x.png", 64},
		{"icon_128x128.png", 128},
		{"icon_128x128@2x.png", 256},
		{"icon_256x256.png", 256},
		{"icon_256x256@2x.png", 512},
		{"icon_512x512.png", 512},
		{"icon_512x512@2x.png", 1024},
	}
	for _, size := range sizes {
		if err := writePNG(filepath.Join(set, size.name), scale(full, size.px)); err != nil {
			panic(err)
		}
	}
	if err := writeICO(filepath.Join(dir, "icon.ico"), full, []int{16, 32, 48, 256}); err != nil {
		panic(err)
	}
}

func draw(n int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, n, n))
	roundRect(img, 0, 0, n, n, int(float64(n)*0.22), ink)
	roundRect(img, scaleN(300, n), scaleN(188, n), scaleN(470, n), scaleN(560, n), scaleN(40, n), back)
	fx, fy := scaleN(214, n), scaleN(276, n)
	fw, fh := scaleN(470, n), scaleN(560, n)
	roundRect(img, fx, fy, fw, fh, scaleN(40, n), paper)
	fold := scaleN(92, n)
	for y := 0; y < fold; y++ {
		for x := 0; x < fold-y; x++ {
			set(img, fx+fw-fold+x, fy+y, back)
		}
	}
	lines := []struct{ y, w int }{
		{460, 300},
		{540, 210},
		{620, 260},
	}
	for _, item := range lines {
		roundRect(img, scaleN(268, n), scaleN(item.y, n), scaleN(item.w, n), scaleN(28, n), scaleN(14, n), line)
	}
	return img
}

func scaleN(v, n int) int {
	return v * n / canvas
}

func roundRect(img *image.NRGBA, x, y, w, h, r int, c color.NRGBA) {
	if r < 1 {
		r = 1
	}
	for py := 0; py < h; py++ {
		for px := 0; px < w; px++ {
			if insideRound(px, py, w, h, r) {
				set(img, x+px, y+py, c)
			}
		}
	}
}

func insideRound(px, py, w, h, r int) bool {
	cx, cy := px, py
	if px < r {
		cx = r
	} else if px >= w-r {
		cx = w - r - 1
	}
	if py < r {
		cy = r
	} else if py >= h-r {
		cy = h - r - 1
	}
	dx := float64(px - cx)
	dy := float64(py - cy)
	return dx*dx+dy*dy <= float64(r*r)
}

func set(img *image.NRGBA, x, y int, c color.NRGBA) {
	if image.Pt(x, y).In(img.Bounds()) {
		img.SetNRGBA(x, y, c)
	}
}

func scale(src *image.NRGBA, n int) *image.NRGBA {
	dst := image.NewNRGBA(image.Rect(0, 0, n, n))
	sb := src.Bounds()
	sw, sh := sb.Dx(), sb.Dy()
	for y := 0; y < n; y++ {
		y0 := y * sh / n
		y1 := (y + 1) * sh / n
		if y1 <= y0 {
			y1 = y0 + 1
		}
		for x := 0; x < n; x++ {
			x0 := x * sw / n
			x1 := (x + 1) * sw / n
			if x1 <= x0 {
				x1 = x0 + 1
			}
			var r, g, b, a, count float64
			for yy := y0; yy < y1; yy++ {
				for xx := x0; xx < x1; xx++ {
					c := src.NRGBAAt(sb.Min.X+xx, sb.Min.Y+yy)
					r += float64(c.R)
					g += float64(c.G)
					b += float64(c.B)
					a += float64(c.A)
					count++
				}
			}
			dst.SetNRGBA(x, y, color.NRGBA{
				R: uint8(math.Round(r / count)),
				G: uint8(math.Round(g / count)),
				B: uint8(math.Round(b / count)),
				A: uint8(math.Round(a / count)),
			})
		}
	}
	return dst
}

func writePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

func writeICO(path string, full *image.NRGBA, sizes []int) error {
	var blobs [][]byte
	for _, size := range sizes {
		var buf bytesBuffer
		if err := png.Encode(&buf, scale(full, size)); err != nil {
			return err
		}
		blobs = append(blobs, buf.b)
	}
	out := make([]byte, 6)
	out[2] = 1
	out[4] = byte(len(blobs))
	out[5] = byte(len(blobs) >> 8)
	offset := 6 + 16*len(blobs)
	for i, blob := range blobs {
		entry := make([]byte, 16)
		dim := sizes[i]
		if dim >= 256 {
			dim = 0
		}
		entry[0] = byte(dim)
		entry[1] = byte(dim)
		entry[4] = 1
		entry[6] = 32
		n := len(blob)
		entry[8] = byte(n)
		entry[9] = byte(n >> 8)
		entry[10] = byte(n >> 16)
		entry[11] = byte(n >> 24)
		entry[12] = byte(offset)
		entry[13] = byte(offset >> 8)
		entry[14] = byte(offset >> 16)
		entry[15] = byte(offset >> 24)
		out = append(out, entry...)
		offset += n
	}
	for _, blob := range blobs {
		out = append(out, blob...)
	}
	return os.WriteFile(path, out, 0o644)
}

type bytesBuffer struct{ b []byte }

func (b *bytesBuffer) Write(p []byte) (int, error) {
	b.b = append(b.b, p...)
	return len(p), nil
}
