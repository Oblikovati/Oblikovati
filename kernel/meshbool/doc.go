// SPDX-License-Identifier: GPL-2.0-only

// Package meshbool is the exact-arithmetic mesh-arrangement boolean core
// (ADR-0052 Track A). It replaces the classify-each-operand-independently-then-
// weld planar boolean, whose near-tangent grazing seams tear (Oblikovati#2084),
// with a unified co-refined arrangement whose output is manifold by construction.
//
// # Exact coordinates, then round once
//
// Arrangement vertices are [Point]s with rational (math/big.Rat) coordinates. An
// original mesh vertex converts to a Point with no rounding (a float64 is a dyadic
// rational); a constructed intersection vertex — an edge crossing a plane — is
// built by exact rational arithmetic, so it lands EXACTLY on the surfaces that
// define it. Equality, orientation, and containment therefore stay exact through
// the entire co-refinement: two faces that share an edge get the SAME split point
// on it, by construction, which is exactly the conforming property whose absence
// tears #2084. Precision is lost only at the very end, in [Point.Round], when the
// finished manifold is written back to float64 positions.
//
// This is the libigl/CGAL discipline (exact predicates AND exact constructions);
// it builds on the exact sign predicates in kernel/predicates.
package meshbool
