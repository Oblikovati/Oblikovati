// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Two-parallel-cylinder host trihedral corner (R4 curved-corner-patch campaign, simple/O9 P7): where
// three equal-radius fillets meet over TWO cylinder hosts sharing a PARALLEL axis (two bosses on the
// same axis line, or a boss + a coaxial bore) and one plane PERPENDICULAR to that shared axis (a cap),
// the corner blend is again an analytic geom.Sphere of radius r — the OCCT corner KPart, exactly like
// the M5/SP2/CN3 single-curved-host corners. What makes this tractable in closed form rather than the
// general skew-cylinder quartic: because the cap plane's normal is PARALLEL to the shared cylinder axis,
// the plane tangency alone fixes the ball centre's AXIAL coordinate, and a cylinder's distance-to-axis is
// independent of axial height — so the two cylinder-tangency conditions reduce to the classic 2D
// circle∩circle intersection in that fixed-height plane (reusing intersectCoplanarCircles, the exact same
// closed form fillet_miter_curved_cylarm.go's parallel-axis MITER ARM already uses for its own tangent-
// ball centre). A crossing/skew axis pair (O4) fails the parallel-axis gate here and keeps the historical
// "corner face must be planar" reject — that general case is a follow-on (see package doc note).

// twoParallelCylinderHostCorner recognises the corner host set: exactly two cylinder faces and one plane
// face. Returns the two cylinder faces (order preserved) and the plane face. ok=false for any other host
// mix (1 cylinder as M5 already owns, 0, or ≥3 cylinders) — so solveBlend keeps the earlier paths /
// eventual planar reject untouched. Axis-parallel and plane-perpendicular are NOT checked here (that
// needs Resolution) — checked by solveTwoCylinderBlend, mirroring cylinderHostCorner/sphereHostCorner's
// split between "shape recognition" and "solve".
func twoParallelCylinderHostCorner(faces []*topo.Face) ([2]*topo.Face, *topo.Face, bool) {
	if len(faces) != 3 {
		return [2]*topo.Face{}, nil, false
	}
	var cylFaces [2]*topo.Face
	var planeFace *topo.Face
	nCyl, nPl := 0, 0
	for _, f := range faces {
		if _, isCyl := f.Geometry().(geom.Cylinder); isCyl && nCyl < 2 {
			cylFaces[nCyl] = f
			nCyl++
			continue
		}
		if _, isPl := f.Geometry().(geom.Plane); isPl && nPl < 1 {
			planeFace = f
			nPl++
		}
	}
	return cylFaces, planeFace, nCyl == 2 && nPl == 1
}

// solveTwoCylinderBlend solves the analytic sphere corner for a two-parallel-cylinder-host trihedral
// corner. Returns the "corner face must be planar" reject (do-no-harm) when the axes are not parallel,
// the plane is not perpendicular to the shared axis, either offset radius spindles out, the two offset
// circles miss, or the solved centre fails its consistency certificate — so a declined corner errors
// exactly as before. Mirrors solveCurvedBlend/solveSphereBlend/solveConeBlend for the two-cylinder host.
func solveTwoCylinderBlend(v *topo.Vertex, cylFaces [2]*topo.Face, planeFace *topo.Face, r float64) (*cornerBlend, error) {
	cylA := cylFaces[0].Geometry().(geom.Cylinder)
	cylB := cylFaces[1].Geometry().(geom.Cylinder)
	pl := planeFace.Geometry().(geom.Plane)
	res := twoCylCornerResolution(v, cylA, cylB, pl)
	c, ok := twoCylCornerCenter(v, cylFaces, cylA, cylB, planeFace, pl, r, res)
	if !ok || !twoCylCornerConsistent(c, cylA, cylB, planeFace, pl, r, res) {
		return nil, fmt.Errorf("fillet: corner face must be planar")
	}
	sph, err := geom.NewSphere(c, r)
	if err != nil {
		return nil, err
	}
	tan := map[uint64]math.Point3{
		cylFaces[0].ID(): cylinderWallPoint(cylA, c),
		cylFaces[1].ID(): cylinderWallPoint(cylB, c),
		planeFace.ID():   planeFootPoint(planeFace, c),
	}
	return &cornerBlend{vertex: v, center: c, sphere: sph, tan: tan}, nil
}

// twoCylAxisParallelTol is the dimensionless floor on |â₁·â₂| (and |n̂_plane·â₁|) below which the two
// axes (or the plane normal and the shared axis) are not parallel enough for the closed-form reduction —
// a scale-free angular floor, sibling of curvedCornerAxisTiny.
const twoCylAxisParallelTol = 1e-9

// twoCylCornerCenter solves the ball centre tangent to the two parallel cylinders and the perpendicular
// cap plane. ok=false when the axes are not parallel, the plane is not perpendicular to them (the general
// crossing-axis corner, e.g. O4, is out of scope here), a radial sign is unreadable, either offset radius
// spindles (R∓r collapses), or the two offset circles miss.
func twoCylCornerCenter(v *topo.Vertex, cylFaces [2]*topo.Face, cylA, cylB geom.Cylinder, planeFace *topo.Face, pl geom.Plane, r float64, res Resolution) (math.Point3, bool) {
	if stdmath.Abs(stdmath.Abs(float64(cylA.AxisDir.Dot(cylB.AxisDir)))-1) > twoCylAxisParallelTol {
		return math.Point3{}, false // not a parallel-axis pair — the crossing-axis corner is out of scope
	}
	n := outwardPlaneNormal(planeFace, pl)
	sigma := float64(cylA.AxisDir.AsVector().Dot(n))
	if stdmath.Abs(stdmath.Abs(sigma)-1) > twoCylAxisParallelTol {
		return math.Point3{}, false // plane not perpendicular to the shared axis — the general corner
	}
	rhoA, okA := twoCylOffsetRadius(cylFaces[0], cylA, v.Point(), r, res)
	rhoB, okB := twoCylOffsetRadius(cylFaces[1], cylB, v.Point(), r, res)
	if !okA || !okB {
		return math.Point3{}, false
	}
	p, q, ok := intersectCoplanarCircles(cylA.Origin, rhoA, cylB.Origin, rhoB, cylA.AxisDir.AsVector(), res)
	if !ok {
		return math.Point3{}, false
	}
	axialC := stdmath.Copysign(1, sigma) * (float64(n.Dot(pl.Origin.AsVector())) - r) // n·c = n·origin − r, n ∥ ±â
	return nearerAtAxialHeight(v.Point(), cylA.AxisDir.AsVector(), axialC, p, q), true
}

// twoCylOffsetRadius is the convex host-tangency offset radius ρ = R − ε·r for one cylinder of the pair,
// ε read from the SAME material-outward sign convention the single-cylinder M5 corner uses
// (cornerWallRadialSign, scoped to just this face so it cannot match the OTHER cylinder of the pair).
// ok=false when the face's normal is unreadable or the offset spindles below the weld band.
func twoCylOffsetRadius(f *topo.Face, cyl geom.Cylinder, vp math.Point3, r float64, res Resolution) (float64, bool) {
	eps := cornerWallRadialSign([]*topo.Face{f}, cyl, vp)
	rho := cyl.Radius - eps*r
	if rho < curvedCornerBandK*res.Weld() {
		return 0, false // spindle: the convex ball reaches this cylinder's axis, no fillet
	}
	return rho, true
}

// nearerAtAxialHeight re-heights a circle-intersection candidate p to axial coordinate axialC along
// axis a (translating along a leaves the radial/perpendicular components — the actual circle-intersection
// answer — unchanged) and returns whichever of the two candidates (p, q) lands nearer the corner vertex.
func nearerAtAxialHeight(vertex math.Point3, a math.Vector3, axialC float64, p, q math.Point3) math.Point3 {
	reheight := func(c math.Point3) math.Point3 {
		h := float64(a.Dot(c.AsVector()))
		return c.TranslateBy(a.Scale(axialC - h))
	}
	cp, cq := reheight(p), reheight(q)
	if cp.DistanceTo(vertex) <= cq.DistanceTo(vertex) {
		return cp
	}
	return cq
}

// twoCylCornerConsistent verifies the solved centre truly sits r from the plane (two-sided, per the N7
// reflected-root lesson) and R∓r from each cylinder's axis. A magnitude failure makes solveTwoCylinderBlend
// return the do-no-harm reject rather than emit a bad corner.
func twoCylCornerConsistent(c math.Point3, cylA, cylB geom.Cylinder, planeFace *topo.Face, pl geom.Plane, r float64, res Resolution) bool {
	n := outwardPlaneNormal(planeFace, pl)
	if stdmath.Abs(stdmath.Abs(float64(pl.Origin.VectorTo(c).Dot(n)))-r) > res.Weld() {
		return false // not at distance r from the plane (either side)
	}
	return cylinderAxisDistanceMatches(cylA, c, r, res) && cylinderAxisDistanceMatches(cylB, c, r, res)
}

// cylinderAxisDistanceMatches reports whether c sits at the cylinder's OWN required distance from its
// axis — recomputed from c and cyl directly (not threaded rho) so the certificate is an independent check
// of the solved centre, not a tautology. The centre must sit on ONE of the two convex offsets (R−r or
// R+r) — whichever twoCylOffsetRadius actually solved against; re-deriving eps here would duplicate that
// sign read, so accept either convex branch (both are valid rolling-ball tangencies; a wrong one fails
// the plane arm or the downstream weld instead, per curvedCornerConsistent's own two-sided-plane
// precedent).
func cylinderAxisDistanceMatches(cyl geom.Cylinder, c math.Point3, r float64, res Resolution) bool {
	a := cyl.AxisDir.AsVector()
	w := cyl.Origin.VectorTo(c)
	dist := float64(w.Sub(a.Scale(w.Dot(a))).Length())
	return stdmath.Abs(dist-(cyl.Radius-r)) < res.Weld() || stdmath.Abs(dist-(cyl.Radius+r)) < res.Weld()
}

// twoCylCornerResolution builds the model-relative weld tolerance for the corner from its own geometry
// (the vertex, both cylinder origins, and the plane origin) — ADR-0042.
func twoCylCornerResolution(v *topo.Vertex, cylA, cylB geom.Cylinder, pl geom.Plane) Resolution {
	return ResolutionForPoints([]math.Point3{v.Point(), cylA.Origin, cylB.Origin, pl.Origin})
}
