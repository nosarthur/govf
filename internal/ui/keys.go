package ui

import (
	"strings"

	"github.com/gdamore/tcell/v2"
)

// keyName maps an event to a vim-ish key name.
func keyName(ev *tcell.EventKey) string {
	switch ev.Key() {
	case tcell.KeyRune:
		return string(ev.Rune())
	case tcell.KeyEnter:
		return "<Enter>"
	case tcell.KeyEscape:
		return "<Esc>"
	case tcell.KeyTab:
		return "<Tab>"
	case tcell.KeyUp:
		return "<Up>"
	case tcell.KeyDown:
		return "<Down>"
	case tcell.KeyLeft:
		return "<Left>"
	case tcell.KeyRight:
		return "<Right>"
	case tcell.KeyPgUp:
		return "<PgUp>"
	case tcell.KeyPgDn:
		return "<PgDn>"
	case tcell.KeyHome:
		return "<Home>"
	case tcell.KeyEnd:
		return "<End>"
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		return "<BS>"
	case tcell.KeyDelete:
		return "<Del>"
	case tcell.KeyCtrlD:
		return "<C-d>"
	case tcell.KeyCtrlU:
		return "<C-u>"
	case tcell.KeyCtrlF:
		return "<C-f>"
	case tcell.KeyCtrlB:
		return "<C-b>"
	case tcell.KeyCtrlL:
		return "<C-l>"
	case tcell.KeyCtrlC:
		return "<C-c>"
	}
	return ""
}

var bindings map[string]func(*App)

func init() {
	bindings = map[string]func(*App){
		"j": func(a *App) { a.cur().Move(1) }, "<Down>": func(a *App) { a.cur().Move(1) },
		"k": func(a *App) { a.cur().Move(-1) }, "<Up>": func(a *App) { a.cur().Move(-1) },
		"h": func(a *App) { a.cur().Up() }, "<Left>": func(a *App) { a.cur().Up() },
		"<BS>": func(a *App) { a.cur().Up() },
		"l":    (*App).enter, "<Right>": (*App).enter, "<Enter>": (*App).enter,
		"gg": func(a *App) { a.cur().MoveTo(0) }, "<Home>": func(a *App) { a.cur().MoveTo(0) },
		"G":     func(a *App) { a.cur().MoveTo(len(a.cur().Entries) - 1) },
		"<End>": func(a *App) { a.cur().MoveTo(len(a.cur().Entries) - 1) },
		"<C-d>": func(a *App) { a.cur().Move(a.viewH() / 2) },
		"<C-u>": func(a *App) { a.cur().Move(-a.viewH() / 2) },
		"<C-f>": func(a *App) { a.cur().Move(a.viewH()) }, "<PgDn>": func(a *App) { a.cur().Move(a.viewH()) },
		"<C-b>": func(a *App) { a.cur().Move(-a.viewH()) }, "<PgUp>": func(a *App) { a.cur().Move(-a.viewH()) },
		"<Tab>": (*App).switchPanel,
		"w":     func(a *App) { a.preview = !a.preview },
		"t":     func(a *App) { a.cur().ToggleSel() },
		" ":     func(a *App) { a.cur().ToggleSel(); a.cur().Move(1) },
		"yy":    func(a *App) { a.yank(false) },
		"dd":    func(a *App) { a.yank(true) },
		"p":     (*App).paste,
		"D":     (*App).deleteTargets, "<Del>": (*App).deleteTargets,
		"cw":    (*App).rename,
		"za":    (*App).toggleHidden,
		"S":     func(a *App) { a.openMenu(a.sortMenu()) },
		"~":     func(a *App) { a.cd("~") },
		":":     func(a *App) { a.startLine(ModeCmd, ":", "", nil) },
		"/":     func(a *App) { a.startLine(ModeSearch, "/", "", nil) },
		"n":     func(a *App) { a.searchNext(false) },
		"N":     func(a *App) { a.searchNext(true) },
		"<C-l>": func(a *App) { a.reloadAll(); a.scr.Sync(); a.invalidateImage() },
		"q":     func(a *App) { a.quit = true },
		"<C-c>": func(a *App) { a.quit = true },
		"<Esc>": func(a *App) { a.cur().Selected = map[string]bool{}; a.msg = "" },
	}
}

// hasPrefix: any binding starts with s (and is longer).
func hasPrefix(s string) bool {
	for k := range bindings {
		if len(k) > len(s) && strings.HasPrefix(k, s) {
			return true
		}
	}
	return false
}

func (a *App) handleNormal(ev *tcell.EventKey) {
	name := keyName(ev)
	if name == "" {
		return
	}
	seq := a.pending + name
	if fn, ok := bindings[seq]; ok {
		a.pending = ""
		fn(a)
		return
	}
	if hasPrefix(seq) {
		a.pending = seq
		return
	}
	a.pending = ""
	// retry bare key if prefix combo failed
	if seq != name {
		if fn, ok := bindings[name]; ok {
			fn(a)
		}
	}
}
