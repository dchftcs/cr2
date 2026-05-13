# cr2

Bubble Tea terminal code review tool.

- `internal/domain`: review, diff, comment, and repository capability types
- `internal/gitrepo`: git repository adapters
- `internal/diffengine`: unified diff parser
- `internal/reviewapp`: application use cases
- `internal/ui/tui`: terminal adapter
- `internal/export`: markdown output

## Build

```bash
make
```

This writes the binary to `bin/cr2`.

## Run

```bash
bin/cr2
bin/cr2 --branch
bin/cr2 --unstaged
bin/cr2 HEAD~1
bin/cr2 -o REVIEW.md
```

## Keys

| Key | Action |
| --- | --- |
| `j` / `k`, arrows | Move in the diff; counts work, e.g. `5j` |
| `Ctrl+d` / `Ctrl+u` | Move half a page down/up |
| `PgUp` / `PgDn`, `Ctrl+b` / `Ctrl+f` | Page through the diff |
| `g` / `G`, `Home` / `End` | Jump to top/bottom; count + `G` jumps to a source line |
| `J` / `K`, `}` / `{` | Next/previous hunk |
| `h` / `l`, `]` / `[`, left/right arrows | Previous/next file; counts work |
| `/`, `n` / `N` | Search the diff and repeat next/previous match |
| `Tab` | Toggle side-by-side/unified diff view |
| `Ctrl+n` | Cycle line numbers between both, relative-only, and absolute-only |
| `c` | Add inline comment at the selected line |
| `v` | Start/clear a block selection for a range comment |
| `R` | Edit general review comment |
| `Ctrl+s` | Submit a comment while editing |
| `Esc` | Cancel comment/help/selection |
| `m` | Mark selected file read/unread |
| `a` | Stage/unstage selected file |
| `s` | Save review and exit |
| `?` | Help |
| `q` | Quit without saving |
