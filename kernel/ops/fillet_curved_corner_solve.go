// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// M5 Slice A, Task 5.1 (m5-weld-setback-retrim-derivation.md §A): the pure-geometry corner
// solver for the curved-arm trihedral fillet weld. It reads the solved curved arms and the
// corner sphere at ONE trihedral vertex and produces, in closed form, every quantity the weld
// (T5.2–T5.5) needs: each arm's setback station (the spine parameter where the moving ball-centre
// equals C), the host-tangent points (the spherical-triangle vertices), and the per-arm weld-rail
// directions. It wires into nothing yet — it is the certified geometry the assembler will consume.
//
// The one fact that dissolves the problem: C = sphere.Center is the common intersection of all
// three arm spines, so each arm's station is the closed-form root of spine(t)=C. No surface–surface
// intersection is attempted (the arm and sphere are internally tangent along the whole rail, so an
// SSI marcher stalls — §Candidate methods); the station is read straight off the spine geometry.

// closureAngleTol is the absolute angular tolerance (radians) for the closure certificate's
// pure-angle asserts (a rail subtense == its geodesic). An angle carries no length, so ADR-0042's
// model-relative rule does not apply — a scale-free constant is correct here. 1e-4 rad sits far
// inside the exact quadric geometry's residual yet rejects any real traversal error (a 5°
// mis-orientation is 0.087 rad, three orders larger).
const closureAngleTol = 1e-4

// armSetback is one curved arm's handoff to the corner sphere: the arm surface, the spine
// parameter where spine(station)=C, and the two unit rail directions (C→host-tangent-point)/r
// of its two hosts — the endpoints of the great-circle arc the arm welds to the sphere along.
type armSetback struct {
	arm      geom.Surface     // ef.armSurface: a geom.Torus (axis ⊥ plane) or geom.Cylinder (axis ∥ plane)
	station  float64          // spine parameter where spine(station)=C (torus: major angle; cyl: axial)
	railDir0 math.UnitVector3 // unit (T_hostA − C)/r
	railDir1 math.UnitVector3 // unit (T_hostB − C)/r
}

// cornerWeld is the solved trihedral corner: the sphere (centre C, radius r), the per-arm setbacks,
// and the distinct host-tangent points (the spherical-triangle vertices, each shared by two arms).
type cornerWeld struct {
	center  math.Point3   // C — the common intersection of the three arm spines
	radius  float64       // r — the rolling-ball / corner-sphere radius
	arms    []armSetback  // one per curved arm meeting at the vertex
	tPoints []math.Point3 // the distinct host-tangent points (spherical-triangle vertices)
}

// solveCurvedCorner gathers the curved arms + sphere at a shared trihedral vertex and solves the
// setback stations + host-tangent points, then certifies the result with curvedClosureValid.
// It honest-rejects (ok=false) when fewer than three arms meet, an arm's station has no in-domain
// root (C not on that spine — a gap), a host is not tangent to the sphere at radius r, or the
// closure certificate fails. Example:
//
//	w, ok := solveCurvedCorner(sphere, arms, ResolutionForBody(body))
//	if !ok { /* decline the weld — do-no-harm, keep the clean unwelded error */ }
func solveCurvedCorner(sphere geom.Sphere, arms []edgeFillet, res Resolution) (cornerWeld, bool) {
	if len(arms) < 3 {
		return cornerWeld{}, false // a trihedral corner needs ≥3 arms; fewer cannot close a spherical triangle
	}
	c, r := sphere.Center, sphere.Radius
	scale := cornerRScale(sphere, arms)
	sets := make([]armSetback, 0, len(arms))
	for _, ef := range arms {
		set, ok := solveArmSetback(ef, c, r, scale, res)
		if !ok {
			return cornerWeld{}, false
		}
		sets = append(sets, set)
	}
	w := cornerWeld{center: c, radius: r, arms: sets, tPoints: distinctTangentPoints(sets, c, r, res)}
	if !curvedClosureValid(w, res) {
		return cornerWeld{}, false
	}
	return w, true
}

// solveArmSetback solves one arm's station and its two rail directions from its two host faces.
func solveArmSetback(ef edgeFillet, c math.Point3, r, scale float64, res Resolution) (armSetback, bool) {
	station, ok := armStation(ef.armSurface, c, scale, res)
	if !ok {
		return armSetback{}, false // C not on this arm's spine (gap) — the arm does not reach the corner
	}
	d0, ok0 := railDir(ef.a, c, r, res)
	d1, ok1 := railDir(ef.b, c, r, res)
	if !ok0 || !ok1 {
		return armSetback{}, false // a host is not tangent to the sphere at radius r
	}
	return armSetback{arm: ef.armSurface, station: station, railDir0: d0, railDir1: d1}, true
}

// armStation reads the setback station off the arm spine in closed form (spine(station)=C).
func armStation(surf geom.Surface, c math.Point3, scale float64, res Resolution) (float64, bool) {
	switch s := surf.(type) {
	case geom.Torus:
		return torusStation(s, c, scale, res)
	case geom.Cylinder:
		return cylinderStation(s, c, scale, res)
	default:
		return 0, false // only torus / cylinder arms carry a rolling-ball spine
	}
}

// torusStation solves the major angle φ* with (ρcosφ*, ρsinφ*) = C−centre in the torus plane.
// Rejects when C is off the spine circle by more than res.Weld·R (R = ρ + minor = host wall radius).
func torusStation(t geom.Torus, c math.Point3, scale float64, res Resolution) (float64, bool) {
	d := t.Center.VectorTo(c) // C − centre
	axis := t.AxisDir.AsVector()
	ref := t.Ref.AsVector()
	inPlane := d.Sub(axis.Scale(d.Dot(axis)))
	if stdmath.Abs(inPlane.Length()-t.MajorRadius) > res.Weld()*scale {
		return 0, false // C not on the torus spine circle (|‖C−centre‖_inPlane − ρ| too large)
	}
	return stdmath.Atan2(d.Dot(axis.Cross(ref)), d.Dot(ref)), true
}

// cylinderStation projects C onto the arm's axis line; the axial coordinate is the station.
// Rejects when C's perpendicular distance to the spine line exceeds res.Weld·R.
func cylinderStation(cyl geom.Cylinder, c math.Point3, scale float64, res Resolution) (float64, bool) {
	axis := cyl.AxisDir.AsVector()
	w := cyl.Origin.VectorTo(c) // C − origin
	station := w.Dot(axis)
	perp := w.Sub(axis.Scale(station))
	if perp.Length() > res.Weld()*scale {
		return 0, false // C not on the cylinder spine line (dist(C, axis) too large)
	}
	return station, true
}

// railDir returns the unit direction from C to the sphere's tangent point with host face h —
// the endpoint direction of the great-circle rail the arm welds to the sphere along.
func railDir(h *topo.Face, c math.Point3, r float64, res Resolution) (math.UnitVector3, bool) {
	tp, ok := hostTangentPoint(h.Geometry(), c, r, res)
	if !ok {
		return math.UnitVector3{}, false
	}
	dir, err := math.UnitVector3FromVector(c.VectorTo(tp))
	if err != nil {
		return math.UnitVector3{}, false // C coincides with the tangent point (degenerate corner)
	}
	return dir, true
}

// hostTangentPoint is the point where the corner sphere touches host surface h — the foot of the
// perpendicular from C onto h, which for the exact geometry sits at distance r from C.
func hostTangentPoint(surf geom.Surface, c math.Point3, r float64, res Resolution) (math.Point3, bool) {
	switch s := surf.(type) {
	case geom.Plane:
		return planeTangentPoint(s, c, r, res)
	case geom.Cylinder:
		return cylinderTangentPoint(s, c, r, res)
	default:
		return math.Point3{}, false // only planar / cylindrical hosts are supported in Slice A
	}
}

// planeTangentPoint is the foot of the perpendicular from C onto the plane; the sphere is tangent
// there iff the signed distance has magnitude r. The result is sign-independent (either plane sense).
func planeTangentPoint(p geom.Plane, c math.Point3, r float64, res Resolution) (math.Point3, bool) {
	n := p.Normal()
	signed := p.Origin.VectorTo(c).Dot(n) // (C − origin)·n̂
	if stdmath.Abs(stdmath.Abs(signed)-r) > res.Weld()*r {
		return math.Point3{}, false // sphere not tangent to this plane at radius r (dist=|signed|, want r)
	}
	return c.TranslateBy(n.Scale(-signed)), true
}

// cylinderTangentPoint is the radial projection of C onto the wall at radius R (cyl.Radius); the
// sphere is tangent there iff |dist(C, axis) − R| == r (internal or external tangency).
func cylinderTangentPoint(cyl geom.Cylinder, c math.Point3, r float64, res Resolution) (math.Point3, bool) {
	axis := cyl.AxisDir.AsVector()
	w := cyl.Origin.VectorTo(c)
	axisPoint := cyl.Origin.TranslateBy(axis.Scale(w.Dot(axis)))
	radial := axisPoint.VectorTo(c) // C − foot, perpendicular to the axis
	dir, err := math.UnitVector3FromVector(radial)
	if err != nil {
		return math.Point3{}, false // C on the axis (degenerate)
	}
	if stdmath.Abs(stdmath.Abs(radial.Length()-cyl.Radius)-r) > res.Weld()*cyl.Radius {
		return math.Point3{}, false // sphere not tangent to the wall at radius r (want |dist−R|=r)
	}
	return axisPoint.TranslateBy(dir.AsVector().Scale(cyl.Radius)), true
}

// cornerRScale is the corner-wide length scale R for the station-coincidence gate: the host wall
// radius, recovered from the torus arm as ρ+minor (§A.1). Falls back to the sphere radius when no
// torus arm is present (not the B3 case, but keeps the gate model-relative regardless).
func cornerRScale(sphere geom.Sphere, arms []edgeFillet) float64 {
	for _, ef := range arms {
		if t, ok := ef.armSurface.(geom.Torus); ok {
			return t.MajorRadius + t.MinorRadius // = R, the host cylinder wall radius
		}
	}
	return sphere.Radius
}

// distinctTangentPoints collects the arms' rail endpoints (C + r·railDir) and dedups them within
// res.Weld·r; a closed trihedral corner yields exactly three, each shared by two arms.
func distinctTangentPoints(arms []armSetback, c math.Point3, r float64, res Resolution) []math.Point3 {
	tol := res.Weld() * r
	pts := []math.Point3{}
	for _, a := range arms {
		for _, d := range [2]math.UnitVector3{a.railDir0, a.railDir1} {
			ep := endpointOf(c, r, d)
			if matchPoint(pts, ep, tol) < 0 {
				pts = append(pts, ep)
			}
		}
	}
	return pts
}

// endpointOf is the rail endpoint C + r·d.
func endpointOf(c math.Point3, r float64, d math.UnitVector3) math.Point3 {
	return c.TranslateBy(d.AsVector().Scale(r))
}

// matchPoint returns the index of the first point within tol of q, or −1.
func matchPoint(pts []math.Point3, q math.Point3, tol float64) int {
	for i, p := range pts {
		if p.DistanceTo(q) <= tol {
			return i
		}
	}
	return -1
}
