// Package ui: tcell-based two-panel file manager.
package ui

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/gdamore/tcell/v2"

	"github.com/nosarthur/govf/internal/fsx"
	"github.com/nosarthur/govf/internal/preview"
)

// Mode of the bottom line.
type Mode int

const (
	ModeNormal Mode = iota
	ModeCmd
	ModeSearch
	ModeConfirm
	ModeInput // generic prompt (rename etc)
	ModeMenu
)

// placement: a native image drawn at a screen rect.
type placement struct {
	key        string
	x, y, w, h int
	data       []byte
}

type clipboard struct {
	paths []string
	cut   bool
}

// App holds all state.
type App struct {
	scr     tcell.Screen
	panels  [2]*Panel
	active  int
	preview bool // inactive side shows preview
	mode    Mode
	prompt  string
	input   string
	msg     string
	msgErr  bool
	pending string // multi-key prefix
	clip    clipboard
	lastQ   string
	onInput func(string) // ModeInput/ModeConfirm callback
	quit    bool
	pvCache pvCache
	menu    *menu

	clipTool []string // external clipboard cmd, if any

	trash     string // trash dir for dd
	undoStack []op
	redoStack []op

	proto    preview.Protocol
	raw      io.Writer  // tty for image/clipboard escapes; nil => blocks, no OSC52
	imgWant  *placement // image requested this frame
	imgShown *placement // image currently on screen
}

// SetTerminal sets raw tty writer (image + OSC52 escapes) and image protocol.
func (a *App) SetTerminal(raw io.Writer, p preview.Protocol) {
	a.proto, a.raw = p, raw
	if raw == nil {
		a.proto = preview.ProtoBlocks
	}
}

// invalidateImage forces re-send of native image on next draw.
func (a *App) invalidateImage() { a.imgShown = nil }

type pvCache struct {
	key string
	res preview.Result
}

// New builds app with left/right start dirs.
func New(scr tcell.Screen, left, right string) *App {
	a := &App{scr: scr}
	a.panels[0] = NewPanel(left)
	a.panels[1] = NewPanel(right)
	a.preview = true
	a.clipTool = findClipTool()
	a.trash = TrashDir()
	return a
}

func (a *App) cur() *Panel   { return a.panels[a.active] }
func (a *App) other() *Panel { return a.panels[1-a.active] }

func (a *App) setMsg(format string, args ...any) {
	a.msg, a.msgErr = fmt.Sprintf(format, args...), false
}

func (a *App) setErr(err error) {
	if err == nil {
		return
	}
	a.msg, a.msgErr = err.Error(), true
}

// Run drives the event loop until quit.
func (a *App) Run() {
	for !a.quit {
		a.draw()
		ev := a.scr.PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventResize:
			a.scr.Sync()
			a.invalidateImage()
		case *tcell.EventKey:
			a.handleKey(ev)
		}
	}
}

func (a *App) handleKey(ev *tcell.EventKey) {
	switch a.mode {
	case ModeNormal:
		a.handleNormal(ev)
	case ModeMenu:
		a.handleMenu(ev)
	default:
		a.handleLine(ev)
	}
}

// handleLine: editing of the bottom input line.
func (a *App) handleLine(ev *tcell.EventKey) {
	switch ev.Key() {
	case tcell.KeyEscape, tcell.KeyCtrlC:
		a.mode = ModeNormal
		a.input = ""
		a.onInput = nil
	case tcell.KeyEnter:
		text := a.input
		m, cb := a.mode, a.onInput
		a.mode, a.input, a.onInput = ModeNormal, "", nil
		switch m {
		case ModeCmd:
			a.runCommand(text)
		case ModeSearch:
			a.lastQ = text
			a.searchNext(false)
		case ModeInput, ModeConfirm:
			if cb != nil {
				cb(text)
			}
		}
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if a.input == "" {
			a.mode, a.onInput = ModeNormal, nil
			return
		}
		r := []rune(a.input)
		a.input = string(r[:len(r)-1])
	case tcell.KeyCtrlU:
		a.input = ""
	case tcell.KeyRune:
		if a.mode == ModeConfirm {
			// single-key y/n
			cb := a.onInput
			a.mode, a.input, a.onInput = ModeNormal, "", nil
			if cb != nil {
				cb(string(ev.Rune()))
			}
			return
		}
		a.input += string(ev.Rune())
	}
}

func (a *App) startLine(m Mode, prompt, initial string, cb func(string)) {
	a.mode, a.prompt, a.input, a.onInput = m, prompt, initial, cb
}

func (a *App) confirm(q string, yes func()) {
	a.startLine(ModeConfirm, q+" [y/N] ", "", func(s string) {
		if strings.ToLower(s) == "y" {
			yes()
		}
	})
}

// ---- actions ----

func (a *App) switchPanel() { a.active = 1 - a.active }

func (a *App) enter() {
	p := a.cur()
	e := p.Current()
	if e == nil {
		return
	}
	if e.IsDir {
		p.Enter()
		a.setErr(p.Err)
		return
	}
	a.openFile(e.Path)
}

// openFile: text in $EDITOR, else OS opener.
func (a *App) openFile(path string) {
	r := preview.Text(path, 1)
	var cmd *exec.Cmd
	if r.Kind == preview.KindText {
		ed := os.Getenv("EDITOR")
		if ed == "" {
			ed = "vi"
		}
		cmd = exec.Command(ed, path)
	} else {
		opener := "xdg-open"
		if runtime.GOOS == "darwin" {
			opener = "open"
		}
		cmd = exec.Command(opener, path)
	}
	a.runExternal(cmd)
}

func (a *App) runExternal(cmd *exec.Cmd) {
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := a.scr.Suspend(); err != nil {
		a.setErr(err)
		return
	}
	err := cmd.Run()
	a.scr.Resume()
	a.invalidateImage()
	a.setErr(err)
	a.reloadAll()
}

func (a *App) reloadAll() {
	for _, p := range a.panels {
		p.Reload()
	}
	a.pvCache = pvCache{}
}

func (a *App) yank(cut bool) {
	t := a.cur().Targets()
	if len(t) == 0 {
		return
	}
	a.clip = clipboard{paths: t, cut: cut}
	verb := "yanked"
	if cut {
		verb = "cut"
	}
	a.setMsg("%d %s", len(t), verb)
	a.cur().Selected = map[string]bool{}
}

func (a *App) paste() {
	if len(a.clip.paths) == 0 {
		a.setMsg("clipboard empty")
		return
	}
	dst := a.cur().Dir
	var steps []step
	for _, src := range a.clip.paths {
		var to string
		var err error
		if a.clip.cut {
			to, err = fsx.Move(src, dst)
			a.removeEmptyTrashSlot(src)
			steps = append(steps, step{stepMove, src, to})
		} else {
			to, err = fsx.Copy(src, dst)
			steps = append(steps, step{stepCopy, src, to})
		}
		if err != nil {
			steps = steps[:len(steps)-1]
			a.setErr(err)
			break
		}
	}
	if a.clip.cut {
		a.record("move", steps...)
		a.clip = clipboard{}
	} else {
		a.record("copy", steps...)
	}
	a.reloadAll()
	if !a.msgErr {
		a.setMsg("%d pasted", len(steps))
	}
}

// renameTo renames cursor entry's path old to name (same dir); undoable.
func (a *App) renameTo(old, name string) {
	dst := filepath.Join(filepath.Dir(old), name)
	if err := fsx.Rename(old, dst); err != nil {
		a.setErr(err)
		return
	}
	a.record("rename", step{stepMove, old, dst})
	a.reloadAll()
	a.cur().SeekName(name)
}

func (a *App) deleteTargets() {
	t := a.cur().Targets()
	if len(t) == 0 {
		return
	}
	q := fmt.Sprintf("permanently delete %d items?", len(t))
	if len(t) == 1 {
		q = fmt.Sprintf("permanently delete %s?", filepath.Base(t[0]))
	}
	a.confirm(q, func() {
		for _, p := range t {
			if err := fsx.Delete(p); err != nil {
				a.setErr(err)
				break
			}
		}
		a.reloadAll()
	})
}

// rename prompts for full name (cw).
func (a *App) rename() { a.renameWith(false) }

// renameStem prompts for name without extension; ext kept (a).
func (a *App) renameStem() { a.renameWith(true) }

// splitExt: ("a.tar", ".gz"); dirs and dotfiles like ".bashrc" have no ext.
func splitExt(e *fsx.Entry) (stem, ext string) {
	if e.IsDir {
		return e.Name, ""
	}
	ext = filepath.Ext(e.Name)
	if ext == e.Name {
		return e.Name, ""
	}
	return strings.TrimSuffix(e.Name, ext), ext
}

func (a *App) renameWith(keepExt bool) {
	e := a.cur().Current()
	if e == nil {
		return
	}
	old := e.Path
	initial, ext, prompt := e.Name, "", "rename: "
	if keepExt {
		initial, ext = splitExt(e)
		if ext != "" {
			prompt = "rename (" + ext + "): "
		}
	}
	a.startLine(ModeInput, prompt, initial, func(name string) {
		name = strings.TrimSpace(name)
		if name == "" {
			return
		}
		name += ext
		if name == filepath.Base(old) {
			return
		}
		a.renameTo(old, name)
	})
}

func (a *App) mkdir(name string) {
	if name == "" {
		a.startLine(ModeInput, "mkdir: ", "", a.mkdir)
		return
	}
	p := filepath.Join(a.cur().Dir, name)
	if _, err := os.Lstat(p); err == nil {
		a.setErr(fmt.Errorf("%s exists", name))
		return
	}
	if err := fsx.Mkdir(p); err != nil {
		a.setErr(err)
		return
	}
	a.record("mkdir", step{stepCreate, "", p})
	a.reloadAll()
	a.cur().SeekName(name)
}

func (a *App) touch(name string) {
	if name == "" {
		a.startLine(ModeInput, "touch: ", "", a.touch)
		return
	}
	p := filepath.Join(a.cur().Dir, name)
	if err := fsx.Touch(p); err != nil {
		a.setErr(err)
		return
	}
	a.record("touch", step{stepCreate, "", p})
	a.reloadAll()
	a.cur().SeekName(name)
}

func (a *App) setSort(k fsx.SortKey) {
	p := a.cur()
	p.Sort.Key = k
	p.Reload()
	a.setMsg("sort %s", p.Sort)
}

func (a *App) toggleReverse() {
	p := a.cur()
	p.Sort.Reverse = !p.Sort.Reverse
	p.Reload()
	a.setMsg("sort %s", p.Sort)
}

func (a *App) toggleHidden() {
	p := a.cur()
	p.Hidden = !p.Hidden
	p.Reload()
}

func (a *App) cd(dir string) {
	if dir == "" || dir == "~" {
		dir, _ = os.UserHomeDir()
	} else if strings.HasPrefix(dir, "~/") {
		h, _ := os.UserHomeDir()
		dir = filepath.Join(h, dir[2:])
	} else if !filepath.IsAbs(dir) {
		dir = filepath.Join(a.cur().Dir, dir)
	}
	st, err := os.Stat(dir)
	if err != nil {
		a.setErr(err)
		return
	}
	if !st.IsDir() {
		a.setMsg("not a dir: %s", dir)
		return
	}
	a.cur().remember()
	a.cur().Load(dir)
	a.setErr(a.cur().Err)
}

func (a *App) searchNext(backward bool) {
	if a.lastQ == "" {
		return
	}
	if !a.cur().Search(a.lastQ, backward) {
		a.setMsg("not found: %s", a.lastQ)
	}
}

func (a *App) viewH() int {
	_, h := a.scr.Size()
	return h - 3 // header + status + cmdline
}
