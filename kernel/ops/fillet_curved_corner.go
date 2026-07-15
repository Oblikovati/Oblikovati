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
	c, ok := curvedCornerCenter(cyl, planes, r, v.Point(), res)
	if !ok || !curvedCornerConsistent(c, cyl, planes, r, res) {
		return nil, fmt.Errorf("fillet: corner face must be planar")
	}
	sph, err := geom.NewSphere(c, r)
	if err != nil {
		return nil, err
	}
	return &cornerBlend{vertex: v, center: c, sphere: sph, tan: curvedCornerTangents(faces, cyl, c, r)}, nil
}

// curvedCornerCenter solves the ball centre tangent to the two planes and the cylinder. The two plane
// constraints n̂ᵢ·c = n̂ᵢ·oᵢ − r pin c to a line (direction d = n̂₁×n̂₂); planePairLine returns a point on
// it, then the convex tangency dist(c, axis) = R−r becomes a quadratic in the line parameter. Picks
// the root nearer the corner vertex (the ball sits in the wedge, not on its mirror side). ok=false on
// a spindle (R−r collapses), parallel planes (no line), or the line clearing the offset cylinder.
func curvedCornerCenter(cyl geom.Cylinder, planes [2]*topo.Face, r float64, vertex math.Point3, res Resolution) (math.Point3, bool) {
	rho := cyl.Radius - r
	if rho < curvedCornerBandK*res.Weld() {
		return math.Point3{}, false // spindle: the convex ball reaches the axis, no fillet
	}
	p0, d, ok := planePairLine(planes, r, vertex)
	if !ok {
		return math.Point3{}, false
	}
	t, ok := cylinderLineParam(cyl, p0, d, rho, vertex)
	if !ok {
		return math.Point3{}, false
	}
	return p0.TranslateBy(d.Scale(t)), true
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
// it is the centre pushed r along the material-outward normal (as in the all-planar corner); on the
// cylinder it is the centre projected radially onto the wall (to radius R from the axis).
func curvedCornerTangents(faces []*topo.Face, cyl geom.Cylinder, c math.Point3, r float64) map[uint64]math.Point3 {
	tan := make(map[uint64]math.Point3, 3)
	for _, f := range faces {
		if pl, ok := f.Geometry().(geom.Plane); ok {
			tan[f.ID()] = c.TranslateBy(outwardPlaneNormal(f, pl).Scale(r))
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

// curvedCornerConsistent verifies the solved centre truly sits r inside both planes and R−r from the
// cylinder axis (m5 §D5), within the model weld tolerance — the "valid equal-r sphere" gate. A failure
// (ill-conditioned solve, or the far/wrong quadratic root) makes solveCurvedBlend return the do-no-harm
// reject rather than emit a bad corner.
func curvedCornerConsistent(c math.Point3, cyl geom.Cylinder, planes [2]*topo.Face, r float64, res Resolution) bool {
	for _, f := range planes {
		pl := f.Geometry().(geom.Plane)
		n := outwardPlaneNormal(f, pl)
		if stdmath.Abs(float64(pl.Origin.VectorTo(c).Dot(n))+r) > res.Weld() {
			return false // not r inside this plane
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
