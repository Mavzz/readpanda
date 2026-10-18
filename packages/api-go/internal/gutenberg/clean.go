package gutenberg

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	"golang.org/x/net/html/charset"
)

var (
	startMarker = regexp.MustCompile(`(?i)\*\*\*\s*START OF (THE|THIS) PROJECT GUTENBERG[^*]*\*\*\*`)
	endMarker   = regexp.MustCompile(`(?i)\*\*\*\s*END OF (THE|THIS) PROJECT GUTENBERG[^*]*\*\*\*`)
	// Credit lines volunteers leave near the top ("Produced by … and the
	// Online Distributed Proofreading Team at https://www.pgdp.net") and
	// links back to the site.
	credit = regexp.MustCompile(`(?i)project gutenberg|gutenberg\.org|pgdp\.net|distributed proofread|^\s*produced by\b|^\s*e-?text prepared by\b|transcribed from the`)
	// Anything left after cleaning stops the book: their licence wants every
	// mention gone, and a person should look at what's left.
	mention = regexp.MustCompile(`(?i)gutenberg`)
)

// A credit is a short note; a block longer than this is the book's own text,
// even if it happens to match.
const maxCreditLen = 600

// A marker's own block is a line or two, or in older files the opening
// blurb ("This eBook is for the use of anyone anywhere…"). A block longer
// than this holds the book as well.
const maxMarkerBlockLen = 3000

// ErrNeedsReview means the HTML still names Gutenberg after cleaning, or
// doesn't have the markers cleaning relies on. The book is skipped, not
// published, so a person can look at it.
var ErrNeedsReview = errors.New("needs review")

// printCSS lays the book out for a phone-sized page. It goes after the
// book's own styles: Gutenberg's are written for a wide browser window
// (body margins of 10–20%), which wastes a small page.
const printCSS = `
@page { size: 5in 8in; margin: 0.6in 0.55in 0.7in; }
html { font-size: 11.5pt; }
body { margin: 0 !important; padding: 0 !important; max-width: none !important;
       font-family: Georgia, "Times New Roman", serif; line-height: 1.45; color: #111; background: #fff; }
img { max-width: 100%; height: auto; page-break-inside: avoid; }
h1, h2, h3, h4 { page-break-after: avoid; }
p { orphans: 2; widows: 2; }
.pagenum, .pageno, .pagenumber { display: none; }
`

// Clean returns the book's HTML edition without Gutenberg's header, licence,
// credit lines and links, retitled, as UTF-8 and with print styles. src is
// the HTML as downloaded, in whatever encoding it declares.
func Clean(src []byte, title string) ([]byte, error) {
	enc, _, _ := charset.DetermineEncoding(src, "text/html")
	utf8, err := enc.NewDecoder().Bytes(src)
	if err != nil {
		return nil, fmt.Errorf("decoding: %w", err)
	}
	doc, err := html.Parse(bytes.NewReader(utf8))
	if err != nil {
		return nil, fmt.Errorf("parsing: %w", err)
	}
	head, body := find(doc, atom.Head), find(doc, atom.Body)
	if head == nil || body == nil {
		return nil, fmt.Errorf("%w: no head or body", ErrNeedsReview)
	}

	start := findText(body, startMarker)
	end := findText(body, endMarker)
	if start == nil || end == nil {
		return nil, fmt.Errorf("%w: no START/END OF THE PROJECT GUTENBERG EBOOK markers", ErrNeedsReview)
	}
	cut(body, start, startMarker, true)
	cut(body, end, endMarker, false)

	removeCredits(body)
	removeAll(doc, atom.Script)
	removeComments(doc)
	unwrapLinks(body)
	trimLeading(body)
	rewriteHead(head, title)

	if m := firstMention(doc); m != "" {
		return nil, fmt.Errorf("%w: still mentions Gutenberg: %q", ErrNeedsReview, m)
	}

	var out bytes.Buffer
	if err := html.Render(&out, doc); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// cut removes a marker's block and everything before it (or, for the end
// marker, after it: the licence). When the marker shares a large block with
// the book, as in older files where the whole text can sit in one <pre> or
// wrapper <div>, only the text up to the end of the marker's line goes
// (from the start of it, for the end marker), and the rest of the block stays.
func cut(body, text *html.Node, re *regexp.Regexp, start bool) {
	block := boilerplate(text, body)
	if block == nil {
		if b := nearestBlock(text, body); len(textOf(b)) <= maxMarkerBlockLen {
			block = b
		}
	}
	if block != nil {
		removeSiblings(body, block, start)
		block.Parent.RemoveChild(block)
		return
	}
	loc := re.FindStringIndex(text.Data)
	if start {
		rest := text.Data[loc[1]:]
		if nl := strings.IndexByte(rest, '\n'); nl >= 0 {
			rest = rest[nl+1:]
		}
		text.Data = rest
	} else {
		text.Data = text.Data[:loc[0]]
	}
	removeSiblings(body, text, start)
}

// boilerplate is the header or footer section newer files wrap the markers
// and licence in (<section class="pg-boilerplate" id="pg-header">), or nil.
func boilerplate(n, body *html.Node) *html.Node {
	for p := n.Parent; p != nil && p != body; p = p.Parent {
		id, class := attr(p, "id"), attr(p, "class")
		if id == "pg-header" || id == "pg-footer" || strings.Contains(class, "pg-boilerplate") {
			return p
		}
	}
	return nil
}

var blocks = map[atom.Atom]bool{
	atom.P: true, atom.Div: true, atom.Pre: true, atom.Section: true, atom.Table: true,
	atom.Center: true, atom.Blockquote: true, atom.H1: true, atom.H2: true, atom.H3: true,
	atom.H4: true, atom.H5: true, atom.H6: true, atom.Hr: true,
}

func nearestBlock(n, body *html.Node) *html.Node {
	for p := n.Parent; p != nil && p != body; p = p.Parent {
		if p.Type == html.ElementNode && blocks[p.DataAtom] {
			return p
		}
	}
	return n
}

// removeSiblings removes everything before n (or after it) in document
// order, up to body: n's earlier siblings, then its parent's, and so on.
func removeSiblings(body, n *html.Node, before bool) {
	for cur := n; cur != nil && cur != body; cur = cur.Parent {
		for {
			s := cur.NextSibling
			if before {
				s = cur.PrevSibling
			}
			if s == nil {
				break
			}
			cur.Parent.RemoveChild(s)
		}
	}
}

// removeCredits drops short blocks that credit Gutenberg or its volunteers.
func removeCredits(body *html.Node) {
	var doomed []*html.Node
	walk(body, func(n *html.Node) bool {
		if n.Type == html.ElementNode && blocks[n.DataAtom] && n.DataAtom != atom.Section {
			t := strings.TrimSpace(textOf(n))
			if len(t) <= maxCreditLen && credit.MatchString(t) {
				doomed = append(doomed, n)
				return false
			}
		}
		return true
	})
	for _, n := range doomed {
		n.Parent.RemoveChild(n)
	}
}

// trimLeading drops what cutting the header leaves at the very start: rules
// and empty blocks that would otherwise print as a blank first page.
func trimLeading(body *html.Node) {
	for {
		n := firstContent(body)
		if n == nil || !blank(n) {
			return
		}
		n.Parent.RemoveChild(n)
	}
}

// firstContent is the first node at the start of the book that isn't
// whitespace, descending into wrappers whose first child it is.
func firstContent(n *html.Node) *html.Node {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.TextNode && strings.TrimSpace(c.Data) == "" {
			continue
		}
		if c.Type == html.ElementNode && !blank(c) && c.DataAtom != atom.Img && c.FirstChild != nil {
			if inner := firstContent(c); inner != nil && blank(inner) {
				return inner
			}
		}
		return c
	}
	return nil
}

// blank is a rule, a line break or an element with no text and no image.
func blank(n *html.Node) bool {
	if n.Type != html.ElementNode {
		return false
	}
	switch n.DataAtom {
	case atom.Hr, atom.Br:
		return true
	case atom.Img, atom.Svg, atom.Table:
		return false
	}
	if strings.TrimSpace(textOf(n)) != "" {
		return false
	}
	hasImage := false
	walk(n, func(c *html.Node) bool {
		if c.Type == html.ElementNode && (c.DataAtom == atom.Img || c.DataAtom == atom.Svg) {
			hasImage = true
		}
		return !hasImage
	})
	return !hasImage
}

// unwrapLinks turns links to Gutenberg into plain text; the text itself is
// left to removeCredits and the final check.
func unwrapLinks(body *html.Node) {
	var links []*html.Node
	walk(body, func(n *html.Node) bool {
		if n.Type == html.ElementNode && n.DataAtom == atom.A && mention.MatchString(attr(n, "href")) {
			links = append(links, n)
		}
		return true
	})
	for _, a := range links {
		for c := a.FirstChild; c != nil; c = a.FirstChild {
			a.RemoveChild(c)
			a.Parent.InsertBefore(c, a)
		}
		a.Parent.RemoveChild(a)
	}
}

// rewriteHead keeps the book's own stylesheets and drops the rest of the
// head — Gutenberg's title, meta tags (which name it) and links — then adds
// a UTF-8 charset, the title and the print styles.
func rewriteHead(head *html.Node, title string) {
	for c := head.FirstChild; c != nil; {
		next := c.NextSibling
		keep := c.Type == html.ElementNode && c.DataAtom == atom.Style
		if c.Type == html.ElementNode && c.DataAtom == atom.Link &&
			strings.EqualFold(attr(c, "rel"), "stylesheet") && !strings.Contains(attr(c, "href"), "//") {
			keep = true
		}
		if !keep {
			head.RemoveChild(c)
		}
		c = next
	}
	meta := &html.Node{Type: html.ElementNode, Data: "meta", DataAtom: atom.Meta,
		Attr: []html.Attribute{{Key: "charset", Val: "utf-8"}}}
	head.InsertBefore(meta, head.FirstChild)
	t := &html.Node{Type: html.ElementNode, Data: "title", DataAtom: atom.Title}
	t.AppendChild(&html.Node{Type: html.TextNode, Data: title})
	head.InsertBefore(t, meta.NextSibling)
	style := &html.Node{Type: html.ElementNode, Data: "style", DataAtom: atom.Style}
	style.AppendChild(&html.Node{Type: html.TextNode, Data: printCSS})
	head.AppendChild(style)
}

// firstMention is a little context around the first remaining mention of
// Gutenberg in the text, a link or an image's alt text; "" if none.
func firstMention(doc *html.Node) string {
	var found string
	walk(doc, func(n *html.Node) bool {
		if found != "" {
			return false
		}
		switch n.Type {
		case html.TextNode:
			if n.Parent != nil && n.Parent.DataAtom == atom.Style {
				return true
			}
			if loc := mention.FindStringIndex(n.Data); loc != nil {
				found = around(n.Data, loc)
			}
		case html.ElementNode:
			for _, a := range n.Attr {
				if loc := mention.FindStringIndex(a.Val); loc != nil {
					found = around(a.Key+"="+a.Val, loc)
				}
			}
		}
		return true
	})
	return found
}

func around(s string, loc []int) string {
	from, to := max(0, loc[0]-40), min(len(s), loc[1]+40)
	return strings.Join(strings.Fields(s[from:to]), " ")
}

// findText is the first text node matching re.
func findText(root *html.Node, re *regexp.Regexp) *html.Node {
	var found *html.Node
	walk(root, func(n *html.Node) bool {
		if found == nil && n.Type == html.TextNode && re.MatchString(n.Data) {
			found = n
		}
		return found == nil
	})
	return found
}

func find(root *html.Node, a atom.Atom) *html.Node {
	var found *html.Node
	walk(root, func(n *html.Node) bool {
		if found == nil && n.Type == html.ElementNode && n.DataAtom == a {
			found = n
		}
		return found == nil
	})
	return found
}

func removeAll(root *html.Node, a atom.Atom) {
	var doomed []*html.Node
	walk(root, func(n *html.Node) bool {
		if n.Type == html.ElementNode && n.DataAtom == a {
			doomed = append(doomed, n)
			return false
		}
		return true
	})
	for _, n := range doomed {
		n.Parent.RemoveChild(n)
	}
}

func removeComments(root *html.Node) {
	var doomed []*html.Node
	walk(root, func(n *html.Node) bool {
		if n.Type == html.CommentNode {
			doomed = append(doomed, n)
		}
		return true
	})
	for _, n := range doomed {
		n.Parent.RemoveChild(n)
	}
}

// walk visits n and its descendants in document order; returning false skips
// a node's children.
func walk(n *html.Node, fn func(*html.Node) bool) {
	if !fn(n) {
		return
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		walk(c, fn)
	}
}

func textOf(n *html.Node) string {
	var b strings.Builder
	walk(n, func(c *html.Node) bool {
		if c.Type == html.TextNode {
			b.WriteString(c.Data)
		}
		return true
	})
	return b.String()
}

func attr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}
