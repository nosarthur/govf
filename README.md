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
| `yy` / `dd` / `p` | yank / cut / paste into current dir |
| `D`, `Del` | delete selection or cursor (confirm) |
| `cw` | rename |
| `za` | toggle dotfiles |
| `S` | sort menu: `n` name, `s` size, `t` time, `r` reverse (j/k + Enter also work) |
| `/`, `n`, `N` | search by substring, next / prev |
| `~` | home dir |
| `Ctrl-l` | reload both panels |
| `Esc` | clear selection / message |
| `q` | quit |

## Commands (`:`)

`q` `cd PATH` `mkdir NAME` `touch NAME` `rename NAME` `delete` `yank` `cut`
`paste` `sort name|size|time|reverse` `reverse` `hidden` `view` `sync` `help`

`mkdir`/`touch`/`rename` without an argument prompt for input.

## Preview

- Text: first screenful of lines, tabs expanded; binary files flagged.
- Images: png, jpeg, gif, bmp, tiff, webp. On iTerm2/WezTerm the file is sent
  with the OSC 1337 inline-image protocol (same as `imgcat`); on kitty the
  kitty graphics protocol. Elsewhere (or under tmux) images fall back to `▀`
  truecolor half-block cells. Override with `GOVF_IMAGES=iterm|kitty|blocks`.
  Large or non-png/jpeg/gif files are re-encoded as PNG at most 1600px.
