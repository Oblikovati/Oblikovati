// SPDX-License-Identifier: GPL-2.0-only

// Package probe holds the read-only geometric questions the operation families all
// ask: a face's outward normal at a point, whether a triangle's winding opposes its
// vertex normals, the overlap of two boxes, and how far along a curve to sample.
//
// None of them changes anything, and none needs the operation layer. They sat in
// kernel/ops only because that is where their first caller was, which made every
// package that wanted to ask one of these questions — validate, query, blend — depend
// on the whole operation layer.
//
//	n := probe.OutwardFaceNormalAt(f, p)
package probe

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// CurveTrimSamples is the interior-sample count along an intersection curve (a count, not a tolerance):
// enough that a short in-trim crossing arc of a curve clipped to the whole shared face box is not
// stepped over.
const CurveTrimSamples = 16

// BoxOverlap returns the intersection box of a and b (their per-axis overlap). Its callers reach it
// only after Box.Intersects reported true, so every axis has Min ≤ Max.
func BoxOverlap(a, b math.Box) math.Box {
	return math.Box{
		Min: math.P3(stdmath.Max(a.Min.X, b.Min.X), stdmath.Max(a.Min.Y, b.Min.Y), stdmath.Max(a.Min.Z, b.Min.Z)),
		Max: math.P3(stdmath.Min(a.Max.X, b.Max.X), stdmath.Min(a.Max.Y, b.Max.Y), stdmath.Min(a.Max.Z, b.Max.Z)),
	}
}

// SampleRange returns a finite parameter interval of curve c to sample, bounded to the box overlap. A
// bounded intersection curve (a closed loop from two curved faces) uses its own finite domain (ok=true).
// An UNBOUNDED curve — the infinite line a closed-form plane∩plane returns, domain [-Inf,+Inf] — is
// bounded by projecting overlap's eight corners onto the line: the line is affine, so the parameter of a
// point q is (q−P0)·d / |d|² with d = P(1)−P(0), and the corner projections bracket the box. An unbounded
// curve that is NOT a straight line (a cone section — parabola/hyperbola) cannot be bracketed this way,
// so ok=false and the caller conservatively treats the pair as crossing rather than risk a missed hit.
func SampleRange(c geom.Curve3, overlap math.Box) (lo, hi float64, ok bool) {
	dlo, dhi := c.Domain()
	if !stdmath.IsInf(dlo, 0) && !stdmath.IsInf(dhi, 0) {
		return dlo, dhi, true
	}
	p0, p1, pmid := c.PointAt(0), c.PointAt(1), c.PointAt(0.5)
	d := p0.VectorTo(p1)
	dd := d.Dot(d)
	if dd == 0 || !IsColinearMidpoint(p0, p1, pmid) {
		return 0, 0, false
	}
	lo, hi = stdmath.Inf(1), stdmath.Inf(-1)
	for _, q := range overlap.Corners() {
		t := p0.VectorTo(q).Dot(d) / dd
		lo, hi = stdmath.Min(lo, t), stdmath.Max(hi, t)
	}
	return lo, hi, true
}

// OutwardFaceNormalAt is a face's outward unit normal at point p on it (the surface normal,
// flipped when the face is reversed).
func OutwardFaceNormalAt(f *topo.Face, p math.Point3) math.Vector3 {
	u, v := f.Geometry().ParamAt(p)
	n := f.Geometry().NormalAt(u, v)
	if f.Reversed() {
		n = n.Scale(-1)
	}
	return UnitOr(n)
}

// WindingOpposesNormals reports whether triangle abc's geometric (cross-product) normal opposes
// the triangle's per-vertex shading normals (their sum). Used to wind a patch consistently with
// the normals each vertex actually carries — robust on a curved patch where the flat triangle's
// centroid is off the surface (unlike triangleFlipped's centroid sample).
func WindingOpposesNormals(a, b, c math.Point3, na, nb, nc math.Vector3) bool {
	gn := a.VectorTo(b).Cross(a.VectorTo(c))
	return gn.Dot(na.Add(nb).Add(nc)) < 0
}

// IsColinearMidpoint reports whether c's midpoint sample lies on the chord P0→P1 — i.e. the curve is a
// straight line over [0,1]. An intersection conic that is colinear at three parameters is degenerate to
// a line, so this cleanly separates the affine plane∩plane line (bracketable by corner projection) from
// a curved unbounded section that is not.
func IsColinearMidpoint(p0, p1, pmid math.Point3) bool {
	chord := p0.VectorTo(p1)
	mid := math.P3((p0.X+p1.X)/2, (p0.Y+p1.Y)/2, (p0.Z+p1.Z)/2)
	return pmid.VectorTo(mid).Length() <= colinearRelTol*chord.Length()
}

// UnitOr normalizes v, returning it unchanged when degenerate (zero length).
func UnitOr(v math.Vector3) math.Vector3 {
	l := v.Length()
	if l < math.Scalar(stdmath.Sqrt(float64(math.DefaultTolerance))) {
		return v
	}
	return v.Scale(1 / l)
}

// colinearRelTol is the chord-relative straightness bound for an unbounded section curve: a true line's
// midpoint deviates only by float rounding, far below this, while any conic bows well above it.
const colinearRelTol = 1e-9 // tol:relative — dimensionless straightness fraction of the chord length
