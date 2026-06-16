// SPDX-License-Identifier: GPL-2.0-only

// Package dwg is a clean-room decoder/encoder for the AutoCAD DWG binary drawing
// format, used by the host to import, edit and export .dwg files into the native
// drawing/sketch model.
//
// # Clean-room provenance
//
// This package is implemented from the publicly published Open Design Alliance
// "Open Design Specification for .dwg files" and the Autodesk DXF group-code
// reference. The GPL-3.0 LibreDWG project is used ONLY as a behavioural oracle in
// tests (to cross-check decoded values); no LibreDWG source is copied or
// translated here. Keeping it clean-room is what lets this package stay
// GPL-2.0-only alongside the rest of /source rather than being forced to GPL-3.0.
//
// # Layout
//
// The decoder is built bottom-up, each layer validated independently:
//
//   - bitreader.go  — the unaligned ODA bit-stream primitives (B, RC, BS, BL, BD,
//     MC, H, …). DWG packs values across byte boundaries, so every higher layer
//     reads through [BitReader].
//   - crc.go        — the CRC8 (16-bit), CRC32 and Reed-Solomon integrity checks
//     that frame DWG sections and pages.
//   - version.go    — magic-byte detection of the format generation, which selects
//     the structural branch (R2000 flat sections vs R2004+ paged/compressed).
//
// Higher layers (section/page layout, LZ77 decompression, the object map and the
// entity decoders that populate the [model] object graph) build on these.
package dwg
