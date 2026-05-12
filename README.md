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

## Run

```bash
go run ./cmd/cr
go run ./cmd/cr --branch
go run ./cmd/cr --unstaged
go run ./cmd/cr HEAD~1
go run ./cmd/cr -o REVIEW.md
```

## Keys

| Key | Action |
| --- | --- |
| `j` / `k`, arrows | Move in the diff |
| `PgUp` / `PgDn` | Page through the diff |
| `g` / `G` | Jump to top/bottom |
| `]` / `[` | Next/previous file |
| `c` | Add inline comment at the selected line |
| `R` | Edit general review comment |
| `Ctrl+s` | Submit a comment while editing |
| `Esc` | Cancel comment/help |
| `m` | Mark selected file read/unread |
| `a` | Stage/unstage selected file |
| `s` | Save review and exit |
| `?` | Help |
| `q` | Quit without saving |
