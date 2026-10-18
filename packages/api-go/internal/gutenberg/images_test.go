package gutenberg

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestShrinkImages(t *testing.T) {
	dir := t.TempDir()
	// A noisy 2400px scan: heavy at quality 100, like Gutenberg's.
	img := image.NewRGBA(image.Rect(0, 0, 2400, 1600))
	for y := 0; y < 1600; y++ {
		for x := 0; x < 2400; x++ {
			v := uint8((x*7 + y*13 + x*y) % 251)
			img.Set(x, y, color.RGBA{v, v, v, 255})
		}
	}
	var buf bytes.Buffer
	jpeg.Encode(&buf, img, &jpeg.Options{Quality: 100})
	p := filepath.Join(dir, "images", "i_001.jpg")
	os.MkdirAll(filepath.Dir(p), 0o755)
	os.WriteFile(p, buf.Bytes(), 0o644)

	saved, err := ShrinkImages(dir, 1000, 70)
	if err != nil {
		t.Fatal(err)
	}
	out, _ := os.ReadFile(p)
	got, err := jpeg.DecodeConfig(bytes.NewReader(out))
	if err != nil {
		t.Fatal(err)
	}
	if got.Width != 1000 || got.Height != 666 || saved <= 0 || len(out) >= buf.Len() {
		t.Errorf("got %dx%d, %d bytes (was %d), saved %d", got.Width, got.Height, len(out), buf.Len(), saved)
	}
}

func TestWithoutImages(t *testing.T) {
	src := `<html><head></head><body><p>Text.</p><figure><img src="a.jpg"><figcaption>"Edmond!"</figcaption></figure><p><img src="b.jpg"> More.</p></body></html>`
	out, err := WithoutImages([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if strings.Contains(s, "<img") || strings.Contains(s, "Edmond") || !strings.Contains(s, "Text.") || !strings.Contains(s, "More.") {
		t.Errorf("got %s", s)
	}
}
