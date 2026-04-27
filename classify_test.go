package classify

import "testing"

func TestSanitize(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		// Square-bracket release tag.
		{"The.Show.S01E01.720p.WEBRip.x264-GROUP[rartv]", "The Show S01E01 720p WEBRip x264"},
		// Parenthesized year kept.
		{"Some.Movie.(2020).1080p.BluRay", "Some Movie 2020 1080p BluRay"},
		// Parenthesized non-year stripped.
		{"Some.Movie.(DirCut).1080p.BluRay", "Some Movie 1080p BluRay"},
		// Underscores + collapse whitespace.
		{"Sleep_Token_-_Sundowning__2019_FLAC", "Sleep Token - Sundowning 2019 FLAC"},
		// Trailing -GROUP.
		{"Some.Movie.2020.1080p.BluRay-RARBG", "Some Movie 2020 1080p BluRay"},
		// Mixed.
		{"[CrewName] Show Title (2019) [x265] S01E05-GROUP", "Show Title 2019 S01E05"},
	}
	for _, tc := range cases {
		got := Sanitize(tc.in)
		if got != tc.want {
			t.Errorf("Sanitize(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestClassifyTVStripsYear(t *testing.T) {
	// Year should not end up in the show name but should be reported as a
	// separate field.
	res := Classify("The.Office.2005.S01E05.HDTV.x264-LOL", []File{
		{RelativePath: "the.office.2005.s01e05.hdtv.x264-lol.mkv", SizeBytes: 300 * 1024 * 1024},
	})
	if res.Category != CategoryTV {
		t.Fatalf("category = %q, want tv", res.Category)
	}
	if res.Show != "The Office" {
		t.Errorf("show = %q, want %q", res.Show, "The Office")
	}
	if res.Year != "2005" {
		t.Errorf("year = %q, want %q", res.Year, "2005")
	}
}

func TestClassifyTVNoYear(t *testing.T) {
	res := Classify("The.Office.S01E05.HDTV.x264-LOL", []File{
		{RelativePath: "episode.mkv", SizeBytes: 1},
	})
	if res.Category != CategoryTV {
		t.Fatalf("category = %q, want tv", res.Category)
	}
	if res.Show != "The Office" {
		t.Errorf("show = %q, want %q", res.Show, "The Office")
	}
	if res.Year != "" {
		t.Errorf("year = %q, want empty", res.Year)
	}
}

func TestClassifyTVSeasonWord(t *testing.T) {
	res := Classify("Great British Bake Off Season 12 Complete", []File{
		{RelativePath: "ep1.mkv", SizeBytes: 1},
		{RelativePath: "ep2.mkv", SizeBytes: 1},
	})
	if res.Category != CategoryTV {
		t.Fatalf("category = %q, want tv", res.Category)
	}
	if res.Show != "Great British Bake Off" {
		t.Errorf("show = %q", res.Show)
	}
}

func TestClassifyTV2x03(t *testing.T) {
	res := Classify("Some Show 2x03", []File{
		{RelativePath: "ep.mkv", SizeBytes: 1},
	})
	if res.Category != CategoryTV {
		t.Fatalf("category = %q, want tv", res.Category)
	}
	if res.Show != "Some Show" {
		t.Errorf("show = %q", res.Show)
	}
}

func TestClassifyTVBeatsMovieYearConfusion(t *testing.T) {
	// TV ordering is load-bearing: a torrent whose name carries BOTH a year
	// AND an SxxExx marker (common for remake shows) must land in TV, not
	// Movies. Because TV runs before Movies in the switch, the year regex
	// never gets to claim this.
	res := Classify("The.Office.2023.S01E01.1080p.WEBRip", []File{
		{RelativePath: "e1.mkv", SizeBytes: 1},
		{RelativePath: "e2.mkv", SizeBytes: 1},
	})
	if res.Category != CategoryTV {
		t.Fatalf("category = %q, want tv (must beat Movies despite the 2023)", res.Category)
	}
	if res.Year != "2023" {
		t.Errorf("year = %q, want 2023", res.Year)
	}
	if res.Show != "The Office" {
		t.Errorf("show = %q, want %q", res.Show, "The Office")
	}
}

func TestClassifyMovieMultiFile(t *testing.T) {
	// Multi-file movie (feature + sample + subs). Majority of files are
	// video, so movie detection still fires.
	res := Classify("Great.Movie.2019.1080p.BluRay.x264-GROUP", []File{
		{RelativePath: "great.movie.2019.1080p.bluray.x264-group.mkv", SizeBytes: 12e9},
		{RelativePath: "sample.mkv", SizeBytes: 50e6},
		{RelativePath: "subs/en.srt", SizeBytes: 40000},
	})
	if res.Category != CategoryMovies {
		t.Fatalf("category = %q, want movies", res.Category)
	}
	if res.Year != "2019" {
		t.Errorf("year = %q, want 2019", res.Year)
	}
	if res.Title != "Great Movie" {
		t.Errorf("title = %q, want %q", res.Title, "Great Movie")
	}
}

// TestClassifyMovieSizeWeightedMajority: scene-style movie releases ship a
// single huge .mkv alongside a handful of small companion files (.nfo, .srt,
// a sample clip, screenshots). By count the .mkv is a minority — 1 of 6 —
// but by bytes it's >99%. The count-based gate bailed and the torrent fell
// through to Other; the size-weighted check catches it.
func TestClassifyMovieSizeWeightedMajority(t *testing.T) {
	res := Classify("The.Housemaid.2025.1080p.WEBRip.x264-GROUP", []File{
		{RelativePath: "The.Housemaid.2025.1080p.WEBRip.x264-GROUP/The.Housemaid.2025.1080p.WEBRip.x264-GROUP.mkv", SizeBytes: 2_000_000_000},
		{RelativePath: "The.Housemaid.2025.1080p.WEBRip.x264-GROUP/The.Housemaid.2025.1080p.WEBRip.x264-GROUP.nfo", SizeBytes: 4_000},
		{RelativePath: "The.Housemaid.2025.1080p.WEBRip.x264-GROUP/The.Housemaid.2025.1080p.WEBRip.x264-GROUP.srt", SizeBytes: 80_000},
		{RelativePath: "The.Housemaid.2025.1080p.WEBRip.x264-GROUP/Sample/sample.mkv", SizeBytes: 30_000_000},
		{RelativePath: "The.Housemaid.2025.1080p.WEBRip.x264-GROUP/Screens/screen1.jpg", SizeBytes: 400_000},
		{RelativePath: "The.Housemaid.2025.1080p.WEBRip.x264-GROUP/Screens/screen2.jpg", SizeBytes: 400_000},
	})
	if res.Category != CategoryMovies {
		t.Fatalf("category = %q, want movies (video is 99%%+ by bytes, minority by count)", res.Category)
	}
	if res.Title != "The Housemaid" {
		t.Errorf("title = %q, want %q", res.Title, "The Housemaid")
	}
	if res.Year != "2025" {
		t.Errorf("year = %q, want 2025", res.Year)
	}
}

// TestClassifyMovieReleaseSitePrefixed reproduces codeberg issue #1. The
// "www <site> <tld> - " watermark combined with the realistic multi-file
// release structure shipped the torrent to Other: the prefix pushed the
// real title past the year in the sanitized form, and the count-based video
// gate bailed. Sanitizer now strips the watermark and detectMovies weights
// by bytes, so this lands in Movies cleanly.
func TestClassifyMovieReleaseSitePrefixed(t *testing.T) {
	res := Classify("www UIndex org - The Housemaid 2025 1080p WEBRip 5 1", []File{
		{RelativePath: "www UIndex org - The Housemaid 2025 1080p WEBRip 5 1/The.Housemaid.2025.1080p.WEBRip.5.1.mkv", SizeBytes: 2_100_000_000},
		{RelativePath: "www UIndex org - The Housemaid 2025 1080p WEBRip 5 1/The.Housemaid.2025.1080p.WEBRip.5.1.nfo", SizeBytes: 3_200},
		{RelativePath: "www UIndex org - The Housemaid 2025 1080p WEBRip 5 1/The.Housemaid.2025.1080p.WEBRip.5.1.srt", SizeBytes: 72_000},
		{RelativePath: "www UIndex org - The Housemaid 2025 1080p WEBRip 5 1/Sample/sample.mkv", SizeBytes: 30_000_000},
		{RelativePath: "www UIndex org - The Housemaid 2025 1080p WEBRip 5 1/Screens/screen1.jpg", SizeBytes: 400_000},
		{RelativePath: "www UIndex org - The Housemaid 2025 1080p WEBRip 5 1/Screens/screen2.jpg", SizeBytes: 400_000},
	})
	if res.Category != CategoryMovies {
		t.Fatalf("category = %q, want movies (codeberg issue #1)", res.Category)
	}
	if res.Title != "The Housemaid" {
		t.Errorf("title = %q, want %q (release-site prefix must be stripped)", res.Title, "The Housemaid")
	}
	if res.Year != "2025" {
		t.Errorf("year = %q, want 2025", res.Year)
	}
}

func TestClassifyMovieSingleFile(t *testing.T) {
	res := Classify("Great.Movie.2019.1080p.BluRay.x264-YIFY", []File{
		{RelativePath: "Great.Movie.2019.1080p.BluRay.x264-YIFY.mp4", SizeBytes: 2e9},
	})
	if res.Category != CategoryMovies {
		t.Fatalf("category = %q, want movies", res.Category)
	}
	if res.Year != "2019" {
		t.Errorf("year = %q", res.Year)
	}
}

func TestClassifyMusicArtistSplit(t *testing.T) {
	res := Classify("Sleep Token - Sundowning 2019 FLAC", []File{
		{RelativePath: "01 The Night Does Not Belong To God.flac", SizeBytes: 40e6},
		{RelativePath: "02 The Offering.flac", SizeBytes: 44e6},
		{RelativePath: "cover.jpg", SizeBytes: 1e6},
	})
	if res.Category != CategoryMusic {
		t.Fatalf("category = %q, want music", res.Category)
	}
	if res.Artist != "Sleep Token" {
		t.Errorf("artist = %q, want %q", res.Artist, "Sleep Token")
	}
}

func TestClassifyMusicNoArtistSplit(t *testing.T) {
	res := Classify("Various Hits 2020", []File{
		{RelativePath: "01.mp3", SizeBytes: 1},
		{RelativePath: "02.mp3", SizeBytes: 1},
	})
	if res.Category != CategoryMusic {
		t.Fatalf("category = %q, want music", res.Category)
	}
	if res.Artist != "" {
		t.Errorf("artist = %q, want empty", res.Artist)
	}
}

func TestClassifyGameFitGirl(t *testing.T) {
	// FitGirl repacks carry .exe / .iso but are games. Games must win.
	res := Classify("Cyberpunk.2077.v2.1-FITGIRL", []File{
		{RelativePath: "setup.exe", SizeBytes: 100e6},
		{RelativePath: "data.iso", SizeBytes: 60e9},
	})
	if res.Category != CategoryGames {
		t.Fatalf("category = %q, want games", res.Category)
	}
}

func TestClassifyGameExtension(t *testing.T) {
	res := Classify("Some Switch Title", []File{
		{RelativePath: "title.nsp", SizeBytes: 1e9},
	})
	if res.Category != CategoryGames {
		t.Fatalf("category = %q, want games", res.Category)
	}
}

func TestClassifySoftware(t *testing.T) {
	res := Classify("Ubuntu 22.04 LTS", []File{
		{RelativePath: "ubuntu-22.04.iso", SizeBytes: 3e9},
	})
	if res.Category != CategorySoftware {
		t.Fatalf("category = %q, want software", res.Category)
	}
}

func TestClassifyBooks(t *testing.T) {
	res := Classify("The Terry Pratchett Pack", []File{
		{RelativePath: "discworld_01.epub", SizeBytes: 1e6},
		{RelativePath: "discworld_02.epub", SizeBytes: 1e6},
		{RelativePath: "cover.jpg", SizeBytes: 50e3},
	})
	if res.Category != CategoryBooks {
		t.Fatalf("category = %q, want books", res.Category)
	}
}

func TestClassifyOtherFallback(t *testing.T) {
	res := Classify("Some.Random.Archive", []File{
		{RelativePath: "readme.txt", SizeBytes: 1},
		{RelativePath: "data.zip", SizeBytes: 1},
	})
	if res.Category != CategoryOther {
		t.Fatalf("category = %q, want other", res.Category)
	}
}

// TestClassifyMovieMimeOnly: cloud-rewritten name with no extension but a
// video MIME type. Year is in the display name, so the year gate is
// satisfied; majorityBytes must accept the file via MIME prefix even
// though the extension table doesn't know it.
func TestClassifyMovieMimeOnly(t *testing.T) {
	res := Classify("Great Movie 2019", []File{
		{RelativePath: "asset-uuid-1234", SizeBytes: 2e9, MimeType: "video/mp4"},
	})
	if res.Category != CategoryMovies {
		t.Fatalf("category = %q, want movies (MIME video/* should drive detection)", res.Category)
	}
	if res.Year != "2019" {
		t.Errorf("year = %q, want 2019", res.Year)
	}
	if res.Title != "Great Movie" {
		t.Errorf("title = %q, want %q", res.Title, "Great Movie")
	}
}

// TestClassifyMovieOctetStreamFallsBackToExt: cloud emits the generic
// application/octet-stream sniff but the file still has a real video
// extension. Extension wins; result is Movies. This is the most common
// shape for cloud-stored .mkv files.
func TestClassifyMovieOctetStreamFallsBackToExt(t *testing.T) {
	res := Classify("Great Movie 2019", []File{
		{RelativePath: "Great.Movie.2019.mkv", SizeBytes: 2e9, MimeType: "application/octet-stream"},
	})
	if res.Category != CategoryMovies {
		t.Fatalf("category = %q, want movies (extension must still win when MIME is generic)", res.Category)
	}
}

// TestClassifyMusicMimeOnly: extensionless audio file, MIME audio/mpeg.
// Music detection has no year requirement, so MIME alone is sufficient.
func TestClassifyMusicMimeOnly(t *testing.T) {
	res := Classify("Various Hits", []File{
		{RelativePath: "track-1", SizeBytes: 5e6, MimeType: "audio/mpeg"},
		{RelativePath: "track-2", SizeBytes: 5e6, MimeType: "audio/mpeg"},
	})
	if res.Category != CategoryMusic {
		t.Fatalf("category = %q, want music", res.Category)
	}
}

// TestClassifyMimeIgnoredWhenEmpty confirms backwards compat: a File with
// no MimeType behaves identically to pre-MIME callers — the existing
// extension-based classification still fires.
func TestClassifyMimeIgnoredWhenEmpty(t *testing.T) {
	res := Classify("Great.Movie.2019.1080p.BluRay.x264-YIFY", []File{
		{RelativePath: "great.movie.2019.mkv", SizeBytes: 2e9}, // no MimeType
	})
	if res.Category != CategoryMovies {
		t.Fatalf("category = %q, want movies", res.Category)
	}
}

func TestIsVideo(t *testing.T) {
	cases := []struct {
		name string
		f    File
		want bool
	}{
		{"video extension", File{RelativePath: "x.mkv"}, true},
		{"video MIME prefix", File{RelativePath: "uuid", MimeType: "video/mp4"}, true},
		{"both ext and MIME", File{RelativePath: "x.mp4", MimeType: "video/mp4"}, true},
		{"audio file is not video", File{RelativePath: "x.mp3", MimeType: "audio/mpeg"}, false},
		{"plain text", File{RelativePath: "readme.txt"}, false},
		{"empty", File{}, false},
		{"upper-case ext", File{RelativePath: "MOVIE.MKV"}, true},
		{"video/x-matroska prefix", File{MimeType: "video/x-matroska"}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsVideo(tc.f); got != tc.want {
				t.Errorf("IsVideo(%+v) = %v, want %v", tc.f, got, tc.want)
			}
		})
	}
}

func TestIsAudio(t *testing.T) {
	cases := []struct {
		name string
		f    File
		want bool
	}{
		{"audio extension", File{RelativePath: "x.mp3"}, true},
		{"audio MIME prefix", File{RelativePath: "uuid", MimeType: "audio/mpeg"}, true},
		{"flac ext", File{RelativePath: "track.flac"}, true},
		{"video file is not audio", File{RelativePath: "x.mkv", MimeType: "video/mp4"}, false},
		{"empty", File{}, false},
		{"upper-case ext", File{RelativePath: "SONG.FLAC"}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsAudio(tc.f); got != tc.want {
				t.Errorf("IsAudio(%+v) = %v, want %v", tc.f, got, tc.want)
			}
		})
	}
}

func TestFuzzyEqual(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"Music", "music", true},
		{"Music", "Musics", true},
		{"Movies", "Movie", true},
		{"Music", "Cinema", false},
		{"TV Shows", "tv shows", true},
		{"TV Shows", "TV Show", true},
	}
	for _, tc := range cases {
		got := FuzzyEqual(tc.a, tc.b)
		if got != tc.want {
			t.Errorf("FuzzyEqual(%q,%q) = %v, want %v", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestCategoryFolderName(t *testing.T) {
	cases := []struct {
		c    Category
		want string
	}{
		{CategoryTV, "TV Shows"},
		{CategoryMovies, "Movies"},
		{CategoryMusic, "Music"},
		{CategoryGames, "Games"},
		{CategorySoftware, "Software"},
		{CategoryBooks, "Books"},
		{CategoryOther, "Other"},
	}
	for _, tc := range cases {
		if got := CategoryFolderName(tc.c); got != tc.want {
			t.Errorf("CategoryFolderName(%q) = %q, want %q", tc.c, got, tc.want)
		}
	}
}

func TestIsSingleFile(t *testing.T) {
	if !IsSingleFile([]File{{RelativePath: "x.mkv", SizeBytes: 1}}) {
		t.Error("single-file should return true")
	}
	if IsSingleFile([]File{{RelativePath: "a", SizeBytes: 1}, {RelativePath: "b", SizeBytes: 1}}) {
		t.Error("multi-file should return false")
	}
	if IsSingleFile(nil) {
		t.Error("empty should return false")
	}
}
