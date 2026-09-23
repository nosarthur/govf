# govf

Minimal vifm-like two-panel terminal file manager in Go.

```
go build -o govf . && ./govf [LEFT_DIR [RIGHT_DIR]]
```

Two panels; by default the inactive panel previews the file under the cursor
(text, image via half-block cells, dir listing). `w` turns preview off to get
two independent panels.

## Keys

| Key | Action |
| --- | --- |
| `j`/`k`, arrows | move cursor |
| `h`/`l`, `Enter`, `Backspace` | parent / enter dir (Enter on file opens `$EDITOR` or `open`/`xdg-open`) |
| `gg`/`G`, `Ctrl-d/u`, `Ctrl-f/b`, PgUp/PgDn | jump / page |
| `Tab` | switch panel |
| `w` | toggle preview panel |
| `t`, `Space` | toggle selection (Space also moves down) |
| `yy` / `p` | yank / paste (copy) into current dir |
| `dd` | delete to trash (`~/.local/share/govf/trash`, or `$GOVF_TRASH`); `p` afterwards puts it back into current dir |
| `x` | move selection / cursor entry to the other panel's dir (like vifm `:move`) |
| `u` / `Ctrl-r` | undo / redo last file op (dd, p, x, rename, mkdir, touch) |
| `DD`, `D`, `Del` | delete permanently (confirm, not undoable) |
| `yf` / `yd` / `yn` | copy full path / current dir path / file name to system clipboard |
| `A`, `cw` | rename, editing full name incl. extension |
| `a` | rename, editing name without its extension |
| `cc` | rename, starting from an empty name |
| `za` | toggle dotfiles |
| `S` | sort menu: `n` name, `s` size, `t` time, `r` reverse (j/k + Enter also work). Default: time, newest first. Right column shows mtime under time sort, size otherwise |
| `/`, `n`, `N` | search by substring, next / prev |
| `~` | home dir |
| `Ctrl-l` | reload both panels |
| `Esc` | clear selection / message |
| `q` | quit |

## Commands (`:`)

`q` `cd PATH` `mkdir NAME` `touch NAME` `rename NAME` `delete` (to trash)
`yank` `paste` `move` / `copy` (to other panel) `undo` `redo` `empty` (trash)
`trash` (cd there)
`sort name|size|time|reverse` `reverse` `hidden` `view` `sync` `help`

`mkdir`/`touch`/`rename` without an argument prompt for input. In prompts,
Enter confirms (empty is rejected) and Esc cancels; Backspace never closes them.

## Clipboard

`yf`/`yd`/`yn` send an OSC 52 escape (reaches your local clipboard over ssh in
iTerm2/kitty/WezTerm; iTerm2 needs Prefs → General → Selection → "Applications
in terminal may access clipboard") and also pipe to `pbcopy`/`wl-copy`/`xclip`/
`xsel` if one exists.

## Preview

- Text: first screenful of lines, tabs expanded; binary files flagged.
- Images: png, jpeg, gif, bmp, tiff, webp. On iTerm2/WezTerm the file is sent
  with the OSC 1337 inline-image protocol (same as `imgcat`); on kitty the
  kitty graphics protocol. Elsewhere (or under tmux) images fall back to `▀`
  truecolor half-block cells. Override with `GOVF_IMAGES=iterm|kitty|blocks`.
  Large or non-png/jpeg/gif files are re-encoded as PNG at most 1600px.
