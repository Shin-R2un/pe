# Snippet file format (v1)

`pe` persists all snippets in a single JSON file at:

- `$PE_DIR/pe.json`, or
- `~/.pe/pe.json` (default)

The file is human-readable so you can `grep`, hand-edit, version-control,
or sync it as you like.

## Schema

```json
{
  "version": 1,
  "snippets": [
    {
      "key": "claude",
      "value": "claude --dangerously-skip-permissions",
      "description": "Claude Code, skip permission prompts",
      "tags": ["ai", "claude", "cli"],
      "createdAt": "2026-05-05T12:34:56Z",
      "updatedAt": "2026-05-05T12:34:56Z",
      "lastUsedAt": "2026-05-05T13:00:00Z",
      "useCount": 4
    }
  ]
}
```

### Fields

| Field         | Type             | Required | Notes                                                       |
| ------------- | ---------------- | -------- | ----------------------------------------------------------- |
| `version`     | int              | yes      | Format version. Currently `1`.                              |
| `snippets`    | array            | yes      | Array of snippet objects.                                   |
| `key`         | string           | yes      | Unique identifier. No whitespace. Cannot be a reserved word. |
| `value`       | string           | yes      | The text that gets copied to the clipboard.                 |
| `description` | string           | no       | Short human-friendly summary.                               |
| `tags`        | string[]         | no       | Optional tags for searching / filtering.                    |
| `createdAt`   | RFC 3339 (UTC)   | yes      | When the snippet was added.                                 |
| `updatedAt`   | RFC 3339 (UTC)   | yes      | When the snippet was last modified.                         |
| `lastUsedAt`  | RFC 3339 (UTC)   | no       | When `pe <key>` last copied it. `null` until first use.     |
| `useCount`    | int              | yes      | Number of times the snippet has been copied.                |

### Reserved keys

These words cannot be used as a `key` because they collide with subcommands:

```
a add l list ls s search find e edit d delete rm ? show
help version -h --help -v --version
```

## Compatibility commitment

Until `pe` reaches `v1.0.0`, breaking changes to this format may happen
between minor versions but will always include a clear migration path
in the release notes. After `v1.0.0`, the on-disk format will follow
SemVer compatibility rules.

## Writing semantics

Writes are atomic: `pe` writes to `pe.json.tmp` first and then
`rename(2)`s into place, so a crash mid-write leaves the previous file
intact.

The directory is created with `0700` and the file with `0600` (owner
read/write only) — snippets are treated as user-private even though the
format is plain text.

## Security note

`pe` does **not** encrypt this file. Do not store API keys, passwords,
tokens, or any secret material in it. For secrets, use a dedicated tool
such as [kpot](https://github.com/Shin-R2un/kpot).
