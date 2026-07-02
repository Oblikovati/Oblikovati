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
// (kernel/brep), and curved operands go through the EXACT analytic curved-boolean
// paths (curvedExactBoolean: surface–surface intersection → (u,v) arrangement →
// classify → stitch, plus analytic specials — ADR-0027; kind taxonomy in
// ADR-0045). Triangle-soup BSP CSG (booleanCSG) survives only as the residual
// fallback when no exact path applies, and taking it is recorded as a tracked
// Defect diagnostic (CodeBooleanCSGFallback), never a silent degradation.
package ops
