// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/internal/tol"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Torus-host trihedral corner (R4 curved-corner-patch campaign, simple/E6 E8 F1 F3): where three
// equal-radius fillets meet over a host TORUS (a pre-existing donut-shaped primitive face already
// part of the base solid — NOT a fillet arm this package builds) and two planes, the corner blend
// is again an analytic geom.Sphere of radius r, exactly like the M5/SP2/CN3 single-curved-host
// corners. Unlike those (each reduces to a QUADRATIC in the plane-pair line parameter), a torus's
// distance-to-surface is not a polynomial of a point's coordinates without one squaring, so the
// host tangency is a QUARTIC — this file derives it directly from the offset torus's implicit
// equation (ONE characterized squaring, ONE extraneous-root class: geometry-math-advisor
// consultation, R4 wave). Port target: OCCT ChFi3d_Builder's corner tiers / ChFiKPart special
// cases (BREP surface code 4, the same analytic-sphere corner KPart the sibling files port) —
// /home/vmiguel/git/oblikovati-workspace/OCCT/src/ChFi3d/.
//
// Two planes pin the centre to a line C(t) = p0 + t·d exactly as the sibling files (planePairLine).
// The torus tangency — distance from C to the CORE CIRCLE (radius Rm about the torus axis through
// centre O) equals ρ = Rt∓r — expands to the offset-torus implicit polynomial evaluated along the
// line:
//
//	F(t) = (Q(t) + Rm² − ρ²)² − 4·Rm²·(Q(t) − axial(t)²) = 0
//
// with Q(t) = |C(t)−O|² (quadratic) and axial(t) = â·(C(t)−O) (linear) — the SAME u, d, â the
// sibling cylinder/cone files already build. This is the offset torus's own implicit equation (a
// torus IS the distance-Rt-from-core-circle level set; ρ replaces Rt for the r-offset ball
// tangency), so the ONE squaring it costs introduces exactly ONE extraneous-root class: solutions
// where the pre-squaring RHS G(t) = Q(t)+Rm²−ρ² was negative (radial(t) = −K(t), not a real
// distance) — filtered by keeping only real roots with G(t) ≥ 0.

// torusHostCorner recognises the corner host set: exactly one torus face and two planar faces.
// Returns the torus geometry, the torus FACE (its material-outward normal fixes the convex sign),
// and the two plane faces. ok=false for any other host mix — so solveBlend keeps the earlier
// single-curved-host paths / eventual planar reject untouched. Sibling of sphereHostCorner /
// coneHostCorner.
func torusHostCorner(faces []*topo.Face) (geom.Torus, *topo.Face, [2]*topo.Face, bool) {
	if len(faces) != 3 {
		return geom.Torus{}, nil, [2]*topo.Face{}, false
	}
	var tor geom.Torus
	var torusFace *topo.Face
	var planes [2]*topo.Face
	nTor, nPl := 0, 0
	for _, f := range faces {
		if t, isTor := f.Geometry().(geom.Torus); isTor {
			tor, torusFace, nTor = t, f, nTor+1
			continue
		}
		if _, isPl := f.Geometry().(geom.Plane); isPl && nPl < 2 {
			planes[nPl], nPl = f, nPl+1
		}
	}
	return tor, torusFace, planes, nTor == 1 && nPl == 2
}

// solveTorusBlend solves the analytic sphere corner for a torus-host trihedral corner. Returns the
// "corner face must be planar" reject (do-no-harm) when no equal-r ball fits (spindle, the
// plane-pair line missing the offset torus, an ambiguous multi-root corner the station-domain
// witness cannot resolve, or an inconsistent centre) — so a declined torus corner errors exactly
// as before. Mirrors solveSphereBlend/solveConeBlend for the torus host.
func solveTorusBlend(v *topo.Vertex, faces []*topo.Face, tor geom.Torus, torusFace *topo.Face, planes [2]*topo.Face, r float64) (*cornerBlend, error) {
	res := torusCornerResolution(v, tor, planes)
	c, sign, ok := torusHostCornerCenter(v, tor, torusFace, planes, r, res)
	if !ok || !torusCornerConsistent(c, tor, planes, r, sign, res) {
		return nil, fmt.Errorf("fillet: corner face must be planar")
	}
	sph, err := geom.NewSphere(c, r)
	if err != nil {
		return nil, err
	}
	return &cornerBlend{vertex: v, center: c, sphere: sph, tan: torusCornerTangents(faces, torusFace, tor, c)}, nil
}

// torusHostCornerCenter solves the ball centre tangent to the two planes and the host torus, for
// EITHER sense: convex boss (material inside the tube, sign=+1, ρ=Rt−r) or concave bore (material
// outside, sign=−1, ρ=Rt+r) — read from the torus face's material-outward normal exactly like the
// sibling sign reads (never from Rm vs Rt, so a horn torus, Rm=Rt, classifies identically to a ring
// torus). ok=false when the sign is unreadable, the offset ρ spindles, the plane-pair line is
// degenerate, or the quartic yields no admissible root.
func torusHostCornerCenter(v *topo.Vertex, tor geom.Torus, torusFace *topo.Face, planes [2]*topo.Face, r float64, res tol.Resolution) (math.Point3, float64, bool) {
	sign, ok := torusCornerSign(tor, torusFace, v.Point())
	if !ok || sign == 0 {
		return math.Point3{}, 0, false
	}
	rho := tor.MinorRadius - sign*r
	if rho < curvedCornerBandK*res.Weld() {
		return math.Point3{}, 0, false // spindle: the convex ball collapses the tube, no fillet
	}
	p0, d, ok := planePairLine(planes, r, v.Point())
	if !ok {
		return math.Point3{}, 0, false
	}
	c, ok := torusCornerCenterOnLine(v, tor, p0, d, rho, r, res)
	return c, sign, ok
}

// torusCornerSign is the material-outward sign the host torus's face carries at the corner vertex:
// sign = n̂·ê_out where ê_out is the outward tube-radial direction (from the core-circle point at
// V's azimuth TOWARD V) — sign>0 boss (material inside the tube, this slice's convex case), sign≤0
// bore (concave, out of scope here — the torus-host mirror of sphereCornerRho/coneCornerMaterialSign).
// Reading it from the FACE, not from Rm vs Rt, is exactly what keeps a horn torus (Rm=Rt)
// classifying identically to a ring torus.
func torusCornerSign(tor geom.Torus, torusFace *topo.Face, v math.Point3) (float64, bool) {
	n, ok := outwardFaceNormal(torusFace, v)
	if !ok {
		return 0, false
	}
	core := torusCorePoint(tor, v)
	outward, err := math.UnitVector3FromVector(core.VectorTo(v))
	if err != nil {
		return 0, false // v sits exactly on the core circle — degenerate, unreachable for a valid host face
	}
	return float64(n.Dot(outward.AsVector())), true
}

// torusCorePoint is the nearest point on the torus's CORE CIRCLE (radius Rm about the axis through
// Center, in the equatorial plane) to p — Center pushed Rm along p's radial (axis-perpendicular)
// direction. The core circle is planar, so this is p's nearest ring point regardless of p's own
// axial offset (minimizing in-plane distance is independent of the fixed axial separation).
func torusCorePoint(tor geom.Torus, p math.Point3) math.Point3 {
	a := tor.AxisDir.AsVector()
	w := tor.Center.VectorTo(p)
	radial, err := math.UnitVector3FromVector(w.Sub(a.Scale(w.Dot(a))))
	if err != nil {
		return tor.Center // p on the axis — degenerate, unreachable once torusCornerSign holds
	}
	return tor.Center.TranslateBy(radial.AsVector().Scale(tor.MajorRadius))
}

// torusCornerConsistent verifies the solved centre truly sits r from both planes (two-sided, per
// the N7 reflected-root lesson) and the required ρ = Rt∓r from the torus's core circle. A magnitude
// failure makes solveTorusBlend return the do-no-harm reject rather than emit a bad corner.
func torusCornerConsistent(c math.Point3, tor geom.Torus, planes [2]*topo.Face, r, sign float64, res tol.Resolution) bool {
	for _, f := range planes {
		pl := f.Geometry().(geom.Plane)
		n := outwardPlaneNormal(f, pl)
		if stdmath.Abs(stdmath.Abs(float64(pl.Origin.VectorTo(c).Dot(n)))-r) > res.Weld() {
			return false // not at distance r from this plane (either side)
		}
	}
	dist := float64(torusCorePoint(tor, c).DistanceTo(c))
	return stdmath.Abs(dist-(tor.MinorRadius-sign*r)) < res.Weld()
}

// torusCornerTangents places the ball's tangent point on each host face, keyed by face id: on a
// plane it is the perpendicular foot of the centre (planeFootPoint); on the host torus it is the
// TRUE surface foot — the core-circle point at c's azimuth, pushed Rt toward c (sign-agnostic:
// works whether c sits inside or outside the tube, mirroring coneTangentPoint's use of the real
// apex rather than the offset one).
func torusCornerTangents(faces []*topo.Face, torusFace *topo.Face, tor geom.Torus, c math.Point3) map[uint64]math.Point3 {
	tan := make(map[uint64]math.Point3, 3)
	for _, f := range faces {
		if _, ok := f.Geometry().(geom.Plane); ok {
			tan[f.ID()] = planeFootPoint(f, c)
			continue
		}
	}
	tan[torusFace.ID()] = torusSurfaceFoot(tor, c)
	return tan
}

// torusSurfaceFoot is the point on the ACTUAL torus surface nearest c: the core-circle point at c's
// azimuth, pushed the tube radius Rt toward c. Falls back to c when c coincides with its core point
// (degenerate, unreachable once torusCornerConsistent holds for r>0).
func torusSurfaceFoot(tor geom.Torus, c math.Point3) math.Point3 {
	core := torusCorePoint(tor, c)
	dir, err := math.UnitVector3FromVector(core.VectorTo(c))
	if err != nil {
		return c
	}
	return core.TranslateBy(dir.AsVector().Scale(tor.MinorRadius))
}

// torusCornerResolution builds the model-relative weld tolerance for the corner from its own
// geometry (the vertex, the torus centre, and the two plane origins) — ADR-0042.
func torusCornerResolution(v *topo.Vertex, tor geom.Torus, planes [2]*topo.Face) tol.Resolution {
	return tol.ForPoints([]math.Point3{
		v.Point(), tor.Center,
		planes[0].Geometry().(geom.Plane).Origin,
		planes[1].Geometry().(geom.Plane).Origin,
	})
}
