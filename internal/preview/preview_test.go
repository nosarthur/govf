package preview

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestIsBinary(t *testing.T) {
	if IsBinary([]byte("hello\nworld\n")) {
		t.Error("text flagged binary")
	}
	if !IsBinary([]byte("ab\x00cd")) {
		t.Error("NUL not binary")
	}
	if IsBinary([]byte("héllo wörld ✓")) {
		t.Error("utf8 flagged binary")
	}
	if !IsBinary([]byte{0xff, 0xfe, 0xfd, 0xfc, 0xfb, 0xfa, 'a'}) {
		t.Error("garbage not binary")
	}
}

func TestExpand(t *testing.T) {
	if got := Expand("a\tb\r", 4); got != "a   b" {
		t.Errorf("got %q", got)
	}
	if got := Expand("abcd\te", 4); got != "abcd    e" {
		t.Errorf("got %q", got)
	}
}

func TestText(t *testing.T) {
	d := t.TempDir()
	p := filepath.Join(d, "f.txt")
	os.WriteFile(p, []byte("l1\nl2\nl3\n"), 0o644)
	r := Text(p, 2)
	if r.Kind != KindText || len(r.Lines) != 2 || r.Lines[1] != "l2" {
		t.Errorf("got %+v", r)
	}
	b := filepath.Join(d, "b.bin")
	os.WriteFile(b, []byte{1, 2, 0, 3}, 0o644)
	if r := Text(b, 5); r.Kind != KindBinary {
		t.Errorf("binary kind: %v", r.Kind)
	}
	if r := Text(filepath.Join(d, "nope"), 5); r.Kind != KindError {
		t.Errorf("missing kind: %v", r.Kind)
	}
}

func TestRenderAndImage(t *testing.T) {
	// 4x4: top half red, bottom half blue
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			c := color.RGBA{255, 0, 0, 255}
			if y >= 2 {
				c = color.RGBA{0, 0, 255, 255}
			}
			img.Set(x, y, c)
		}
	}
	cells := Render(img, 4, 2)
	if len(cells) != 2 || len(cells[0]) != 4 {
		t.Fatalf("dims %dx%d", len(cells), len(cells[0]))
	}
	if cells[0][0].FG != 0xff0000 || cells[1][3].BG != 0x0000ff {
		t.Errorf("colors: %+v", cells)
	}
	// aspect: 40x10 image into 10x10 cells -> 10 cols, 2px tall = 1 row
	wide := image.NewRGBA(image.Rect(0, 0, 40, 10))
	c2 := Render(wide, 10, 10)
	if len(c2) != 1 || len(c2[0]) != 10 {
		t.Errorf("aspect dims %dx%d", len(c2), len(c2[0]))
	}
	// via file
	d := t.TempDir()
	p := filepath.Join(d, "i.png")
	f, _ := os.Create(p)
	png.Encode(f, img)
	f.Close()
	r := File(p, 4, 2)
	if r.Kind != KindImage || len(r.Cells) != 2 {
		t.Errorf("file image: %+v", r)
	}
	if !IsImagePath("x.JPG") || IsImagePath("x.txt") {
		t.Error("IsImagePath")
	}
	// bad image with image ext falls back to text/binary path
	bad := filepath.Join(d, "bad.png")
	os.WriteFile(bad, []byte("not an image"), 0o644)
	if r := File(bad, 4, 2); r.Kind != KindText {
		t.Errorf("bad png kind: %v", r.Kind)
	}
}
