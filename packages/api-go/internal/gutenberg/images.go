package gutenberg

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// ShrinkImages recompresses an unpacked edition's JPEGs in place: no wider
// than maxWidth, at the given quality. Chrome embeds a JPEG in the PDF as it
// is, and some editions carry hundreds of heavy scans (The Count of Monte
// Cristo's 450 come to 100 MB). A file is only replaced when the new one is
// smaller. It returns the bytes saved.
func ShrinkImages(dir string, maxWidth, quality int) (saved int64, err error) {
	err = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		ext := strings.ToLower(filepath.Ext(p))
		if ext != ".jpg" && ext != ".jpeg" {
			return nil
		}
		src, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		img, err := jpeg.Decode(bytes.NewReader(src))
		if err != nil {
			return nil // left as it is; Chrome may still manage it
		}
		if img.Bounds().Dx() > maxWidth {
			img = downscale(img, maxWidth)
		}
		var out bytes.Buffer
		if err := jpeg.Encode(&out, img, &jpeg.Options{Quality: quality}); err != nil {
			return nil
		}
		if out.Len() >= len(src) {
			return nil
		}
		saved += int64(len(src) - out.Len())
		return os.WriteFile(p, out.Bytes(), 0o644)
	})
	return saved, err
}

// downscale shrinks img to width w by averaging the source pixels each
// destination pixel covers, which keeps line engravings from going jagged.
func downscale(img image.Image, w int) image.Image {
	b := img.Bounds()
	h := b.Dy() * w / b.Dx()
	if h < 1 {
		h = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		y0, y1 := b.Min.Y+y*b.Dy()/h, b.Min.Y+(y+1)*b.Dy()/h
		if y1 == y0 {
			y1 = y0 + 1
		}
		for x := 0; x < w; x++ {
			x0, x1 := b.Min.X+x*b.Dx()/w, b.Min.X+(x+1)*b.Dx()/w
			if x1 == x0 {
				x1 = x0 + 1
			}
			var r, g, bl, a, n uint64
			for sy := y0; sy < y1; sy++ {
				for sx := x0; sx < x1; sx++ {
					cr, cg, cb, ca := img.At(sx, sy).RGBA()
					r, g, bl, a, n = r+uint64(cr), g+uint64(cg), bl+uint64(cb), a+uint64(ca), n+1
				}
			}
			dst.Set(x, y, color.RGBA64{uint16(r / n), uint16(g / n), uint16(bl / n), uint16(a / n)})
		}
	}
	return dst
}

// WithoutImages takes the illustrations out of cleaned HTML, for a book that
// prints too large with them. Figures go whole, captions included, since a
// caption without its picture reads as a stray line.
func WithoutImages(src []byte) ([]byte, error) {
	doc, err := html.Parse(bytes.NewReader(src))
	if err != nil {
		return nil, err
	}
	for _, a := range []atom.Atom{atom.Figure, atom.Img, atom.Svg} {
		removeAll(doc, a)
	}
	var out bytes.Buffer
	if err := html.Render(&out, doc); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
