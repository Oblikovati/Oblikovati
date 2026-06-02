// SPDX-License-Identifier: GPL-2.0-only

// Package geom holds ownerless, immutable analytic geometry value types —
// curves (line, arc, circle, ellipse, polyline) and, later, surfaces and NURBS.
//
// It is the modern, cgo-free form of the COM TransientGeometry factory
// (architecture/core/03-geometry-kernel.md): because Go has value semantics,
// these are plain structs created by ordinary constructors, with no factory and
// no marshaling. Evaluators are methods — PointAt(t), TangentAt(t) — and every
// type is immutable, so the kernel can evaluate them in parallel (ADR-0007).
//
// Coordinates and parameters are float64 in canonical database units (cm,
// radians). Curve construction lives here; persistent B-rep topology that
// *references* this geometry lives in kernel/topo (M07).
package geom
