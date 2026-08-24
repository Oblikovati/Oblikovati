// SPDX-License-Identifier: GPL-2.0-only

// Package predicates provides adaptive-precision, exact-sign geometric
// predicates: the orientation and in-sphere tests a robust boolean/arrangement
// core needs so that every topological decision (which side of a plane, do these
// cross, are these coplanar) is provably consistent and never contradicts itself.
//
// # Why exact signs
//
// The planar B-rep boolean tears at near-tangent grazing seams (Oblikovati#2084,
// ADR-0052) because two operands are arranged and classified with tolerance-based
// floating-point tests, made independently, that then disagree at facet
// resolution. Any distance tolerance is a compromise that eventually lands a case
// in its gap. An exact-sign predicate removes the gap: orient3d(a,b,c,d) returns a
// value whose SIGN is exactly the sign of the true determinant, with no epsilon,
// so "d is above plane(a,b,c)" is a single global truth all consumers agree on.
//
// # Design: static filter, then exact rational
//
// Two-stage, smallest-trusted-base:
//
//   - Fast path — a floating-point estimate of the determinant, returned only when
//     its sign is certified by Shewchuk's a-priori forward-error bound (the
//     "static filter"; error-bound constants from "Adaptive Precision
//     Floating-Point Arithmetic and Fast Robust Geometric Predicates", J. R.
//     Shewchuk, Discrete & Computational Geometry 18:305-363, 1997). This resolves
//     every non-degenerate input at floating-point speed and allocation-free.
//   - Exact path — when the filter cannot certify the sign, the determinant is
//     recomputed over math/big.Rat. Each binary64 coordinate converts to a
//     rational with NO rounding, so the rational determinant's sign is the exact
//     mathematical sign. This is the entire correctness guarantee; it has no
//     epsilon and no porting risk.
//
// Shewchuk's adaptive expansion stages (a cheaper exact path than big.Rat) are a
// deliberate future optimization: they would shrink the big.Rat call rate without
// changing any result, and should be added only if profiling the arrangement core
// shows the exact path is hot. Keeping them out now keeps the trusted base minimal.
//
// # FMA hazard (Oblikovati#2020) — load-bearing
//
// Shewchuk's error-free transforms (Two-Product, Two-Sum) assume every individual
// multiply and add is separately rounded to binary64. Go MAY fuse a*b+c into a
// single unrounded FMA — and DOES on arm64 but not amd64 (see the #2020 note in
// kernel/ops/fillet_test.go). A fused product silently violates the transforms and
// makes these predicates neither exact nor cross-platform-stable. Every product
// here that feeds a later add/sub is therefore wrapped with the identity
// conversion float64(...), which the Go spec guarantees rounds to binary64 and so
// blocks the fusion. Do NOT remove those conversions; see rounded() in filter.go.
// (The exact big.Rat path is immune to fusion; the guard protects the filter's
// estimate, whose certified error bound assumes each op is separately rounded.)
package predicates
