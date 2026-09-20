package ui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"

	"github.com/nosarthur/govf/internal/fsx"
	"github.com/nosarthur/govf/internal/preview"
)

var (
	stDefault = tcell.StyleDefault
	stHeader  = tcell.StyleDefault.Bold(true).Foreground(tcell.ColorYellow)
	stDir     = tcell.StyleDefault.Foreground(tcell.NewRGBColor(0x5f, 0xaf, 0xff)).Bold(true)
	stLink    = tcell.StyleDefault.Foreground(tcell.ColorTeal)
	stImage   = tcell.StyleDefault.Foreground(tcell.NewRGBColor(0xd7, 0x5f, 0xd7))
	stSel     = tcell.StyleDefault.Foreground(tcell.ColorYellow).Bold(true)
	stCursor  = tcell.StyleDefault.Reverse(true)
	stErr     = tcell.StyleDefault.Foreground(tcell.ColorRed).Bold(true)
	stDim     = tcell.StyleDefault.Dim(true)
	stSep     = tcell.StyleDefault.Foreground(tcell.ColorGray)
)

// joinDir: sibling path of p named name.
func joinDir(p, name string) string { return filepath.Join(filepath.Dir(p), name) }

func (a *App) puts(x, y, w int, s string, st tcell.Style) {
	col := 0
	for _, r := range s {
		rw := runewidth.RuneWidth(r)
		if col+rw > w {
			break
		}
		a.scr.SetContent(x+col, y, r, nil, st)
		col += rw
	}
	for ; col < w; col++ {
		a.scr.SetContent(x+col, y, ' ', nil, st)
	}
}

func (a *App) draw() {
	s := a.scr
	s.Clear()
	W, H := s.Size()
	if W < 10 || H < 5 {
		s.Show()
		return
	}
	lw := W / 2
	rw := W - lw - 1
	listH := H - 3
	a.imgWant = nil
	// two sides
	a.drawSide(0, 0, lw, listH, 0)
	for y := 0; y < listH+1; y++ {
		s.SetContent(lw, y, '│', nil, stSep)
	}
	a.drawSide(lw+1, 0, rw, listH, 1)
	if a.mode == ModeMenu {
		px := 0
		if a.active == 1 {
			px = lw + 1
		}
		a.drawMenu(px, 1, lw, listH)
	}
	a.drawStatus(H-2, W)
	a.drawCmdline(H-1, W)
	a.flush()
}

// flush pushes cells, then (re)sends native image if its placement changed.
func (a *App) flush() {
	changed := (a.imgWant == nil) != (a.imgShown == nil) ||
		(a.imgWant != nil && a.imgWant.key != a.imgShown.key)
	if !changed {
		a.scr.Show()
		return
	}
	if a.imgShown != nil && a.proto == preview.ProtoKitty {
		a.raw.Write(preview.KittyDelete())
	}
	a.scr.Sync() // full repaint clears old image cells
	if p := a.imgWant; p != nil {
		var seq []byte
		switch a.proto {
		case preview.ProtoITerm:
			seq = preview.ITermSeq(p.data, p.w, p.h)
		case preview.ProtoKitty:
			seq = preview.KittySeq(p.data, p.w, p.h)
		}
		fmt.Fprintf(a.raw, "\x1b[%d;%dH", p.y+1, p.x+1)
		a.raw.Write(seq)
		if a.mode != ModeNormal { // restore cmdline cursor moved by image
			_, H := a.scr.Size()
			fmt.Fprintf(a.raw, "\x1b[%d;%dH", H, runewidth.StringWidth(a.prompt+a.input)+1)
		}
	}
	a.imgShown = a.imgWant
}

func (a *App) drawSide(x, y, w, h, idx int) {
	if a.preview && idx != a.active {
		a.drawPreview(x, y, w, h)
		return
	}
	a.drawPanel(x, y, w, h, a.panels[idx], idx == a.active)
}

func (a *App) drawPanel(x, y, w, h int, p *Panel, active bool) {
	hs := stHeader
	if !active {
		hs = stDim
	}
	a.puts(x, y, w, trimPathLeft(p.Dir, w), hs)
	p.Scroll(h)
	if p.Err != nil {
		a.puts(x, y+1, w, p.Err.Error(), stErr)
		return
	}
	for i := 0; i < h; i++ {
		idx := p.Top + i
		if idx >= len(p.Entries) {
			break
		}
		e := p.Entries[idx]
		st := entryStyle(e)
		name := e.Name
		if e.IsDir {
			name += "/"
		}
		prefix := " "
		if p.Selected[e.Path] {
			st = stSel
			prefix = "*"
		}
		if active && idx == p.Cur {
			st = st.Reverse(true)
		}
		right := fsx.HumanSize(e.Size)
		if e.IsDir {
			right = ""
		}
		nameW := w - len(prefix) - len(right) - 1
		if nameW < 1 {
			nameW = 1
		}
		line := prefix + runewidth.FillRight(runewidth.Truncate(name, nameW, "~"), nameW) + " " + right
		a.puts(x, y+1+i, w, line, st)
	}
}

// render: native image payload when protocol supports it, else cells/text.
func (a *App) render(path string, w, h int) preview.Result {
	if a.proto != preview.ProtoBlocks {
		if r := preview.Native(path, a.proto); r.Kind == preview.KindImage {
			return r
		}
	}
	return preview.File(path, w, h)
}

// entryStyle: dir > link > image > plain.
func entryStyle(e fsx.Entry) tcell.Style {
	switch {
	case e.IsDir:
		return stDir
	case e.IsLink:
		return stLink
	case preview.IsImagePath(e.Name):
		return stImage
	}
	return stDefault
}

// trimPathLeft keeps the tail of long paths.
func trimPathLeft(p string, w int) string {
	if runewidth.StringWidth(p) <= w {
		return p
	}
	return runewidth.TruncateLeft(p, runewidth.StringWidth(p)-w+1, "…")
}

func (a *App) drawPreview(x, y, w, h int) {
	e := a.cur().Current()
	if e == nil {
		a.puts(x, y, w, "(preview)", stDim)
		return
	}
	a.puts(x, y, w, trimPathLeft(e.Path, w), stDim)
	if e.IsDir {
		es, err := fsx.ReadDir(e.Path, a.cur().Hidden, a.cur().Sort)
		if err != nil {
			a.puts(x, y+1, w, err.Error(), stErr)
			return
		}
		for i := 0; i < h && i < len(es); i++ {
			n := es[i].Name
			if es[i].IsDir {
				n += "/"
			}
			a.puts(x, y+1+i, w, " "+n, entryStyle(es[i]))
		}
		return
	}
	key := fmt.Sprintf("%s|%d|%d|%d|%d", e.Path, e.Size, e.ModTime.UnixNano(), w, h)
	if a.pvCache.key != key {
		a.pvCache = pvCache{key: key, res: a.render(e.Path, w, h)}
	}
	r := a.pvCache.res
	if r.Kind == preview.KindImage && r.Data != nil {
		a.imgWant = &placement{key: fmt.Sprintf("%s|%d|%d", key, x, y), x: x, y: y + 1, w: w, h: h, data: r.Data}
		return
	}
	switch r.Kind {
	case preview.KindText:
		for i, ln := range r.Lines {
			if i >= h {
				break
			}
			a.puts(x, y+1+i, w, ln, stDefault)
		}
	case preview.KindImage:
		for ry, row := range r.Cells {
			if ry >= h {
				break
			}
			for rx, c := range row {
				if rx >= w {
					break
				}
				st := tcell.StyleDefault.Foreground(tcell.NewHexColor(int32(c.FG))).Background(tcell.NewHexColor(int32(c.BG)))
				a.scr.SetContent(x+rx, y+1+ry, '▀', nil, st)
			}
		}
	case preview.KindBinary:
		a.puts(x, y+1, w, "(binary file)", stDim)
	case preview.KindError:
		a.puts(x, y+1, w, r.Err.Error(), stErr)
	}
}

func (a *App) drawStatus(y, w int) {
	p := a.cur()
	left := ""
	if e := p.Current(); e != nil {
		left = fmt.Sprintf("%s %s %s", e.Mode.String(), fsx.HumanSize(e.Size), e.ModTime.Format("2006-01-02 15:04"))
	}
	nsel := len(p.Selected)
	right := fmt.Sprintf("sort:%s", p.Sort)
	if p.Hidden {
		right += " hidden"
	}
	if nsel > 0 {
		right += fmt.Sprintf(" sel:%d", nsel)
	}
	if len(a.clip.paths) > 0 {
		op := "yank"
		if a.clip.cut {
			op = "cut"
		}
		right += fmt.Sprintf(" clip:%d(%s)", len(a.clip.paths), op)
	}
	right += fmt.Sprintf(" %d/%d", min(p.Cur+1, len(p.Entries)), len(p.Entries))
	gap := w - runewidth.StringWidth(left) - runewidth.StringWidth(right)
	if gap < 1 {
		gap = 1
	}
	a.puts(0, y, w, left+strings.Repeat(" ", gap)+right, stDim.Dim(false).Reverse(true))
}

func (a *App) drawCmdline(y, w int) {
	switch a.mode {
	case ModeNormal:
		st := stDefault
		if a.msgErr {
			st = stErr
		}
		txt := a.msg
		if a.pending != "" {
			txt = a.pending
		}
		a.puts(0, y, w, txt, st)
		a.scr.HideCursor()
	default:
		line := a.prompt + a.input
		a.puts(0, y, w, line, stDefault)
		a.scr.ShowCursor(runewidth.StringWidth(line), y)
	}
}
