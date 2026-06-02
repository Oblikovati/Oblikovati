// SPDX-License-Identifier: GPL-2.0-only

// Package ops is the kernel's modeling operations layer: tessellation, validation
// and healing, and boolean combination of bodies (architecture core/03, kernel
// phase A). All pure Go and cgo-free; functions take immutable topology and return
// new results, so recompute and tessellation parallelize trivially (ADR-0007).
//
// Phase A scope (this milestone): tolerance-driven tessellation of analytic faces
// and edges, body validation/healing, and the boolean framework with the
// non-overlapping topological cases. General robust booleans on intersecting NURBS
// solids (face-face intersection + classification) are the hardest part of the
// kernel and are phased to later (Phase C), behind the same API.
package ops
