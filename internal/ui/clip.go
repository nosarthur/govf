package ui

import (
	"encoding/base64"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// clipTools: first found is used, in order.
var clipTools = [][]string{
	{"pbcopy"},
	{"wl-copy"},
	{"xclip", "-selection", "clipboard"},
	{"xsel", "--clipboard", "--input"},
}

func findClipTool() []string {
	for _, t := range clipTools {
		if _, err := exec.LookPath(t[0]); err == nil {
			return t
		}
	}
	return nil
}

// osc52 builds the terminal clipboard-set escape.
func osc52(s string) []byte {
	return []byte("\x1b]52;c;" + base64.StdEncoding.EncodeToString([]byte(s)) + "\a")
}

// copyText: OSC 52 to tty (reaches local clipboard over ssh) + local tool.
func (a *App) copyText(what, s string) {
	sent := false
	if a.raw != nil {
		if _, err := a.raw.Write(osc52(s)); err == nil {
			sent = true
		}
	}
	if len(a.clipTool) > 0 {
		cmd := exec.Command(a.clipTool[0], a.clipTool[1:]...)
		cmd.Stdin = strings.NewReader(s)
		if err := cmd.Run(); err == nil {
			sent = true
		} else if !sent {
			a.setErr(fmt.Errorf("%s: %v", a.clipTool[0], err))
			return
		}
	}
	if !sent {
		a.setMsg("no clipboard available")
		return
	}
	a.setMsg("copied %s: %s", what, s)
}

func (a *App) yankPath() {
	if e := a.cur().Current(); e != nil {
		a.copyText("path", e.Path)
	}
}

func (a *App) yankDir() { a.copyText("dir", filepath.Clean(a.cur().Dir)) }

func (a *App) yankName() {
	if e := a.cur().Current(); e != nil {
		a.copyText("name", e.Name)
	}
}
