// SPDX-License-Identifier: GPL-2.0-only

// Package brep implements the exact B-rep boolean (M20·F01, the Option-A kernel).
// The planar core — [Boolean] / [BooleanDiag] over an [Op] — imprints face–face
// intersections, splits faces along them via the 2D planar arrangement
// ([Arrange]), classifies the sub-faces against the other solid, and stitches the
// kept faces into a clean, low-face-count, chainable B-rep. On top of that core
// sit the exact analytic curved paths: half-space cuts ([HalfSpaceCut]),
// cylindrical/conical hole and boss primitives (CutCylindricalHole,
// JoinCylindricalBoss, …), and the crossing-cylinder / cone / Steinmetz general
// intersect–cut–join specials (ADR-0027; kind taxonomy in ADR-0045). Dispatch
// between these paths — and the residual triangle-CSG fallback — lives one level
// up in kernel/ops.
//
// The invariant that matters: every result face carries its source lineage
// forward (and new intersection edges are named by their generating face pair,
// ADR-0043), so reference keys survive the boolean — topological naming, not
// addresses (K1a; architecture core/05).
package brep
