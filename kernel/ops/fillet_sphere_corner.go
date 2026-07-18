// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Sphere-host trihedral corner — the sphere-host campaign Slice SP2 (sphere-host-corner-derivation.md).
// Where three equal-radius fillets meet over a host SPHERE (centre O, radius R, material inside) and
// two planes, the corner blend is an analytic geom.Sphere of the same radius r — OCCT's corner KPart
// (BREP surface code 4), exactly like the M5 cylinder-host corner. Its centre is the rolling ball
// tangent to the two planes (r inside each) AND to the host sphere at distance ρ = R−r from O (the
// convex-external offset the torus arms use). This generalises solveBlend's dispatch: the M5 cylinder
// path (cylinderHostCorner→solveCurvedBlend) is UNTOUCHED and ordered first, the sphere recognizer
// runs after it, and the all-planar corner still reaches solvePlanarBlend — so every non-sphere corner
// is byte-identical (unreachable-by-construction). Concave bore (ρ = R+r), spindle (R ≤ r), a grazing
// tangency, and any inconsistency honest-reject with the exact "corner face must be planar" string.
//
// This file solves the corner SPHERE + its tangent points; the arm↔sphere weld/setback and the
// sphere-host retrim/tessellation are SP3's concern.

// sphereHostCorner recognises the SP2 sphere corner: exactly one host-sphere face and two planar host
// faces. Returns the sphere geometry, the sphere FACE (its material-outward normal fixes the convex
// sign), and the two plane faces. ok=false for any other host mix — all-planar (solvePlanarBlend) or a
// cylinder/≥2-curved host — so solveBlend keeps the planar path/reject. Sibling of cylinderHostCorner.
func sphereHostCorner(faces []*topo.Face) (geom.Sphere, *topo.Face, [2]*topo.Face, bool) {
	if len(faces) != 3 {
		return geom.Sphere{}, nil, [2]*topo.Face{}, false
	}
	var sph geom.Sphere
	var sphereFace *topo.Face
	var planes [2]*topo.Face
	nSph, nPl := 0, 0
	for _, f := range faces {
		if s, isSph := f.Geometry().(geom.Sphere); isSph {
			sph, sphereFace, nSph = s, f, nSph+1
			continue
		}
		if _, isPl := f.Geometry().(geom.Plane); isPl && nPl < 2 {
			planes[nPl], nPl = f, nPl+1
		}
	}
	return sph, sphereFace, planes, nSph == 1 && nPl == 2
}

// solveSphereBlend solves the analytic sphere corner for a sphere-host trihedral corner
// (sphere-host-corner-derivation.md §D5, OCCT BREP code 4). Returns the "corner face must be planar"
// reject (do-no-harm) when no equal-r ball fits (concave bore, spindle R≤r, the plane-pair line missing
// or grazing the offset sphere, or an inconsistent centre) — so a declined sphere corner errors exactly
// as before. Mirrors solveCurvedBlend for the sphere host.
func solveSphereBlend(v *topo.Vertex, faces []*topo.Face, sph geom.Sphere, sphereFace *topo.Face, planes [2]*topo.Face, r float64) (*cornerBlend, error) {
	res := sphereCornerResolution(v, sph, planes)
	c, ok := sphereHostCornerCenter(sph, sphereFace, planes, r, v, res)
	if !ok || !sphereCornerConsistent(c, sph, planes, r, res) {
		return nil, fmt.Errorf("fillet: corner face must be planar")
	}
	corner, err := geom.NewSphere(c, r)
	if err != nil {
		return nil, err
	}
	return &cornerBlend{vertex: v, center: c, sphere: corner, tan: sphereCornerTangents(faces, sph, c)}, nil
}

// sphereHostCornerCenter solves the ball centre tangent to the two planes and the host sphere. The two
// plane constraints pin c to a line (planePairLine, direction d = n̂₁×n̂₂, point p₀ r-inside both); the
// host tangency |c−O| = ρ = R−r becomes the FULL-3D quadratic qa·t²+qb·t+qc = 0 in the line parameter
// (sphereLineParam). ok=false on a concave bore / spindle (sphereCornerRho), parallel planes (no line),
// or the line clearing/grazing the offset sphere. sphereCornerRoot then reduces the reflected pair to
// the material-outward seed c0 (no cylinder arms witness a sphere-host corner — derivation §Reflected).
func sphereHostCornerCenter(sph geom.Sphere, sphereFace *topo.Face, planes [2]*topo.Face, r float64, v *topo.Vertex, res Resolution) (math.Point3, bool) {
	rho, ok := sphereCornerRho(sph, sphereFace, v, r, res)
	if !ok {
		return math.Point3{}, false
	}
	p0, d, ok := planePairLine(planes, r, v.Point())
	if !ok {
		return math.Point3{}, false
	}
	t, ok := sphereLineParam(sph, p0, d, rho, v.Point(), res)
	if !ok {
		return math.Point3{}, false
	}
	return sphereCornerRoot(v, r, res, p0.TranslateBy(d.Scale(t)))
}

// sphereCornerRho returns the convex host-tangency radius ρ = R−r, guarding the two out-of-slice
// degeneracies. The convex sign s = (v−O)·n̂_host,out (the host sphere face's material-outward normal
// at the corner vertex, Reversed-aware, sibling of SP1's sphereHostMaterialSign): s > 0 ⇒ material
// INSIDE the sphere (convex-external, ρ = R−r, this slice); s ≤ 0 ⇒ concave bore (material outside,
// ρ = R+r) — a follow-on slice, honest-reject. ok=false also on a spindle (R−r ≤ band, the ball
// engulfs the host), reusing the M5/arm spindle band (curvedCornerBandK·res.Weld(), ADR-0042).
func sphereCornerRho(sph geom.Sphere, sphereFace *topo.Face, v *topo.Vertex, r float64, res Resolution) (float64, bool) {
	n, ok := outwardFaceNormal(sphereFace, v.Point())
	if !ok {
		return 0, false
	}
	if s := float64(sph.Center.VectorTo(v.Point()).Dot(n)); s <= 0 {
		return 0, false // concave bore (material outside the sphere): ρ = R+r is out of this slice
	}
	rho := sph.Radius - r
	if rho < curvedCornerBandK*res.Weld() {
		return 0, false // spindle: the convex ball reaches the sphere centre, no fillet
	}
	return rho, true
}

// sphereLineParam returns the line parameter t so that p₀+t·d lies at distance ρ from the sphere centre
// O — the convex tangency |c−O| = R−r as the FULL-3D quadratic |u+t·d|² = ρ² (u = p₀−O). Unlike the
// cylinder's axis-perpendicular quadratic there is no axis-parallel degeneracy (qa = |d|² > 0 whenever
// planePairLine succeeded), so the only rejects are disc ≤ 0 (the line clears the offset sphere) and a
// grazing tangency where the two roots coalesce (sphereRootsSeparated). Returns the nearer-vertex root.
func sphereLineParam(sph geom.Sphere, p0 math.Point3, d math.Vector3, rho float64, vertex math.Point3, res Resolution) (float64, bool) {
	u := sph.Center.VectorTo(p0)
	qa := float64(d.Dot(d))
	qb := 2 * float64(u.Dot(d))
	qc := float64(u.Dot(u)) - rho*rho
	if !sphereRootsSeparated(qa, qb, qc, d, res) {
		return 0, false
	}
	return nearerRoot(qa, qb, qc, p0, d, vertex)
}

// sphereRootsSeparated reports whether the offset line meets the offset sphere in two well-separated
// points. disc ≤ 0 ⇒ the line clears (or exactly grazes) C_ρ — no real tangency. When disc > 0 the
// two roots' POINT-space separation is √disc/|d| (Δt·|d|, Δt = √disc/qa, qa = |d|²); below
// curvedCornerBandK·res.Weld() the near/far pick is noise and the corner is a grazing degeneracy —
// honest-reject rather than emit an ill-conditioned centre (derivation §Numerical pitfalls, ADR-0042).
func sphereRootsSeparated(qa, qb, qc float64, d math.Vector3, res Resolution) bool {
	disc := qb*qb - 4*qa*qc
	if disc <= 0 {
		return false
	}
	sep := stdmath.Sqrt(disc) / float64(d.Length())
	return sep >= curvedCornerBandK*res.Weld()
}

// sphereCornerRoot reduces the reflected-root family to the material-outward seed c0. A sphere-host
// corner bounds only Sphere∧Plane and Plane∧Plane edges — never a Plane∧Cylinder LINE arm — so the
// reflected-root station witness (cornerCylinderArms, the very machinery selectCornerRoot runs) is
// empty and the selector keeps the legacy c0, which the oracle confirms is correct in all three cases
// (sphere-host-corner-derivation.md §"Reflected-root ambiguity"). A future sphere corner with a genuine
// tangent dihedron (a picked arm running r-outside a centre-through plane) would extend the witness to
// Plane∧Plane cylinder arms — machinery exists, evidence does not — so any arm present here is out of
// scope: honest-reject rather than pick a possibly-wrong root.
func sphereCornerRoot(v *topo.Vertex, r float64, res Resolution, c0 math.Point3) (math.Point3, bool) {
	if len(cornerCylinderArms(v, r, res)) != 0 {
		return math.Point3{}, false
	}
	return c0, true
}

// sphereCornerConsistent verifies the solved centre truly sits r from both planes and ρ = R−r from the
// sphere centre (the "valid equal-r sphere" gate, sibling of curvedCornerConsistent). The plane test is
// two-sided (|dist| = r, either side) per the N7 reflected-root lesson; a magnitude failure makes
// solveSphereBlend return the do-no-harm reject rather than emit a bad corner.
func sphereCornerConsistent(c math.Point3, sph geom.Sphere, planes [2]*topo.Face, r float64, res Resolution) bool {
	for _, f := range planes {
		pl := f.Geometry().(geom.Plane)
		n := outwardPlaneNormal(f, pl)
		if stdmath.Abs(stdmath.Abs(float64(pl.Origin.VectorTo(c).Dot(n)))-r) > res.Weld() {
			return false // not at distance r from this plane (either side)
		}
	}
	return stdmath.Abs(float64(sph.Center.VectorTo(c).Length())-(sph.Radius-r)) < res.Weld()
}

// sphereCornerTangents places the ball's tangent point on each host face, keyed by face id: on a plane
// it is the perpendicular foot of the centre (planeFootPoint, valid either side); on the host sphere it
// is the centre projected radially onto the wall — O + R·(c−O)/|c−O| (sphereWallPoint, the internal
// tangency point c·R/(R−r) that the derivation's corner-rail forensics identify as the pinch vertex).
func sphereCornerTangents(faces []*topo.Face, sph geom.Sphere, c math.Point3) map[uint64]math.Point3 {
	tan := make(map[uint64]math.Point3, 3)
	for _, f := range faces {
		if _, ok := f.Geometry().(geom.Plane); ok {
			tan[f.ID()] = planeFootPoint(f, c)
			continue
		}
		tan[f.ID()] = sphereWallPoint(sph, c)
	}
	return tan
}

// sphereWallPoint projects centre c radially onto the host sphere wall (radius R about O) — the ball's
// tangent point on the sphere host. Falls back to c when c coincides with O (degenerate, unreachable
// once the spindle guard holds); sphereCornerConsistent then rejects the corner.
func sphereWallPoint(sph geom.Sphere, c math.Point3) math.Point3 {
	radial, err := math.UnitVector3FromVector(sph.Center.VectorTo(c))
	if err != nil {
		return c
	}
	return sph.Center.TranslateBy(radial.AsVector().Scale(sph.Radius))
}

// sphereCornerResolution builds the model-relative weld tolerance for the sphere corner from its own
// geometry (the vertex, the sphere centre, and the two plane origins) — ADR-0042, so the tangency
// checks scale with the model rather than a bare 1e-6. Sibling of curvedCornerResolution.
func sphereCornerResolution(v *topo.Vertex, sph geom.Sphere, planes [2]*topo.Face) Resolution {
	return ResolutionForPoints([]math.Point3{
		v.Point(), sph.Center,
		planes[0].Geometry().(geom.Plane).Origin,
		planes[1].Geometry().(geom.Plane).Origin,
	})
}
