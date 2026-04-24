// Package classify inspects a torrent's display name and file list and picks
// a category (TV / Movies / Music / Games / Software / Books / Other) plus
// best-effort sub-info (show, artist, title, year).
//
// It is a pure-Go, regex-only classifier: no external metadata services are
// consulted, no network traffic is produced, no persistence is performed.
// Feed it a name and a slice of File and you get back a Result.
//
// The ordering of the category switch in Classify is load-bearing: TV is
// checked before Movies because a torrent like "Some.Show.S01.2023.Complete"
// would match the movie year regex but belongs under TV.
//
// For directory-layout decisions on multi-file torrents, see the sibling
// package codeberg.org/garoafiles/gf-classify/layout.
package classify
