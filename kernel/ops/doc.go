// SPDX-License-Identifier: GPL-2.0-only

// Package ops is the kernel's modeling operations layer (architecture core/03, kernel phase A).
// All pure Go and cgo-free; functions take immutable topology and return new results, so
// recompute and tessellation parallelize trivially (ADR-0007).
//
// # Package by operation
//
// The layer is split by OPERATION, not by case (#2183), so a change to one family neither
// rebuilds nor re-tests the others:
//
//	kernel/ops/boolean    combining bodies: classification, the curved surface-pair paths, the
//	                      CSG fallback, and the mesh-arrangement / analytic-reconstruction stack
//	kernel/ops/blend      fillet, chamfer and draft — one blend engine (ADR-0050/0051)
//	kernel/ops/tessellate the tolerance-driven mesher for analytic and NURBS faces and edges
//	kernel/ops/validate   the ordered validity levels every operation runs as a post-condition
//	kernel/ops/heal       sew, stitch, snap, cap, fill, drop faces, void classification
//	kernel/ops/query      the read-only questions: pick, mass properties, boxes, containment
//	kernel/ops/surface    the face operations: extend, offset, replace, rebuild, fair, untrim
//	kernel/ops/transform  moving, deforming and re-surfacing a body
//
// Substrate several families share lives in kernel/ops/internal: mesh (the Mesh type and the
// point welder), probe (read-only geometric questions), retopo (rebuild helpers), tol (the
// model-relative tolerance constructors) and disjoint (union-find).
//
// # What stays here
//
// The general operations that belong to no one family — section by plane, shell, split and
// thicken — plus the aliases that hold the boolean enum and entry points steady for their
// ~1100 call sites (boolean_alias.go). Everything else is reached by naming its package.
//
// Booleans cover the non-overlapping topological cases (disjoint / one-contains-the-other)
// plus general intersecting solids: planar-faceted operands go through the exact planar B-rep
// boolean (kernel/brep), and curved operands through the EXACT analytic curved-boolean paths
// (surface–surface intersection → (u,v) arrangement → classify → stitch, plus analytic
// specials — ADR-0027; kind taxonomy in ADR-0045). Triangle-soup BSP CSG survives only as the
// residual fallback when no exact path applies, and taking it is recorded as a tracked Defect
// diagnostic (CodeBooleanCSGFallback), never a silent degradation.
package ops
