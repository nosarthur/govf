package ui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"

	"github.com/nosarthur/govf/internal/fsx"
)

type menuItem struct {
	key   rune
	label string
	act   func(*App)
}

// menu: modal popup list; item key or Enter runs action and closes.
type menu struct {
	title string
	items []menuItem
	cur   int
}

func (a *App) openMenu(m *menu) {
	a.menu = m
	a.mode = ModeMenu
}

func (a *App) closeMenu() {
	a.menu = nil
	a.mode = ModeNormal
}

func (a *App) handleMenu(ev *tcell.EventKey) {
	m := a.menu
	if m == nil {
		a.mode = ModeNormal
		return
	}
	run := func(i int) {
		a.closeMenu()
		m.items[i].act(a)
	}
	switch ev.Key() {
	case tcell.KeyEscape, tcell.KeyCtrlC:
		a.closeMenu()
	case tcell.KeyEnter:
		run(m.cur)
	case tcell.KeyDown:
		m.cur = (m.cur + 1) % len(m.items)
	case tcell.KeyUp:
		m.cur = (m.cur + len(m.items) - 1) % len(m.items)
	case tcell.KeyRune:
		r := ev.Rune()
		for i, it := range m.items {
			if it.key == r {
				run(i)
				return
			}
		}
		switch r {
		case 'j':
			m.cur = (m.cur + 1) % len(m.items)
		case 'k':
			m.cur = (m.cur + len(m.items) - 1) % len(m.items)
		case 'q':
			a.closeMenu()
		}
	}
}

// sortMenu builds the sort chooser reflecting current panel state.
func (a *App) sortMenu() *menu {
	p := a.cur()
	mark := func(k fsx.SortKey) string {
		if p.Sort.Key == k {
			return " *"
		}
		return ""
	}
	rev := "off"
	if p.Sort.Reverse {
		rev = "on"
	}
	m := &menu{title: "Sort by", items: []menuItem{
		{'n', "name" + mark(fsx.SortName), func(a *App) { a.setSort(fsx.SortName) }},
		{'s', "size" + mark(fsx.SortSize), func(a *App) { a.setSort(fsx.SortSize) }},
		{'t', "time" + mark(fsx.SortTime), func(a *App) { a.setSort(fsx.SortTime) }},
		{'r', "reverse: " + rev, (*App).toggleReverse},
	}}
	m.cur = int(p.Sort.Key)
	return m
}

func (a *App) drawMenu(px, py, pw, ph int) {
	m := a.menu
	if m == nil {
		return
	}
	w := runewidth.StringWidth(m.title) + 4
	for _, it := range m.items {
		if l := runewidth.StringWidth(it.label) + 8; l > w {
			w = l
		}
	}
	h := len(m.items) + 2
	if w > pw {
		w = pw
	}
	x := px + (pw-w)/2
	y := py + max((ph-h)/2, 0)
	border := stDefault.Foreground(tcell.ColorYellow)
	a.puts(x, y, w, "┌"+runewidth.FillRight(" "+m.title+" ", w-2)+"┐", border)
	for i, it := range m.items {
		st := stDefault
		if i == m.cur {
			st = stCursor
		}
		line := fmt.Sprintf(" [%c] %s", it.key, it.label)
		a.scr.SetContent(x, y+1+i, '│', nil, border)
		a.puts(x+1, y+1+i, w-2, line, st)
		a.scr.SetContent(x+w-1, y+1+i, '│', nil, border)
	}
	a.puts(x, y+h-1, w, "└"+runewidth.FillRight("", w-2)+"┘", border)
	// fix bottom border fill
	for i := 1; i < w-1; i++ {
		a.scr.SetContent(x+i, y+h-1, '─', nil, border)
	}
	for i := runewidth.StringWidth(m.title) + 3; i < w-1; i++ {
		a.scr.SetContent(x+i, y, '─', nil, border)
	}
}
