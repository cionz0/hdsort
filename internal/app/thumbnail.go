package app

import (
	"bytes"
	"image"
	"image/jpeg"
	"io"
	"path/filepath"
	"strings"

	_ "image/gif" // register GIF for image.Decode
	_ "image/png" // register PNG for image.Decode
)

const (
	thumbMaxWidth = 320
	thumbMaxBytes = 1 << 20
)

// Thumbnail returns a preview for a JPEG, PNG, or GIF. An image wider than
// 320 pixels, or larger than 1 MiB, comes back as a JPEG no wider than 320.
// Other types report ok false.
func Thumbnail(path string, r io.Reader, size int64) ([]byte, bool, error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".jpg", ".jpeg", ".png", ".gif":
	default:
		return nil, false, nil
	}
	body, err := io.ReadAll(r)
	if err != nil {
		return nil, false, err
	}
	img, _, err := image.Decode(bytes.NewReader(body))
	if err != nil {
		return nil, false, nil
	}
	bounds := img.Bounds()
	if bounds.Dx() <= thumbMaxWidth && size <= thumbMaxBytes && len(body) <= thumbMaxBytes {
		return body, true, nil
	}
	scaled := scaleWidth(img, thumbMaxWidth)
	var out bytes.Buffer
	if err := jpeg.Encode(&out, scaled, &jpeg.Options{Quality: 80}); err != nil {
		return nil, false, err
	}
	return out.Bytes(), true, nil
}

func scaleWidth(src image.Image, maxW int) image.Image {
	b := src.Bounds()
	w := b.Dx()
	if w <= maxW {
		return src
	}
	h := b.Dy() * maxW / w
	if h < 1 {
		h = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, maxW, h))
	for y := 0; y < h; y++ {
		sy := b.Min.Y + y*b.Dy()/h
		for x := 0; x < maxW; x++ {
			sx := b.Min.X + x*w/maxW
			dst.Set(x, y, src.At(sx, sy))
		}
	}
	return dst
}
