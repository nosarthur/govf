package ui

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"

	"github.com/nosarthur/govf/internal/fsx"
	"github.com/nosarthur/govf/internal/preview"
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
		case '\x08':
			ev = tcell.NewEventKey(tcell.KeyBackspace, 0, 0)
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
	// sort menu
	keys(a, "S")
	if a.mode != ModeMenu || a.menu == nil {
		t.Fatal("S menu")
	}
	a.draw()
	if out := screenText(s); !strings.Contains(out, "Sort by") || !strings.Contains(out, "[s] size") {
		t.Errorf("menu not drawn:\n%s", out)
	}
	keys(a, "s")
	if a.mode != ModeNormal || a.cur().Sort.Key.String() != "size" {
		t.Error("menu s")
	}
	keys(a, "Sr")
	if !a.cur().Sort.Reverse || a.cur().Entries[2].Name != "beta.txt" {
		t.Errorf("menu r: %v", a.cur().Entries[2].Name)
	}
	keys(a, "S\x1b")
	if a.mode != ModeNormal || a.menu != nil {
		t.Error("menu esc")
	}
	keys(a, "Sjjj\n") // cursor starts on size(1) -> wraps to name(0)
	if a.cur().Sort.Key.String() != "name" {
		t.Errorf("menu nav: %s", a.cur().Sort.Key)
	}
	keys(a, ":sort time\n")
	if a.cur().Sort.Key.String() != "time" {
		t.Error(":sort time")
	}
	keys(a, "Sn")
	keys(a, "Sr")
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
	// a: edit stem, keep ext
	keys(a, "/gamma\n")
	keys(a, "a")
	if a.mode != ModeInput || a.input != "gamma" || a.prompt != "rename (.txt): " {
		t.Errorf("a prompt: mode=%v input=%q prompt=%q", a.mode, a.input, a.prompt)
	}
	keys(a, "\x08\x08\x08\x08\x08delta\n")
	if _, err := os.Stat(filepath.Join(d, "dir1", "delta.txt")); err != nil {
		t.Error("a rename kept ext")
	}
	if a.cur().Current().Name != "delta.txt" {
		t.Error("cursor after a rename")
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

func TestNativeImageFlush(t *testing.T) {
	d := t.TempDir()
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	f, _ := os.Create(filepath.Join(d, "g.png"))
	png.Encode(f, img)
	f.Close()
	os.WriteFile(filepath.Join(d, "a.txt"), []byte("hi"), 0o644)
	a, s := newTestApp(t, d)
	var raw bytes.Buffer
	a.SetTerminal(&raw, preview.ProtoITerm)
	keys(a, "j") // g.png
	a.draw()
	out := raw.String()
	if !strings.Contains(out, "\x1b[2;42H\x1b]1337;File=inline=1;") {
		t.Errorf("iterm seq not written at right panel origin: %q", out[:min(60, len(out))])
	}
	if strings.Contains(screenText(s), "▀") {
		t.Error("blocks drawn in native mode")
	}
	raw.Reset()
	a.draw() // unchanged: no resend
	if raw.Len() != 0 {
		t.Error("image resent without change")
	}
	keys(a, "k") // a.txt: image removed
	a.draw()
	if a.imgShown != nil {
		t.Error("image still shown")
	}
	// blocks fallback when no writer
	a.SetTerminal(nil, preview.ProtoITerm)
	a.pvCache = pvCache{}
	keys(a, "j")
	a.draw()
	if !strings.Contains(screenText(s), "▀") {
		t.Error("blocks fallback")
	}
}

func TestYankClipboard(t *testing.T) {
	d := setup(t)
	a, _ := newTestApp(t, d)
	a.clipTool = nil // don't touch real clipboard
	keys(a, "yf")
	if a.msg != "no clipboard available" {
		t.Errorf("no tty msg: %q", a.msg)
	}
	var raw bytes.Buffer
	a.SetTerminal(&raw, preview.ProtoBlocks)
	b64 := func(s string) string { return base64.StdEncoding.EncodeToString([]byte(s)) }
	keys(a, "yf")
	if want := "\x1b]52;c;" + b64(filepath.Join(d, "dir1")) + "\a"; raw.String() != want {
		t.Errorf("yf: %q", raw.String())
	}
	raw.Reset()
	keys(a, "yd")
	if want := "\x1b]52;c;" + b64(d) + "\a"; raw.String() != want {
		t.Errorf("yd: %q", raw.String())
	}
	raw.Reset()
	keys(a, "jjyn")
	if want := "\x1b]52;c;" + b64("alpha.txt") + "\a"; raw.String() != want {
		t.Errorf("yn: %q", raw.String())
	}
	if !strings.HasPrefix(a.msg, "copied name: alpha.txt") {
		t.Errorf("msg %q", a.msg)
	}
	// yy still works after y-prefix additions
	keys(a, "yy")
	if len(a.clip.paths) != 1 {
		t.Error("yy")
	}
}

func TestSplitExt(t *testing.T) {
	cases := []struct {
		name      string
		dir       bool
		stem, ext string
	}{
		{"a.txt", false, "a", ".txt"},
		{"a.tar.gz", false, "a.tar", ".gz"},
		{".bashrc", false, ".bashrc", ""},
		{"README", false, "README", ""},
		{"v1.0", true, "v1.0", ""},
	}
	for _, c := range cases {
		s, e := splitExt(&fsx.Entry{Name: c.name, IsDir: c.dir})
		if s != c.stem || e != c.ext {
			t.Errorf("%s: got %q %q", c.name, s, e)
		}
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
