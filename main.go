// govf: minimal vifm-like two-panel file manager.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gdamore/tcell/v2"

	"github.com/nosarthur/govf/internal/preview"
	"github.com/nosarthur/govf/internal/ui"
)

const version = "0.1.0"

func main() {
	showVer := flag.Bool("version", false, "print version")
	chooseDir := flag.String("choose-dir", "", "on quit write active panel dir to FILE ('-' = stdout)")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: govf [-version] [LEFT_DIR [RIGHT_DIR]]\n")
		flag.PrintDefaults()
	}
	flag.Parse()
	if *showVer {
		fmt.Println("govf", version)
		return
	}
	cwd, _ := os.Getwd()
	left, right := cwd, cwd
	if flag.NArg() > 0 {
		left = absDir(flag.Arg(0))
		right = left
	}
	if flag.NArg() > 1 {
		right = absDir(flag.Arg(1))
	}
	scr, err := tcell.NewScreen()
	if err != nil {
		fatal(err)
	}
	if err := scr.Init(); err != nil {
		fatal(err)
	}
	defer scr.Fini()
	app := ui.New(scr, left, right)
	if ts, ok := scr.(interface{ Tty() (tcell.Tty, bool) }); ok {
		if tty, ok := ts.Tty(); ok {
			app.SetTerminal(tty, preview.DetectProtocol(os.Getenv))
		}
	}
	app.Run()
	scr.Fini()
	if *chooseDir != "" {
		if err := writeChosenDir(*chooseDir, app.CurrentDir()); err != nil {
			fatal(err)
		}
	}
}

func writeChosenDir(dst, dir string) error {
	if dst == "-" {
		_, err := fmt.Println(dir)
		return err
	}
	return os.WriteFile(dst, []byte(dir+"\n"), 0o644)
}

func absDir(p string) string {
	if a, err := filepath.Abs(p); err == nil {
		return a
	}
	return p
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "govf:", err)
	os.Exit(1)
}
