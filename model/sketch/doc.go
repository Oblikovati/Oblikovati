// SPDX-License-Identifier: GPL-2.0-only

// Package sketch is the constrained-sketch environment: a 2D (or 3D) program of
// geometry plus geometric and dimensional constraints that a solver resolves into
// curves, exporting [Profile]/[Path] — the only thing features consume
// (parametric-cad §10, architecture modeling/00, ADR-0009).
//
// The solver lives entirely behind that profile boundary: it consumes entities +
// constraints and emits solved positions; the kernel never sees the solver and the
// solver never sees the B-rep, so the two evolve independently. Everything here is
// pure Go and headless-testable.
//
// Design highlights (modeling/00):
//   - Entities carry shared constrainable points ([Point]); a shared endpoint *is*
//     a coincidence, structurally.
//   - A planar [Sketch] hosts on a [Plane] and maps points 2D↔3D
//     ([Plane.ToModel]/[Plane.ToSketch]).
//   - Dimensional constraints own a parameter (core/04): the dimension value is an
//     editable expression in the parameter DAG; editing it re-solves the sketch.
//   - The solver is decompose-plus-Newton (ADR-0009); a whole-system Newton solve
//     ships first behind the same Solve() API, with DOF analysis as a byproduct.
package sketch
