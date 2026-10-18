// Package gutenberg imports public-domain books from Project Gutenberg.
//
// It reads Gutenberg's RDF catalogue, decides which books ReadPanda can host
// in every country the app ships to (rights.go) and where they go in the
// library (genres.go), and turns a book's HTML edition into a PDF with
// Gutenberg's name taken out (clean.go, render.go). Their licence lets anyone
// use the texts freely, paid or not, once every mention of Project Gutenberg
// is gone; keeping the name means paying them royalties.
package gutenberg

import (
	"archive/tar"
	"compress/bzip2"
	"encoding/xml"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// CatalogURL is the RDF dump Gutenberg publishes for programs like this one,
// one file per book. The main site blocks scrapers; the feeds are the
// sanctioned way in.
const CatalogURL = "https://www.gutenberg.org/cache/epub/feeds/rdf-files.tar.bz2"

const marcrelNS = "http://id.loc.gov/vocabulary/relators/"

// Contributor is one person credited on a book. Death is 0 when the
// catalogue doesn't know it.
type Contributor struct {
	Name  string `json:"name"`
	Role  string `json:"role"`
	Birth int    `json:"birth,omitempty"`
	Death int    `json:"death,omitempty"`
}

// Book is one catalogue entry, reduced to what the import needs.
type Book struct {
	ID        int
	Title     string
	Type      string
	Rights    string
	Languages []string
	LoCC      []string
	Subjects  []string
	// Bookshelves include Gutenberg's own categories ("Category: Romance").
	Bookshelves  []string
	Summary      string
	Downloads    int
	Contributors []Contributor
	// HTMLZip is the catalogue's link to the HTML edition with images, or ""
	// when the book has none.
	HTMLZip string
}

// Authors are the contributors credited as authors.
func (b Book) Authors() []Contributor {
	var out []Contributor
	for _, c := range b.Contributors {
		if c.Role == "author" {
			out = append(out, c)
		}
	}
	return out
}

// Roles Gutenberg uses most, by MARC relator code. Anything else keeps its code.
var roleNames = map[string]string{
	"trl": "translator",
	"ill": "illustrator",
	"edt": "editor",
	"aui": "author of introduction",
	"aft": "author of afterword",
	"ann": "annotator",
	"com": "compiler",
	"cmm": "commentator",
	"ctb": "contributor",
	"adp": "adapter",
	"pht": "photographer",
	"oth": "other",
}

type rdfAgent struct {
	Name  string `xml:"name"`
	Birth string `xml:"birthdate"`
	Death string `xml:"deathdate"`
}

type rdfRole struct {
	XMLName xml.Name
	Agents  []rdfAgent `xml:"agent"`
}

type rdfDescription struct {
	MemberOf struct {
		Resource string `xml:"resource,attr"`
	} `xml:"memberOf"`
	Value string `xml:"value"`
}

type rdfFile struct {
	About string `xml:"about,attr"`
}

type rdfEbook struct {
	About       string           `xml:"about,attr"`
	Titles      []string         `xml:"title"`
	Rights      string           `xml:"rights"`
	Summary     string           `xml:"marc520"`
	Downloads   string           `xml:"downloads"`
	Languages   []string         `xml:"language>Description>value"`
	Types       []string         `xml:"type>Description>value"`
	Subjects    []rdfDescription `xml:"subject>Description"`
	Bookshelves []string         `xml:"bookshelf>Description>value"`
	Files       []rdfFile        `xml:"hasFormat>file"`
	Creators    []rdfAgent       `xml:"creator>agent"`
	// Everything else, which is where the marcrel:* roles (translator,
	// illustrator…) land, since their element names vary.
	Other []rdfRole `xml:",any"`
}

type rdfDoc struct {
	Ebooks []rdfEbook `xml:"ebook"`
}

// ParseRDF reads one book's RDF file. ok is false for files that describe no
// book.
func ParseRDF(r io.Reader) (b Book, ok bool, err error) {
	var doc rdfDoc
	if err := xml.NewDecoder(r).Decode(&doc); err != nil {
		return Book{}, false, err
	}
	if len(doc.Ebooks) == 0 {
		return Book{}, false, nil
	}
	e := doc.Ebooks[0]

	id, err := strconv.Atoi(strings.TrimPrefix(e.About, "ebooks/"))
	if err != nil {
		return Book{}, false, fmt.Errorf("unexpected ebook id %q", e.About)
	}
	b = Book{
		ID:          id,
		Rights:      strings.TrimSpace(e.Rights),
		Summary:     strings.TrimSpace(e.Summary),
		Languages:   trimAll(e.Languages),
		Bookshelves: trimAll(e.Bookshelves),
		Downloads:   atoi(e.Downloads),
	}
	if len(e.Titles) > 0 {
		// Subtitles come on a second line: "Frankenstein\nOr, The Modern Prometheus".
		var lines []string
		for _, l := range strings.Split(e.Titles[0], "\n") {
			if l = strings.Join(strings.Fields(l), " "); l != "" {
				lines = append(lines, l)
			}
		}
		b.Title = strings.Join(lines, ": ")
	}
	if len(e.Types) > 0 {
		b.Type = strings.TrimSpace(e.Types[0])
	}
	for _, s := range e.Subjects {
		v := strings.TrimSpace(s.Value)
		if strings.HasSuffix(s.MemberOf.Resource, "/LCC") {
			b.LoCC = append(b.LoCC, v)
		} else {
			b.Subjects = append(b.Subjects, v)
		}
	}
	for _, a := range e.Creators {
		b.Contributors = append(b.Contributors, contributor(a, "author"))
	}
	for _, o := range e.Other {
		if o.XMLName.Space != marcrelNS {
			continue
		}
		role := roleNames[o.XMLName.Local]
		if role == "" {
			role = o.XMLName.Local
		}
		for _, a := range o.Agents {
			b.Contributors = append(b.Contributors, contributor(a, role))
		}
	}
	// Older records list the HTML zip twice: under /files/, which Gutenberg
	// no longer serves, and under /cache/epub/, which it does.
	for _, f := range e.Files {
		if strings.HasSuffix(f.About, "-h.zip") && (b.HTMLZip == "" || strings.Contains(f.About, "/cache/epub/")) {
			b.HTMLZip = f.About
		}
	}
	return b, true, nil
}

func contributor(a rdfAgent, role string) Contributor {
	return Contributor{
		Name:  strings.TrimSpace(a.Name),
		Role:  role,
		Birth: atoi(a.Birth),
		Death: atoi(a.Death),
	}
}

// ReadCatalog calls fn for every book in the catalogue at path: the
// rdf-files.tar.bz2 download, or a directory it was unpacked into. Files that
// don't parse are reported to warn and skipped; one bad entry in 75,000
// shouldn't stop an import.
func ReadCatalog(path string, fn func(Book) error, warn func(name string, err error)) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	handle := func(name string, r io.Reader) error {
		b, ok, err := ParseRDF(r)
		if err != nil {
			warn(name, err)
			return nil
		}
		if !ok {
			return nil
		}
		return fn(b)
	}

	if info.IsDir() {
		return filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(p, ".rdf") {
				return err
			}
			f, err := os.Open(p)
			if err != nil {
				return err
			}
			defer f.Close()
			return handle(p, f)
		})
	}

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	var r io.Reader = f
	if strings.HasSuffix(path, ".bz2") {
		r = bzip2.NewReader(f)
	}
	tr := tar.NewReader(r)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("reading %s: %w", path, err)
		}
		if h.Typeflag != tar.TypeReg || !strings.HasSuffix(h.Name, ".rdf") {
			continue
		}
		if err := handle(h.Name, tr); err != nil {
			return err
		}
	}
}

func atoi(s string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(s))
	return n
}

func trimAll(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}
	return out
}
