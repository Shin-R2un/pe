# pe

`pe` — paste-friendly snippet & phrase manager.

A tiny CLI for registering frequently-used words, commands, boilerplate,
and AI prompt fragments, then searching and copying them to the
clipboard in one shot. The name comes from Japanese **「ぺっと貼る」**
(*pe-tto haru*, "paste it on").

## Status

Early scaffold. Subcommands print "not implemented yet" until the
core flows land.

## Install

```sh
go install github.com/Shin-R2un/pe/cmd/pe@latest
```

Or build from source:

```sh
git clone https://github.com/Shin-R2un/pe
cd pe
make build
./pe -v
```

## Planned usage

```sh
pe add                  # register a snippet (interactive)
pe ls                   # list snippets
pe find <query>         # search by title / body / tag
pe cp <id>              # copy snippet to clipboard
pe rm <id>              # delete a snippet
```

## Storage

Snippets live in a single JSON file:

- `$XDG_CONFIG_HOME/pe/snippets.json`, falling back to
- `~/.config/pe/snippets.json`

The file is human-readable, `grep`-friendly, and easy to back up or
sync via your preferred mechanism (git, Dropbox, etc.). Format spec:
[`docs/format.md`](docs/format.md).

## Clipboard backends

`pe` shells out to a native helper — no CGO, no third-party deps:

| OS              | Helper                              |
| --------------- | ----------------------------------- |
| macOS           | `pbcopy`                            |
| Linux (Wayland) | `wl-copy`                           |
| Linux (X11)     | `xclip` (preferred) or `xsel`       |
| Windows         | `clip`                              |

## License

MIT — see [LICENSE](LICENSE).
