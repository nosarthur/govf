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
| `sn` `ss` `st` `sr` | sort by name / size / time; reverse |
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
- Images: png, jpeg, gif, bmp, tiff, webp scaled to fit and drawn with `▀`
  truecolor cells (needs a truecolor terminal).
