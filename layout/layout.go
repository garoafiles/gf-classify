package layout

import (
	"strings"

	classify "github.com/garoafiles/gf-classify"
)

// Plan describes the directory layout to apply under a caller-selected
// target folder for a multi-file torrent. Fields are only meaningful when
// Wrap is true; a zero-value Plan means "place files directly under the
// target" (single-file or degenerate torrents).
type Plan struct {
	// Wrap is true for multi-file torrents: the caller should create
	// <target>/<WrapperName>/ and place files underneath.
	Wrap bool

	// WrapperName is the directory name to lazy-create when Wrap is true.
	// Typically the sanitized torrent display name.
	WrapperName string

	// Collapse tells the caller to drop the first relative-path segment
	// from each file's path when building the final placement. Used to
	// avoid <target>/<wrapper>/<wrapper>/... double-nesting when the
	// torrent's on-disk root already matches the wrapper.
	Collapse bool
}

// DecideLayout returns the layout Plan for a torrent whose display name and
// on-disk root directory name agree (after sanitization) — the common case.
// Equivalent to DecideLayoutSplit(sanitizedName, sanitizedName, files).
//
// If your torrent's on-disk root can differ from the display name (for
// example: a magnet with a rich dn= pointing at a sparse info-dict root),
// call DecideLayoutSplit directly so the wrapper carries the richer name
// while collapse detection still fires on the real disk root.
func DecideLayout(sanitizedName string, files []classify.File) Plan {
	return DecideLayoutSplit(sanitizedName, sanitizedName, files)
}

// DecideLayoutSplit takes two sanitized forms:
//
//   - sanitizedDisk:    the torrent's info-dict root directory name after
//     Sanitize (equivalently, the common first-segment of
//     the file paths). Drives collapse detection.
//   - sanitizedWrapper: the display name after Sanitize. Becomes the
//     wrapper folder created under the target.
//
// When the two forms agree, behavior matches the single-arg DecideLayout.
// When they differ, the wrapper carries the richer text while collapse
// still fires against the filesystem reality.
func DecideLayoutSplit(sanitizedDisk, sanitizedWrapper string, files []classify.File) Plan {
	if len(files) <= 1 {
		return Plan{}
	}
	p := Plan{Wrap: true, WrapperName: sanitizedWrapper}
	if Collapse(sanitizedDisk, files) {
		p.Collapse = true
	}
	return p
}

// Collapse reports whether the torrent's root is already a single directory
// whose name matches sanitizedName. Layout callers should strip that first
// path segment from every file when Collapse returns true, avoiding
// <wrapper>/<wrapper>/... double-nesting.
//
// Detection is structural: every file's relative path must share the same
// first segment, and that segment must match sanitizedName after also being
// run through classify.Sanitize. Running both sides through Sanitize means
// differences in whitespace runs, bracketed release tags, or scene-dot
// naming on the filesystem don't produce false negatives against an already
// sanitized torrent name.
func Collapse(sanitizedName string, files []classify.File) bool {
	if len(files) == 0 {
		return false
	}
	first := ""
	for i, f := range files {
		parts := splitPath(f.RelativePath)
		if len(parts) < 2 {
			// A file at the top level rules collapse out.
			return false
		}
		if i == 0 {
			first = parts[0]
			continue
		}
		if parts[0] != first {
			return false
		}
	}
	return classify.FuzzyEqual(classify.Sanitize(first), sanitizedName)
}

// CommonRoot returns the common first-segment of every file's relative
// path, or "" if not every file shares one (or any file sits at the top
// level). Useful for callers that need to know the torrent's on-disk root
// directory without re-implementing the walk — for example, to pass into
// DecideLayoutSplit as sanitizedDisk = classify.Sanitize(CommonRoot(files)).
func CommonRoot(files []classify.File) string {
	if len(files) == 0 {
		return ""
	}
	first := ""
	for i, f := range files {
		parts := strings.SplitN(f.RelativePath, "/", 2)
		if len(parts) < 2 || parts[0] == "" {
			return ""
		}
		if i == 0 {
			first = parts[0]
			continue
		}
		if parts[0] != first {
			return ""
		}
	}
	return first
}

// SplitRel splits a relative path by forward slash (torrent protocol always
// reports POSIX separators). Empty segments, "." and ".." are dropped. The
// ".." drop is defensive: in practice a caller's output directory
// containment stops escapes, but layout callers often build filesystem
// paths from the split, and belt-and-suspenders is cheap here.
func SplitRel(p string) []string {
	raw := strings.Split(p, "/")
	out := raw[:0]
	for _, seg := range raw {
		if seg == "" || seg == "." || seg == ".." {
			continue
		}
		out = append(out, seg)
	}
	return out
}

// JoinVirtual joins a parent path with a new name using forward slash. An
// empty or "/" parent yields "/name"; otherwise "parent/name". Intended for
// materialized-path layouts where the root is "/".
func JoinVirtual(parent, name string) string {
	if parent == "" || parent == "/" {
		return "/" + name
	}
	return parent + "/" + name
}

// splitPath is the Collapse-internal splitter. Drops empty segments. Keeps
// "." and ".." so Collapse's "file at top level" check stays honest (a path
// like "./foo.mkv" still has only one real segment after the drop, but its
// structure is unusual enough that we treat it as a top-level file — which
// Collapse already does by returning false when parts < 2).
func splitPath(p string) []string {
	raw := strings.Split(p, "/")
	out := raw[:0]
	for _, seg := range raw {
		if seg == "" {
			continue
		}
		out = append(out, seg)
	}
	return out
}
