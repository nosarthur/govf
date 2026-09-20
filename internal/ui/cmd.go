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
			e := a.cur().Current()
			if e != nil {
				old := e.Path
				if err := fsx.Rename(old, joinDir(old, arg)); err != nil {
					a.setErr(err)
					return
				}
				a.reloadAll()
				a.cur().SeekName(arg)
			}
		}
	case "delete", "d":
		a.deleteTargets()
	case "yank", "y":
		a.yank(false)
	case "cut":
		a.yank(true)
	case "paste":
		a.paste()
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
		a.setMsg("j/k h/l gg/G Tab w t yy dd p D cw za S(sort) / n N :q")
	default:
		a.setMsg("unknown command: %s", name)
	}
}
