package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/Shin-R2un/pe/internal/editor"
	"github.com/Shin-R2un/pe/internal/store"
)

// editForm is the JSON payload presented to the user in the editor.
// Only fields users should edit are included; metadata (createdAt,
// useCount, etc.) is preserved transparently.
type editForm struct {
	Key         string   `json:"key"`
	Value       string   `json:"value"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
}

const editFormHint = `// Edit the JSON below. Save and exit.
// 'key' must be unique and contain no whitespace.
// 'value' is what gets copied to the clipboard.
`

func (a *App) cmdEdit(args []string) int {
	if len(args) < 1 {
		return a.usage("usage: pe e <key>")
	}
	key := args[0]
	f, p, err := a.load()
	if err != nil {
		return a.errorf("load %s: %v", p, err)
	}
	current, err := f.Get(key)
	if err != nil {
		return a.notFound(f, key)
	}

	form := editForm{
		Key:         current.Key,
		Value:       current.Value,
		Description: current.Description,
		Tags:        current.Tags,
	}
	if form.Tags == nil {
		form.Tags = []string{}
	}
	body, err := json.MarshalIndent(form, "", "  ")
	if err != nil {
		return a.errorf("marshal: %v", err)
	}
	initial := []byte(editFormHint + string(body) + "\n")

	edited, err := editor.Edit("pe-snippet", "json", initial)
	if err != nil {
		if errors.Is(err, editor.ErrUnchanged) {
			fmt.Fprintln(a.out(), "no changes")
			return 0
		}
		return a.errorf("editor: %v", err)
	}

	cleaned := stripJSONComments(string(edited))
	var next editForm
	if err := json.Unmarshal([]byte(cleaned), &next); err != nil {
		return a.errorf("parse edited JSON: %v", err)
	}
	if err := validateKey(next.Key); err != nil {
		return a.errorf("%v", err)
	}

	updated := store.Snippet{
		Key:         next.Key,
		Value:       next.Value,
		Description: next.Description,
		Tags:        next.Tags,
		CreatedAt:   current.CreatedAt,
		LastUsedAt:  current.LastUsedAt,
		UseCount:    current.UseCount,
	}
	if err := f.Update(key, updated); err != nil {
		switch err {
		case store.ErrExists:
			return a.errorf("already exists: %s", next.Key)
		default:
			return a.errorf("%v", err)
		}
	}
	if err := store.Save(p, f); err != nil {
		return a.errorf("save %s: %v", p, err)
	}
	if next.Key != key {
		fmt.Fprintf(a.out(), "updated: %s → %s\n", key, next.Key)
	} else {
		fmt.Fprintf(a.out(), "updated: %s\n", key)
	}
	return 0
}

// stripJSONComments removes leading // line comments before the first '{'.
// We only strip leading comments because JSON itself does not allow them
// and we want users to see the hint without it breaking the parse.
func stripJSONComments(s string) string {
	lines := strings.Split(s, "\n")
	start := 0
	for i, line := range lines {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "//") || t == "" {
			continue
		}
		start = i
		break
	}
	return strings.Join(lines[start:], "\n")
}
