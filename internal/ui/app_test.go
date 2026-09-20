package ui

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"

	"github.com/nosarthur/govf/internal/fsx"
)

func newTestApp(t *testing.T, dir string) (*App, tcell.SimulationScreen) {
	t.Helper()
	s := tcell.NewSimulationScreen("")
	if err := s.Init(); err != nil {
		t.Fatal(err)
	}
	s.SetSize(80, 24)
	a := New(s, dir, dir)
	return a, s
}

func keys(a *App, seq string) {
	for _, r := range seq {
		var ev *tcell.EventKey
		switch r {
		case '\n':
			ev = tcell.NewEventKey(tcell.KeyEnter, 0, 0)
		case '\t':
			ev = tcell.NewEventKey(tcell.KeyTab, 0, 0)
		case '\x1b':
			ev = tcell.NewEventKey(tcell.KeyEscape, 0, 0)
		default:
			ev = tcell.NewEventKey(tcell.KeyRune, r, 0)
		}
		a.handleKey(ev)
	}
}

func screenText(s tcell.SimulationScreen) string {
	cells, w, h := s.GetContents()
	var b strings.Builder
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := cells[y*w+x]
			if len(c.Runes) > 0 {
				b.WriteRune(c.Runes[0])
			} else {
				b.WriteByte(' ')
			}
		}
		b.WriteByte('\n')
	}
	return b.String()
}

func TestAppNavAndOps(t *testing.T) {
	d := setup(t)
	a, s := newTestApp(t, d)
	a.draw()
	out := screenText(s)
	for _, want := range []string{"dir1/", "alpha.txt", "sort:+name", "1/4"} {
		if !strings.Contains(out, want) {
			t.Errorf("screen missing %q:\n%s", want, out)
		}
	}
	// preview of dir1 on right side
	if !strings.Contains(out, "inner") {
		t.Errorf("dir preview missing:\n%s", out)
	}
	// text preview
	keys(a, "jj") // alpha.txt
	a.draw()
	if !strings.Contains(screenText(s), "alpha.txt") {
		t.Error("cursor move")
	}
	if a.cur().Current().Name != "alpha.txt" {
		t.Errorf("cur %s", a.cur().Current().Name)
	}
	// gg / G
	keys(a, "G")
	if a.cur().Current().Name != "beta.txt" {
		t.Error("G")
	}
	keys(a, "gg")
	if a.cur().Cur != 0 {
		t.Error("gg")
	}
	// enter dir, go up
	keys(a, "l")
	if filepath.Base(a.cur().Dir) != "dir1" {
		t.Error("enter")
	}
	keys(a, "h")
	if a.cur().Dir != d || a.cur().Current().Name != "dir1" {
		t.Error("up")
	}
	// sort keys
	keys(a, "ss")
	if a.cur().Sort.Key.String() != "size" {
		t.Error("ss")
	}
	keys(a, "sr")
	if !a.cur().Sort.Reverse || a.cur().Entries[2].Name != "beta.txt" {
		t.Errorf("sr: %v", a.cur().Entries[2].Name)
	}
	keys(a, ":sort time\n")
	if a.cur().Sort.Key.String() != "time" {
		t.Error(":sort time")
	}
	keys(a, "sn")
	keys(a, "sr")
	// hidden toggle
	keys(a, "za")
	if len(a.cur().Entries) != 5 {
		t.Error("za")
	}
	keys(a, "za")
	// preview toggle + tab
	keys(a, "w")
	if a.preview {
		t.Error("w")
	}
	keys(a, "\t")
	if a.active != 1 {
		t.Error("tab")
	}
	keys(a, "\t")
	keys(a, "w")
	// mkdir via cmd
	keys(a, ":mkdir newd\n")
	if _, err := os.Stat(filepath.Join(d, "newd")); err != nil {
		t.Error("mkdir")
	}
	if a.cur().Current().Name != "newd" {
		t.Error("cursor on new dir")
	}
	// touch via prompt
	keys(a, ":touch\n")
	if a.mode != ModeInput {
		t.Error("touch prompt")
	}
	keys(a, "new.txt\n")
	if _, err := os.Stat(filepath.Join(d, "new.txt")); err != nil {
		t.Error("touch")
	}
	// yank + paste into dir1
	keys(a, "/alpha\n")
	if a.cur().Current().Name != "alpha.txt" {
		t.Error("search")
	}
	keys(a, "yy")
	if len(a.clip.paths) != 1 || a.clip.cut {
		t.Error("yy")
	}
	keys(a, "gg")
	keys(a, "l")
	keys(a, "p")
	if _, err := os.Stat(filepath.Join(d, "dir1", "alpha.txt")); err != nil {
		t.Error("paste copy")
	}
	// cut + paste
	keys(a, "h")
	keys(a, "/beta\n")
	keys(a, "dd")
	keys(a, "gg")
	keys(a, "l")
	keys(a, "p")
	if _, err := os.Stat(filepath.Join(d, "beta.txt")); !os.IsNotExist(err) {
		t.Error("cut left source")
	}
	if _, err := os.Stat(filepath.Join(d, "dir1", "beta.txt")); err != nil {
		t.Error("paste move")
	}
	// rename via cw
	keys(a, "/beta\n")
	keys(a, "cw")
	if a.mode != ModeInput || a.input != "beta.txt" {
		t.Errorf("cw prompt: mode=%v input=%q", a.mode, a.input)
	}
	keys(a, "\x1b")
	keys(a, ":rename gamma.txt\n")
	if _, err := os.Stat(filepath.Join(d, "dir1", "gamma.txt")); err != nil {
		t.Error("rename")
	}
	// selection + delete with confirm
	keys(a, "gg")
	keys(a, "tj")
	keys(a, "t")
	if len(a.cur().Selected) != 2 {
		t.Error("select")
	}
	keys(a, "D")
	if a.mode != ModeConfirm {
		t.Error("confirm prompt")
	}
	keys(a, "n")
	if len(a.cur().Entries) != 3 {
		t.Error("delete not cancelled")
	}
	keys(a, "D")
	keys(a, "y")
	if len(a.cur().Entries) != 1 {
		t.Errorf("delete: %d left", len(a.cur().Entries))
	}
	// sync + quit
	keys(a, ":sync\n")
	if a.other().Dir != a.cur().Dir {
		t.Error("sync")
	}
	keys(a, ":bogus\n")
	if !strings.Contains(a.msg, "unknown") {
		t.Error("unknown cmd msg")
	}
	keys(a, "q")
	if !a.quit {
		t.Error("q")
	}
}

func TestEntryStyle(t *testing.T) {
	fg := func(st tcell.Style) tcell.Color { f, _, _ := st.Decompose(); return f }
	if fg(entryStyle(fsx.Entry{Name: "d", IsDir: true})) != fg(stDir) {
		t.Error("dir style")
	}
	if fg(entryStyle(fsx.Entry{Name: "x.PNG"})) != fg(stImage) {
		t.Error("image style")
	}
	if fg(entryStyle(fsx.Entry{Name: "x.txt"})) != fg(stDefault) {
		t.Error("plain style")
	}
}

func TestImagePreviewDraw(t *testing.T) {
	d := t.TempDir()
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			img.Set(x, y, color.RGBA{0, 255, 0, 255})
		}
	}
	f, _ := os.Create(filepath.Join(d, "g.png"))
	png.Encode(f, img)
	f.Close()
	os.WriteFile(filepath.Join(d, "bin"), []byte{0, 1, 2}, 0o644)
	a, s := newTestApp(t, d)
	keys(a, "j") // g.png (bin sorts first)
	a.draw()
	cells, w, _ := s.GetContents()
	// right panel starts at col 41, row 1
	c := cells[1*w+41]
	if len(c.Runes) == 0 || c.Runes[0] != '▀' {
		t.Fatalf("no block glyph: %+v", c)
	}
	fg, _, _ := c.Style.Decompose()
	if fg.Hex() != 0x00ff00 {
		t.Errorf("fg %06x", fg.Hex())
	}
	keys(a, "k")
	a.draw()
	if !strings.Contains(screenText(s), "(binary file)") {
		t.Error("binary preview")
	}
}
