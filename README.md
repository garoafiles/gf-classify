# gf-classify

A pure-Go, regex-only torrent classifier and layout planner.

Given a torrent's display name and file list, `gf-classify` returns a category
(TV Shows, Movies, Music, Games, Software, Books, Other), best-effort sub-info
(show, artist, title, year), and a layout plan describing where a multi-file
torrent should unpack under a caller-chosen target directory. No external
metadata lookups, no persistence, no tenancy — feed it a name and files, get
back a classification and a plan.

Designed for torrent-flavored apps: self-hosted seedbox front-ends,
classification pipelines, Plex/Jellyfin pre-processors.

## Install

```sh
go get github.com/garoafiles/gf-classify
```

Requires Go 1.25 or later.

## Usage

```go
import (
    "github.com/garoafiles/gf-classify"
    "github.com/garoafiles/gf-classify/layout"
)

files := []classify.File{
    {RelativePath: "Great.Movie.2019/great.movie.2019.1080p.mkv", SizeBytes: 2e9},
}
res := classify.Classify("Great.Movie.2019.1080p.BluRay.x264-YIFY", files)
// res.Category      == classify.CategoryMovies
// res.SanitizedName == "Great Movie 2019 1080p BluRay x264"
// res.Title         == "Great Movie"
// res.Year          == "2019"

plan := layout.DecideLayout(res.SanitizedName, files)
// plan.Wrap         == false (single file)
```

For multi-file torrents:

```go
plan := layout.DecideLayout(res.SanitizedName, files)
if plan.Wrap {
    // Create <target>/<plan.WrapperName>/ and place files under it.
    // If plan.Collapse, strip the first relative-path segment on each file.
}
```

When the torrent's on-disk root directory differs from the display name (for
example: a magnet with a rich `dn=` parameter pointing at an info-dict whose
root dir is sparse), use the two-argument form so the wrapper folder reflects
the display name while collapse detection still fires against the real disk
root:

```go
diskRoot := layout.CommonRoot(files)
plan := layout.DecideLayoutSplit(
    classify.Sanitize(diskRoot),
    res.SanitizedName,
    files,
)
```

## Scope

**In.**

- `Classify(name, files) Result` — pure function, name + files → category + sub-info.
- `Sanitize(name) string` — torrent-name cleaner shared with the layout package.
- `FuzzyEqual(a, b) bool`, `NormalizeFolder(s) string` — folder-match rule
  (case-insensitive + English plural tolerance).
- `CategoryFolderName(Category) string` — canonical folder names.
- `layout.DecideLayout`, `layout.DecideLayoutSplit`, `layout.Collapse`,
  `layout.SplitRel`, `layout.JoinVirtual`, `layout.CommonRoot` — layout helpers.

**Out.**

- External metadata services (guessit, TMDB, TVDB).
- User / tenant concepts, persistence, DB rows.
- Non-English plural tolerance.
- Synonym tables (e.g. "Cinema" ≈ "Movies").

The library never sees users, folders-as-rows, or routing decisions. Input is
a torrent name + file list; output is a classification and a layout plan.
Callers are responsible for directory creation, target-folder selection, and
applying the plan.

## Versioning

`v0.x.y` — API may still shift between minor versions as the plugin surface
(custom detectors, richer range visibility) is designed. Once `v1.0.0` lands
the top-level `Classify` / `Sanitize` / `layout.DecideLayout` contract is
stable.

## License

Apache-2.0. See [LICENSE](LICENSE).
