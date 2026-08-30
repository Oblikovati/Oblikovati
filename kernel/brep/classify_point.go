// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Containment is where a point sits relative to a solid: strictly outside the material, on its
// boundary surface, or strictly inside. It is brep's own tri-state because brep cannot import ops
// (ops depends on brep); ops.BodyContainment maps it to the public PointContainment.
type Containment uint8

const (
	Outside Containment = iota
	OnSurface
	Inside
)

// rayTangentCos rejects a ray that meets a face almost tangentially (|d·n| below this): the two
// pierce parameters coincide there, so the crossing count is ambiguous. d and n are unit vectors,
// so the threshold is a dimensionless angular one.
const rayTangentCos = 1e-6 // tol:angular — ray-vs-face grazing (unit dir · unit normal)

// boundarySampleCount is the per-edge sampling of a loop when measuring a pierce point's distance
// to a face boundary, to catch a ray that crosses through a trim edge (shared by two faces), where
// parity would double- or under-count.
const boundarySampleCount = 24

// rayDirections are the fixed candidate cast directions, tried in order until one crosses the body
// cleanly. They are deliberately non-axis-aligned and mutually non-parallel (integer vectors with no
// shared ratios), and the list is fixed — no randomness — so classification is byte-identical across
// runs and platforms (the kernel determinism rule).
var rayDirections = [][3]float64{
	{2, 3, 5}, {3, -5, 2}, {-5, 2, 3}, {1, -1, 4}, {4, 4, -1}, {-3, 6, -2},
}

// ClassifyPoint reports whether p is inside, on, or outside the solid body b, by exact analytic ray
// casting — the OCCT BRepClass3d_SolidClassifier analog. It counts the boundary crossings of a ray
// from p (each crossing a ray∩surface pierce that lies within its face's trimmed region, via the
// exact [RaySurfaceHits] and the parameter-space [pointInTrimUV]); even–odd parity gives
// material membership independent of face orientation. A ray that grazes a face boundary or runs
// tangent is discarded for the next candidate direction. It reads no tessellation — the analytic
// replacement for the mesh winding oracle (M48/C3 #3427).
//
// Example — the centre of a unit cube is inside, a far point outside:
//
//	cube, _ := brep.Box(math.P3(0,0,0), math.P3(1,1,1))
//	brep.ClassifyPoint(cube, math.P3(0.5, 0.5, 0.5)) // Inside
func ClassifyPoint(b *topo.Body, p math.Point3) Containment {
	faces := facesOfAny(b)
	if len(faces) == 0 {
		return Outside
	}
	tol := geom.ResolutionForBox(b.RangeBox()).Sew()
	if onAnyFace(faces, p, tol) {
		return OnSurface
	}
	tMax := 2 * float64(b.RangeBox().Diagonal().Length())
	for _, d := range rayDirections {
		if crossings, ok := rayCrossings(faces, p, d, tMax, tol); ok {
			return insideFromParity(crossings)
		}
	}
	return Outside // every candidate direction grazed a boundary: astronomically unlikely
}

// insideFromParity maps an odd crossing count to Inside, even to Outside.
func insideFromParity(crossings int) Containment {
	if crossings%2 == 1 {
		return Inside
	}
	return Outside
}

// rayCrossings counts the boundary crossings of the ray p→dir within [0, tMax]. ok is false when a
// pierce grazes a face boundary or runs tangent, signalling the caller to try another direction
// (a reselection, not a geometry nudge).
func rayCrossings(faces []curvedFace, p math.Point3, dir [3]float64, tMax, tol float64) (int, bool) {
	ray, err := geom.NewLine(p, math.V3(dir[0], dir[1], dir[2]))
	if err != nil {
		return 0, false
	}
	count := 0
	for i := range faces {
		n, ok := faceRayCrossings(faces[i], ray, tMax, tol)
		if !ok {
			return 0, false
		}
		count += n
	}
	return count, true
}

// faceRayCrossings counts how many of the ray's pierces of one face's surface land inside that
// face's trim. ok is false on a grazing/tangent pierce (see rayCrossings).
func faceRayCrossings(f curvedFace, ray geom.Line, tMax, tol float64) (int, bool) {
	count := 0
	for _, hit := range geom.RaySurfaceHits(f.surface, ray, tMax) {
		if rayGrazes(f, ray, hit, tol) {
			return 0, false
		}
		if pointInTrimUV(f, hit.Point) {
			count++
		}
	}
	return count, true
}

// rayGrazes reports an ambiguous pierce: the ray meets the surface almost tangentially, or the
// pierce point sits within tol of the face's trim boundary (where a shared edge would be counted on
// two faces or none). Either case is resolved by choosing a different ray direction.
func rayGrazes(f curvedFace, ray geom.Line, hit geom.RayHit, tol float64) bool {
	n := f.surface.NormalAt(hit.U, hit.V)
	if n.LengthSquared() == 0 || stdmath.Abs(float64(ray.Dir.AsVector().Dot(n))) < rayTangentCos {
		return true
	}
	return nearFaceBoundary(f, hit.Point, tol)
}

// nearFaceBoundary reports whether q lies within tol of any of the face's boundary loops, sampling
// each edge so a pierce through a curved trim edge is caught.
func nearFaceBoundary(f curvedFace, q math.Point3, tol float64) bool {
	for _, loop := range f.loops {
		for _, e := range loop.edges {
			if edgeDistanceTo(e, q) < tol {
				return true
			}
		}
	}
	return false
}

// edgeDistanceTo returns the least distance from q to the loop edge, sampled as a polyline over its
// oriented parameter interval.
func edgeDistanceTo(e loopEdge, q math.Point3) float64 {
	best := stdmath.Inf(1)
	prev := e.curve.PointAt(e.t0)
	for k := 1; k <= boundarySampleCount; k++ {
		t := e.t0 + (e.t1-e.t0)*float64(k)/boundarySampleCount
		cur := e.curve.PointAt(t)
		if d := pointSegmentDistance(q, prev, cur); d < best {
			best = d
		}
		prev = cur
	}
	return best
}

// onAnyFace reports whether p sits on the trimmed boundary of any face — within tol of the surface
// and inside that face's trim region.
func onAnyFace(faces []curvedFace, p math.Point3, tol float64) bool {
	for i := range faces {
		_, _, foot := geom.ClosestPointOnSurface(faces[i].surface, p)
		if float64(foot.DistanceTo(p)) < tol && pointInTrimUV(faces[i], p) {
			return true
		}
		if nearFaceBoundary(faces[i], p, tol) {
			return true
		}
	}
	return false
}

// pointSegmentDistance returns the distance from q to the segment ab (the projection clamped to the
// segment).
func pointSegmentDistance(q, a, b math.Point3) float64 {
	ab := a.VectorTo(b)
	lenSq := float64(ab.LengthSquared())
	if lenSq == 0 {
		return float64(a.DistanceTo(q))
	}
	t := float64(a.VectorTo(q).Dot(ab)) / lenSq
	t = stdmath.Max(0, stdmath.Min(1, t))
	return float64(a.TranslateBy(ab.Scale(t)).DistanceTo(q))
}
