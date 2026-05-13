# CLAUDE.md

## Project

`cr2` is a clean-room Go reimplementation of a terminal code review tool for reviewing git diffs and exporting markdown feedback that coding agents can act on.

## Architecture

Keep the application layered:

- `cmd/cr`: CLI argument parsing and process startup only.
- `internal/domain`: stable review, diff, comment, status, and capability types. This package should not import infrastructure or UI packages.
- `internal/gitrepo`: repository adapters. Local git is implemented here; future SSH/container adapters should satisfy the same application-facing interface.
- `internal/diffengine`: unified diff parsing into domain objects. Avoid UI-specific row concepts here.
- `internal/reviewapp`: application use cases such as loading a review, toggling stage state, and refreshing status.
- `internal/ui/tui`: terminal interaction. This should stay a thin adapter over `reviewapp` and `domain`.
- `internal/export`: markdown or other output formats from domain state.

## Development Commands

```bash
go test ./...
go build ./cmd/cr
go run ./cmd/cr --help
```

## Dependency Policy

Minimize external dependencies. At this stage the project should mainly use the Go standard library plus the Bubble Tea ecosystem needed for the terminal UI.

When introducing any new external dependency:

- Prefer the standard library first.
- Reuse the existing Bubble Tea/Charmbracelet stack for TUI behavior instead of adding a second UI framework.
- Add the dependency only when it removes substantial implementation risk or complexity.
- Update this section with the new dependency, why it is needed, and whether it is direct or transitive.

Direct external imports currently used by source code:

- `github.com/charmbracelet/bubbletea`: Bubble Tea TUI runtime.
- `github.com/charmbracelet/bubbles/textarea`: textarea component for comment editing.
- `github.com/charmbracelet/lipgloss`: terminal styling.

External modules currently listed in `go.mod`:

- `github.com/atotto/clipboard`
- `github.com/aymanbagabas/go-osc52/v2`
- `github.com/charmbracelet/bubbles`
- `github.com/charmbracelet/bubbletea`
- `github.com/charmbracelet/colorprofile`
- `github.com/charmbracelet/lipgloss`
- `github.com/charmbracelet/x/ansi`
- `github.com/charmbracelet/x/cellbuf`
- `github.com/charmbracelet/x/term`
- `github.com/clipperhouse/displaywidth`
- `github.com/clipperhouse/stringish`
- `github.com/clipperhouse/uax29/v2`
- `github.com/erikgeiser/coninput`
- `github.com/lucasb-eyer/go-colorful`
- `github.com/mattn/go-isatty`
- `github.com/mattn/go-localereader`
- `github.com/mattn/go-runewidth`
- `github.com/muesli/ansi`
- `github.com/muesli/cancelreader`
- `github.com/muesli/termenv`
- `github.com/rivo/uniseg`
- `github.com/xo/terminfo`
- `golang.org/x/sys`
- `golang.org/x/text`

## Design Rules

- Keep git behavior behind repository adapters. Do not shell out to git from UI or export code.
- Keep terminal UI state out of `domain`; domain types should be usable by a future Bubble Tea UI, web UI, or tests.
- Preserve useful edge cases in tests, especially unborn repositories, untracked files, staged files, and revision diffs.
- Prefer explicit capabilities over target-type checks. For example, use `CanStage` rather than assuming local/remote behavior.
- Markdown output should be deterministic: sort comments by file and line before rendering.

## Current UX

The terminal UI is Bubble Tea based. It currently has a split file-list/diff view and supports:

- `j` / `k`, arrows: move in the diff; numeric counts work
- `Ctrl+d` / `Ctrl+u`: move half a page down/up
- `PgUp` / `PgDn`, `Ctrl+b` / `Ctrl+f`: page through the diff
- `g` / `G`, `Home` / `End`: jump to top/bottom; count + `G` jumps to a source line
- `J` / `K`, `}` / `{`: next/previous hunk
- `h` / `l`, `]` / `[`: next/previous file; numeric counts work
- `/`, `n` / `N`: search the diff and repeat next/previous match
- `Tab`: toggle side-by-side/unified diff view
- `Ctrl+n`: cycle line numbers between both, relative-only, and absolute-only
- `c`: add an inline comment at the selected line
- `R`: edit the review-level comment
- `Ctrl+s`: submit comment while editing
- `m`: mark selected file read/unread
- `a`: stage/unstage selected file
- `s`: save review and exit
- `?`: help
- `q`: quit without saving

Future UI work should improve ergonomics without moving git, diff parsing, or export logic into the UI layer.

## Known Constraints

- The first implementation supports local git only.
- There is no persistent review-session file format yet.
- Mouse support and editor integration are intentionally deferred until the layered core is firmer.
