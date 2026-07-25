// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The CLOSED-FORM rolling-ball spine of a CLOSED rim where a PLANE meets a geom.EllipticalCylinder
// wall — the elliptic-prism rim vein (J6/J8; T5/U2 are its spilling siblings, see the gate in
// fillet_elliptic_rim_canal.go). This is NOT F4. F4's edge was a STRAIGHT RULING of the same wall, so
// its spine was a straight line and the fillet an EXACT right circular cylinder
// (fillet_ellipticalarm.go). Here the rim is CLOSED and the spine is a closed NON-ANALYTIC curve, so
// the rolling-ball envelope is a genuine canal (pipe) surface of VARIABLE section — not a torus, not a
// cylinder.
//
// Derivation. The wall is a translational sweep, so its unit normal N̂(u) = normalize(∂P/∂u × â) is
// constant along every ruling. A ball of radius r tangent to BOTH hosts has its centre
//   (i)  on the plane offset r to the ball's side:  n̂·C = n̂·O + side·r, and
//   (ii) on the wall offset r to the ball's side:   C = S(u,v) + side·σ·r·N̂(u).
// Substituting (ii) into (i) and solving the single linear unknown v gives the EXACT closed form
//   v(u) = (n̂·O + side·r − n̂·S(u,0) − side·σ·r·(n̂·N̂(u))) / (n̂·â),
// so C(u), the wall foot S(u,v(u)) and the plane foot C(u) − side·r·n̂ are all exact — no marching, no
// SSI. Because C stays in a fixed plane (C′·n̂ = 0) and stays at distance r from the wall
// (C′·N̂ = 0), the sphere family's CHARACTERISTIC circle at u lies in span{N̂(u), n̂} through C(u) —
// which contains BOTH feet. The cross-section is therefore the exact radius-r arc between them, and
// the band is the loft of those arcs (geom.LoftCanalStations), reused verbatim from the cone canal
// arm rather than re-derived.

// ellipticRimAxisTiltTol floors |n̂·â| — the cap plane must not be PARALLEL to the wall's rulings, or
// the linear solve for v(u) has no solution (the plane would never close a rim on the wall at all).
// Dimensionless (a dot of two unit vectors), so it needs no model scale.
const ellipticRimAxisTiltTol = 1e-9 // tol:angular — plane-parallel-to-rulings guard

// ellipticRimProbeFraction is the fraction of r used to step a material-side probe OFF the wall and
// ALONG it, away from the rim. It must be big enough that the winding-number point test is decisive
// (not sitting in the surface's own numerical skin) and small enough to stay on the picked wall.
const ellipticRimProbeFraction = 0.25

// ellipticRimSpine is the closed-form rolling-ball spine above, bound to one rim: the wall, the cap
// plane's material-outward frame, the radius, and the two SIGNS that select the physical branch.
type ellipticRimSpine struct {
	ec  geom.EllipticalCylinder
	nPl math.UnitVector3 // the cap plane's MATERIAL-OUTWARD unit normal (planeHostNormal — Reversed-aware)
	cPl float64          // n̂·O, the cap plane's signed offset along n̂
	r   float64
	// side is −1 for a CONVEX rim (the ball nestles INSIDE the material, the fillet removes material)
	// and +1 for a CONCAVE rim (the ball rolls in the reentrant VOID, the fillet adds material).
	side float64
	// sigma is +1 when the wall's GEOMETRIC outward normal (away from the axis) IS its material-outward
	// normal, −1 when the material is outside the wall (a bore). It is derived by probing the SOLID,
	// never from the elliptic face's Reversed flag — imported oblique extrusions carry an unreliable one
	// (the STEP EllipticalCylinder orientation defect that F4 already worked around).
	sigma float64
	den   float64 // n̂·â, the linear solve's denominator
}

// newEllipticRimSpine binds the closed-form spine to a rim, deriving the material side GEOMETRICALLY.
// ok=false — so every caller falls through to the byte-identical flat refusal — when the plane normal
// is unreadable, the plane is parallel to the wall's rulings, the wall's material side is undecidable,
// or the rim is neither cleanly convex nor cleanly concave.
func newEllipticRimSpine(body *topo.Body, e *topo.Edge, ec geom.EllipticalCylinder, pl geom.Plane, wallF *topo.Face, r float64) (ellipticRimSpine, bool) {
	nPl, ok := planeHostNormal(e, pl)
	if !ok {
		return ellipticRimSpine{}, false
	}
	den := float64(nPl.AsVector().Dot(ec.AxisDir.AsVector()))
	if stdmath.Abs(den) < ellipticRimAxisTiltTol {
		return ellipticRimSpine{}, false // cap plane ∥ the rulings — no closed rim to round
	}
	sigma, ok := ellipticWallMaterialSign(body, e, ec, wallF, r)
	if !ok {
		return ellipticRimSpine{}, false
	}
	side, ok := ellipticRimConvexitySide(body, e, ec, nPl, sigma, r)
	if !ok {
		return ellipticRimSpine{}, false
	}
	cPl := float64(math.P3(0, 0, 0).VectorTo(pl.Origin).Dot(nPl.AsVector()))
	spine := ellipticRimSpine{ec: ec, nPl: nPl, cPl: cPl, r: r, side: side, sigma: sigma, den: den}
	if side < 0 && r >= spine.minSectionCurvatureRadius() {
		return ellipticRimSpine{}, false // convex offset past the wall's evolute — no rolling ball fits
	}
	return spine, true
}

// ellipticWallMaterialSign returns +1 when the elliptic wall's GEOMETRIC outward normal is also its
// material-outward normal, −1 when the material lies outside the wall. It probes the SOLID a quarter
// radius inside the surface at a point stepped along the ruling away from the rim (so the probe is
// clear of the rim's own corner), and requires the two opposite probes to disagree — an undecidable
// pair (both in / both out: a wall thinner than the probe, or a degenerate pick) returns ok=false.
//
// This is the sound geometric replacement for ClassifyEdgeConvexity on an elliptic host: the imported
// oblique-extrusion face's Reversed flag mis-calls the dihedral (the STEP extrusion→EllipticalCylinder
// orientation defect), so an elliptic rim must never be classified from it.
func ellipticWallMaterialSign(body *topo.Body, e *topo.Edge, ec geom.EllipticalCylinder, wallF *topo.Face, r float64) (float64, bool) {
	probe, nGeo, ok := ellipticWallProbeFrame(e, ec, wallF, r)
	if !ok {
		return 0, false
	}
	step := nGeo.Scale(ellipticRimProbeFraction * r)
	in := PointInsideBody(body, probe.TranslateBy(step.Scale(-1)))
	out := PointInsideBody(body, probe.TranslateBy(step))
	if in == out {
		return 0, false // undecidable: the wall is thinner than the probe, or the pick is degenerate
	}
	if in {
		return 1, true // material inside the wall — geometric outward IS material-outward
	}
	return -1, true
}

// ellipticWallProbeFrame returns a point ON the wall stepped a quarter radius along the ruling AWAY
// from the rim (toward the wall's far seam vertex) and the wall's unit GEOMETRIC normal there. The
// step direction comes from the wall's seam edge, the one boundary that certainly runs across the
// wall. ok=false when the wall has no seam at the rim vertex or the normal degenerates.
func ellipticWallProbeFrame(e *topo.Edge, ec geom.EllipticalCylinder, wallF *topo.Face, r float64) (math.Point3, math.Vector3, bool) {
	_, farV := wallSeam(wallF, e, e.StartVertex())
	if farV == nil {
		return math.Point3{}, math.Vector3{}, false
	}
	mid := edgeMidpoint(e)
	u, vRim := ec.ParamAt(mid)
	_, vFar := ec.ParamAt(farV.Point())
	along := ellipticRimProbeFraction * r * stdmath.Copysign(1, vFar-vRim)
	n, err := math.UnitVector3FromVector(ec.NormalAt(u, vRim))
	if err != nil {
		return math.Point3{}, math.Vector3{}, false
	}
	return ec.PointAt(u, vRim+along), n.AsVector(), true
}

// ellipticRimConvexitySide returns −1 for a CONVEX rim and +1 for a CONCAVE one, decided GEOMETRICALLY
// off the solid — never off the elliptic face's Reversed flag, which imported oblique extrusions get
// wrong (ClassifyEdgeConvexity mis-calls BOTH of this vein's rims).
//
// The discriminator is the QUADRANT probe, not the bisector: in the plane across the rim the two hosts
// cut four quadrants, and the material is their INTERSECTION when the edge is convex but their UNION
// when it is concave. So the two MIXED quadrants — inside one host's half-space, outside the other's,
// i.e. ±(nB − nA) — are VOID on a convex rim and MATERIAL on a concave one. (The inward-bisector
// quadrant is material either way, which is exactly why probing it cannot tell them apart.) Both mixed
// probes must agree, or the rim is not a clean dihedral and the caller declines.
func ellipticRimConvexitySide(body *topo.Body, e *topo.Edge, ec geom.EllipticalCylinder, nPl math.UnitVector3, sigma, r float64) (float64, bool) {
	mid := edgeMidpoint(e)
	u, v := ec.ParamAt(mid)
	nW, err := math.UnitVector3FromVector(ec.NormalAt(u, v))
	if err != nil {
		return 0, false
	}
	nA, nB := nPl.AsVector(), nW.AsVector().Scale(sigma) // both MATERIAL-outward unit normals
	mixed, err := math.UnitVector3FromVector(nB.Sub(nA))
	if err != nil {
		return 0, false // the hosts are tangent at the rim — no dihedral to classify
	}
	// A SHORT step (a quarter radius): convexity is a LOCAL property of the dihedral, and a full-radius
	// probe can leave the part altogether across a small host face.
	step := mixed.AsVector().Scale(ellipticRimProbeFraction * r)
	one := PointInsideBody(body, mid.TranslateBy(step))
	other := PointInsideBody(body, mid.TranslateBy(step.Scale(-1)))
	if one != other {
		return 0, false // the two mixed quadrants disagree — not a clean dihedral, decline
	}
	if one {
		return 1, true // CONCAVE: the mixed quadrants are material, so the hosts' UNION is the solid
	}
	return -1, true // CONVEX: the mixed quadrants are void, so the hosts' INTERSECTION is the solid
}

// station returns the EXACT rolling-ball station at wall parameter u: the ball centre, its wall foot
// and its plane foot. Both feet are at distance exactly r from the centre by construction (the wall
// foot by the offset in (ii), the plane foot by the offset in (i)), which is what
// geom.LoftCanalStations asserts before lofting.
func (s ellipticRimSpine) station(u float64) (center, wallFoot, planeFoot math.Point3, ok bool) {
	nW, err := math.UnitVector3FromVector(s.ec.NormalAt(u, 0))
	if err != nil {
		return math.Point3{}, math.Point3{}, math.Point3{}, false
	}
	off := s.side * s.sigma * s.r
	p0 := s.ec.PointAt(u, 0)
	origin := math.P3(0, 0, 0)
	v := (s.cPl + s.side*s.r - float64(origin.VectorTo(p0).Dot(s.nPl.AsVector())) -
		off*float64(nW.AsVector().Dot(s.nPl.AsVector()))) / s.den
	wallFoot = s.ec.PointAt(u, v)
	center = wallFoot.TranslateBy(nW.AsVector().Scale(off))
	planeFoot = center.TranslateBy(s.nPl.AsVector().Scale(-s.side * s.r))
	return center, wallFoot, planeFoot, true
}

// tangencyError is the do-no-harm certificate at one station: how far the TRUE distance from the ball
// centre to the wall strays from r. It is > 0 only when the offset has passed the wall's evolute (the
// ball no longer touches the surface at the algebraic foot but cuts through it) — the exact failure a
// constant-radius fillet on a variable-curvature host must refuse, not loft. It reads the same generic
// point inversion the weld's foot tests use, so a station accepted here is tangent there too.
func (s ellipticRimSpine) tangencyError(center math.Point3) float64 {
	_, _, foot := geom.ClosestPointOnSurface(s.ec, center)
	return stdmath.Abs(float64(foot.DistanceTo(center)) - s.r)
}

// minSectionCurvatureRadius is the smallest radius of curvature of the wall's PERPENDICULAR section
// (an ellipse with semi-axes MajorRadius ≥ MinorRadius): b²/a, attained at the major-axis vertices. A
// CONVEX rim's ball is offset INWARD, so the inner offset — hence the fillet — exists only while r
// stays under this; newEllipticRimSpine gates on it up front so an impossible radius declines at the
// closed form instead of failing station by station. tangencyError remains the per-station certificate.
//
// TODO(bore-sigma): this bound assumes σ=+1, i.e. the material is INSIDE the wall, so "convex rim" means
// the ball is offset toward the wall's CONCAVE side and the evolute limits it. On a BORE (σ=−1) a convex
// rim's ball is offset toward the wall's CONVEX side, where the offset never degenerates and no such
// bound applies — so the gate is over-restrictive there and declines radii that are in fact buildable.
// It fails SAFE (an honest refusal, never a mis-built band), which is why it stands as-is; making it
// σ-aware needs the offset-side analysis done properly, not a sign flipped on a hunch.
func (s ellipticRimSpine) minSectionCurvatureRadius() float64 {
	return s.ec.MinorRadius * s.ec.MinorRadius / s.ec.MajorRadius
}
