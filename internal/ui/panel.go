package ui

import (
	"path/filepath"
	"strings"

	"github.com/nosarthur/govf/internal/fsx"
)

// Panel is one directory view.
type Panel struct {
	Dir      string
	Entries  []fsx.Entry
	Cur      int
	Top      int
	Hidden   bool
	Sort     fsx.SortSpec
	Selected map[string]bool   // by path
	lastPos  map[string]string // dir -> cursor name
	Err      error
}

func NewPanel(dir string) *Panel {
	p := &Panel{Selected: map[string]bool{}, lastPos: map[string]string{}}
	p.Load(dir)
	return p
}

// Load reads dir; cursor restored from history if seen before.
func (p *Panel) Load(dir string) {
	dir = filepath.Clean(dir)
	want := p.lastPos[dir]
	if p.Dir == dir && p.Cur < len(p.Entries) {
		want = p.Entries[p.Cur].Name
	}
	p.Dir = dir
	p.Cur, p.Top = 0, 0
	p.Selected = map[string]bool{}
	p.Entries, p.Err = fsx.ReadDir(dir, p.Hidden, p.Sort)
	if want != "" {
		p.SeekName(want)
	}
}

// Reload rereads current dir keeping cursor + selection.
func (p *Panel) Reload() {
	sel := p.Selected
	p.Load(p.Dir)
	for k := range sel {
		if p.index(k) >= 0 {
			p.Selected[k] = true
		}
	}
}

func (p *Panel) index(path string) int {
	for i, e := range p.Entries {
		if e.Path == path {
			return i
		}
	}
	return -1
}

// SeekName moves cursor to entry named name if present.
func (p *Panel) SeekName(name string) bool {
	for i, e := range p.Entries {
		if e.Name == name {
			p.Cur = i
			return true
		}
	}
	return false
}

// Current entry or nil.
func (p *Panel) Current() *fsx.Entry {
	if p.Cur < 0 || p.Cur >= len(p.Entries) {
		return nil
	}
	return &p.Entries[p.Cur]
}

func (p *Panel) remember() {
	if e := p.Current(); e != nil {
		p.lastPos[p.Dir] = e.Name
	}
}

// Move cursor by delta, clamped.
func (p *Panel) Move(delta int) {
	n := len(p.Entries)
	if n == 0 {
		p.Cur = 0
		return
	}
	p.Cur += delta
	if p.Cur < 0 {
		p.Cur = 0
	}
	if p.Cur >= n {
		p.Cur = n - 1
	}
}

func (p *Panel) MoveTo(i int) {
	p.Cur = i
	p.Move(0)
}

// Enter descends into dir under cursor. Returns false if not a dir.
func (p *Panel) Enter() bool {
	e := p.Current()
	if e == nil || !e.IsDir {
		return false
	}
	p.remember()
	p.Load(e.Path)
	return true
}

// Up goes to parent, cursor on the dir we left.
func (p *Panel) Up() {
	parent := filepath.Dir(p.Dir)
	if parent == p.Dir {
		return
	}
	p.remember()
	child := filepath.Base(p.Dir)
	p.Load(parent)
	p.SeekName(child)
}

// ToggleSel flips selection under cursor.
func (p *Panel) ToggleSel() {
	e := p.Current()
	if e == nil {
		return
	}
	if p.Selected[e.Path] {
		delete(p.Selected, e.Path)
	} else {
		p.Selected[e.Path] = true
	}
}

// Targets: selected paths in list order, else cursor entry.
func (p *Panel) Targets() []string {
	var out []string
	for _, e := range p.Entries {
		if p.Selected[e.Path] {
			out = append(out, e.Path)
		}
	}
	if len(out) == 0 {
		if e := p.Current(); e != nil {
			out = append(out, e.Path)
		}
	}
	return out
}

// Search finds next entry (from cur+dir) whose name contains q (case-insens).
func (p *Panel) Search(q string, backward bool) bool {
	n := len(p.Entries)
	if n == 0 || q == "" {
		return false
	}
	q = strings.ToLower(q)
	step := 1
	if backward {
		step = -1
	}
	for k := 1; k <= n; k++ {
		i := ((p.Cur+step*k)%n + n) % n
		if strings.Contains(strings.ToLower(p.Entries[i].Name), q) {
			p.Cur = i
			return true
		}
	}
	return false
}

// Scroll adjusts Top so Cur is visible in a window of h rows.
func (p *Panel) Scroll(h int) {
	if h <= 0 {
		return
	}
	if p.Cur < p.Top {
		p.Top = p.Cur
	}
	if p.Cur >= p.Top+h {
		p.Top = p.Cur - h + 1
	}
	if p.Top < 0 {
		p.Top = 0
	}
}
