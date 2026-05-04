# Snippet file format (v1)

`pe` persists all snippets in a single JSON file at:

- `$XDG_CONFIG_HOME/pe/snippets.json`, or
- `~/.config/pe/snippets.json` (fallback)

The file is human-readable so you can `grep`, hand-edit, version-control,
or sync it as you like.

## Schema

```json
{
  "version": 1,
  "snippets": [
    {
      "id": "abc123",
      "title": "git: rebase onto upstream/main",
      "body": "git fetch upstream && git rebase upstream/main",
      "tags": ["git", "rebase"],
      "created_at": "2026-05-05T12:34:56Z",
      "updated_at": "2026-05-05T12:34:56Z"
    }
  ]
}
```

### Fields

| Field        | Type      | Required | Notes                                           |
| ------------ | --------- | -------- | ----------------------------------------------- |
| `version`    | int       | yes      | Format version. Currently `1`.                  |
| `snippets`   | array     | yes      | Array of snippet objects.                       |
| `id`         | string    | yes      | Stable opaque identifier. Short, URL-safe.      |
| `title`      | string    | yes      | Short human-friendly label.                     |
| `body`       | string    | yes      | The text that gets copied to the clipboard.     |
| `tags`       | string[]  | no       | Optional tags for searching / filtering.        |
| `created_at` | RFC 3339  | yes      | UTC timestamp when added.                       |
| `updated_at` | RFC 3339  | yes      | UTC timestamp when last modified.               |

## Compatibility commitment

Until `pe` reaches `v1.0.0`, breaking changes to this format may happen
between minor versions but will always include a clear migration path
in the release notes. After `v1.0.0`, the on-disk format will follow
SemVer compatibility rules.

## Writing semantics

Writes are atomic: `pe` writes to `snippets.json.tmp` first and then
`rename(2)`s into place, so a crash mid-write leaves the previous file
intact.

File permissions are `0600` (owner read/write only) — snippets are
treated as user-private even though the format is plain text.
