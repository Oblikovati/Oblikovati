// SPDX-License-Identifier: GPL-2.0-only

// Package math is the mathematical vocabulary of the whole system: ownerless,
// immutable value types (points, vectors, unit vectors, matrices) in 2D and 3D.
//
// These types are the modern, cgo-free form of the COM TransientGeometry
// factory (see architecture/core/03-geometry-kernel.md). Because Go gives value
// semantics for free, there is no factory and no marshaling — a [Point3] is just
// a struct passed by value. Every type is immutable: operations return new
// values rather than mutating the receiver, which is what makes the kernel safe
// to run in parallel over shared inputs (ADR-0007).
//
// All coordinates are float64 in canonical database units (centimetres for
// length, radians for angle); see implementation-plan/CONVENTIONS.md. The
// renderer narrows to float32 only at the GPU boundary, never here.
//
// Naming maps to the public contract (ADR-0006) as: Point3→Point, Vector3→Vector,
// UnitVector3→UnitVector, Matrix4→Matrix; Point2→Point2d, Vector2→Vector2d,
// UnitVector2→UnitVector2d, Matrix3→Matrix2d.
package math
