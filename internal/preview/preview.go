// Package preview: renders text/image files into terminal-agnostic cells.
package preview

import (
	"bufio"
	"bytes"
	"image"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	_ "golang.org/x/image/bmp"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

// Kind of preview content.
type Kind int

const (
	KindNone Kind = iota
	KindText
	KindImage
	KindBinary
	KindError
)

// Cell is one image cell: upper-half block glyph, fg=top px, bg=bottom px.
type Cell struct {
	FG, BG uint32 // 0xRRGGBB
}

// Result of rendering a file.
type Result struct {
	Kind  Kind
	Lines []string // KindText
	Cells [][]Cell // KindImage, [row][col]
	Err   error
}

const (
	maxTextBytes = 256 << 10
	sniffBytes   = 8 << 10
)

var imageExts = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true,
	".bmp": true, ".tiff": true, ".tif": true, ".webp": true,
}

// IsImagePath: by extension.
func IsImagePath(p string) bool { return imageExts[strings.ToLower(filepath.Ext(p))] }

// IsBinary sniffs a sample: NUL byte or lots of invalid utf8 => binary.
func IsBinary(sample []byte) bool {
	if bytes.IndexByte(sample, 0) >= 0 {
		return true
	}
	if utf8.Valid(sample) {
		return false
	}
	// tolerate a truncated trailing rune
	bad := 0
	for len(sample) > 0 {
		r, n := utf8.DecodeRune(sample)
		if r == utf8.RuneError && n == 1 {
			bad++
		}
		sample = sample[n:]
	}
	return bad > 4
}

// File renders path for a cols x rows viewport.
func File(path string, cols, rows int) Result {
	if IsImagePath(path) {
		if r := Image(path, cols, rows); r.Kind == KindImage {
			return r
		}
	}
	return Text(path, rows)
}

// Text reads up to maxLines lines; detects binary.
func Text(path string, maxLines int) Result {
	f, err := os.Open(path)
	if err != nil {
		return Result{Kind: KindError, Err: err}
	}
	defer f.Close()
	br := bufio.NewReaderSize(f, sniffBytes)
	sample, _ := br.Peek(sniffBytes)
	if IsBinary(sample) {
		return Result{Kind: KindBinary}
	}
	lr := io.LimitReader(br, maxTextBytes)
	sc := bufio.NewScanner(lr)
	sc.Buffer(make([]byte, 64<<10), 1<<20)
	var lines []string
	for len(lines) < maxLines && sc.Scan() {
		lines = append(lines, Expand(sc.Text(), 4))
	}
	return Result{Kind: KindText, Lines: lines}
}

// Expand replaces tabs with spaces to next tab stop; drops CR.
func Expand(s string, tab int) string {
	s = strings.TrimSuffix(s, "\r")
	if !strings.ContainsRune(s, '\t') {
		return s
	}
	var b strings.Builder
	col := 0
	for _, r := range s {
		if r == '\t' {
			n := tab - col%tab
			b.WriteString(strings.Repeat(" ", n))
			col += n
			continue
		}
		b.WriteRune(r)
		col++
	}
	return b.String()
}

// Image decodes + fits into cols x rows cells (2 px rows per cell).
func Image(path string, cols, rows int) Result {
	f, err := os.Open(path)
	if err != nil {
		return Result{Kind: KindError, Err: err}
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return Result{Kind: KindError, Err: err}
	}
	return Result{Kind: KindImage, Cells: Render(img, cols, rows)}
}

// Render scales img to fit cols x (2*rows) px, keeping aspect, into cells.
func Render(img image.Image, cols, rows int) [][]Cell {
	if cols <= 0 || rows <= 0 {
		return nil
	}
	b := img.Bounds()
	iw, ih := b.Dx(), b.Dy()
	if iw == 0 || ih == 0 {
		return nil
	}
	maxW, maxH := cols, rows*2
	w, h := maxW, ih*maxW/iw
	if h > maxH {
		h = maxH
		w = iw * maxH / ih
	}
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, b, draw.Src, nil)
	nrows := (h + 1) / 2
	out := make([][]Cell, nrows)
	for y := 0; y < nrows; y++ {
		row := make([]Cell, w)
		for x := 0; x < w; x++ {
			row[x].FG = rgb(dst, x, 2*y)
			if 2*y+1 < h {
				row[x].BG = rgb(dst, x, 2*y+1)
			} else {
				row[x].BG = row[x].FG
			}
		}
		out[y] = row
	}
	return out
}

func rgb(m *image.RGBA, x, y int) uint32 {
	i := m.PixOffset(x, y)
	p := m.Pix[i : i+4 : i+4]
	// composite over black for alpha
	a := uint32(p[3])
	r := uint32(p[0]) * a / 255
	g := uint32(p[1]) * a / 255
	bl := uint32(p[2]) * a / 255
	return r<<16 | g<<8 | bl
}
