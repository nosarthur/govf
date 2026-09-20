package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/nosarthur/govf/internal/fsx"
)

type stepKind int

const (
	stepMove   stepKind = iota // from -> to; undo moves back
	stepCopy                   // from copied to `to`; undo trashes `to`
	stepCreate                 // `to` created; undo trashes it
)

type step struct {
	kind     stepKind
	from, to string
}

// op: one undoable user action.
type op struct {
	desc  string
	steps []step
}

// TrashDir: $GOVF_TRASH, else $XDG_DATA_HOME/govf/trash, else ~/.local/share/govf/trash.
func TrashDir() string {
	if t := os.Getenv("GOVF_TRASH"); t != "" {
		return t
	}
	base := os.Getenv("XDG_DATA_HOME")
	if base == "" {
		h, _ := os.UserHomeDir()
		base = filepath.Join(h, ".local", "share")
	}
	return filepath.Join(base, "govf", "trash")
}

var trashSeq atomic.Int64

// trashPath: unique slot trash/<nanos>-<seq>/<name>; keeps original name.
func (a *App) trashPath(name string) string {
	slot := strconv.FormatInt(time.Now().UnixNano(), 36) + "-" + strconv.FormatInt(trashSeq.Add(1), 36)
	return filepath.Join(a.trash, slot, name)
}

// record pushes an op and clears redo history.
func (a *App) record(desc string, steps ...step) {
	if len(steps) == 0 {
		return
	}
	a.undoStack = append(a.undoStack, op{desc: desc, steps: steps})
	a.redoStack = nil
}

// trashTargets moves selection/cursor entries to trash; p then puts from trash.
func (a *App) trashTargets() {
	t := a.cur().Targets()
	if len(t) == 0 {
		return
	}
	var steps []step
	var trashed []string
	for _, p := range t {
		dst := a.trashPath(filepath.Base(p))
		if err := fsx.MoveTo(p, dst); err != nil {
			a.setErr(err)
			break
		}
		steps = append(steps, step{stepMove, p, dst})
		trashed = append(trashed, dst)
	}
	a.record("delete", steps...)
	if len(trashed) > 0 {
		a.clip = clipboard{paths: trashed, cut: true}
	}
	a.reloadAll()
	if !a.msgErr {
		a.setMsg("%d trashed (u undo, p put back)", len(trashed))
	}
}

// removeEmptyTrashSlot drops the per-item slot dir once its item is moved out.
func (a *App) removeEmptyTrashSlot(p string) {
	if d := filepath.Dir(p); filepath.Dir(d) == a.trash {
		os.Remove(d) // only succeeds when empty
	}
}

func (a *App) undo() {
	n := len(a.undoStack)
	if n == 0 {
		a.setMsg("nothing to undo")
		return
	}
	o := a.undoStack[n-1]
	a.undoStack = a.undoStack[:n-1]
	for i := len(o.steps) - 1; i >= 0; i-- {
		s := &o.steps[i]
		var err error
		switch s.kind {
		case stepMove:
			err = fsx.MoveTo(s.to, s.from)
			a.removeEmptyTrashSlot(s.to)
		case stepCopy, stepCreate:
			t := a.trashPath(filepath.Base(s.to))
			if err = fsx.MoveTo(s.to, t); err == nil {
				*s = step{stepMove, t, s.to} // redo = move back out of trash
			}
		}
		if err != nil {
			a.undoStack = append(a.undoStack, o)
			a.setErr(fmt.Errorf("undo %s: %v", o.desc, err))
			a.reloadAll()
			return
		}
	}
	a.redoStack = append(a.redoStack, o)
	a.clip = clipboard{}
	a.reloadAll()
	a.setMsg("undone: %s", o.desc)
}

func (a *App) redo() {
	n := len(a.redoStack)
	if n == 0 {
		a.setMsg("nothing to redo")
		return
	}
	o := a.redoStack[n-1]
	a.redoStack = a.redoStack[:n-1]
	for _, s := range o.steps {
		if err := fsx.MoveTo(s.from, s.to); err != nil {
			a.redoStack = append(a.redoStack, o)
			a.setErr(fmt.Errorf("redo %s: %v", o.desc, err))
			a.reloadAll()
			return
		}
		a.removeEmptyTrashSlot(s.from)
	}
	a.undoStack = append(a.undoStack, o)
	a.clip = clipboard{}
	a.reloadAll()
	a.setMsg("redone: %s", o.desc)
}

// emptyTrash permanently removes trash contents (confirm).
func (a *App) emptyTrash() {
	a.confirm("empty trash permanently?", func() {
		des, err := os.ReadDir(a.trash)
		if err != nil {
			a.setErr(err)
			return
		}
		for _, de := range des {
			if err := os.RemoveAll(filepath.Join(a.trash, de.Name())); err != nil {
				a.setErr(err)
				return
			}
		}
		a.undoStack, a.redoStack = nil, nil
		a.clip = clipboard{}
		a.reloadAll()
		a.setMsg("trash emptied")
	})
}
