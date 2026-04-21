# postx

CLI for scheduling and posting to **X** using the X API v2, with a [Bubble Tea](https://github.com/charmbracelet/bubbletea) + [Bubbles](https://github.com/charmbracelet/bubbles) UI.

## Commands

| Command | Description |
| -------- | ------------- |
| `postx channels` | Interactive list: navigate channels, **Enter** to configure (X) or see preview-only notice |
| `postx channels configure [x]` | Non-interactive configure: credentials, persist to `env`, shell profile, then login |
| `postx login` | OAuth 2.0 with PKCE (opens browser; local callback server) |
| `postx logout` | Remove stored tokens |
| `postx status` | Calendar + detail pane for scheduled posts |
| `postx post` | Interactive flow: content type → compose → **channels** → schedule / post now (X live; others preview) |
| `postx flush` | Process all due posts once (for cron or systemd) |
| `postx daemon` | Poll on an interval and run `flush` logic |
| `postx cancel ID` | Soft-cancel a **pending** post |
| `postx theme` | Show the active TUI theme and list presets |
| `postx theme ls` | List themes (`violet`, `sky`, `orange`, `neutral`, `green`) |
| `postx theme set NAME` | Save a theme to `$XDG_CONFIG_HOME/postcli/theme` (aliases: `blue`→sky, `pink`→violet, …) |

In **`postx status`**: **←/→** or **h/l** moves the selected day by one; **↑/↓** or **j/k** moves by one week; **`[` / `]`** changes month; **`t`** jumps to today (UTC).

Data lives under `$XDG_CONFIG_HOME/postcli` (fallback: `~/.config/postcli`): `queue.db`, `oauth.json`, `env`.

**`postx post` steps:** choose **content type** (text-only or text + image), write the body, pick an image if needed, then pick one or more **channels** (only **X** publishes today; Mastodon, Bluesky, and Threads are preview placeholders), then choose immediate post or a scheduled time. Each selected channel gets its **own queue row** (same text, time, and media path).

## Environment

| Variable | Meaning |
| ---------- | --------- |
| `POSTX_CLIENT_ID` | OAuth 2.0 client ID from the X developer portal |
| `POSTX_CLIENT_SECRET` | **Often required:** X’s token URL expects an `Authorization: Basic` header. Use the **OAuth 2.0 Client Secret** from your app (not the old API Key Secret unless that’s what the portal shows for OAuth 2). If login still fails with `401` / `invalid_client`, ensure this matches the portal exactly. |
| `POSTX_REDIRECT_URI` | Default `http://127.0.0.1:8080/callback` — must match the app settings exactly |
| `POSTX_DRY_RUN` | If `1` / `true`, log tweet payloads and skip HTTP (no API calls) |

`postx channels configure x` stores these values in `$XDG_CONFIG_HOME/postcli/env` and also writes them to your active shell profile (`.zshrc`, `.bashrc`, or fish config) so future sessions pick them up automatically.

## X developer setup

1. Create a project and app in the [X developer portal](https://developer.twitter.com/).
2. Enable **OAuth 2.0** with type **Native App** or **Web** as appropriate; note the **Client ID** (and **Client Secret** if confidential).
3. Add the redirect URL you will use (default `http://127.0.0.1:8080/callback`) under the app’s callback / redirect list.
4. Request user-auth scopes that allow posting, for example: `tweet.read`, `tweet.write`, `users.read`, `offline.access` (for refresh tokens).

Posting requires API access that allows creating tweets (per X’s current product tiers).

## Build

```bash
go build -o postx ./cmd/postx
```

## Troubleshooting posting

When `postx post`, `postx flush`, or `postx daemon` cannot publish to X, use these messages:

- **`Missing POSTX_CLIENT_ID`**  
  Set `POSTX_CLIENT_ID` to your OAuth 2.0 client ID from the X portal.
- **`Missing POSTX_CLIENT_SECRET`**  
  Set `POSTX_CLIENT_SECRET` to your app’s OAuth 2.0 client secret.
- **`You are not logged in`**  
  Run `postx channels configure x` (or `postx login`) then retry posting.
- **`X rejected this request as unauthorized (401)`**  
  Re-check `POSTX_CLIENT_ID` / `POSTX_CLIENT_SECRET` and run `postx login` again.
- **`X API returned 402 Payment Required`**  
  Your X API project likely needs billing/payment enabled or a higher tier; add payment method in the X portal and retry.
- **`X rejected this request (403 Forbidden)`**  
  Confirm your app has `tweet.write` and the required product access.
- **`X rate limit reached (429)`**  
  Wait and retry later.

On immediate posting, success feedback now includes the tweet ID:

- `Success: posted #<local_post_id> to X (tweet <tweet_id>).`

## systemd user timer (flush every minute)

Replace `/path/to/postx` and ensure `POSTX_CLIENT_ID` (and other env vars) are available to the service (e.g. an `EnvironmentFile`).

`~/.config/systemd/user/postx-flush.service`:

```ini
[Unit]
Description=postx flush due posts

[Service]
Type=oneshot
EnvironmentFile=%h/.config/postcli/env
ExecStart=/path/to/postx flush
```

`~/.config/systemd/user/postx-flush.timer`:

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

Then:

```bash
systemctl --user daemon-reload
systemctl --user enable --now postx-flush.timer
```

## Notes

- **`postx login` seems to hang:** The terminal waits until your browser completes the redirect. **WSL2:** the callback server binds **`0.0.0.0:port`** (not only `127.0.0.1`) so traffic from a **Windows** browser to `http://127.0.0.1:8080/callback` can reach the Linux process after OS port forwarding. Your **redirect URI in the X portal** must still be exactly `http://127.0.0.1:8080/callback` (or whatever you set). The command prints the authorize URL and uses a **5-minute timeout** (`postx login --timeout 10m` to change).
- Scheduled times in the **post** wizard use **local** time (`2006-01-02 15:04`); they are stored in UTC in the database.
- **Media**: small images are uploaded via `upload.twitter.com` v1.1 simple upload, then attached to a v2 tweet. Large video or chunked upload is not implemented here.
- If `login` fails with redirect or TLS issues, confirm the redirect URI in the portal matches `POSTX_REDIRECT_URI` exactly (including host, port, and path).
