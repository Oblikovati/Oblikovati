// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// M5 Slice A (m5-curved-arm-derivation.md §D5, m5-trihedral-spike.md): the trihedral corner where
// three equal-radius fillets meet over a CURVED (cylinder) host is an analytic geom.Sphere of the
// same radius r — precisely the OCCT corner KPart (BREP surface code 4), not a BSpline. Its centre is
// the rolling ball tangent to the two host planes (r inside each) AND to the cylinder wall (distance
// R−r from the axis, the same convex-external offset the arm surfaces use). This generalises
// solveBlend's three-plane centre solve; the all-planar path is untouched (byte-identical). The
// arm↔sphere weld/setback is a separate concern (Task 5) — this file only builds the corner sphere.

// curvedCornerBandK is k in the spindle existence guard R−r < k·res.Weld() (§Numerical pitfalls),
// the SAME length band the torus arm uses (armSpindleBand): below it the convex ball reaches the axis
// and the sphere would sit on a self-intersecting rim, so the corner is rejected rather than emitted.
const curvedCornerBandK = armSpindleBand

// curvedCornerAxisTiny is the dimensionless floor on the corner line's axis-perpendicular content
// (|d⊥|²/|d|² = sin²∠(d, axis)): below it the plane-pair line runs parallel to the cylinder axis and
// the tangency quadratic degenerates (distance to the axis is constant along the line), so the corner
// is rejected. A scale-free angular floor, not a length tolerance — sibling of arcBisectorTiny.
const curvedCornerAxisTiny = 1e-12

// cylinderHostCorner recognises the M5 Slice-A curved corner: exactly one cylinder host face and two
// planar host faces (a boss rim meeting two flats). Returns the cylinder geometry and the two plane
// faces (needed for their material-outward normals). ok=false for any other host mix — all-planar (the
// historical sphere corner) or ≥2 curved (unsupported) — so solveBlend keeps the planar path/reject.
func cylinderHostCorner(faces []*topo.Face) (geom.Cylinder, [2]*topo.Face, bool) {
	if len(faces) != 3 {
		return geom.Cylinder{}, [2]*topo.Face{}, false
	}
	var cyl geom.Cylinder
	var planes [2]*topo.Face
	nCyl, nPl := 0, 0
	for _, f := range faces {
		if c, isCyl := f.Geometry().(geom.Cylinder); isCyl {
			cyl, nCyl = c, nCyl+1
			continue
		}
		if _, isPl := f.Geometry().(geom.Plane); isPl && nPl < 2 {
			planes[nPl], nPl = f, nPl+1
		}
	}
	return cyl, planes, nCyl == 1 && nPl == 2
}

// solveCurvedBlend solves the analytic sphere corner for a cylinder-host trihedral corner
// (m5-curved-arm-derivation.md §D5, OCCT BREP code 4). Returns the "corner face must be planar" reject
// (do-no-harm) when no equal-r ball fits (spindle R≤r, the plane-pair line misses the offset cylinder,
// or the solved centre is inconsistent) — so a declined curved corner still errors exactly as before.
func solveCurvedBlend(v *topo.Vertex, faces []*topo.Face, cyl geom.Cylinder, planes [2]*topo.Face, r float64) (*cornerBlend, error) {
	res := curvedCornerResolution(v, cyl, planes)
	c, ok := curvedCornerCenter(cyl, planes, r, v, res)
	if !ok || !curvedCornerConsistent(c, cyl, planes, r, res) {
		return nil, fmt.Errorf("fillet: corner face must be planar")
	}
	sph, err := geom.NewSphere(c, r)
	if err != nil {
		return nil, err
	}
	return &cornerBlend{vertex: v, center: c, sphere: sph, tan: curvedCornerTangents(faces, cyl, c)}, nil
}

// curvedCornerCenter solves the ball centre tangent to the two planes and the cylinder, then selects
// the correct reflected root. The two plane constraints pin c to a line (direction d = n̂₁×n̂₂);
// planePairLine returns a point on it, then the convex tangency dist(c, axis) = R−r becomes a quadratic
// (cylinderLineParam) whose nearer-vertex root is the legacy seed c0. At a tangent/diametral dihedron
// (N7: the x=50 plane passes through the wall axis) c0 is the WRONG reflected root — r inside the plane
// but on the wrong side of the corner — so selectCornerRoot re-picks by station domain (see below).
// ok=false on a spindle (R−r collapses), parallel planes (no line), the line clearing the offset
// cylinder, or an ambiguous corner where the station witness admits neither/both roots (do-no-harm).
func curvedCornerCenter(cyl geom.Cylinder, planes [2]*topo.Face, r float64, v *topo.Vertex, res Resolution) (math.Point3, bool) {
	rho := cyl.Radius - r
	if rho < curvedCornerBandK*res.Weld() {
		return math.Point3{}, false // spindle: the convex ball reaches the axis, no fillet
	}
	p0, d, ok := planePairLine(planes, r, v.Point())
	if !ok {
		return math.Point3{}, false
	}
	t, ok := cylinderLineParam(cyl, p0, d, rho, v.Point())
	if !ok {
		return math.Point3{}, false
	}
	return selectCornerRoot(cyl, planes, r, v, p0.TranslateBy(d.Scale(t)), res)
}

// selectCornerRoot re-roots the corner ball at a tangent/diametral dihedron (n7-runout-rederivation.md
// §"tangent-dihedron reflected-root trap"). The ball-tangent system has a reflected pair — c0 and its
// mirror across each plane — all valid equal-r tangent balls, so curvedCornerConsistent (tangency) and
// the closure certificate accept BOTH; the tiebreak is the station domain. For each candidate it
// demands that on EVERY straight (cylinder) arm at V the ball sit on the arm's material-inward spine
// AND station on the same side of V as the far vertex (rootStationsInDomain). Reduces to c0 on a clean
// corner (B3: c0 is the unique in-domain root). When no straight arm can discriminate (none built at
// V) it keeps the legacy c0 unchanged; when a straight arm is present but neither/both roots qualify it
// honest-rejects (ok=false) rather than emit a wrong corner. A re-picked root is additionally area-
// witnessed (curvedCornerTriangleArea) so a degenerate flip is never accepted.
func selectCornerRoot(cyl geom.Cylinder, planes [2]*topo.Face, r float64, v *topo.Vertex, c0 math.Point3, res Resolution) (math.Point3, bool) {
	arms := cornerCylinderArms(v, r, res)
	if len(arms) == 0 {
		return c0, true // no straight arm to root against — keep the legacy nearer-vertex root
	}
	scale := cyl.Radius
	var chosen math.Point3
	n := 0
	for _, c := range curvedCornerRootCandidates(c0, planes) {
		if rootStationsInDomain(arms, v.Point(), c, scale, res) {
			chosen, n = c, n+1
		}
	}
	if n != 1 {
		return math.Point3{}, false // neither/both roots in-domain: ambiguous — honest-reject (do-no-harm)
	}
	if chosen.DistanceTo(c0) > res.Weld()*r && curvedCornerTriangleArea(cyl, planes, chosen) < res.Weld()*r*r {
		return math.Point3{}, false // area witness: a re-picked root that collapses the corner triangle
	}
	return chosen, true
}

// cornerArm is a built straight (cylinder) arm at the corner vertex plus its far edge terminus — the
// station-domain frame the reflected-root selector tests each candidate ball centre against.
type cornerArm struct {
	spine geom.Cylinder // the material-inward rolling-ball cylinder about the filleted ruling
	far   math.Point3   // the filleted edge's vertex away from the corner (the runout terminus)
}

// cornerCylinderArms builds the material-inward straight arm at every Plane∧Cylinder line edge at V
// (the arms whose station cleanly separates the reflected pair; torus arms are excluded because the
// corner ball's angular station on a torus is near-antipodal, not a reliable domain signal). It reuses
// the production arm builder so each spine is the exact branch the weld will use.
func cornerCylinderArms(v *topo.Vertex, r float64, res Resolution) []cornerArm {
	var arms []cornerArm
	for _, e := range v.Edges() {
		wc, pl, ok := cylinderPlaneEdge(e)
		if !ok || classifyCurvedArm(wc, pl, res) != armCylinder {
			continue
		}
		spine, ok := cylinderArmSurface(e, wc, pl, r)
		if !ok {
			continue
		}
		arms = append(arms, cornerArm{spine: spine, far: farVertexNotVid(e, v.ID())})
	}
	return arms
}

// curvedCornerRootCandidates enumerates the reflected root family: the legacy centre c0 and its mirror
// across each planar host. Mirroring across a plane flips that plane's offset sign while preserving
// tangency to the wall and the other plane, so it is a valid alternate tangent ball — exactly the
// z=5↔z=15 reflected pair the tangent dihedron admits.
func curvedCornerRootCandidates(c0 math.Point3, planes [2]*topo.Face) [3]math.Point3 {
	return [3]math.Point3{c0, reflectAcrossFace(c0, planes[0]), reflectAcrossFace(c0, planes[1])}
}

// reflectAcrossFace mirrors point c across the plane of face f (c − 2·((c−o)·n̂)·n̂).
func reflectAcrossFace(c math.Point3, f *topo.Face) math.Point3 {
	pl := f.Geometry().(geom.Plane)
	n := pl.Normal()
	signed := float64(pl.Origin.VectorTo(c).Dot(n))
	return c.TranslateBy(n.Scale(-2 * signed))
}

// rootStationsInDomain reports whether candidate ball centre c is in-domain adjacent to V on EVERY
// straight arm: c must lie on the arm's (material-inward) spine — off-spine means the reflected branch,
// r-outside that arm's plane host — AND its axial station must fall on the same side of V as the far
// vertex (the filleted extent), not on V's mirror side. A valid corner root satisfies both on all arms;
// the wrong reflected root fails one (off-spine, or station past V away from the edge).
func rootStationsInDomain(arms []cornerArm, vp math.Point3, c math.Point3, scale float64, res Resolution) bool {
	for _, a := range arms {
		if _, onSpine := cylinderStation(a.spine, c, scale, res); !onSpine {
			return false // c off the arm spine: the reflected branch (r-outside this arm's plane host)
		}
		if !stationTowardFar(a.spine, vp, a.far, c) {
			return false // station on V's mirror side, away from the far terminus — out of the arm domain
		}
	}
	return true
}

// stationTowardFar reports whether c's axial station lies on the same side of V as the far vertex —
// (station(c)−station(V)) and (station(far)−station(V)) agree in sign — i.e. c sits in the filleted
// extent adjacent to V, not on the reflected mirror side.
func stationTowardFar(spine geom.Cylinder, vp, far, c math.Point3) bool {
	axis := spine.AxisDir.AsVector()
	cst := float64(spine.Origin.VectorTo(c).Dot(axis))
	vst := float64(spine.Origin.VectorTo(vp).Dot(axis))
	fst := float64(spine.Origin.VectorTo(far).Dot(axis))
	return (cst-vst)*(fst-vst) >= 0
}

// curvedCornerTriangleArea is the area of the host-tangent triangle (the ball's three tangent feet on
// the two planes and the wall) — the area witness that rejects a re-picked root collapsing the corner.
func curvedCornerTriangleArea(cyl geom.Cylinder, planes [2]*topo.Face, c math.Point3) float64 {
	t0 := planeFootPoint(planes[0], c)
	t1 := planeFootPoint(planes[1], c)
	tw := cylinderWallPoint(cyl, c)
	return 0.5 * float64(t0.VectorTo(t1).Cross(t0.VectorTo(tw)).Length())
}

// planeFootPoint is the foot of the perpendicular from c onto face f's plane (the ball's tangent point).
func planeFootPoint(f *topo.Face, c math.Point3) math.Point3 {
	pl := f.Geometry().(geom.Plane)
	n := pl.Normal()
	signed := float64(pl.Origin.VectorTo(c).Dot(n))
	return c.TranslateBy(n.Scale(-signed))
}

// planePairLine returns a point p0 on the intersection of the two r-offset planes plus the line
// direction d = n̂₁×n̂₂. Each plane contributes n̂·c = n̂·origin − r (material-outward normal, centre r
// inside — identical to the all-planar solve). A third row d·c = d·vertex fixes p0 at the vertex's
// d-station so the downstream root pick is well-conditioned. ok=false when the planes are near-parallel.
func planePairLine(planes [2]*topo.Face, r float64, vertex math.Point3) (math.Point3, math.Vector3, bool) {
	pl0, pl1 := planes[0].Geometry().(geom.Plane), planes[1].Geometry().(geom.Plane)
	n0, n1 := outwardPlaneNormal(planes[0], pl0), outwardPlaneNormal(planes[1], pl1)
	d := n0.Cross(n1)
	a := [3][3]float64{{n0.X, n0.Y, n0.Z}, {n1.X, n1.Y, n1.Z}, {d.X, d.Y, d.Z}}
	b := [3]float64{
		n0.Dot(pl0.Origin.AsVector()) - r,
		n1.Dot(pl1.Origin.AsVector()) - r,
		d.Dot(vertex.AsVector()),
	}
	x, ok := solve3(a, b)
	if !ok {
		return math.Point3{}, math.Vector3{}, false
	}
	return math.P3(x[0], x[1], x[2]), d, true
}

// cylinderLineParam returns the line parameter t so that p0+t·d lies at distance rho from the cylinder
// axis — the convex tangency dist(c, axis) = R−r as a quadratic |u⊥+t·d⊥|² = rho² in the
// axis-perpendicular components (u = p0−A). Returns the root nearer the corner vertex. ok=false when
// the line is axis-parallel (degenerate quadratic) or the discriminant is negative (line clears C_ρ).
func cylinderLineParam(cyl geom.Cylinder, p0 math.Point3, d math.Vector3, rho float64, vertex math.Point3) (float64, bool) {
	a := cyl.AxisDir.AsVector()
	u := cyl.Origin.VectorTo(p0)
	uPerp := u.Sub(a.Scale(u.Dot(a)))
	dPerp := d.Sub(a.Scale(d.Dot(a)))
	qa := float64(dPerp.Dot(dPerp))
	if qa < curvedCornerAxisTiny*float64(d.Dot(d)) {
		return 0, false // line parallel to the axis: distance to the axis is constant, no tangency
	}
	qb := 2 * float64(uPerp.Dot(dPerp))
	qc := float64(uPerp.Dot(uPerp)) - rho*rho
	return nearerRoot(qa, qb, qc, p0, d, vertex)
}

// nearerRoot solves qa·t²+qb·t+qc = 0 and returns whichever root's point p0+t·d lies closer to the
// corner vertex (the physical ball sits in the wedge; its mirror root is the far intersection with the
// offset cylinder). ok=false when the discriminant is negative (no real tangency).
func nearerRoot(qa, qb, qc float64, p0 math.Point3, d math.Vector3, vertex math.Point3) (float64, bool) {
	disc := qb*qb - 4*qa*qc
	if disc < 0 {
		return 0, false
	}
	root := stdmath.Sqrt(disc)
	tLo, tHi := (-qb-root)/(2*qa), (-qb+root)/(2*qa)
	if p0.TranslateBy(d.Scale(tLo)).DistanceTo(vertex) <= p0.TranslateBy(d.Scale(tHi)).DistanceTo(vertex) {
		return tLo, true
	}
	return tHi, true
}

// curvedCornerTangents places the ball's tangent point on each host face, keyed by face id: on a plane
// it is the perpendicular foot of the centre (the true tangent point, valid whichever side of the plane
// the ball sits — the re-picked reflected root is r OUTSIDE one plane); on the cylinder it is the centre
// projected radially onto the wall (radius R). For a material-inward centre the foot equals the old
// centre+outward·r, so the historical planar/B3 corners are byte-identical.
func curvedCornerTangents(faces []*topo.Face, cyl geom.Cylinder, c math.Point3) map[uint64]math.Point3 {
	tan := make(map[uint64]math.Point3, 3)
	for _, f := range faces {
		if _, ok := f.Geometry().(geom.Plane); ok {
			tan[f.ID()] = planeFootPoint(f, c)
			continue
		}
		tan[f.ID()] = cylinderWallPoint(cyl, c)
	}
	return tan
}

// cylinderWallPoint projects centre c radially onto the cylinder wall (to radius R about the axis) —
// the ball's tangent point on the cylinder host. Falls back to c when c lies on the axis (degenerate);
// curvedCornerConsistent then rejects the corner.
func cylinderWallPoint(cyl geom.Cylinder, c math.Point3) math.Point3 {
	a := cyl.AxisDir.AsVector()
	w := cyl.Origin.VectorTo(c)
	foot := cyl.Origin.TranslateBy(a.Scale(w.Dot(a)))
	radial, err := math.UnitVector3FromVector(w.Sub(a.Scale(w.Dot(a))))
	if err != nil {
		return c
	}
	return foot.TranslateBy(radial.AsVector().Scale(cyl.Radius))
}

// curvedCornerConsistent verifies the solved centre truly sits r from both planes and R−r from the
// cylinder axis (m5 §D5), within the model weld tolerance — the "valid equal-r sphere" gate. The plane
// test is two-sided (|dist| = r, either side): at a tangent/diametral dihedron the correct root is r
// OUTSIDE one plane (N7's z=15 root), a valid tangent ball whose side is already fixed by the station
// witness (selectCornerRoot); a material-inward-only test would wrongly reject it. A magnitude failure
// (ill-conditioned solve, or a non-tangent centre) still makes solveCurvedBlend return the do-no-harm
// reject rather than emit a bad corner.
func curvedCornerConsistent(c math.Point3, cyl geom.Cylinder, planes [2]*topo.Face, r float64, res Resolution) bool {
	for _, f := range planes {
		pl := f.Geometry().(geom.Plane)
		n := outwardPlaneNormal(f, pl)
		if stdmath.Abs(stdmath.Abs(float64(pl.Origin.VectorTo(c).Dot(n)))-r) > res.Weld() {
			return false // not at distance r from this plane (either side)
		}
	}
	a := cyl.AxisDir.AsVector()
	w := cyl.Origin.VectorTo(c)
	dist := float64(w.Sub(a.Scale(w.Dot(a))).Length())
	return stdmath.Abs(dist-(cyl.Radius-r)) < res.Weld()
}

// curvedCornerResolution builds the model-relative weld tolerance for the corner from its own geometry
// (the vertex, the cylinder axis point, and the two plane origins) — ADR-0042, so the tangency checks
// scale with the model rather than a bare 1e-6.
func curvedCornerResolution(v *topo.Vertex, cyl geom.Cylinder, planes [2]*topo.Face) Resolution {
	return ResolutionForPoints([]math.Point3{
		v.Point(), cyl.Origin,
		planes[0].Geometry().(geom.Plane).Origin,
		planes[1].Geometry().(geom.Plane).Origin,
	})
}
