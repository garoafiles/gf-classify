// Package layout plans where a multi-file torrent's files should land under
// a caller-chosen target directory. It does not consult any filesystem, does
// not create directories, and does not know about users, folders-as-rows, or
// storage. A caller feeds it a classified torrent name plus the torrent's
// file list; it returns a Plan describing whether to wrap the files in a
// new directory and whether to strip a redundant top-level segment.
//
// The package depends on codeberg.org/garoafiles/gf-classify for its Sanitize
// and FuzzyEqual helpers, which are the same rules used by the root
// Classify function. Using the same rules here means the wrapper-name
// comparison in Collapse behaves identically to Classify's own name
// handling.
package layout
