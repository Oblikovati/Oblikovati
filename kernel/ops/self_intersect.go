// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/geom"
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
// face pair whose triangles cross. The check is mesh-accurate: a crossing no
// deeper than the two faces' own faceting error is not reported, because the
// mesh carries no evidence that the true surfaces cross at all — which matches
// what every downstream consumer (booleans, mass properties, export) sees anyway.
// Two planar faces are meshed exactly and so are held to a crossing of zero.
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
			// Two meshes may each stray from their true surface, so they can appear to cross by up
			// to the SUM of their deviations with no real interpenetration behind it — which is what
			// made every tangent blend (fillet/sphere/cone weld) report a false self-intersection
			// (#2077). Planar faces contribute nothing, so plane-on-plane stays exact.
			allow := faceMeshDeviation(faces[i], q) + faceMeshDeviation(faces[j], q)
			if p, hit := meshCrossesOffBoundary(tris[i], bvhs[j], shared, q.tol(), allow); hit {
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
func meshCrossesOffBoundary(aTris [][3]math.Point3, bBVH *triBVH, shared [][2]math.Point3, tol, allow float64) (math.Point3, bool) {
	var witness math.Point3
	found := false
	for _, t1 := range aTris {
		box := math.BoxFromPoints(t1[0], t1[1], t1[2])
		bBVH.query(box, func(j int) bool {
			p, kind := triangleCrossing(t1, bBVH.tris[j])
			if kind == crossNone || onSharedBoundary(p, shared, tol) {
				return false
			}
			if kind == crossStraddle && crossingThickness(t1, bBVH.tris[j]) <= allow {
				return false // shallower than the two meshes' own faceting error — no evidence
			}
			witness, found = p, true
			return true // stop the BVH walk at the first real crossing
		})
		if found {
			return witness, true
		}
	}
	return math.Point3{}, false
}

// crossingThickness is how deep the shallower of the two triangles pokes through the other's
// plane — the honest size of a straddling crossing, and what the faceting allowance is compared
// against (#2077).
func crossingThickness(a, b [3]math.Point3) float64 {
	return stdmath.Min(pokeDepth(a, b), pokeDepth(b, a))
}

// pokeDepth is how far t reaches past the plane of other, on the side it reaches less far.
func pokeDepth(t, other [3]math.Point3) float64 {
	n, d := triPlaneEq(other)
	scale := float64(n.Length())
	if scale == 0 {
		return 0 // a degenerate triangle has no plane to poke through
	}
	hi, lo := stdmath.Inf(-1), stdmath.Inf(1)
	for _, p := range t {
		s := (float64(n.Dot(p.AsVector())) - d) / scale
		hi, lo = stdmath.Max(hi, s), stdmath.Min(lo, s)
	}
	return stdmath.Min(hi, -lo)
}

// faceMeshDeviation is how far a face's TESSELLATION may stray from its true surface. A planar
// face is meshed exactly, so it gets no allowance at all; a curved face's chords sit up to the
// chord tolerance away from the surface.
func faceMeshDeviation(f *topo.Face, q Quality) float64 {
	if _, planar := f.Geometry().(geom.Plane); planar {
		return 0
	}
	return q.tol()
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

// selfIntersectEps keeps grazing contact from separating two coplanar triangles that in fact
// overlap. It now guards ONLY the 2D separating-axis test in coplanarOverlap, whose coordinates
// come from unit plane axes and so are already true lengths (#2075 normalised the rest).
const selfIntersectEps = 1e-9 // tol:numeric — a true length on unit plane axes, not a scaled residual

// crossRatio is how deep, and how far along, two triangles must cross before the crossing counts
// as interpenetration rather than contact. It multiplies the smaller triangle's own size, so it
// carries no model scale (ADR-0042) — see Oblikovati#2075.
const crossRatio = 1e-9 // tol:numeric — dimensionless; scaled by the local triangle size at use

// trianglesIntersect is the Möller interval test: T1 must straddle T2's plane
// and vice versa, and their intervals on the planes' intersection line must
// overlap. Coplanar triangles are handled by 2D overlap on their shared plane.
//
// Every comparison is made in TRUE length units, against a tolerance scaled by the smaller
// triangle. Until Oblikovati#2075 the plane distances were left unnormalised (n·p−d, inflated by
// |n| ≈ 2·area) and the interval was projected on the unnormalised n1×n2, so a fixed 1e-9 meant a
// different thing for every pair of triangles — and for small ones it meant nothing at all. Two
// faces that merely TOUCH along a line then read as interpenetrating, because the overlap interval
// is a single point whose inflated length still cleared the threshold. That is what made a mitered
// sheet-metal corner report a self-intersection where the gap cut abuts the wall it cuts.
func trianglesIntersect(t1, t2 [3]math.Point3) (math.Point3, bool) {
	p, kind := triangleCrossing(t1, t2)
	return p, kind != crossNone
}

// crossKind says WHICH branch of the Möller test produced a hit. The two need different evidence:
// a straddling crossing is judged by how DEEP it is, a coplanar one by how much AREA it shares —
// a coplanar pair has no depth at all, so a depth gate would silently erase every coplanar
// overlap (#2077).
type crossKind int

const (
	crossNone crossKind = iota
	crossStraddle
	crossCoplanar
)

// triangleCrossing is trianglesIntersect with the branch reported. See trianglesIntersect.
func triangleCrossing(t1, t2 [3]math.Point3) (math.Point3, crossKind) {
	eps := crossRatio * stdmath.Min(triScale(t1), triScale(t2))
	n2, d2 := triPlaneEq(t2)
	s1, straddles1 := signedDistances(t1, n2, d2, eps)
	if !straddles1 {
		return coplanarCrossing(t1, t2, n2, s1, eps)
	}
	n1, d1 := triPlaneEq(t1)
	s2, straddles2 := signedDistances(t2, n1, d1, eps)
	if !straddles2 {
		return math.Point3{}, crossNone
	}
	// Cross the UNIT normals, not the raw ones. |n| is ~2·area, so n1×n2 shrinks as the fourth
	// power of the model and underflows the zero-length guard on small parts — which would drop a
	// real crossing rather than report it. The unit cross product has magnitude sin θ, which
	// depends only on the angle between the planes.
	line, err := unitCross(n1, n2)
	if err != nil {
		return math.Point3{}, crossNone // planes too near parallel to define an intersection line
	}
	p, hit := intervalOverlap(t1, s1, t2, s2, line, eps)
	if !hit {
		return math.Point3{}, crossNone
	}
	return p, crossStraddle
}

// coplanarCrossing is the branch taken when t1 does not straddle t2's plane: either the two lie in
// that plane and overlap in area, or they do not meet at all.
func coplanarCrossing(t1, t2 [3]math.Point3, n2 math.Vector3, s1 [3]float64, eps float64) (math.Point3, crossKind) {
	if !triCoplanar(s1, eps) {
		return math.Point3{}, crossNone
	}
	p, hit := coplanarOverlap(t1, t2, n2)
	if !hit {
		return math.Point3{}, crossNone
	}
	return p, crossCoplanar
}

// parallelSinRatio is how far from parallel two planes must be before their intersection line is
// worth computing. It is |sin θ| — dimensionless — so it says nothing about the model's size.
const parallelSinRatio = 1e-12 // tol:numeric — a sine, not a length

// unitCross returns the unit direction of the two planes' intersection line. It divides by the
// normals' magnitudes explicitly rather than calling UnitVector3FromVector, whose zero-length guard
// is an ABSOLUTE 1e-9: |n| is ~2·area, so on a part measured in microns every face normal looks
// like a zero vector to that guard and a real crossing is silently dropped (#2075).
func unitCross(n1, n2 math.Vector3) (math.Vector3, error) {
	m1, m2 := float64(n1.Length()), float64(n2.Length())
	if m1 == 0 || m2 == 0 {
		return math.Vector3{}, fmt.Errorf("degenerate triangle normal: |n1|=%g |n2|=%g, want both > 0", m1, m2)
	}
	c := n1.Cross(n2)
	sin := float64(c.Length()) / (m1 * m2)
	if sin < parallelSinRatio {
		return math.Vector3{}, fmt.Errorf("planes are parallel to within |sin| = %g, want >= %g", sin, parallelSinRatio)
	}
	return c.Scale(math.Scalar(1 / (m1 * m2 * sin))), nil
}

// triScale is a triangle's characteristic length — its longest edge. It turns the dimensionless
// crossRatio into a length tolerance that follows the geometry instead of the model's units.
func triScale(t [3]math.Point3) float64 {
	longest := 0.0
	for i := 0; i < 3; i++ {
		if d := float64(t[i].DistanceTo(t[(i+1)%3])); d > longest {
			longest = d
		}
	}
	return longest
}

// triPlaneEq returns the triangle's (unnormalized) plane normal and offset.
func triPlaneEq(t [3]math.Point3) (math.Vector3, float64) {
	n := t[0].VectorTo(t[1]).Cross(t[0].VectorTo(t[2]))
	return n, float64(n.Dot(t[0].AsVector()))
}

// signedDistances returns each vertex's TRUE perpendicular distance to the plane and whether the
// triangle genuinely straddles it (signs on both sides beyond eps). Normalising by |n| is what
// makes eps a length the caller can reason about (#2075).
func signedDistances(t [3]math.Point3, n math.Vector3, d, eps float64) ([3]float64, bool) {
	var s [3]float64
	scale := float64(n.Length())
	if scale == 0 {
		return s, false // a degenerate triangle has no plane to straddle
	}
	pos, neg := false, false
	for i, p := range t {
		s[i] = (float64(n.Dot(p.AsVector())) - d) / scale
		if s[i] > eps {
			pos = true
		}
		if s[i] < -eps {
			neg = true
		}
	}
	return s, pos && neg
}

func triCoplanar(s [3]float64, eps float64) bool {
	for _, v := range s {
		if v > eps || v < -eps {
			return false
		}
	}
	return true
}

// intervalOverlap projects both triangles' plane crossings onto the planes' intersection line —
// a UNIT direction, so the interval is a true arc length — and reports a witness point only when
// the intervals share more than eps of it. Faces that touch along a line share a single POINT of
// that line, so requiring positive length is what separates contact from interpenetration (#2075).
func intervalOverlap(t1 [3]math.Point3, s1 [3]float64, t2 [3]math.Point3, s2 [3]float64, line math.Vector3, eps float64) (math.Point3, bool) {
	a0, a1, pa := crossingInterval(t1, s1, line)
	b0, b1, _ := crossingInterval(t2, s2, line)
	lo, hi := max(a0, b0), min(a1, b1)
	if lo+eps >= hi {
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

// coplanarOverlap projects both triangles onto their shared plane's dominant axes, tests 2D
// overlap by separating axes, and reports a witness INSIDE the shared region.
//
// The witness has to be truthful. Returning t1's centre instead (as this did until
// Oblikovati#2074) put the witness far from where the triangles actually met, and the caller
// filters legitimate contact by asking whether the witness lies on the two faces' shared
// boundary — a centre never does. Two coplanar faces meeting along a shared edge, which every
// sheet-metal wall makes where its end cap meets the sheet's side, were therefore reported as
// interpenetrating on every part that had one.
func coplanarOverlap(t1, t2 [3]math.Point3, n math.Vector3) (math.Point3, bool) {
	u, v := planeAxes(n)
	a := projectTriangle(t1, u, v)
	b := projectTriangle(t2, u, v)
	if separated(a, b) || separated(b, a) {
		return math.Point3{}, false
	}
	region, flat := clipToTriangle(t2, b, a)
	if len(region) < 3 || degenerateShare(flat, a, b) {
		return math.Point3{}, false
	}
	return centroidOf(region), true // centroidOf is shared with the sphere-cap mesher
}

// coplanarShareRatio is the least share of the smaller triangle two coplanar triangles must
// overlap by to count as interpenetrating. It is a RATIO of areas, so it carries no model
// scale. Contact along an edge overlaps by ~1e-16 of a triangle (rounding alone) and a real
// overlap by ~1e-2 or more, so every threshold between about 1e-12 and 1e-6 gives the same
// answer — see TestCoplanarShareThresholdHasAPlateau.
const coplanarShareRatio = 1e-9 // tol:numeric — a ratio of two areas, so it carries no model scale

// degenerateShare reports whether the clipped region is contact rather than overlap: two
// coplanar triangles meeting along an edge or at a corner share a region of no area, which is
// how every face meets its neighbour and is not an interpenetration.
func degenerateShare(region [][2]float64, a, b [3][2]float64) bool {
	smaller := stdmath.Min(triArea2D(a), triArea2D(b))
	if smaller <= 0 {
		return true // a degenerate triangle cannot overlap anything
	}
	return polygonArea2D(region)/smaller < coplanarShareRatio
}

// triArea2D is the projected triangle's area.
func triArea2D(t [3][2]float64) float64 {
	return polygonArea2D([][2]float64{t[0], t[1], t[2]})
}

// polygonArea2D is the shoelace area, unsigned so winding does not matter.
func polygonArea2D(p [][2]float64) float64 {
	sum := 0.0
	for i := range p {
		j := (i + 1) % len(p)
		sum += p[i][0]*p[j][1] - p[j][0]*p[i][1]
	}
	return stdmath.Abs(sum) / 2
}

// clipToTriangle clips triangle t (with projection tp) against clip's three edge half-planes,
// returning the shared region's corners in 3D and their projections. It is Sutherland–Hodgman
// run on the projected coordinates while interpolating the 3D points, so the corners land on
// the real surface and the projections stay available to measure the region's area.
func clipToTriangle(t [3]math.Point3, tp, clip [3][2]float64) ([]math.Point3, [][2]float64) {
	poly := []math.Point3{t[0], t[1], t[2]}
	proj := [][2]float64{tp[0], tp[1], tp[2]}
	for i := 0; i < 3 && len(poly) > 0; i++ {
		j := (i + 1) % 3
		nx, ny := clip[j][1]-clip[i][1], clip[i][0]-clip[j][0]
		inside := func(p [2]float64) float64 {
			return nx*(p[0]-clip[i][0]) + ny*(p[1]-clip[i][1])
		}
		poly, proj = clipHalfPlane(poly, proj, inside, insideSign(clip, nx, ny, i))
	}
	return poly, proj
}

// insideSign reports which side of the clip edge the triangle's own third corner is on, so the
// clip works for either winding.
func insideSign(clip [3][2]float64, nx, ny float64, i int) float64 {
	third := clip[(i+2)%3]
	if nx*(third[0]-clip[i][0])+ny*(third[1]-clip[i][1]) < 0 {
		return -1
	}
	return 1
}

// clipHalfPlane keeps the part of the polygon on the inside of one half-plane, interpolating
// both the 3D point and its projection at each crossing.
func clipHalfPlane(poly []math.Point3, proj [][2]float64, dist func([2]float64) float64,
	sign float64) ([]math.Point3, [][2]float64) {
	var outP []math.Point3
	var outQ [][2]float64
	for i := range poly {
		j := (i + 1) % len(poly)
		di, dj := sign*dist(proj[i]), sign*dist(proj[j])
		if di >= 0 {
			outP, outQ = append(outP, poly[i]), append(outQ, proj[i])
		}
		if (di >= 0) == (dj >= 0) || di == dj {
			continue
		}
		f := di / (di - dj)
		outP = append(outP, poly[i].TranslateBy(poly[i].VectorTo(poly[j]).Scale(math.Scalar(f))))
		outQ = append(outQ, [2]float64{
			proj[i][0] + f*(proj[j][0]-proj[i][0]),
			proj[i][1] + f*(proj[j][1]-proj[i][1]),
		})
	}
	return outP, outQ
}

// planeAxes picks two in-plane axes for the dominant-normal projection.
func planeAxes(n math.Vector3) (math.Vector3, math.Vector3) {
	ref := math.V3(1, 0, 0)
	if stdmath.Abs(float64(n.X)) > stdmath.Abs(float64(n.Y)) && stdmath.Abs(float64(n.X)) > stdmath.Abs(float64(n.Z)) {
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
		lo, hi = min(lo, v), max(hi, v)
	}
	return lo, hi
}
