# Contributing to postx

Thanks for helping improve postx. This document describes how to work on the codebase and what we look for in contributions.

## Prerequisites

- **Go** (version in [`go.mod`](go.mod))
- A normal terminal with reasonable Unicode and color support (for TUI work)
- Optional: an X Developer account if you are testing live posting (see [README.md](README.md))

## Quick start

```bash
git clone <your-fork-or-repo-url>
cd postcli
go build -o postx ./cmd/postx
go test ./...
go vet ./...
```

Install into your `PATH`:

```bash
go install ./cmd/postx
```

## Project layout

| Path | Role |
| ---- | ---- |
| [`cmd/postx`](cmd/postx) | Program entrypoint |
| [`internal/cli`](internal/cli) | Cobra commands |
| [`internal/config`](internal/config) | Config paths (XDG, theme file, DB, tokens) |
| [`internal/store`](internal/store) | SQLite persistence and migrations |
| [`internal/xapi`](internal/xapi) | OAuth 2.0 and X API v2 client |
| [`internal/schedule`](internal/schedule) | Due-post runner (`flush` / `daemon`) |
| [`internal/theme`](internal/theme) | TUI color presets and persistence |
| [`internal/tui/post`](internal/tui/post) | Compose wizard (Bubble Tea) |
| [`internal/tui/status`](internal/tui/status) | Calendar status view (Bubble Tea) |
| [`internal/tui/channelsui`](internal/tui/channelsui) | Interactive channel browser (`postx channels`) |
| [`internal/channels`](internal/channels) | Channel catalog for the post wizard (X + preview destinations) |

## Tests and quality

Before opening a PR, run:

```bash
go test ./...
go vet ./...
go build ./...
```

Add or extend tests when you change behavior in **`internal/store`**, **`internal/schedule`**, **`internal/theme`**, or other logic-heavy packages. TUI packages may rely on manual testing; describe what you checked in the PR.

## Code style

- Match existing patterns in the file you edit (naming, error handling, imports).
- Prefer small, focused changes over large refactors unless discussed first.
- Use **`context.Context`** for I/O that crosses API boundaries where the codebase already does.
- Do not commit secrets, tokens, or real OAuth client credentials.

## TUI and themes

- Bubble Tea is **v2** (`charm.land/bubbletea/v2`). Alt screen is controlled via `tea.View` fields (for example `AltScreen`), not legacy program options.
- Shared colors come from **`internal/theme`**. New UI surfaces should read **`theme.Current()`** after **`theme.Load()`** (see existing `post` and `status` flows). Add new presets in [`internal/theme/theme.go`](internal/theme/theme.go) if you introduce another named theme.

## X API and OAuth

- Posting and login behavior depends on X’s current API products and errors (for example billing or tier limits). When changing error handling, prefer clear, user-facing messages and accurate HTTP error surfacing.
- **`POSTX_DRY_RUN=1`** is the supported way to exercise flows without calling X.

## Pull requests

1. Open an issue first for large or ambiguous changes, unless the fix is trivial.
2. One logical change per PR when possible.
3. In the PR description, explain **what** changed and **why**, and note any manual TUI or OAuth testing you performed.
4. Keep commits readable; maintainers may squash on merge.

## Security

If you find a security issue (for example token handling or unsafe defaults), please report it privately to the maintainers instead of filing a public issue with exploit details.

## License

By contributing, you agree your contributions will be licensed under the same terms as the project (see [LICENSE](LICENSE)).
