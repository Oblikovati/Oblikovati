// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/internal/tol"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Corner-blend-weld Slice-1 Piece A — the PER-FACE-SIGN concave-aware corner-sphere solve
// (slice1-pieceA-concave-sphere-brief.md). The committed foundation places the corner sphere with a
// SINGLE wall-ε, which cannot serve a MIXED concave-arm trihedral corner: for M5 (box − r30 pocket,
// DRAWEXE sphere (45,14.49,45)) the ball is r into MATERIAL on plane0 (−r), r into the VOID on plane1
// (+r, the plane through the pocket axis — the N7-style through-axis reflected-root trap), and R−r from
// the cylinder axis (inside the pocket). The convex seed's ε cannot reach that combination: a plane
// reflection cannot flip the CYLINDER side, and the pocket's wall-ε returns R+r where the ball needs
// R−r. So this branch drops the single ε entirely and solves the ball tangent to the three faces under
// EVERY per-face sign, selecting the physical root by a geometric witness rather than a heuristic.
//
// Root selection (dodging the N7 through-axis trap): a fillet on a concave/reentrant corner ADDS
// material, so its rolling ball sits in the VOID — brep.PointInside(centre) == false. Among the ≤8
// per-face-sign candidates (each the near-vertex root of its own tangency system) exactly ONE
// void-side centre sits adjacent to the vertex; the others are inside the solid (the wrong reflected
// root that armStation rejected) or on the far intersection. This is the concave dual of
// selectCornerRoot's material-inward station witness, and needs no arm spine (the production arms are
// themselves mis-sensed on these corners — the sphere is anchored on the solid, not on an arm).

// cornerHasConcaveArm reports whether ANY filleted edge at the trihedral vertex v is reentrant
// (ClassifyEdgeConvexity == EdgeConcave). It is the do-no-harm GATE: an all-convex corner (B3, the
// bore-wall greens N1/L9, and every convex curved-weld green) returns false and keeps the untouched
// single-ε path in solveCurvedBlend, so its corner sphere is byte-identical. Only a genuinely mixed/
// concave trihedral corner (M5/L8/M8/N4/O1/H7) opens the per-face-sign branch. Classification is used
// ONLY to gate — the sign itself is derived geometrically below, so a misclassified rim (the #2005
// concern) still lands the correct sphere; a false rim merely opens a branch that reproduces the same
// centre.
func cornerHasConcaveArm(v *topo.Vertex) bool {
	for _, e := range v.Edges() {
		if _, _, ok := cylinderPlaneEdge(e); !ok {
			continue
		}
		if ClassifyEdgeConvexity(e) == EdgeConcave {
			return true
		}
	}
	return false
}

// solveConcaveCurvedBlend is solveCurvedBlend's concave/mixed branch: the per-face-sign corner sphere.
// It honest-rejects with the EXACT historical "corner face must be planar" string (do-no-harm) when no
// void-side equal-r ball fits or the solved centre is not tangent to all three faces, so a declined
// concave corner still errors as before.
func solveConcaveCurvedBlend(body *topo.Body, v *topo.Vertex, faces []*topo.Face, cyl geom.Cylinder, planes [2]*topo.Face, r float64, res tol.Resolution) (*cornerBlend, error) {
	c, ok := concaveCornerCenter(body, cyl, planes, r, v)
	if !ok || !curvedCornerTangent(c, cyl, planes, r, res) {
		return nil, fmt.Errorf("fillet: corner face must be planar")
	}
	sph, err := geom.NewSphere(c, r)
	if err != nil {
		return nil, err
	}
	return &cornerBlend{vertex: v, center: c, sphere: sph, tan: curvedCornerTangents(faces, cyl, c)}, nil
}

// concaveCornerCenter solves the corner ball under every per-face tangency sign and returns the unique
// VOID-side centre nearest the corner vertex (the material-adding fillet's rolling ball, §root
// selection). Each (s0,s1,sc) picks the plane offsets and the cylinder side (ρ = R + sc·r); the
// near-vertex root of that system is a candidate. PointInsideBody rejects the material-side reflected
// roots (the armStation-declined ones); nearest-to-V breaks any residual tie. ok=false when no
// void-side candidate exists (the corner admits no reentrant ball — do-no-harm reject).
func concaveCornerCenter(body *topo.Body, cyl geom.Cylinder, planes [2]*topo.Face, r float64, v *topo.Vertex) (math.Point3, bool) {
	best := math.Point3{}
	bestD := stdmath.Inf(1)
	found := false
	for _, s0 := range [2]float64{-1, 1} {
		for _, s1 := range [2]float64{-1, 1} {
			for _, sc := range [2]float64{-1, 1} {
				c, ok := signedCornerCandidate(cyl, planes, r, s0, s1, sc, v)
				if !ok || brep.PointInside(body, c) {
					continue // no real root, or the material-side reflected root (ball not in the void)
				}
				if d := float64(c.DistanceTo(v.Point())); d < bestD {
					best, bestD, found = c, d, true
				}
			}
		}
	}
	return best, found
}

// signedCornerCandidate is the near-vertex ball centre tangent to the two planes (offsets s0·r, s1·r)
// and the cylinder at radius ρ = R + sc·r. It reuses planePairLineSigned (the plane-pair line) and
// cylinderLineParam (the axis-distance quadratic's near-vertex root), so a candidate is exactly the
// centre the downstream weld would ride. ok=false on parallel planes, an axis-parallel line, or a line
// that clears the offset cylinder (no real tangency).
func signedCornerCandidate(cyl geom.Cylinder, planes [2]*topo.Face, r, s0, s1, sc float64, v *topo.Vertex) (math.Point3, bool) {
	p0, d, ok := planePairLineSigned(planes, r, s0, s1, v.Point())
	if !ok {
		return math.Point3{}, false
	}
	t, ok := cylinderLineParam(cyl, p0, d, cyl.Radius+sc*r, v.Point())
	if !ok {
		return math.Point3{}, false
	}
	return p0.TranslateBy(d.Scale(t)), true
}

// curvedCornerTangent is the SIGN-FREE tangency certificate for the per-face-sign centre: the ball is
// distance r from BOTH planes (either side) AND its distance to the cylinder axis is R−r OR R+r (either
// tangency), within the model weld tolerance. Unlike curvedCornerConsistent (which pins the cylinder
// side to a single ε) it is two-sided on the cylinder, because the concave branch chooses that side per
// corner — the certificate only asserts a VALID equal-r tangent ball, not which sense.
func curvedCornerTangent(c math.Point3, cyl geom.Cylinder, planes [2]*topo.Face, r float64, res tol.Resolution) bool {
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
	return stdmath.Abs(stdmath.Abs(dist-cyl.Radius)-r) < res.Weld() // |dist−R| = r: internal (R−r) or external (R+r)
}
