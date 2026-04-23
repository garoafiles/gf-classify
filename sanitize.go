package classify

import (
	"regexp"
	"strings"
)

var (
	// Release-group tags in square brackets: [rartv], [YIFY], [H264], ...
	reSquareTag = regexp.MustCompile(`\[[^\]]*\]`)

	// Parenthesized segments we strip. A separate step keeps 4-digit years.
	reParenSeg = regexp.MustCompile(`\([^)]*\)`)

	// "(19xx)" or "(20xx)" -- preserved because it's a disambiguating year.
	reParenYear = regexp.MustCompile(`\((19|20)\d{2}\)`)

	// Collapse the . / _ separators common in scene names, then whitespace.
	reDotUnderscore = regexp.MustCompile(`[._]`)
	reWhitespace    = regexp.MustCompile(`\s+`)

	// Trailing scene-group tag: "-GROUP" at end of name, uppercase, 2+ chars.
	// Allows digits within the tag (-PLAZA, -RARBG, -X264). Requires a word
	// boundary on the left so we don't eat "-release" type dashes.
	reTrailingGroup = regexp.MustCompile(`\s*-\s*[A-Z][A-Z0-9]{1,}$`)
)

// Sanitize returns the display-friendly form of a torrent name:
//
//  1. Strip [...] release-group tags.
//  2. Strip (...) segments EXCEPT a 4-digit year, which is kept.
//  3. Replace . and _ with space (scene naming).
//  4. Collapse runs of whitespace to a single space.
//  5. Strip a trailing "-GROUP" scene tag.
//
// Exported so callers that fuzzy-match the result can work from the exact
// same string Classify worked from.
func Sanitize(name string) string {
	s := reSquareTag.ReplaceAllString(name, " ")
	s = preserveYears(s)
	s = reDotUnderscore.ReplaceAllString(s, " ")
	s = reWhitespace.ReplaceAllString(s, " ")
	s = strings.TrimSpace(s)
	s = reTrailingGroup.ReplaceAllString(s, "")
	s = reWhitespace.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

// preserveYears keeps "(19xx)" / "(20xx)" verbatim (minus parens) while
// stripping every other parenthesized segment.
func preserveYears(s string) string {
	if !strings.ContainsAny(s, "()") {
		return s
	}
	return reParenSeg.ReplaceAllStringFunc(s, func(seg string) string {
		if reParenYear.MatchString(seg) {
			return " " + seg[1:len(seg)-1] + " "
		}
		return " "
	})
}
