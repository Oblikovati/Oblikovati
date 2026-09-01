// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/internal/tol"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// An oblique elliptical-cylinder boss (STEP SURFACE_OF_LINEAR_EXTRUSION of an ellipse, elementarised
// to geom.EllipticalCylinder on import, T7) footprints on its host plane as an ELLIPSE (geom.EllipseFull,
// forensics §2) — not the circle every other setback boss cuts. These helpers give the setback tiler the
// same conic operations it has for the circle (line∩conic crossings, spine-station point, directed
// sub-arcs) but on the exact ellipse, so the fillet rails to the intact wall along its true footprint. The
// circle path (footprintConic-based, S1/S4) is untouched — solveImprint/footprintSubArc/… dispatch here
// only for a geom.EllipseFull footprint edge (setback-patch-derivation.md D1/D4).

// solveConicLevel solves cos a·p + sin a·q = k for the two angles a where a parametric conic
// P(a)=C+cos a·A+sin a·B meets a level set (a band line, or a fixed spine coordinate): with R=hypot(p,q)
// and φ=atan2(q,p) this is R·cos(a−φ)=k, so a = φ ± acos(k/R). It reports ok=false when the level lies
// outside the conic's projected span (|k|≥R) or grazes it tangentially — the ellipse analogue of
// lineCircleRoots' discriminant guard, scaled to `scale` (the footprint major radius) per ADR-0042 so
// the grazing floor tracks the model, never a bare epsilon.
func solveConicLevel(p, q, k, scale float64) (a0, a1 float64, ok bool) {
	r2 := p*p + q*q
	eps := scale * imprintGrazeEps
	if r2-k*k < eps*eps {
		return 0, 0, false
	}
	phi := stdmath.Atan2(q, p)
	delta := stdmath.Acos(k / stdmath.Sqrt(r2))
	return phi + delta, phi - delta, true
}

// ellipseAxes returns the ellipse's two semi-axis VECTORS (major = MajorRadius·MajorAxis, minor =
// MinorRadius·(Normal×MajorAxis)) — the A,B of the P(a)=C+cos a·A+sin a·B parametrisation the conic-level
// solver and the point/angle helpers share.
func ellipseAxes(e geom.EllipseFull) (major, minor math.Vector3) {
	minorDir := e.Normal.Cross(e.MajorAxis)
	return e.MajorAxis.AsVector().Scale(e.MajorRadius), minorDir.Scale(e.MinorRadius)
}

// ellipseAngleOf returns the parameter angle a (radians) of an on-ellipse point p: with d=p−C, the
// projections d·MajorAxis=MajorR·cos a and d·(Normal×MajorAxis)=MinorR·sin a give a=atan2(·,·) after
// rescaling by the radii (the same inversion geom.EllipticalCylinder.ParamAt uses).
func ellipseAngleOf(e geom.EllipseFull, p math.Point3) float64 {
	d := e.Center.VectorTo(p)
	minorDir := e.Normal.Cross(e.MajorAxis)
	return stdmath.Atan2(d.Dot(minorDir)/e.MinorRadius, d.Dot(e.MajorAxis.AsVector())/e.MajorRadius)
}

// ellipsePointAtAngle evaluates P(a)=C+MajorR·cos a·MajorAxis+MinorR·sin a·(Normal×MajorAxis).
func ellipsePointAtAngle(e geom.EllipseFull, a float64) math.Point3 {
	cos, sin := stdmath.Cos(a), stdmath.Sin(a)
	major, minor := ellipseAxes(e)
	return e.Center.TranslateBy(major.Scale(cos).Add(minor.Scale(sin)))
}

// ellipseSubArc is the directed geom.EllipticalArc from→to on e: the MINOR (shortest-sweep) arc, or its
// >180° complement when major=true (through the far side of the ellipse). It is the exact ellipse
// counterpart of footprintSubArc/footprintMajorArc's geom.Arc3d, feeding the same rail sampling. ok=false
// when the endpoints coincide (the sub-arc collapses), mirroring the near-antipodal guard on the circle.
func ellipseSubArc(e geom.EllipseFull, from, to math.Point3, major bool) (geom.Curve3, bool) {
	a0 := ellipseAngleOf(e, from)
	sweep := signedShortestAngle(ellipseAngleOf(e, to) - a0)
	if stdmath.Abs(sweep) < arcBisectorTiny {
		return nil, false // from ≈ to on the ellipse: degenerate sub-arc
	}
	if major {
		sweep -= stdmath.Copysign(2*stdmath.Pi, sweep) // the long way round, same endpoints
	}
	arc, err := geom.NewEllipticalArc(e.Center, e.Normal.AsVector(), e.MajorAxis.AsVector(),
		e.MajorRadius, e.MinorRadius, a0, sweep)
	return arc, err == nil
}

// signedShortestAngle wraps a raw angle difference into (−π, π] — the shortest signed sweep between two
// ellipse parameters (the minor-arc direction ellipseSubArc extends or complements).
func signedShortestAngle(raw float64) float64 {
	a := stdmath.Mod(raw, 2*stdmath.Pi)
	if a > stdmath.Pi {
		return a - 2*stdmath.Pi
	}
	if a <= -stdmath.Pi {
		return a + 2*stdmath.Pi
	}
	return a
}

// solveImprintEllipse is solveImprint for a geom.EllipseFull footprint: it crosses the receded fillet
// band line (rebuilt from im.nodes) with the exact ellipse (line∩ellipse via solveConicLevel in the host
// plane) and returns the two crossings plus the outboard sub-arc. Same contract as the circle path — the
// crossings are the ±x_s stations setbackStation reads off the spine.
func solveImprintEllipse(im runoutImprint, e geom.EllipseFull, res tol.Resolution) (imprintCut, bool) {
	if im.nodes[0].P.DistanceTo(im.nodes[1].P) <= res.Weld() {
		return imprintCut{}, false // nodes too close to fix a band direction from
	}
	band := bandLineFromNodes(im.nodes)
	o3 := im.back(band.origin)
	d3 := o3.VectorTo(im.back(band.origin.TranslateBy(band.dir)))
	nline := im.plane.Normal().Cross(d3) // in-plane normal to the band line
	major, minor := ellipseAxes(e)
	a0, a1, ok := solveConicLevel(major.Dot(nline), minor.Dot(nline), e.Center.VectorTo(o3).Dot(nline), e.MajorRadius)
	if !ok {
		return imprintCut{}, false
	}
	p0, p1 := ellipsePointAtAngle(e, a0), ellipsePointAtAngle(e, a1)
	return imprintCut{pMinus: p0, pPlus: p1, arc: ellipseOutboardArc(im, e, p0, p1)}, true
}

// ellipseOutboardArc picks the footprint sub-arc between p0 and p1 that bulges to the HOST (outboard)
// side — the minor arc when its angular midpoint sits host-side, else the major one — the exact ellipse
// analogue of outboardArc's exhaustive host/fillet-side test (onHostSide).
func ellipseOutboardArc(im runoutImprint, e geom.EllipseFull, p0, p1 math.Point3) geom.Curve3 {
	if minor, ok := ellipseSubArc(e, p0, p1, false); ok && onHostSide(im, minor.PointAt(0.5)) {
		return minor
	}
	major, _ := ellipseSubArc(e, p0, p1, true)
	return major
}

// ellipseStationPoint is footprintPointAtStation for a geom.EllipseFull footprint: the ellipse point
// whose spine coordinate equals absolute station s, on the edgeward side (toward the fillet band). It
// solves cos a·(A·axis)+sin a·(B·axis)=s−spineParam(center) (line∩ellipse projected on the spine) and,
// of the two roots, keeps the one further along the center→fillet-contact direction. ok=false when the
// station falls outside the footprint's spine span (|s−center-station|≥ its axial extent).
func ellipseStationPoint(e geom.EllipseFull, host *topo.Face, cyl geom.Cylinder, s float64) (math.Point3, bool) {
	plane, ok := host.Geometry().(geom.Plane)
	if !ok {
		return math.Point3{}, false
	}
	axis := cyl.AxisDir.AsVector()
	major, minor := ellipseAxes(e)
	sc := spineParam(e.Center, cyl)
	a0, a1, ok := solveConicLevel(major.Dot(axis), minor.Dot(axis), s-sc, e.MajorRadius)
	if !ok {
		return math.Point3{}, false
	}
	edgeward := e.Center.VectorTo(filletContact(cyl, plane, sc))
	return pickEdgewardPoint(e, a0, a1, edgeward), true
}

// pickEdgewardPoint returns whichever of the two ellipse points (at a0, a1) lies further along edgeward
// (toward the fillet band) — the setback corner sits on the band-facing side of the footprint, exactly as
// footprintPointAtStation's +√ branch places the circle corner.
func pickEdgewardPoint(e geom.EllipseFull, a0, a1 float64, edgeward math.Vector3) math.Point3 {
	p0, p1 := ellipsePointAtAngle(e, a0), ellipsePointAtAngle(e, a1)
	if e.Center.VectorTo(p0).Dot(edgeward) >= e.Center.VectorTo(p1).Dot(edgeward) {
		return p0
	}
	return p1
}
