// SPDX-License-Identifier: GPL-2.0-only

// Package dxf is a clean-room ASCII DXF (Drawing Exchange Format) codec. It decodes a DXF
// file's model-space curve geometry into, and encodes it from, the format-neutral
// kernel/exchange/drawing model — the same model the DWG codec uses, so both share one
// Sketch converter.
//
// DXF is a text format: a flat stream of group-code/value line pairs grouped into named
// sections (HEADER, TABLES, BLOCKS, ENTITIES, OBJECTS). The decoder tolerates the
// real-world variation in those files (code padding, CRLF, omitted/defaulted/reordered
// group codes); the encoder emits the standard section set AutoCAD expects, targeting
// either R2000 (AC1015) or R2018 (AC1032).
//
// Sources: the Autodesk DXF group-code reference and the ODA specification (group codes are
// shared between DWG and DXF). No third-party code is incorporated.
package dxf
