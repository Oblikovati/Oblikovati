// SPDX-License-Identifier: GPL-2.0-only

// Package persistence is the on-disk document format: a single **git-friendly YAML
// text file** (".obk") per document, replacing COM's OLE structured storage with
// something cross-platform, tool-inspectable, and version-controllable (ADR-0020,
// architecture core/05). The earlier ZIP-of-binary-streams container is gone — a
// document is now one text file git can diff and merge.
//
// A [Package] is the in-memory document model: the manifest identity (schema
// version, document kind, display name), the model recipe section, and named text
// data sections (DataIO / attributes). It is read with [OpenPackage] and written
// with [Package.Save], which is atomic (write-temp-then-rename) so an interrupted
// save never corrupts the prior file. On open, [OpenPackage] runs the [Migrate]
// pipeline so older-schema documents are upgraded to the current version.
//
// [PackageStore] adapts a Package into the [doc.Store] seam the document workspace
// depends on (M03-F02): documents save and reopen as real files on disk. [DataIO]
// is the arbitrary text-section read/write surface used by add-ins and the
// attribute layer (M03-F06).
//
// Recipe-only, no binaries (ADR-0020): the file holds the regenerable recipe
// (parameters, sketches, features) as text; realized geometry is recomputed on
// open, vector references are SVG, and thumbnails are generated git-ignored
// sidecars. The one opaque exception is the reference-key context (topological
// naming), carried as a base64 string because correct rebinding requires it.
package persistence
