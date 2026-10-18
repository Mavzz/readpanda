package gutenberg

import (
	"archive/zip"
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestMirrorPath(t *testing.T) {
	for id, want := range map[int]string{1342: "1/3/4/1342", 84: "8/84", 5: "0/5", 10001: "1/0/0/0/10001"} {
		if got := MirrorPath(id); got != want {
			t.Errorf("MirrorPath(%d) = %q, want %q", id, got, want)
		}
	}
}

func zipOf(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		w.Write([]byte(body))
	}
	zw.Close()
	return buf.Bytes()
}

func TestUnpack(t *testing.T) {
	dir := t.TempDir()
	data := zipOf(t, map[string]string{
		"1342-h/1342-h.htm":       strings.Repeat("book ", 100),
		"1342-h/toc.html":         "short",
		"1342-h/images/i_001.jpg": "jpeg",
	})
	main, err := Unpack(data, dir)
	if err != nil {
		t.Fatal(err)
	}
	want, _ := filepath.Abs(filepath.Join(dir, "1342-h", "1342-h.htm"))
	if main != want {
		t.Errorf("main = %q, want %q", main, want)
	}
}

func TestUnpackRefusesEscapes(t *testing.T) {
	for _, name := range []string{"../evil.html", "a/../../evil.html"} {
		data := zipOf(t, map[string]string{name: "x"})
		if _, err := Unpack(data, t.TempDir()); err == nil {
			t.Errorf("%s: unpacked outside the directory", name)
		}
	}
}
