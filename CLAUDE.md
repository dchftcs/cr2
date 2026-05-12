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

## Design Rules

- Keep git behavior behind repository adapters. Do not shell out to git from UI or export code.
- Keep terminal UI state out of `domain`; domain types should be usable by a future Bubble Tea UI, web UI, or tests.
- Preserve useful edge cases in tests, especially unborn repositories, untracked files, staged files, and revision diffs.
- Prefer explicit capabilities over target-type checks. For example, use `CanStage` rather than assuming local/remote behavior.
- Markdown output should be deterministic: sort comments by file and line before rendering.

## Current UX

The terminal UI is Bubble Tea based. It currently has a split file-list/diff view and supports:

- `j` / `k`, arrows: move in the diff
- `PgUp` / `PgDn`: page through the diff
- `g` / `G`: jump to top/bottom
- `]` / `[`: next/previous file
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
- Side-by-side rendering, search, mouse support, and editor integration are intentionally deferred until the layered core is firmer.
