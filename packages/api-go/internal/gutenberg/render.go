package gutenberg

import (
	"bytes"
	"context"
	"fmt"
	"hash/fnv"
	"html"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"time"
)

// Renderer prints HTML to PDF and PNG with headless Chrome. The reader in the
// app only opens PDFs, and Gutenberg doesn't publish them; Chrome lays out
// their HTML (images, tables, poetry indents) better than any converter we
// could vendor in Go.
type Renderer struct {
	Chrome  string
	Timeout time.Duration
}

// FindChrome is the Chrome to use: $CHROME_PATH, then the usual places on a
// Mac, then whatever is on PATH. "" if there is none.
func FindChrome() string {
	if p := os.Getenv("CHROME_PATH"); p != "" {
		return p
	}
	for _, p := range []string{
		"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		"/Applications/Chromium.app/Contents/MacOS/Chromium",
	} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	for _, name := range []string{"google-chrome", "chromium", "chromium-browser", "chrome", "headless_shell", "chrome-headless-shell"} {
		if p, err := exec.LookPath(name); err == nil {
			return p
		}
	}
	return ""
}

// PDF prints the HTML file at htmlPath to outPath.
func (r Renderer) PDF(ctx context.Context, htmlPath, outPath string) error {
	return r.run(ctx, htmlPath, outPath,
		"--print-to-pdf="+outPath,
		"--no-pdf-header-footer",
		"--print-to-pdf-no-header", // the same, for Chrome before 120
	)
}

// Cover draws our own cover for b to outPath as a 1000×1500 PNG. Gutenberg's
// cover images are sometimes a scan of a modern edition's cover, which isn't
// free to use, so every imported book gets this instead.
func (r Renderer) Cover(ctx context.Context, b Book, outPath string) error {
	dir, err := os.MkdirTemp("", "cover-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	page := filepath.Join(dir, "cover.html")
	if err := os.WriteFile(page, []byte(CoverHTML(b)), 0o600); err != nil {
		return err
	}
	return r.run(ctx, page, outPath,
		"--screenshot="+outPath,
		"--window-size=1000,1500",
		"--hide-scrollbars",
		"--force-device-scale-factor=1",
	)
}

// Cover colours: deep, quiet tones that keep white type readable. A book
// always gets the same one.
var coverColors = []string{"#23395B", "#3E5641", "#6B2737", "#2F4858", "#5B3758", "#3D405B", "#7A4419", "#264653"}

// CoverHTML is the page Cover screenshots.
func CoverHTML(b Book) string {
	h := fnv.New32a()
	h.Write([]byte(strconv.Itoa(b.ID)))
	bg := coverColors[h.Sum32()%uint32(len(coverColors))]

	title := DisplayTitle(b.Title)
	size := 96
	switch n := len([]rune(title)); {
	case n > 90:
		size = 56
	case n > 50:
		size = 68
	case n > 25:
		size = 80
	}
	return fmt.Sprintf(`<!doctype html><html><head><meta charset="utf-8"><style>
html, body { margin: 0; width: 1000px; height: 1500px; overflow: hidden; }
body { background: %s; color: #FAF7F0; font-family: Georgia, "Times New Roman", serif;
       display: flex; flex-direction: column; justify-content: space-between; box-sizing: border-box; padding: 150px 110px 140px; }
.rule { width: 120px; height: 6px; background: #FAF7F0; opacity: .7; margin-bottom: 64px; }
h1 { font-size: %dpx; line-height: 1.12; font-weight: normal; margin: 0; }
p { font-size: 46px; margin: 0; letter-spacing: .02em; opacity: .9; }
</style></head><body><div><div class="rule"></div><h1>%s</h1></div><p>%s</p></body></html>`,
		bg, size, html.EscapeString(title), html.EscapeString(AuthorLine(b)))
}

// run loads page in a fresh headless Chrome with the network unreachable: the
// HTML comes from a download, and a book has no business fetching anything.
// Scripts can't be switched off (Chrome then prints nothing); Clean removes
// them instead. out is the file Chrome should write.
func (r Renderer) run(ctx context.Context, page, out string, extra ...string) error {
	if r.Chrome == "" {
		return fmt.Errorf("no Chrome found: install Google Chrome or set CHROME_PATH")
	}
	timeout := r.Timeout
	if timeout == 0 {
		timeout = 3 * time.Minute
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	profile, err := os.MkdirTemp("", "chrome-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(profile)

	args := []string{
		"--headless=new",
		"--disable-gpu",
		"--no-first-run",
		"--no-default-browser-check",
		"--disable-extensions",
		"--user-data-dir=" + profile,
		"--host-resolver-rules=MAP * ~NOTFOUND",
	}
	if os.Geteuid() == 0 {
		// Chrome refuses to sandbox as root, which only happens in containers.
		args = append(args, "--no-sandbox")
	}
	args = append(args, extra...)
	args = append(args, (&url.URL{Scheme: "file", Path: page}).String())

	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, r.Chrome, args...)
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("chrome: %w: %s", err, lastLines(stderr.String(), 5))
	}
	// Chrome exits 0 even when it gave up on the page.
	if info, err := os.Stat(out); err != nil || info.Size() == 0 {
		return fmt.Errorf("chrome wrote nothing: %s", lastLines(stderr.String(), 5))
	}
	return nil
}

var pageObject = regexp.MustCompile(`/Type\s*/Page[^s]`)

// PageCount counts the pages in a PDF Chrome wrote. Chrome doesn't compress
// its object table, so the page objects can be counted as text.
func PageCount(pdf []byte) int {
	return len(pageObject.FindAllIndex(pdf, -1))
}

func lastLines(s string, n int) string {
	lines := bytes.Split(bytes.TrimSpace([]byte(s)), []byte("\n"))
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return string(bytes.Join(lines, []byte(" | ")))
}
