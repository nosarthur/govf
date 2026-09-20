package ui

import (
	"strings"

	"github.com/nosarthur/govf/internal/fsx"
)

// runCommand executes a ":" command line.
func (a *App) runCommand(line string) {
	line = strings.TrimSpace(line)
	if line == "" {
		return
	}
	name, arg, _ := strings.Cut(line, " ")
	arg = strings.TrimSpace(arg)
	switch name {
	case "q", "quit", "exit":
		a.quit = true
	case "cd":
		a.cd(arg)
	case "mkdir":
		a.mkdir(arg)
	case "touch":
		a.touch(arg)
	case "rename":
		if arg == "" {
			a.rename()
		} else {
			if e := a.cur().Current(); e != nil {
				a.renameTo(e.Path, arg)
			}
		}
	case "delete", "d":
		a.trashTargets()
	case "yank", "y":
		a.yank(false)
	case "paste":
		a.paste()
	case "undo":
		a.undo()
	case "redo":
		a.redo()
	case "empty":
		a.emptyTrash()
	case "trash":
		a.cd(a.trash)
	case "sort":
		if k, ok := fsx.ParseSortKey(arg); ok {
			a.setSort(k)
		} else if arg == "r" || arg == "reverse" {
			a.toggleReverse()
		} else {
			a.setMsg("sort: name|size|time|reverse")
		}
	case "reverse", "invert":
		a.toggleReverse()
	case "hidden":
		a.toggleHidden()
	case "view", "preview":
		a.preview = !a.preview
	case "sync":
		a.other().Load(a.cur().Dir)
	case "help", "h":
		a.setMsg("j/k h/l gg/G Tab w t yy dd p u C-r DD cw a A za S(sort) / n N :q")
	default:
		a.setMsg("unknown command: %s", name)
	}
}
