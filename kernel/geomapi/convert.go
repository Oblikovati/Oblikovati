// SPDX-License-Identifier: GPL-2.0-only

// Package geomapi binds the transient-geometry kernel (math + kernel/geom) to the
// public contract (M01-F05, #602): the [Factory] implements
// contract.TransientGeometry, and thin adapters give every kernel curve/surface
// its contract interface — the kernel itself stays free of API conversions. The
// in-proc vs over-the-wire split is recorded in ADR-0018's geometry addendum.
package geomapi

import (
	"oblikovati.org/api/types"
	"oblikovati.org/math"
)

// Conversions between the kernel's math vocabulary and the contract's value
// types. Both are plain float64 triples; only the spelling differs.

func toPoint(p math.Point3) types.Point   { return types.Point{X: p.X, Y: p.Y, Z: p.Z} }
func fromPoint(p types.Point) math.Point3 { return math.P3(p.X, p.Y, p.Z) }

func toPoint2d(p math.Point2) types.Point2d   { return types.Point2d{X: p.X, Y: p.Y} }
func fromPoint2d(p types.Point2d) math.Point2 { return math.P2(p.X, p.Y) }

func toVector(v math.Vector3) types.Vector { return types.Vector{X: v.X, Y: v.Y, Z: v.Z} }

func toVector2d(v math.Vector2) types.Vector2d { return types.Vector2d{X: v.X, Y: v.Y} }

// toUnit converts a kernel unit direction; the invariant is already held, so the
// fields copy straight across.
func toUnit(u math.UnitVector3) types.UnitVector {
	return types.UnitVector{X: u.X(), Y: u.Y(), Z: u.Z()}
}

func toUnit2d(u math.UnitVector2) types.UnitVector2d {
	return types.UnitVector2d{X: u.X(), Y: u.Y()}
}

// fromUnit widens a contract unit direction to the kernel vector the geom
// constructors take (they re-validate and normalize internally).
func fromUnit(u types.UnitVector) math.Vector3 { return math.V3(u.X, u.Y, u.Z) }

func fromUnit2d(u types.UnitVector2d) math.Vector2 { return math.V2(u.X, u.Y) }

func toPoints(ps []math.Point3) []types.Point {
	out := make([]types.Point, len(ps))
	for i, p := range ps {
		out[i] = toPoint(p)
	}
	return out
}

func fromPoints(ps []types.Point) []math.Point3 {
	out := make([]math.Point3, len(ps))
	for i, p := range ps {
		out[i] = fromPoint(p)
	}
	return out
}

func toPoints2d(ps []math.Point2) []types.Point2d {
	out := make([]types.Point2d, len(ps))
	for i, p := range ps {
		out[i] = toPoint2d(p)
	}
	return out
}

func fromPoints2d(ps []types.Point2d) []math.Point2 {
	out := make([]math.Point2, len(ps))
	for i, p := range ps {
		out[i] = fromPoint2d(p)
	}
	return out
}
