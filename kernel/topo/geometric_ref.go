// SPDX-License-Identifier: GPL-2.0-only

package topo

import "oblikovati.org/math"

// SPIKE (M8, NX exporter): resolving a face/edge selection from a SERIALIZED geometric
// descriptor rather than an Oblikovati lineage key. The exact-lineage binder
// ([Body.FindFaceByKey]) and the M31 in-memory geometric fallback both need lineage an
// external author (the NX exporter) cannot supply. A geometric descriptor — a face's
// centroid+outward-normal, an edge's midpoint+direction, all in model space — is what an
// external author CAN supply, and is enough to re-find the entity on a freshly recomputed
// body. This is intentionally a parallel, non-identity path: it never feeds a key's
// identity bytes (preserving the M31-F07 rule that the drift-prone anchor stays out of
// identity); it only recovers a selection at recompute, and the caller maps a hit to
// health.Warning (auto-healed, flagged), exactly like the M31 fallback tiers.

// normalAlignMin is the minimum dot product (unit normals) for a candidate face's outward
// normal to count as aligned with the descriptor's — ~25°, enough to reject the opposite
// face of a thin feature while tolerating tessellation/curvature drift.
const normalAlignMin = 0.9

// GeometricFaceRef names a face by where it sits, not by lineage: a representative point
// (its outer-loop centroid) and its outward unit normal, both in model space.
type GeometricFaceRef struct {
	Centroid math.Point3
	Normal   math.Vector3
}

// GeometricEdgeRef names an edge by its midpoint and (sign-agnostic) direction.
type GeometricEdgeRef struct {
	Midpoint  math.Point3
	Direction math.Vector3
}

// DescribeFace produces the geometric descriptor of a face — the inverse of
// [Body.FindFaceByGeometry]. An external author (the NX exporter) builds the descriptor
// from NX geometry instead; this is for converting an Oblikovati-side selection into a
// portable ref and for round-trip testing.
func DescribeFace(f *Face) GeometricFaceRef {
	c := faceCentroid(f)
	return GeometricFaceRef{Centroid: c, Normal: faceOutwardNormal(f, c)}
}

// DescribeEdge produces the geometric descriptor of an edge.
func DescribeEdge(e *Edge) GeometricEdgeRef {
	return GeometricEdgeRef{Midpoint: edgeMidpoint(e), Direction: edgeDirection(e)}
}

// FindFaceByGeometry returns the face whose centroid is within tol of the descriptor and
// whose outward normal aligns with it, when that face is unambiguous. A tie (two equally
// near aligned faces) returns false rather than binding the wrong face — the same
// "defensible or lost" rule the M31 geometric tier uses.
func (b *Body) FindFaceByGeometry(ref GeometricFaceRef, tol math.Scalar) (*Face, bool) {
	want := unitOrZero(ref.Normal)
	var best, second *Face
	var bestD, secondD math.Scalar
	for _, f := range b.Faces() {
		c := faceCentroid(f)
		d := c.DistanceTo(ref.Centroid)
		if d > tol {
			continue
		}
		if want.LengthSquared() > 0 && faceOutwardNormal(f, c).Dot(want) < normalAlignMin {
			continue
		}
		best, bestD, second, secondD = rank(f, d, best, bestD, second, secondD)
	}
	return decide(best, bestD, second, secondD)
}

// FindEdgeByGeometry returns the edge whose midpoint is within tol of the descriptor and
// whose direction is parallel to it (either sense), when unambiguous.
func (b *Body) FindEdgeByGeometry(ref GeometricEdgeRef, tol math.Scalar) (*Edge, bool) {
	want := unitOrZero(ref.Direction)
	var best, second *Edge
	var bestD, secondD math.Scalar
	for _, e := range b.Edges() {
		d := edgeMidpoint(e).DistanceTo(ref.Midpoint)
		if d > tol {
			continue
		}
		if want.LengthSquared() > 0 && absScalar(edgeDirection(e).Dot(want)) < normalAlignMin {
			continue
		}
		best, bestD, second, secondD = rankEdge(e, d, best, bestD, second, secondD)
	}
	return decideEdge(best, bestD, second, secondD)
}

// rank keeps the nearest and second-nearest face candidates (generic-free to stay
// readable); decide turns them into a unique-or-lost result.
func rank(f *Face, d math.Scalar, best *Face, bestD math.Scalar, second *Face, secondD math.Scalar) (*Face, math.Scalar, *Face, math.Scalar) {
	switch {
	case best == nil || d < bestD:
		return f, d, best, bestD
	case second == nil || d < secondD:
		return best, bestD, f, d
	default:
		return best, bestD, second, secondD
	}
}

func decide(best *Face, bestD math.Scalar, second *Face, secondD math.Scalar) (*Face, bool) {
	if best == nil {
		return nil, false
	}
	if second != nil && bestD == secondD {
		return nil, false // an indefensible geometric tie
	}
	return best, true
}

func rankEdge(e *Edge, d math.Scalar, best *Edge, bestD math.Scalar, second *Edge, secondD math.Scalar) (*Edge, math.Scalar, *Edge, math.Scalar) {
	switch {
	case best == nil || d < bestD:
		return e, d, best, bestD
	case second == nil || d < secondD:
		return best, bestD, e, d
	default:
		return best, bestD, second, secondD
	}
}

func decideEdge(best *Edge, bestD math.Scalar, second *Edge, secondD math.Scalar) (*Edge, bool) {
	if best == nil {
		return nil, false
	}
	if second != nil && bestD == secondD {
		return nil, false
	}
	return best, true
}

// faceCentroid averages the face's outer-loop vertices — the face centre for a planar
// quad, and a stable representative for any face.
func faceCentroid(f *Face) math.Point3 {
	var sx, sy, sz math.Scalar
	var n math.Scalar
	for _, l := range f.Loops() {
		if !l.IsOuter() {
			continue
		}
		for _, u := range l.EdgeUses() {
			p := u.Edge().StartVertex().Point()
			sx, sy, sz, n = sx+p.X, sy+p.Y, sz+p.Z, n+1
		}
	}
	if n == 0 {
		return math.Point3{}
	}
	return math.P3(sx/n, sy/n, sz/n)
}

// faceOutwardNormal evaluates the face's outward unit normal at p (mirrors
// ops.outwardFaceNormalAt, kept local to the spike).
func faceOutwardNormal(f *Face, p math.Point3) math.Vector3 {
	u, v := f.Geometry().ParamAt(p)
	nrm := f.Geometry().NormalAt(u, v)
	if f.Reversed() {
		nrm = nrm.Scale(-1)
	}
	return unitOrZero(nrm)
}

func edgeMidpoint(e *Edge) math.Point3 {
	a, b := e.StartVertex().Point(), e.EndVertex().Point()
	return math.P3((a.X+b.X)/2, (a.Y+b.Y)/2, (a.Z+b.Z)/2)
}

func edgeDirection(e *Edge) math.Vector3 {
	return unitOrZero(e.StartVertex().Point().VectorTo(e.EndVertex().Point()))
}

func unitOrZero(v math.Vector3) math.Vector3 {
	if v.LengthSquared() == 0 {
		return math.Vector3{}
	}
	return v.Scale(1 / v.Length())
}

func absScalar(s math.Scalar) math.Scalar {
	if s < 0 {
		return -s
	}
	return s
}
