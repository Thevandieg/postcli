# postx

Minimal terminal UI for composing, scheduling, and publishing social posts.
In `v1.0.0`, live publishing supports **X (Twitter)** only.

## Contents

- [postx](#postx)
  - [Contents](#contents)
  - [Why postx](#why-postx)
  - [Current platform support](#current-platform-support)
  - [Install](#install)
  - [Quick start](#quick-start)
  - [Commands](#commands)
  - [X setup (OAuth)](#x-setup-oauth)
  - [Environment variables](#environment-variables)
  - [Scheduler automation (systemd)](#scheduler-automation-systemd)
  - [Troubleshooting](#troubleshooting)
  - [Contributing](#contributing)
  - [License](#license)

## Why postx

- Keyboard-first flow for writing, scheduling, and publishing posts.
- Queue-based scheduler with `flush` and `daemon` modes.
- Theme support for visual preferences in terminal workflows.

## Current platform support

- **X (Twitter):** fully supported for live publishing.
- **Mastodon, Bluesky, Threads:** preview placeholders in `v1.0.0`.

## Install

Build from source:

```bash
go build -o postx ./cmd/postx
```

Install to your `PATH`:

```bash
go install ./cmd/postx
```

## Quick start

```bash
# 1) Build or install
go install ./cmd/postx

# 2) Configure X OAuth (interactive)
postx channels configure x

# 3) Create and publish/schedule a post
postx post

# 4) Check scheduled queue
postx status
```

Data is stored in `$XDG_CONFIG_HOME/postcli` (fallback `~/.config/postcli`):
`queue.db`, `oauth.json`, `env`, and `theme`.

## Commands

| Command | Description |
| --- | --- |
| `postx channels` | Browse channels; configure X or view preview-only channels |
| `postx channels configure x` | Interactive setup menu for client ID/secret, OAuth, and redirect URI |
| `postx post` | Compose flow: content type -> text/media -> channels -> post now or schedule |
| `postx status` | Calendar + details for scheduled posts |
| `postx flush` | Process due posts once (good for cron/systemd) |
| `postx daemon` | Poll on an interval and process due posts continuously |
| `postx cancel ID` | Soft-cancel a pending queued post |
| `postx logout` | Remove stored OAuth tokens |
| `postx theme` | Show active theme and available theme commands |
| `postx theme ls` | List themes (`violet`, `sky`, `orange`, `neutral`, `green`) |
| `postx theme set NAME` | Persist selected theme under config dir |

Status view navigation:

- Day: `left/right` or `h/l`
- Week: `up/down` or `j/k`
- Month: `[` and `]`
- Jump to today (UTC): `t`

## X setup (OAuth)

1. Create an app in the [X developer portal](https://developer.twitter.com/).
2. Enable OAuth 2.0 and copy your client credentials.
3. Add redirect URI (default: `http://127.0.0.1:8080/callback`).
4. Ensure scopes include: `tweet.read`, `tweet.write`, `users.read`,
   and `offline.access`.
5. Run `postx channels configure x` and complete login.

`postx channels configure x` supports:

- full setup flow,
- updating only client ID,
- updating only client secret,
- rerunning OAuth only,
- updating redirect URI only.

## Environment variables

| Variable | Description |
| --- | --- |
| `POSTX_CLIENT_ID` | OAuth 2.0 client ID |
| `POSTX_CLIENT_SECRET` | OAuth 2.0 client secret (required in most setups) |
| `POSTX_REDIRECT_URI` | OAuth callback URI (default `http://127.0.0.1:8080/callback`) |
| `POSTX_DRY_RUN` | If `1` or `true`, skips API calls and logs payloads |

## Scheduler automation (systemd)

Example user service (`~/.config/systemd/user/postx-flush.service`):

```ini
[Unit]
Description=postx flush due posts

[Service]
Type=oneshot
EnvironmentFile=%h/.config/postcli/env
ExecStart=/path/to/postx flush
```

Example user timer (`~/.config/systemd/user/postx-flush.timer`):

```ini
[Unit]
Description=Run postx flush every minute

[Timer]
OnBootSec=1min
OnUnitActiveSec=1min
Unit=postx-flush.service

[Install]
WantedBy=timers.target
```

Enable it:

```bash
systemctl --user daemon-reload
systemctl --user enable --now postx-flush.timer
```

## Troubleshooting

- **`Missing POSTX_CLIENT_ID`**
  Set `POSTX_CLIENT_ID` and retry.
- **`Missing POSTX_CLIENT_SECRET`**
  Set `POSTX_CLIENT_SECRET` and retry.
- **`You are not logged in`**
  Run `postx channels configure x`.
- **`401 unauthorized`**
  Re-check client credentials and redo OAuth login.
- **`402 payment required`**
  Your X project may need billing-enabled API access.
- **`403 forbidden`**
  Verify app permissions include `tweet.write`.
- **`429 rate limit`**
  Wait and retry later.

WSL2 note:
If login appears stuck, keep the terminal open until browser redirect completes.
The callback listener binds to `0.0.0.0:port` for WSL2 compatibility.

Media note:
Simple image upload is supported; chunked large media/video upload is not in
`v1.0.0`.

## Contributing

Please read [`CONTRIBUTING.md`](CONTRIBUTING.md) before opening pull requests.

## License

MIT. See [`LICENSE`](LICENSE).
