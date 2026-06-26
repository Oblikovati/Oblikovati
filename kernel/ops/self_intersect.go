// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Self-intersection detection (M07 PBI-084, Oblikovati/Oblikovati#300): pairs
// of faces of one body that pass through each other. Topologically adjacent
// faces (sharing an edge or vertex) legitimately touch along that boundary and
// are excluded; what remains crossing is real interpenetration — the classic
// outcome of a bad import or an over-folded offset.

// SelfIntersection is one interpenetrating, non-adjacent face pair, with a
// witness point on the crossing.
type SelfIntersection struct {
	FaceA, FaceB *topo.Face
	Witness      math.Point3
}

// SelfIntersections tessellates each face at q and reports every non-adjacent
// face pair whose triangles cross. The check is mesh-accurate: crossings
// thinner than the chord tolerance can escape it, which matches what every
// downstream consumer (booleans, mass properties, export) would see anyway.
//
// Example: if hits := ops.SelfIntersections(body, ops.DefaultQuality()); len(hits) > 0 { reject(body) }
func SelfIntersections(b *topo.Body, q Quality) []SelfIntersection {
	faces := b.Faces()
	tris := make([][][3]math.Point3, len(faces))
	boxes := make([]math.Box, len(faces))
	bvhs := make([]*triBVH, len(faces)) // built lazily the first time a face is the queried-against side
	for i, f := range faces {
		tris[i] = meshTriangles(TessellateFace(f, q))
		boxes[i] = f.RangeBox()
	}
	var out []SelfIntersection
	for i := range faces {
		for j := i + 1; j < len(faces); j++ {
			if !boxes[i].Intersects(boxes[j]) {
				continue
			}
			// Don't skip the whole pair when the faces merely touch (#1321): test triangle pairs and
			// discard only crossings that land ON the shared boundary (the legitimate edge/vertex
			// contact). A crossing AWAY from the shared topology is a real interpenetration.
			shared := sharedFaceBoundary(faces[i], faces[j])
			if bvhs[j] == nil {
				bvhs[j] = newTriBVH(tris[j])
			}
			if p, hit := meshCrossesOffBoundary(tris[i], bvhs[j], shared, q.tol()); hit {
				out = append(out, SelfIntersection{FaceA: faces[i], FaceB: faces[j], Witness: p})
			}
		}
	}
	return out
}

// sharedFaceBoundary returns the geometry two faces legitimately share: their common edges as
// segments and any common vertex as a degenerate (point) segment. Contact within tol of any of these
// is the faces meeting along their shared topology, not an interpenetration.
func sharedFaceBoundary(a, b *topo.Face) [][2]math.Point3 {
	edgesB := map[*topo.Edge]bool{}
	vertsB := map[*topo.Vertex]bool{}
	for _, e := range b.Edges() {
		edgesB[e] = true
		vertsB[e.StartVertex()], vertsB[e.EndVertex()] = true, true
	}
	var shared [][2]math.Point3
	seenV := map[*topo.Vertex]bool{}
	for _, e := range a.Edges() {
		if edgesB[e] {
			shared = append(shared, [2]math.Point3{e.StartVertex().Point(), e.EndVertex().Point()})
		}
		for _, v := range [2]*topo.Vertex{e.StartVertex(), e.EndVertex()} {
			if vertsB[v] && !seenV[v] {
				seenV[v] = true
				shared = append(shared, [2]math.Point3{v.Point(), v.Point()})
			}
		}
	}
	return shared
}

// meshCrossesOffBoundary tests each triangle of mesh A against only the B triangles its box overlaps
// (found through the B-side BVH, not an all-pairs scan, #1411) and returns the first crossing whose
// witness lies farther than tol from the shared boundary — the first real interpenetration. The exact
// Möller test and the boundary filter are unchanged, so detection is identical to the old scan; only
// the candidates it runs on are pruned. Crossings on the shared boundary are legitimate contact.
func meshCrossesOffBoundary(aTris [][3]math.Point3, bBVH *triBVH, shared [][2]math.Point3, tol float64) (math.Point3, bool) {
	var witness math.Point3
	found := false
	for _, t1 := range aTris {
		box := math.BoxFromPoints(t1[0], t1[1], t1[2])
		bBVH.query(box, func(j int) bool {
			p, hit := trianglesIntersect(t1, bBVH.tris[j])
			if hit && !onSharedBoundary(p, shared, tol) {
				witness, found = p, true
				return true // stop the BVH walk at the first real crossing
			}
			return false
		})
		if found {
			return witness, true
		}
	}
	return math.Point3{}, false
}

// onSharedBoundary reports whether p lies within tol of any shared boundary segment/point.
func onSharedBoundary(p math.Point3, shared [][2]math.Point3, tol float64) bool {
	for _, s := range shared {
		if pointToSegment(p, s[0], s[1]) <= tol {
			return true
		}
	}
	return false
}

func meshTriangle(m *Mesh, i int) [3]math.Point3 {
	return [3]math.Point3{
		m.Positions[m.Indices[i]], m.Positions[m.Indices[i+1]], m.Positions[m.Indices[i+2]],
	}
}

// selfIntersectEps keeps grazing contact (faces meeting within numerical noise
// of each other) from reporting as interpenetration.
const selfIntersectEps = 1e-9 // tol:calibrated — guards an UNNORMALISED plane equation (n·p−d, ~size³); relativising needs normalising by |n| first

// trianglesIntersect is the Möller interval test: T1 must straddle T2's plane
// and vice versa, and their intervals on the planes' intersection line must
// overlap. Coplanar triangles are handled by 2D overlap on their shared plane.
func trianglesIntersect(t1, t2 [3]math.Point3) (math.Point3, bool) {
	n2, d2 := triPlaneEq(t2)
	s1, straddles1 := signedDistances(t1, n2, d2)
	if !straddles1 {
		if triCoplanar(s1) {
			return coplanarOverlap(t1, t2, n2)
		}
		return math.Point3{}, false
	}
	n1, d1 := triPlaneEq(t1)
	s2, straddles2 := signedDistances(t2, n1, d1)
	if !straddles2 {
		return math.Point3{}, false
	}
	return intervalOverlap(t1, s1, t2, s2, n1.Cross(n2))
}

// triPlaneEq returns the triangle's (unnormalized) plane normal and offset.
func triPlaneEq(t [3]math.Point3) (math.Vector3, float64) {
	n := t[0].VectorTo(t[1]).Cross(t[0].VectorTo(t[2]))
	return n, float64(n.Dot(t[0].AsVector()))
}

// signedDistances returns each vertex's signed distance to the plane and
// whether the triangle genuinely straddles it (signs on both sides beyond eps).
func signedDistances(t [3]math.Point3, n math.Vector3, d float64) ([3]float64, bool) {
	var s [3]float64
	pos, neg := false, false
	for i, p := range t {
		s[i] = float64(n.Dot(p.AsVector())) - d
		if s[i] > selfIntersectEps {
			pos = true
		}
		if s[i] < -selfIntersectEps {
			neg = true
		}
	}
	return s, pos && neg
}

func triCoplanar(s [3]float64) bool {
	for _, v := range s {
		if v > selfIntersectEps || v < -selfIntersectEps {
			return false
		}
	}
	return true
}

// intervalOverlap projects both triangles' plane crossings onto the planes'
// intersection line and reports a witness point if the intervals overlap.
func intervalOverlap(t1 [3]math.Point3, s1 [3]float64, t2 [3]math.Point3, s2 [3]float64, line math.Vector3) (math.Point3, bool) {
	a0, a1, pa := crossingInterval(t1, s1, line)
	b0, b1, _ := crossingInterval(t2, s2, line)
	lo, hi := maxFloat(a0, b0), minFloat(a1, b1)
	if lo+selfIntersectEps >= hi {
		return math.Point3{}, false
	}
	return pa((lo + hi) / 2), true
}

// crossingInterval finds where the triangle's edges cross the other plane
// (s changes sign), projected on the intersection line; pa maps a line
// parameter back to a 3D witness point.
func crossingInterval(t [3]math.Point3, s [3]float64, line math.Vector3) (float64, float64, func(float64) math.Point3) {
	var pts []math.Point3
	for i := 0; i < 3; i++ {
		j := (i + 1) % 3
		if (s[i] > 0) == (s[j] > 0) || s[i] == s[j] {
			continue
		}
		f := math.Scalar(s[i] / (s[i] - s[j]))
		pts = append(pts, t[i].TranslateBy(t[i].VectorTo(t[j]).Scale(f)))
	}
	if len(pts) < 2 {
		pts = append(pts, t[0], t[0]) // degenerate grazing — empty interval
	}
	u0 := float64(line.Dot(pts[0].AsVector()))
	u1 := float64(line.Dot(pts[1].AsVector()))
	p0, p1 := pts[0], pts[1]
	if u0 > u1 {
		u0, u1, p0, p1 = u1, u0, p1, p0
	}
	at := func(u float64) math.Point3 {
		if u1 == u0 {
			return p0
		}
		f := math.Scalar((u - u0) / (u1 - u0))
		return p0.TranslateBy(p0.VectorTo(p1).Scale(f))
	}
	return u0, u1, at
}

// coplanarOverlap projects both triangles onto their shared plane's dominant
// axes and tests 2D overlap by separating axes over both triangles' edges.
func coplanarOverlap(t1, t2 [3]math.Point3, n math.Vector3) (math.Point3, bool) {
	u, v := planeAxes(n)
	a := projectTriangle(t1, u, v)
	b := projectTriangle(t2, u, v)
	if separated(a, b) || separated(b, a) {
		return math.Point3{}, false
	}
	c := triangleCenter(t1)
	return c, true
}

// planeAxes picks two in-plane axes for the dominant-normal projection.
func planeAxes(n math.Vector3) (math.Vector3, math.Vector3) {
	ref := math.V3(1, 0, 0)
	if abs(float64(n.X)) > abs(float64(n.Y)) && abs(float64(n.X)) > abs(float64(n.Z)) {
		ref = math.V3(0, 1, 0)
	}
	u := n.Cross(ref)
	return u, n.Cross(u)
}

func projectTriangle(t [3]math.Point3, u, v math.Vector3) [3][2]float64 {
	var out [3][2]float64
	for i, p := range t {
		out[i] = [2]float64{float64(u.Dot(p.AsVector())), float64(v.Dot(p.AsVector()))}
	}
	return out
}

// separated reports whether any edge normal of a separates the projections of
// a and b — disjoint intervals on the axis mean no overlap (plain SAT, winding
// independent).
func separated(a, b [3][2]float64) bool {
	for i := 0; i < 3; i++ {
		j := (i + 1) % 3
		nx, ny := a[j][1]-a[i][1], a[i][0]-a[j][0] // edge normal
		aMin, aMax := axisInterval(a, nx, ny)
		bMin, bMax := axisInterval(b, nx, ny)
		if aMax < bMin-selfIntersectEps || bMax < aMin-selfIntersectEps {
			return true
		}
	}
	return false
}

// axisInterval projects the triangle onto the (nx, ny) axis.
func axisInterval(t [3][2]float64, nx, ny float64) (float64, float64) {
	lo, hi := 1e308, -1e308
	for _, p := range t {
		v := nx*p[0] + ny*p[1]
		lo, hi = minFloat(lo, v), maxFloat(hi, v)
	}
	return lo, hi
}

func triangleCenter(t [3]math.Point3) math.Point3 {
	v := t[0].AsVector().Add(t[1].AsVector()).Add(t[2].AsVector())
	return v.Scale(math.Scalar(1.0 / 3)).AsPoint()
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
