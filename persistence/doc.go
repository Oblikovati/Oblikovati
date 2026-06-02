// SPDX-License-Identifier: GPL-2.0-only

// Package persistence is the on-disk document format: a portable ZIP package
// (".obk") of named binary streams, replacing COM's OLE structured storage with
// something cross-platform and tool-inspectable (architecture core/05).
//
// A [Package] is an ordered set of named streams (e.g. "manifest.json",
// "model/parameters.bin", "thumbnail.png") plus typed manifest access. It is read
// with [OpenPackage] and written with [Package.Save], which is atomic
// (write-temp-then-rename) so an interrupted save never corrupts the prior file.
// On open, [OpenPackage] runs the [Migrate] pipeline so older-schema packages are
// upgraded to the current version.
//
// [PackageStore] adapts a Package directory into the [doc.Store] seam the document
// workspace depends on (M03-F02): documents save and reopen as real files on disk.
// [DataIO] is the arbitrary-stream read/write surface used by add-ins and the
// attribute layer (M03-F06).
//
// Scope note: streams here are opaque byte blobs. The typed, columnar Codec[T]
// serialization keyed by stable TypeID (core/05) arrives once there is real model
// data to serialize — the feature program and sketches land from M07. Until then
// the recipe a document carries is just its manifest identity, which is exactly
// what round-trips today.
package persistence
