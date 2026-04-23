package classify

import (
	"path/filepath"
	"regexp"
	"strings"
)

// Category is the set of top-level folder categories a torrent can land in.
type Category string

const (
	CategoryTV       Category = "tv"
	CategoryMovies   Category = "movies"
	CategoryMusic    Category = "music"
	CategoryGames    Category = "games"
	CategorySoftware Category = "software"
	CategoryBooks    Category = "books"
	CategoryOther    Category = "other"
)

// File is the minimal per-output-file shape callers provide. Typically one
// entry per file in a torrent snapshot.
type File struct {
	RelativePath string
	SizeBytes    int64
}

// Result is what Classify returns. Fields are filled on a best-effort,
// category-specific basis:
//
//   - TV:      Show (year stripped), optional Year.
//   - Movies:  Title, Year.
//   - Music:   Artist (may be empty).
//   - Other categories leave these blank.
//
// SanitizedName is the torrent name after sanitization rules, exposed so
// callers that need a "nice directory name" (multi-file layout) don't have
// to re-run the pipeline.
type Result struct {
	Category      Category
	SanitizedName string
	Show          string
	Artist        string
	Title         string
	Year          string
}

var (
	// TV-show detectors. First-match-wins across these three.
	reTVSE      = regexp.MustCompile(`(?i)\bS(\d{1,2})E(\d{1,3})\b`)
	reTVSeason  = regexp.MustCompile(`(?i)\bSeason\s+(\d+)\b`)
	reTVNumXNum = regexp.MustCompile(`\b(\d{1,2})x(\d{1,2})\b`)

	// Year in the sanitized name (used for movies + TV year stripping).
	reYearAny = regexp.MustCompile(`\b(19|20)\d{2}\b`)

	// Music: "Artist - Album" split. Lazy on both sides so a leading
	// hyphenated artist ("Florence + The Machine - High as Hope") doesn't
	// grab the separator.
	reArtistAlbum = regexp.MustCompile(`^(.+?)\s*-\s*(.+)$`)
)

// Scene-group tokens that, if present anywhere in the name (any case), force
// the Games classification. Deliberately checked before the .exe / .iso
// Software heuristic because FitGirl-style releases frequently carry those
// extensions.
var gameTokens = []string{
	"-FITGIRL", "-RUNE", "-CODEX", "-SKIDROW", "-PLAZA", "-EMPRESS", "REPACK",
}

var (
	videoExts = stringSet(".mkv", ".mp4", ".avi", ".mov", ".webm", ".m4v")
	audioExts = stringSet(".mp3", ".flac", ".m4a", ".ogg", ".wav", ".opus", ".aac")
	gameExts  = stringSet(".nsp", ".xci", ".rom", ".wbfs", ".cso", ".gog")
	softExts  = stringSet(".iso", ".exe", ".msi", ".dmg", ".pkg", ".deb", ".rpm", ".apk", ".app")
	bookExts  = stringSet(".epub", ".pdf", ".mobi", ".azw3", ".cbr", ".cbz")
)

func stringSet(keys ...string) map[string]struct{} {
	m := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		m[k] = struct{}{}
	}
	return m
}

// Classify runs the full pipeline: sanitize -> category -> sub-info. name
// is the torrent's display name; files is the list of output files. Returns
// a Result with Category always populated (CategoryOther if nothing matched).
func Classify(name string, files []File) Result {
	sanitized := Sanitize(name)
	res := Result{SanitizedName: sanitized, Category: CategoryOther}

	if show, year, ok := detectTV(sanitized); ok {
		res.Category = CategoryTV
		res.Show = show
		res.Year = year
		return res
	}
	if title, year, ok := detectMovies(sanitized, files); ok {
		res.Category = CategoryMovies
		res.Title = title
		res.Year = year
		return res
	}
	if artist, ok := detectMusic(sanitized, files); ok {
		res.Category = CategoryMusic
		res.Artist = artist
		return res
	}
	// Games: scene-group tokens (-FITGIRL, etc.) are checked against the
	// ORIGINAL name, not the sanitized form. The sanitizer strips trailing
	// -GROUP tags, which would otherwise erase the signal we need here.
	if detectGames(name, sanitized, files) {
		res.Category = CategoryGames
		return res
	}
	if detectSoftware(files) {
		res.Category = CategorySoftware
		return res
	}
	if detectBooks(files) {
		res.Category = CategoryBooks
		return res
	}
	return res
}

// detectTV matches any of the three TV-show forms and returns the show name
// (year stripped) plus optional year. "The.Office.2005.S01E05.HDTV" -> show
// "The Office", year "2005". "The.Office.S01E05.HDTV" -> show "The Office",
// year "".
func detectTV(sanitized string) (show, year string, ok bool) {
	loc := firstTVMatch(sanitized)
	if loc == nil {
		return "", "", false
	}
	show = strings.TrimSpace(sanitized[:loc[0]])
	if m := reYearAny.FindStringIndex(show); m != nil && m[1] == len(show) {
		year = show[m[0]:m[1]]
		show = strings.TrimSpace(show[:m[0]])
	}
	return show, year, true
}

// firstTVMatch returns the location of the earliest TV pattern in s, or nil.
// We try all three and pick whichever starts first, so "Season 1 S01E01" in
// an oddball name still fires on the earliest marker.
func firstTVMatch(s string) []int {
	best := []int{-1, -1}
	for _, re := range []*regexp.Regexp{reTVSE, reTVSeason, reTVNumXNum} {
		m := re.FindStringIndex(s)
		if m == nil {
			continue
		}
		if best[0] == -1 || m[0] < best[0] {
			best = m
		}
	}
	if best[0] == -1 {
		return nil
	}
	return best
}

// detectMovies needs both a 4-digit year in the name AND ≥50% video files.
// Title = substring before the first year.
func detectMovies(sanitized string, files []File) (title, year string, ok bool) {
	m := reYearAny.FindStringIndex(sanitized)
	if m == nil {
		return "", "", false
	}
	if !majorityExt(files, videoExts, 0.5) {
		return "", "", false
	}
	title = strings.TrimSpace(sanitized[:m[0]])
	year = sanitized[m[0]:m[1]]
	return title, year, true
}

// detectMusic needs ≥50% audio files. Artist is an optional extraction from
// the "Artist - Album" pattern.
func detectMusic(sanitized string, files []File) (artist string, ok bool) {
	if !majorityExt(files, audioExts, 0.5) {
		return "", false
	}
	if m := reArtistAlbum.FindStringSubmatch(sanitized); m != nil {
		artist = strings.TrimSpace(m[1])
	}
	return artist, true
}

// detectGames matches on scene-group tokens first (highest signal), then
// falls back to extension majority. Checked before Software so -FITGIRL
// releases packed with .exe / .iso stay classified correctly.
func detectGames(raw, sanitized string, files []File) bool {
	upperRaw := strings.ToUpper(raw)
	upperSan := strings.ToUpper(sanitized)
	for _, tok := range gameTokens {
		if strings.Contains(upperRaw, tok) || strings.Contains(upperSan, tok) {
			return true
		}
	}
	return majorityExt(files, gameExts, 0.5)
}

func detectSoftware(files []File) bool {
	return majorityExt(files, softExts, 0.5)
}

func detectBooks(files []File) bool {
	return majorityExt(files, bookExts, 0.5)
}

// majorityExt returns true if at least `threshold` of the files (by count)
// have an extension in `set`. Threshold is a fraction in [0, 1].
func majorityExt(files []File, set map[string]struct{}, threshold float64) bool {
	if len(files) == 0 {
		return false
	}
	hit := 0
	for _, f := range files {
		ext := strings.ToLower(filepath.Ext(f.RelativePath))
		if _, ok := set[ext]; ok {
			hit++
		}
	}
	return float64(hit)/float64(len(files)) >= threshold
}

// IsSingleFile reports whether the torrent is a single-file torrent. Used by
// the layout rules (single-file -> T/<leaf>; multi-file -> T/<dir>/...).
func IsSingleFile(files []File) bool {
	return len(files) == 1
}

// CategoryFolderName returns the canonical folder name for a category. Used
// when lazy-creating a category folder under a caller's virtual root.
func CategoryFolderName(c Category) string {
	switch c {
	case CategoryTV:
		return "TV Shows"
	case CategoryMovies:
		return "Movies"
	case CategoryMusic:
		return "Music"
	case CategoryGames:
		return "Games"
	case CategorySoftware:
		return "Software"
	case CategoryBooks:
		return "Books"
	}
	return "Other"
}
