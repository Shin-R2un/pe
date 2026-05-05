package cli

import (
	"sort"
	"strings"

	"github.com/Shin-R2un/pe/internal/store"
)

// suggest returns up to `limit` snippet keys near `query`, ordered by
// "did you mean" usefulness:
//
//	0 — exact case-insensitive match (rare, only when caller already
//	    failed an exact-case Get)
//	1 — case-insensitive prefix match
//	2 — case-insensitive substring match
//	3 — small edit distance (Levenshtein ≤ max(2, len(query)/3))
//
// Lower score wins. Ties broken alphabetically.
func suggest(f *store.File, query string, limit int) []string {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" || len(f.Snippets) == 0 {
		return nil
	}
	type cand struct {
		key   string
		score int
	}
	threshold := len(q) / 3
	if threshold < 2 {
		threshold = 2
	}
	var cands []cand
	for _, s := range f.Snippets {
		k := strings.ToLower(s.Key)
		switch {
		case k == q:
			cands = append(cands, cand{s.Key, 0})
		case strings.HasPrefix(k, q):
			cands = append(cands, cand{s.Key, 1})
		case strings.Contains(k, q):
			cands = append(cands, cand{s.Key, 2})
		default:
			if d := levenshtein(k, q); d <= threshold {
				cands = append(cands, cand{s.Key, 3 + d})
			}
		}
	}
	sort.Slice(cands, func(i, j int) bool {
		if cands[i].score != cands[j].score {
			return cands[i].score < cands[j].score
		}
		return cands[i].key < cands[j].key
	})
	if len(cands) > limit {
		cands = cands[:limit]
	}
	out := make([]string, len(cands))
	for i, c := range cands {
		out[i] = c.key
	}
	return out
}

// levenshtein computes the Levenshtein edit distance between a and b.
// Standard two-row DP, O(len(a)*len(b)) time, O(min) space.
func levenshtein(a, b string) int {
	ar := []rune(a)
	br := []rune(b)
	if len(ar) == 0 {
		return len(br)
	}
	if len(br) == 0 {
		return len(ar)
	}
	prev := make([]int, len(br)+1)
	curr := make([]int, len(br)+1)
	for j := 0; j <= len(br); j++ {
		prev[j] = j
	}
	for i := 1; i <= len(ar); i++ {
		curr[0] = i
		for j := 1; j <= len(br); j++ {
			cost := 1
			if ar[i-1] == br[j-1] {
				cost = 0
			}
			del := prev[j] + 1
			ins := curr[j-1] + 1
			sub := prev[j-1] + cost
			m := del
			if ins < m {
				m = ins
			}
			if sub < m {
				m = sub
			}
			curr[j] = m
		}
		prev, curr = curr, prev
	}
	return prev[len(br)]
}
