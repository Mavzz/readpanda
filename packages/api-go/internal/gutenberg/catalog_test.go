package gutenberg

import (
	"archive/tar"
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func parseFile(t *testing.T, name string) Book {
	t.Helper()
	f, err := os.Open("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	b, ok, err := ParseRDF(f)
	if err != nil || !ok {
		t.Fatalf("ParseRDF(%s): ok=%v err=%v", name, ok, err)
	}
	return b
}

// pg2554.rdf is the catalogue's real record for Crime and Punishment.
func TestParseRDF(t *testing.T) {
	b := parseFile(t, "pg2554.rdf")
	if b.ID != 2554 || b.Title != "Crime and Punishment" || b.Type != "Text" || b.Rights != "Public domain in the USA." {
		t.Errorf("got %+v", b)
	}
	if b.Downloads <= 0 {
		t.Errorf("Downloads = %d", b.Downloads)
	}
	if !reflect.DeepEqual(b.Languages, []string{"en"}) || !reflect.DeepEqual(b.LoCC, []string{"PG"}) {
		t.Errorf("Languages = %v, LoCC = %v", b.Languages, b.LoCC)
	}
	want := []Contributor{
		{Name: "Dostoyevsky, Fyodor", Role: "author", Birth: 1821, Death: 1881},
		{Name: "Garnett, Constance", Role: "translator", Birth: 1861, Death: 1946},
	}
	if !reflect.DeepEqual(b.Contributors, want) {
		t.Errorf("Contributors = %+v", b.Contributors)
	}
	if !strings.HasSuffix(b.HTMLZip, "/2554/pg2554-h.zip") {
		t.Errorf("HTMLZip = %q", b.HTMLZip)
	}
	if !strings.Contains(strings.Join(b.Bookshelves, "|"), "Category: Crime, Thrillers and Mystery") {
		t.Errorf("Bookshelves = %v", b.Bookshelves)
	}
	if !strings.Contains(b.Summary, "Raskolnikov") || strings.Contains(Description(b), "automatically generated") {
		t.Errorf("Summary = %q, Description = %q", b.Summary, Description(b))
	}
}

func TestParseRDFSubtitle(t *testing.T) {
	src := `<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#" xmlns:pgterms="http://www.gutenberg.org/2009/pgterms/" xmlns:dcterms="http://purl.org/dc/terms/">
<pgterms:ebook rdf:about="ebooks/84"><dcterms:title>Frankenstein
Or, The Modern Prometheus</dcterms:title></pgterms:ebook></rdf:RDF>`
	b, _, err := ParseRDF(strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	if b.Title != "Frankenstein: Or, The Modern Prometheus" {
		t.Errorf("Title = %q", b.Title)
	}
}

func TestReadCatalogTar(t *testing.T) {
	rdf, err := os.ReadFile("testdata/pg2554.rdf")
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	add := func(name string, data []byte) {
		tw.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(data)), Typeflag: tar.TypeReg})
		tw.Write(data)
	}
	add("cache/epub/2554/pg2554.rdf", rdf)
	add("cache/epub/9/pg9.rdf", []byte("not xml"))
	add("cache/epub/0/README", []byte("hello"))
	tw.Close()

	path := filepath.Join(t.TempDir(), "rdf-files.tar")
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	var ids []int
	var warned []string
	err = ReadCatalog(path, func(b Book) error { ids = append(ids, b.ID); return nil },
		func(name string, err error) { warned = append(warned, name) })
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(ids, []int{2554}) || !reflect.DeepEqual(warned, []string{"cache/epub/9/pg9.rdf"}) {
		t.Errorf("ids = %v, warned = %v", ids, warned)
	}
}

// Jekyll and Hyde's record lists /files/43/43-h.zip, which is gone, before
// /cache/epub/43/pg43-h.zip, which isn't.
func TestParseRDFPrefersCurrentZip(t *testing.T) {
	src := `<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#" xmlns:pgterms="http://www.gutenberg.org/2009/pgterms/" xmlns:dcterms="http://purl.org/dc/terms/">
<pgterms:ebook rdf:about="ebooks/43">
<dcterms:hasFormat><pgterms:file rdf:about="https://www.gutenberg.org/files/43/43-h.zip"/></dcterms:hasFormat>
<dcterms:hasFormat><pgterms:file rdf:about="https://www.gutenberg.org/cache/epub/43/pg43-h.zip"/></dcterms:hasFormat>
<dcterms:hasFormat><pgterms:file rdf:about="https://www.gutenberg.org/files/43/43-h/43-h.htm"/></dcterms:hasFormat>
</pgterms:ebook></rdf:RDF>`
	b, _, err := ParseRDF(strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	if b.HTMLZip != "https://www.gutenberg.org/cache/epub/43/pg43-h.zip" {
		t.Errorf("HTMLZip = %q", b.HTMLZip)
	}
}
