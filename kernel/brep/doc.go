// SPDX-License-Identifier: GPL-2.0-only

// Package brep implements the exact B-rep boolean (M20·F01, the Option-A kernel).
// The planar core — [Boolean] / [BooleanDiag] over an [Op] — imprints face–face
// intersections, splits faces along them via the 2D planar arrangement
// ([Arrange]), classifies the sub-faces against the other solid, and stitches the
// kept faces into a clean, low-face-count, chainable B-rep. On top of that core
// sits the per-face dispatch that carries curved operands too (ADR-0058): a face
// is trimmed in its own chart by the sections it meets, and the kept faces stitch
// with the planar ones. There is ONE such path — the 26-recognizer ladder above it
// and the triangle-CSG fallback below it are both deleted (ADR-0061 stages 4, 6
// and 7), so a pair the pipeline cannot model is refused by name rather than
// approximated. Modelling PRIMITIVES stay ([HalfSpaceCut], [CutCylindricalHole],
// the revolution and sweep builders): those construct geometry, they do not
// recognise a boolean.
//
// The invariant that matters: every result face carries its source lineage
// forward (and new intersection edges are named by their generating face pair,
// ADR-0043), so reference keys survive the boolean — topological naming, not
// addresses (K1a; architecture core/05).
package brep
