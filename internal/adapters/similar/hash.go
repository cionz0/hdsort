package similar

import (
	"hash/fnv"
	"image"
	"image/color"
	"strings"
	"unicode"
)

// averageHash is Buchner's average_hash: 64 samples, bit set when the
// sample is strictly greater than their mean. See ADR-0010.
func averageHash(img image.Image) uint64 {
	const n = 8
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}
	var samples [n * n]float64
	var sum float64
	for ty := 0; ty < n; ty++ {
		y0 := bounds.Min.Y + ty*height/n
		y1 := bounds.Min.Y + (ty+1)*height/n
		if y1 <= y0 {
			y1 = y0 + 1
		}
		for tx := 0; tx < n; tx++ {
			x0 := bounds.Min.X + tx*width/n
			x1 := bounds.Min.X + (tx+1)*width/n
			if x1 <= x0 {
				x1 = x0 + 1
			}
			var acc float64
			var count int
			for y := y0; y < y1 && y < bounds.Max.Y; y++ {
				for x := x0; x < x1 && x < bounds.Max.X; x++ {
					acc += luminance(img.At(x, y))
					count++
				}
			}
			if count == 0 {
				count = 1
			}
			value := acc / float64(count)
			samples[ty*n+tx] = value
			sum += value
		}
	}
	mean := sum / float64(n*n)
	var hash uint64
	for i, sample := range samples {
		if sample > mean {
			hash |= 1 << uint(i)
		}
	}
	return hash
}

func luminance(c color.Color) float64 {
	r, g, b, _ := c.RGBA()
	return 0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)
}

// simhash is Charikar's 64-bit fingerprint. Each word's FNV-1a-64 bits
// vote with that word's count. A bit is 1 only when its sum is positive.
// See ADR-0010.
func simhash(text string) uint64 {
	counts := tokens(text)
	var sums [64]int
	for word, weight := range counts {
		h := fnv.New64a()
		_, _ = h.Write([]byte(word))
		sum := h.Sum64()
		for bit := 0; bit < 64; bit++ {
			if sum&(1<<uint(bit)) != 0 {
				sums[bit] += weight
			} else {
				sums[bit] -= weight
			}
		}
	}
	var hash uint64
	for bit, sum := range sums {
		if sum > 0 {
			hash |= 1 << uint(bit)
		}
	}
	return hash
}

func tokens(text string) map[string]int {
	counts := map[string]int{}
	var word strings.Builder
	flush := func() {
		if word.Len() == 0 {
			return
		}
		counts[strings.ToLower(word.String())]++
		word.Reset()
	}
	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			word.WriteRune(r)
			continue
		}
		flush()
	}
	flush()
	return counts
}
