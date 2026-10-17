package gutenberg

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Limits on what one book may download and unpack to. The biggest
// illustrated editions run to a few tens of MB.
const (
	maxDownload = 200 << 20
	maxUnpacked = 400 << 20
)

// Fetcher downloads books' HTML editions, politely: Gutenberg asks programs
// to use a mirror and to wait between requests.
type Fetcher struct {
	// Mirror is a Gutenberg mirror's base URL, laid out like the main
	// collection (/1/3/4/1342/1342-h.zip). Empty skips it.
	Mirror string
	Delay  time.Duration
	Client *http.Client

	last time.Time
}

// MirrorPath is where a mirror keeps a book's files: one directory per digit
// but the last, then the number. 1342 → 1/3/4/1342, 84 → 8/84, 5 → 0/5.
func MirrorPath(id int) string {
	s := strconv.Itoa(id)
	if len(s) == 1 {
		return "0/" + s
	}
	return strings.Join(strings.Split(s[:len(s)-1], ""), "/") + "/" + s
}

// HTMLZip downloads b's HTML edition: from the mirror if it has it, otherwise
// from the link in the catalogue.
func (f *Fetcher) HTMLZip(ctx context.Context, b Book) ([]byte, string, error) {
	var urls []string
	if f.Mirror != "" {
		urls = append(urls, fmt.Sprintf("%s/%s/%d-h.zip", strings.TrimRight(f.Mirror, "/"), MirrorPath(b.ID), b.ID))
	}
	if b.HTMLZip != "" {
		urls = append(urls, b.HTMLZip)
	}
	if len(urls) == 0 {
		return nil, "", errors.New("no HTML edition in the catalogue")
	}
	var errs []error
	for _, u := range urls {
		data, err := f.get(ctx, u)
		if err == nil {
			return data, u, nil
		}
		errs = append(errs, err)
	}
	return nil, "", errors.Join(errs...)
}

func (f *Fetcher) get(ctx context.Context, u string) ([]byte, error) {
	if wait := f.Delay - time.Since(f.last); wait > 0 {
		select {
		case <-time.After(wait):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	defer func() { f.last = time.Now() }()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "ReadPandaImporter/1.0")
	client := f.Client
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Minute}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: %s", u, resp.Status)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxDownload+1))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", u, err)
	}
	if len(data) > maxDownload {
		return nil, fmt.Errorf("%s: over %d MB", u, maxDownload>>20)
	}
	return data, nil
}

// Unpack extracts an HTML edition's zip into dir, which should be new and
// empty, and returns the path of the book's HTML file (the largest one, since
// some editions split off notes or a table of contents). Entries that would
// land outside dir, links, and archives past maxUnpacked are refused.
func Unpack(data []byte, dir string) (string, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("not a zip: %w", err)
	}
	root, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}

	var total int64
	var main string
	var mainSize int64
	for _, zf := range zr.File {
		if zf.FileInfo().IsDir() {
			continue
		}
		if !zf.Mode().IsRegular() {
			return "", fmt.Errorf("%s: not a regular file", zf.Name)
		}
		dest := filepath.Join(root, filepath.FromSlash(zf.Name))
		if !strings.HasPrefix(dest, root+string(os.PathSeparator)) {
			return "", fmt.Errorf("%s: outside the archive", zf.Name)
		}
		n, err := extract(zf, dest, maxUnpacked-total)
		if err != nil {
			return "", err
		}
		total += n

		ext := strings.ToLower(filepath.Ext(dest))
		if (ext == ".htm" || ext == ".html") && n > mainSize {
			main, mainSize = dest, n
		}
	}
	if main == "" {
		return "", errors.New("no HTML file in the archive")
	}
	return main, nil
}

func extract(zf *zip.File, dest string, budget int64) (int64, error) {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return 0, err
	}
	rc, err := zf.Open()
	if err != nil {
		return 0, err
	}
	defer rc.Close()
	out, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return 0, err
	}
	defer out.Close()
	n, err := io.Copy(out, io.LimitReader(rc, budget+1))
	if err != nil {
		return n, err
	}
	if n > budget {
		return n, fmt.Errorf("archive unpacks to over %d MB", maxUnpacked>>20)
	}
	return n, nil
}
