// SPDX-License-Identifier: GPL-2.0-only

// Package ops is the kernel's modeling operations layer: tessellation, validation
// and healing, and boolean combination of bodies (architecture core/03, kernel
// phase A). All pure Go and cgo-free; functions take immutable topology and return
// new results, so recompute and tessellation parallelize trivially (ADR-0007).
//
// Scope: tolerance-driven tessellation of analytic faces and edges, body
// validation/healing, and boolean combination. Booleans cover the non-overlapping
// topological cases (disjoint / one-contains-the-other) plus general intersecting
// solids: planar-faceted operands go through the exact planar B-rep boolean
// (kernel/brep), and operands with a curved face fall back to a triangle-soup BSP
// CSG (see booleanCSG). Exact booleans on curved/NURBS solids — analytic face-face
// intersection rather than the faceted CSG approximation — remain future work,
// behind the same API.
package ops
