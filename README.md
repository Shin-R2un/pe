# pe

A tiny CLI snippet launcher for **「ぺっと貼る」**.

> *pe means "ぺっと貼る" — copy a saved snippet and paste it instantly.*

## Concept

`pe` is **not** a text expander.
`pe` is **not** a clipboard history manager.
`pe` is a command-first snippet launcher: **search, copy, and paste.**

The name comes from Japanese **ぺっと貼る** (*pe-tto haru*, "paste it on").
Register frequently-used commands, prompts, URLs, and boilerplate under
short keys, then call them up with a single command.

```sh
pe a claude "claude --dangerously-skip-permissions"
pe claude
# → copied: claude
```

## Install

### Linux / macOS — `go install`

`pe` is a single statically-linked Go binary. With Go 1.18+ on PATH:

```sh
go install github.com/Shin-R2un/pe/cmd/pe@latest
```

The binary lands at `$(go env GOBIN)` / `$(go env GOPATH)/bin/pe`
(typically `~/go/bin/pe`). Make sure that directory is on your `PATH`.

### Windows — Scoop

```powershell
scoop bucket add r2un https://github.com/Shin-R2un/scoop-bucket
scoop install r2un/pe
```

The same bucket also hosts [kpot](https://github.com/Shin-R2un/kpot).

### Pre-built binaries

Each tagged release ships archives for linux/macOS (amd64 + arm64) and
windows (amd64) on the [Releases page](https://github.com/Shin-R2un/pe/releases),
along with a `checksums.txt`.

### From source

```sh
git clone https://github.com/Shin-R2un/pe
cd pe
make build
./pe version
```

### On multiple machines

`pe`'s home is in `~/.pe/pe.json`, so each machine has its own snippet
set unless you point them all at the same dir:

```sh
export PE_DIR=~/dotfiles/pe   # everywhere — sync via Git/Dropbox/etc.
```

For headless servers reached over SSH, the OSC 52 fallback means `pe`
copies to **your** local clipboard — no `xclip`/`wl-copy` needed on the
remote. See [OSC 52 over SSH/tmux](#osc-52-over-sshtmux).

### Updating

Use the same channel you installed from:

| Installed via | Update command           |
| ------------- | ------------------------ |
| `go install`  | `pe update`              |
| Scoop         | `scoop update pe`        |
| Source        | `git pull && make install` |
| Release zip   | redownload the new release |

`pe update` wraps `go install github.com/Shin-R2un/pe/cmd/pe@latest`, so
it needs `go` on the target. It reports the old → new version when it
finishes:

```
$ pe update
running: go install github.com/Shin-R2un/pe/cmd/pe@latest
updated: 0.2.0 → 0.3.0
```

If your machine is behind the public Go module proxy and you've just
pushed a release, the proxy can return a stale 404 for ~30 minutes.
Bypass it with:

```sh
GOPROXY=direct pe update
```

If the target has no `go` toolchain, `pe update` will tell you so —
either install Go (https://go.dev/dl/) or `scoop update pe` on Windows,
or grab a release binary from the Releases page.

### Clipboard backend

`pe` first tries a native helper:

| OS              | Helper                              |
| --------------- | ----------------------------------- |
| macOS           | `pbcopy` (built in)                 |
| Linux (Wayland) | `wl-copy`                           |
| Linux (X11)     | `xclip` or `xsel`                   |
| WSL             | the above, or `clip.exe` fallback   |
| Windows         | `clip` (built in)                   |

If no native helper is found (typical on a headless box reached over
SSH), `pe` falls back to **OSC 52** — an escape sequence that the
terminal emulator forwards to the local clipboard. See [OSC 52 over
SSH/tmux](#osc-52-over-sshtmux) below.

`PE_CLIP` overrides the strategy:

| Value          | Behavior                                          |
| -------------- | ------------------------------------------------- |
| _(unset)_      | auto: native first, OSC 52 fallback (default)     |
| `native`       | only native; fail if absent                       |
| `osc52`        | force OSC 52 (skip native helpers entirely)       |
| `none`         | no-op (useful for testing)                        |

### OSC 52 over SSH/tmux

Working on a remote machine over SSH and want `pe` to put the snippet on
**your local** clipboard? `pe` can do that with no helper installed on
the remote — it just emits the OSC 52 escape sequence and lets the
terminal emulator handle it.

What you need:

1. **A terminal emulator that supports OSC 52.** WezTerm, iTerm2, kitty,
   Alacritty (with `enable_kitty_keyboard = true` not required for OSC 52
   itself, but check the docs), Windows Terminal, modern xterm, etc.
   Most do, often by default.
2. **If you're inside tmux**, enable passthrough so tmux forwards the
   sequence to the outer terminal. Add to `~/.tmux.conf`:
   ```tmux
   set -g allow-passthrough on
   ```
   Then reload (`tmux source-file ~/.tmux.conf`) or restart tmux.
3. **Test it:**
   ```sh
   PE_CLIP=osc52 pe claude
   ```
   You should see `copied: claude` and the value should be on your local
   clipboard. If nothing pastes, recheck step 2 and the terminal's OSC 52
   setting.

In auto mode, `pe` only falls back to OSC 52 when no native helper is
available, so installing `xclip` on a workstation will silently take
priority. Set `PE_CLIP=osc52` if you want to force OSC 52 even when a
local helper exists.

## Quick Start

```sh
pe a claude "claude --dangerously-skip-permissions"
pe a codex  "codex --resume"
pe a clawd  "cd /mnt/s3vo/openclaw && claude"

pe claude          # copy & paste anywhere
pe l               # list snippets
pe s cla           # find by substring
pe ? claude        # show full contents
pe e claude        # edit in $EDITOR
pe d claude        # delete

pe                 # interactive fuzzy picker
```

## Commands

| Short          | Long                  | Description                              |
| -------------- | --------------------- | ---------------------------------------- |
| `pe <key>`     | —                     | Copy snippet to clipboard                |
| `pe`           | —                     | Interactive search (fuzzy filter)        |
| `pe a K T`     | `pe add K T`          | Add a snippet                            |
| `pe l`         | `pe list`             | List snippets (one line each)            |
| `pe s Q`       | `pe search Q`         | Search by key / description / value / tag |
| `pe ? K`       | `pe show K`           | Show full contents of a snippet          |
| `pe e K`       | `pe edit K`           | Edit a snippet in your `$EDITOR`         |
| `pe d K`       | `pe delete K`         | Delete a snippet                         |
| `pe completion <sh>` | —               | Print bash / zsh / fish tab-completion   |
| `pe update`    | —                     | Reinstall latest release via `go install` |
| `pe help`      | `pe -h` / `pe --help` | Print usage                              |
| `pe version`   | `pe -v`               | Print version                            |

Output is intentionally short:

```
$ pe a claude "claude --dangerously-skip-permissions"
added: claude

$ pe claude
copied: claude

$ pe d claude
deleted: claude
```

For unknown keys, `pe` shows a "did you mean" line listing the closest
existing keys (prefix > substring > Levenshtein, top 3):

```
$ pe clude
not found: clude
did you mean: claude?
```

If nothing is close enough, it falls back to suggesting `pe s`:

```
$ pe nope
not found: nope
hint: try `pe s nope`
```

## Tab completion

`pe completion <shell>` emits a completion script for **bash**, **zsh**,
or **fish**. Once installed, Tab on `pe <key>` and `pe e <key>` /
`pe d <key>` / `pe ? <key>` completes against your registered keys.

```sh
# zsh
mkdir -p ~/.zsh/completions
pe completion zsh > ~/.zsh/completions/_pe
# add to ~/.zshrc once:
#   fpath=(~/.zsh/completions $fpath)
#   autoload -Uz compinit && compinit

# bash
pe completion bash > ~/.local/share/bash-completion/completions/pe

# fish
pe completion fish > ~/.config/fish/completions/pe.fish
```

Then reload your shell. `pe cla<TAB>` should expand to all keys
starting with `cla`.

## Editor for `pe e`

`pe e <key>` opens the snippet in your editor as JSON. Resolution order:

1. `$PE_EDITOR` (pe-specific override)
2. `$VISUAL`
3. `$EDITOR`
4. `vi` (Linux/macOS) / `notepad` (Windows)

If your global `$VISUAL` is set to something you don't want for pe
(e.g. `$VISUAL=nano` but you want vim for code-like editing), set
`PE_EDITOR` in your shell rc:

```sh
echo 'export PE_EDITOR=vim' >> ~/.zshrc
```

You can edit `key`, `value`, `description`, and `tags`. Metadata
(`createdAt`, `useCount`, `lastUsedAt`) is preserved automatically.

## Examples

```sh
# AI / agent commands
pe a claude  "claude --dangerously-skip-permissions"
pe a codex   "codex --resume"
pe a clawd   "cd /mnt/s3vo/openclaw && claude"

# Git
pe a rebase  "git fetch upstream && git rebase upstream/main"

# SSH
pe a vps     "ssh root@example.com"

# Frequently-pasted prompts
pe a review  "Please review this code for bugs and clarity:"
pe a fmtjp   "Translate the following to formal Japanese:"

# Long URLs
pe a docs    "https://docs.example.com/very/long/path"
```

## Storage

Snippets live in a single JSON file:

- `$PE_DIR/pe.json` if `PE_DIR` is set, otherwise
- `~/.pe/pe.json` (default).

File permissions are `0600`, the directory `0700`. The format is
human-readable, `grep`-friendly, and easy to back up or sync via your
preferred mechanism (Git, Dropbox, etc.). Format spec:
[`docs/format.md`](docs/format.md).

```sh
PE_DIR=$HOME/dotfiles/pe pe l   # use a synced dotfiles dir
```

## Security Notes

**Do not store API keys, passwords, or tokens in `pe`.** The snippet
file is plain JSON on disk, mode `0600`, but it is **not encrypted**.
Use a secret manager such as [kpot](https://github.com/Shin-R2un/kpot)
for credentials.

`pe` deliberately does not display snippet values during a copy
operation, to reduce the risk of leaking long commands or prompts to
your terminal scrollback / log files. Use `pe ? <key>` when you
explicitly want to see the contents.

## Roadmap

Likely next steps (not yet implemented):

- `pe a <key> --editor` for multi-line prompt registration
- `pe a <key> -d "..." -t tag1,tag2` for description/tags via flags
- `pe tag <tag>` to filter by tag
- `pe export` / `pe import` for migration between machines
- `pe stats` / `pe pin` for usage-aware ordering
- richer fuzzy search in interactive mode (AND, regex, non-ASCII keys)

Out of scope (use a different tool):

- secret management → use `kpot`
- text expansion / shortcuts → use `espanso`, `Alfred`, etc.
- clipboard history → use `cliphist`, `Maccy`, etc.

## License

MIT — see [LICENSE](LICENSE).
