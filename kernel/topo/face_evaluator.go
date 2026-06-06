// SPDX-License-Identifier: GPL-2.0-only

package topo

import (
	stdmath "math"

	"oblikovati/kernel/geom"
	"oblikovati/math"
)

// FaceEvaluator adds face-level queries (area, containment) to the surface
// evaluator.
type FaceEvaluator struct {
	SurfaceEvaluator
	face *Face
}

// NewFaceEvaluator wraps a face.
func NewFaceEvaluator(f *Face) FaceEvaluator {
	return FaceEvaluator{SurfaceEvaluator: NewSurfaceEvaluator(f.surface), face: f}
}

// Area returns the face area and whether it is exact. Planar faces are computed
// exactly from the outer boundary (Newell's formula); non-planar faces need
// tessellation (M07-F03) and return (0, false) for now.
func (e FaceEvaluator) Area() (float64, bool) {
	if _, ok := e.face.surface.(geom.Plane); !ok {
		return 0, false
	}
	return newellArea(outerBoundary(e.face)), true
}

// Contains reports whether p lies within the face's outer boundary and outside its
// holes (planar faces). It first checks p is on the face plane.
func (e FaceEvaluator) Contains(p math.Point3) bool {
	plane, ok := e.face.surface.(geom.Plane)
	if !ok {
		return false
	}
	n := plane.NormalAt(0, 0)
	if stdmath.Abs(plane.Origin.VectorTo(p).Dot(n)) > 1e-6 {
		return false // off the plane
	}
	if !pointInLoop(p, outerBoundary(e.face), n) {
		return false
	}
	for _, l := range e.face.loops {
		if !l.outer && pointInLoop(p, loopBoundary(l), n) {
			return false // inside a hole
		}
	}
	return true
}

// outerBoundary returns the ordered vertices of the face's outer loop.
func outerBoundary(f *Face) []math.Point3 {
	for _, l := range f.loops {
		if l.outer {
			return loopBoundary(l)
		}
	}
	return nil
}

// loopBoundary returns the ordered "from" vertices of a loop's edge uses.
func loopBoundary(l *Loop) []math.Point3 {
	pts := make([]math.Point3, 0, len(l.uses))
	for _, u := range l.uses {
		from := u.edge.start
		if u.reversed {
			from = u.edge.end
		}
		pts = append(pts, from.point)
	}
	return pts
}

// newellArea returns the area of a planar polygon via Newell's method (robust in 3D).
func newellArea(poly []math.Point3) float64 {
	if len(poly) < 3 {
		return 0
	}
	var nx, ny, nz float64
	for i, a := range poly {
		b := poly[(i+1)%len(poly)]
		nx += (a.Y - b.Y) * (a.Z + b.Z)
		ny += (a.Z - b.Z) * (a.X + b.X)
		nz += (a.X - b.X) * (a.Y + b.Y)
	}
	return 0.5 * stdmath.Sqrt(nx*nx+ny*ny+nz*nz)
}

// pointInLoop tests planar containment by dropping the normal's dominant axis and
// ray-casting in 2D.
func pointInLoop(p math.Point3, poly []math.Point3, normal math.Vector3) bool {
	flat := dropAxis(normal)
	pp := flat(p)
	inside := false
	n := len(poly)
	for i, j := 0, n-1; i < n; j, i = i, i+1 {
		vi, vj := flat(poly[i]), flat(poly[j])
		if (vi.Y > pp.Y) != (vj.Y > pp.Y) {
			x := vi.X + (pp.Y-vi.Y)/(vj.Y-vi.Y)*(vj.X-vi.X)
			if pp.X < x {
				inside = !inside
			}
		}
	}
	return inside
}

// dropAxis returns a projection to 2D that drops the axis of the largest normal
// component, preserving in-plane geometry.
func dropAxis(n math.Vector3) func(math.Point3) math.Point2 {
	ax, ay, az := stdmath.Abs(n.X), stdmath.Abs(n.Y), stdmath.Abs(n.Z)
	switch {
	case ax >= ay && ax >= az:
		return func(p math.Point3) math.Point2 { return math.P2(p.Y, p.Z) }
	case ay >= ax && ay >= az:
		return func(p math.Point3) math.Point2 { return math.P2(p.X, p.Z) }
	default:
		return func(p math.Point3) math.Point2 { return math.P2(p.X, p.Y) }
	}
}
