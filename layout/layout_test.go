package layout

import (
	"testing"

	classify "codeberg.org/garoafiles/gf-classify"
)

func TestCollapseHappyPath(t *testing.T) {
	// Music release where every track is under a single top-level dir whose
	// name matches the torrent name. Collapse must fire so callers don't
	// produce <target>/<name>/<name>/track.flac.
	sanitized := classify.Sanitize("Sleep Token - Sundowning 2019 FLAC")
	files := []classify.File{
		{RelativePath: "Sleep Token - Sundowning 2019 FLAC/01.flac", SizeBytes: 1},
		{RelativePath: "Sleep Token - Sundowning 2019 FLAC/02.flac", SizeBytes: 1},
	}
	if !Collapse(sanitized, files) {
		t.Fatalf("expected collapse for single top-level dir matching torrent name")
	}
}

func TestCollapseMixedRoots(t *testing.T) {
	sanitized := classify.Sanitize("Sleep Token - Sundowning 2019 FLAC")
	files := []classify.File{
		{RelativePath: "Sleep Token - Sundowning 2019 FLAC/01.flac", SizeBytes: 1},
		{RelativePath: "extras/bonus.flac", SizeBytes: 1},
	}
	if Collapse(sanitized, files) {
		t.Fatalf("mixed-root torrents must not collapse")
	}
}

func TestCollapseSingleFile(t *testing.T) {
	sanitized := classify.Sanitize("Great.Movie.2019.1080p.mkv")
	files := []classify.File{{RelativePath: "Great.Movie.2019.1080p.mkv", SizeBytes: 1}}
	if Collapse(sanitized, files) {
		t.Fatalf("single-file torrent reported collapsible")
	}
}

func TestCollapseSanitizationMismatch(t *testing.T) {
	// Regression: torrent name "Sleep Token - Sundowning  2019" has a
	// double space; Sanitize collapses it to a single space. The
	// filesystem top-level dir keeps the double space, so a naive
	// FuzzyEqual(rawFirst, sanitized) would never match. Sanitizing the
	// first segment before comparing fixes it.
	sanitized := classify.Sanitize("Sleep Token - Sundowning  2019")
	files := []classify.File{
		{RelativePath: "Sleep Token - Sundowning  2019/01.flac", SizeBytes: 1},
		{RelativePath: "Sleep Token - Sundowning  2019/02.flac", SizeBytes: 1},
	}
	if !Collapse(sanitized, files) {
		t.Fatalf("expected collapse after sanitizing first segment")
	}
}

func TestDecideLayoutSparseRootRichWrapper(t *testing.T) {
	// Regression for the Sleep Token / Even In Arcadia case: the torrent's
	// info-dict root directory is a sparse "Sleep Token" (every file lives
	// at Sleep Token/<track>.flac), but the magnet dn carries the full
	// release name. The caller picks the richer dn as the wrapper; collapse
	// must still strip the redundant "Sleep Token/" segment from each file.
	files := []classify.File{
		{RelativePath: "Sleep Token/01 - Look To Windward.flac", SizeBytes: 1},
		{RelativePath: "Sleep Token/02 - Emergence.flac", SizeBytes: 1},
		{RelativePath: "Sleep Token/03 - Past Self.flac", SizeBytes: 1},
	}
	diskRoot := CommonRoot(files)
	if diskRoot != "Sleep Token" {
		t.Fatalf("CommonRoot = %q, want %q", diskRoot, "Sleep Token")
	}
	sanitizedDisk := classify.Sanitize(diskRoot)
	sanitizedWrapper := classify.Sanitize("Sleep Token - Even In Arcadia - FLAC")
	plan := DecideLayoutSplit(sanitizedDisk, sanitizedWrapper, files)
	if !plan.Wrap {
		t.Fatalf("expected Wrap=true for multi-file torrent")
	}
	if plan.WrapperName != "Sleep Token - Even In Arcadia" {
		t.Errorf("WrapperName = %q, want %q", plan.WrapperName, "Sleep Token - Even In Arcadia")
	}
	if !plan.Collapse {
		t.Errorf("expected Collapse=true so the redundant Sleep Token/ prefix is stripped")
	}
}

func TestDecideLayoutAgreeingNames(t *testing.T) {
	// When disk root and wrapper are the same sanitized form, behavior
	// matches the single-arg DecideLayout: wrap + collapse.
	files := []classify.File{
		{RelativePath: "Sleep Token - Sundowning 2019 FLAC/01.flac", SizeBytes: 1},
		{RelativePath: "Sleep Token - Sundowning 2019 FLAC/02.flac", SizeBytes: 1},
	}
	sanitized := classify.Sanitize("Sleep Token - Sundowning 2019 FLAC")

	plan := DecideLayout(sanitized, files)
	if !plan.Wrap || !plan.Collapse {
		t.Fatalf("DecideLayout expected wrap+collapse, got %+v", plan)
	}
	if plan.WrapperName != sanitized {
		t.Errorf("WrapperName = %q, want %q", plan.WrapperName, sanitized)
	}

	split := DecideLayoutSplit(sanitized, sanitized, files)
	if split != plan {
		t.Errorf("DecideLayoutSplit with same args diverged: split=%+v plan=%+v", split, plan)
	}
}

func TestDecideLayoutMixedRootsNoCollapse(t *testing.T) {
	// Files at mixed top-levels -> no common disk root to collapse.
	files := []classify.File{
		{RelativePath: "disc1/01.flac", SizeBytes: 1},
		{RelativePath: "disc2/01.flac", SizeBytes: 1},
	}
	if r := CommonRoot(files); r != "" {
		t.Errorf("CommonRoot = %q, want empty", r)
	}
	plan := DecideLayoutSplit("", "Some Album", files)
	if !plan.Wrap {
		t.Fatal("expected Wrap=true for multi-file torrent")
	}
	if plan.Collapse {
		t.Error("expected Collapse=false for mixed-root torrent")
	}
}

func TestDecideLayoutSingleFile(t *testing.T) {
	files := []classify.File{{RelativePath: "movie.mkv", SizeBytes: 1}}
	plan := DecideLayout("Great Movie 2019", files)
	if plan.Wrap {
		t.Error("single-file torrent should not wrap")
	}
}

func TestDecideLayoutEmpty(t *testing.T) {
	if plan := DecideLayout("X", nil); plan.Wrap {
		t.Error("empty file list should not wrap")
	}
}

func TestCommonRootFileAtTopLevel(t *testing.T) {
	// At least one file at the top level -> no common root to collapse.
	files := []classify.File{
		{RelativePath: "Readme.txt", SizeBytes: 1},
		{RelativePath: "Album/01.flac", SizeBytes: 1},
	}
	if r := CommonRoot(files); r != "" {
		t.Errorf("CommonRoot = %q, want empty", r)
	}
}

func TestSplitRel(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"a/b/c", []string{"a", "b", "c"}},
		{"/a/b", []string{"a", "b"}},
		{"a//b", []string{"a", "b"}},
		{"./a/./b", []string{"a", "b"}},
		{"../a/b", []string{"a", "b"}},
		{"", nil},
	}
	for _, tc := range cases {
		got := SplitRel(tc.in)
		if !equalStrings(got, tc.want) {
			t.Errorf("SplitRel(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestJoinVirtual(t *testing.T) {
	cases := []struct {
		parent, name, want string
	}{
		{"", "a", "/a"},
		{"/", "a", "/a"},
		{"/Music", "Sleep Token", "/Music/Sleep Token"},
		{"/Music/Sleep Token", "Sundowning", "/Music/Sleep Token/Sundowning"},
	}
	for _, tc := range cases {
		got := JoinVirtual(tc.parent, tc.name)
		if got != tc.want {
			t.Errorf("JoinVirtual(%q,%q) = %q, want %q", tc.parent, tc.name, got, tc.want)
		}
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
