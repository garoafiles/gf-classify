package classify

import "strings"

// FuzzyEqual is the folder-matching rule used for both category lookup and
// sub-info lookup: case-insensitive comparison with English plural tolerance
// (a single trailing 's' is stripped from each side before comparison). It
// is intentionally narrow; Levenshtein distance and synonym lookups are out
// of scope.
func FuzzyEqual(a, b string) bool {
	return NormalizeFolder(a) == NormalizeFolder(b)
}

// NormalizeFolder lowercases, trims, and removes a single trailing 's' so
// "Music", "music", "Musics" all collapse to "music". Exported so callers
// building a normalized index of existing folder names can share the exact
// rule used by FuzzyEqual.
func NormalizeFolder(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	s = strings.TrimSuffix(s, "s")
	return s
}
