// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/internal/retopo"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Full-round K-arm planar corner (OCCT tests/blend simple/X8, tolblend_simple/A1): EVERY edge at a
// K-valent planar vertex is filleted at one constant radius, and the K host planes admit a COMMON
// tangent sphere (a pyramid apex, or the CV-series 4-face corner). OCCT's PerformMoreThreeCorner
// fills this corner with a GeomPlate approximation whose boundary arcs are exactly the K great
// circles of that common sphere (verified against DRAWEXE on A1/X8: every boundary arc lies on the
// sphere at |d−r| < 1e-9·r); we emit the EXACT sphere patch instead of an approximation of it —
// the spherical K-gon bounded by one great-circle arc per arm — per the parity ruling that the
// closed form beats the oracle's own approximant when they describe the same surface.

// solveFullRoundCorner solves the K≥4-arm full-round corner at v, or declines with the reason the
// configuration leaves the sphere underdetermined. It fires only when: every edge at v is picked,
// every pick is a constant CONVEX arm, every face at v is planar, and the least-squares common
// sphere centre (‖n_i·s − (n_i·o_i − r)‖ ≤ weld for ALL K planes) exists. The K tangent points feed
// the ordinary cornerBlend machinery: each arm registers its great-circle arc and chainArcs closes
// the spherical K-gon. Example: cb, err := solveFullRoundCorner(body, v, faces, ps, r).
func solveFullRoundCorner(body *topo.Body, v *topo.Vertex, faces []*topo.Face, ps []filletPick, r float64) (*cornerBlend, error) {
	if err := fullRoundArmsSupported(v, ps); err != nil {
		return nil, err
	}
	weld := ResolutionForBody(body).Weld()
	s, residual, err := commonTangentSphereCentre(faces, r)
	if err != nil {
		return nil, err
	}
	if residual > weld {
		return nil, fmt.Errorf("fillet: corner where %d filleted edges meet a %d-face vertex has no common tangent sphere (radius %g, plane residual %g > weld %g) — the free-form corner filling is not yet supported", len(ps), len(faces), r, residual, weld)
	}
	if !fullRoundArmSpinesConcurrent(v, ps, s, r, weld) {
		// A K>3-plane least-squares fit can pass the per-PLANE residual above while still landing off
		// SOME arm's own 2-face spine (armCornerCentre's independent gate, fillet.go:854) — that arm
		// would then silently keep its frame-derived, non-sphere centre while the registered corner
		// arc still uses the sphere, building a band whose boundary does not lie on the sphere it is
		// supposed to meet. Declining here keeps the corner honest.
		return nil, fmt.Errorf("fillet: corner where %d filleted edges meet a %d-face vertex has a common tangent sphere but it is not concurrent with every arm's own rolling-ball spine (radius %g) — the free-form corner filling is not yet supported", len(ps), len(faces), r)
	}
	if !fullRoundDihedralsEqual(faces, weld) {
		// EMPIRICAL GATE, HONESTLY LABELLED: TestEveryLoopSegmentLiesOnItsFace caught a real B-rep
		// defect on simple/X8 (a host plane's loop bounded by a corner arc 0.0135 off it, ~0.135% of
		// r) that is ABSENT on tolblend_simple/A1 — both solved through this exact function. Every
		// hypothesis that would explain and NARROWLY fix it was checked and falsified with a direct
		// measurement, not assumed: the sphere's per-plane tangency residual is fine on X8 (the
		// commonTangentSphereCentre certificate above passes); armCornerCentre's independent spine-
		// concurrence gate (fillet.go:854) also holds on every arm (fullRoundArmSpinesConcurrent
		// above); and cornerAt's shared great-circle-arc midpoint formula (perpToward(arcCen,
		// v.Point(), axis), fillet.go:813) measures EXACTLY (<1e-9) against the true spherical slerp
		// on both X8 and A1 — so the arc registered on the sphere is not the defect either. The one
		// measured structural difference between the clean case and the broken one is dihedral
		// symmetry: A1's four arms meet at an EQUAL 70.254° dihedral angle (every ordinary K=3
		// trihedral corner reduces to this too — 3 planes pairwise equal by construction of a right-
		// angle box, or trivially "equal" with only 3 terms); X8's is a general asymmetric pyramid
		// (90°/29°/29°/76.4°). Some OTHER step in the shared host-retrim pipeline this function feeds
		// (not yet isolated — the defect is downstream of solveFullRoundCorner, in code this cluster
		// does not own) evidently assumes that symmetry. Gating on it, honestly labelled as measured-
		// not-derived, is safer than shipping the defect or discarding the verified-clean A1/K=3
		// population while the true root cause is found. Revisit with the actual mechanism once found
		// (see TestEveryLoopSegmentLiesOnItsFace for the reproduction).
		return nil, fmt.Errorf("fillet: corner where %d filleted edges meet a %d-face vertex has unequal dihedral angles between adjacent arms — the full-round K-gon patch is only certified for a dihedrally-symmetric corner (radius %g) — the free-form corner filling is not yet supported", len(ps), len(faces), r)
	}
	return tangentSphereBlend(v, faces, s, r)
}

// fullRoundDihedralsEqual reports whether every pair of PLANAR-adjacent faces at the corner (every
// K-gon arm edge) meets at the SAME dihedral angle, within a weld/r-scaled angular tolerance — see
// the WHY at solveFullRoundCorner's call site.
func fullRoundDihedralsEqual(faces []*topo.Face, weld float64) bool {
	var ref float64
	haveRef := false
	angTol := stdmath.Max(1e-9, weld) // weld is a length; the angle check only needs it as a floor
	for i, fa := range faces {
		for _, fb := range faces[i+1:] {
			ang, ok := armDihedralAngle(fa, fb)
			if !ok {
				continue
			}
			if !haveRef {
				ref, haveRef = ang, true
				continue
			}
			if stdmath.Abs(ang-ref) > angTol {
				return false
			}
		}
	}
	return true
}

// armDihedralAngle returns the angle between fa's and fb's outward normals when they are the two
// faces of a shared K-gon arm edge, ok=false otherwise (not adjacent, or not planar).
func armDihedralAngle(fa, fb *topo.Face) (float64, bool) {
	pa, okA := fa.Geometry().(geom.Plane)
	pb, okB := fb.Geometry().(geom.Plane)
	if !okA || !okB || !armEdgeShared(fa, fb) {
		return 0, false
	}
	nA, nB := outwardPlaneNormal(fa, pa), outwardPlaneNormal(fb, pb)
	return stdmath.Acos(math.Clamp(nA.Dot(nB), -1, 1)), true
}

// armEdgeShared reports whether fa and fb are the two faces of some edge of either face — i.e. they
// are PLANAR-ADJACENT (share a K-gon arm), not just any two of the corner's K faces.
func armEdgeShared(fa, fb *topo.Face) bool {
	for _, l := range fa.Loops() {
		for _, u := range l.EdgeUses() {
			ef := u.Edge().Faces()
			if len(ef) == 2 && (ef[0] == fb || ef[1] == fb) {
				return true
			}
		}
	}
	return false
}

// fullRoundArmSpinesConcurrent certifies that the solved sphere centre s lies on EVERY pick's own
// 2-face rolling-ball spine, replicating armCornerCentre's exact spine-concurrence test (fillet.go)
// so a corner this dispatch accepts is GUARANTEED to have that gate adopt the sphere for every arm —
// never the silent per-arm fallback the L6-canal case relies on its own downstream rebuild to fix.
func fullRoundArmSpinesConcurrent(v *topo.Vertex, ps []filletPick, s math.Point3, r, weld float64) bool {
	for _, p := range ps {
		if !armSpineConcurrent(v, p, s, r, weld) {
			return false
		}
	}
	return true
}

// armSpineConcurrent reports whether s is within weld of pick p's own arm spine at v: the line
// through v's frame-derived rolling-ball centre (offset r along the arm's OWN 2-face bisector, the
// same offDir formula edgePlanarFaces/filletFrame use) in the direction of the edge's axis. The
// distance itself (shared with the partial-corner decline diagnostics, fillet_corner_partial.go) is
// armSpinePerpDistance; this just applies the weld cutoff.
func armSpineConcurrent(v *topo.Vertex, p filletPick, s math.Point3, r, weld float64) bool {
	return armSpinePerpDistance(v, p, s, r) <= weld
}

// fullRoundArmsSupported gates the full-round solve to the configuration the sphere expresses:
// every edge at v picked (no sharp gap the patch would have to batten across), every pick a
// constant-radius CONVEX arm (the material-side centre solve assumes one sense).
func fullRoundArmsSupported(v *topo.Vertex, ps []filletPick) error {
	if len(v.Edges()) != len(ps) {
		return fmt.Errorf("fillet: corner where %d filleted edges meet a %d-edge vertex is not a supported blend (a partial corner filling needs the free-form patch)", len(ps), len(v.Edges()))
	}
	if p := varyingPick(ps); p != nil {
		return fmt.Errorf("fillet: a variable-radius edge (radii %g→%g) cannot share a full-round %d-arm corner", p.r0, p.r1, len(ps))
	}
	for _, p := range ps {
		if ClassifyEdgeConvexity(p.edge) != EdgeConvex {
			return fmt.Errorf("fillet: full-round %d-arm corner requires all-convex arms (edge %d is not convex)", len(ps), p.edge.ID())
		}
	}
	return nil
}

// commonTangentSphereCentre solves the least-squares centre s of the sphere at distance r inside
// every plane (normal-equation solve of n_i·s = n_i·o_i − r) and returns the worst per-plane
// residual. K>3 planes overdetermine s, so the residual is the certificate that one sphere really
// is tangent to all of them (a pyramid apex) rather than approximately near them (a drafted wedge).
func commonTangentSphereCentre(faces []*topo.Face, r float64) (math.Point3, float64, error) {
	var ata [3][3]float64
	var atb [3]float64
	for _, f := range faces {
		pl, ok := f.Geometry().(geom.Plane)
		if !ok {
			return math.Point3{}, 0, fmt.Errorf("fillet: full-round corner face must be planar (got %T)", f.Geometry())
		}
		n := outwardPlaneNormal(f, pl)
		d := n.Dot(pl.Origin.AsVector()) - r
		row := [3]float64{n.X, n.Y, n.Z}
		for i := range 3 {
			for j := range 3 {
				ata[i][j] += row[i] * row[j]
			}
			atb[i] += row[i] * d
		}
	}
	x, ok := retopo.Solve3(ata, atb)
	if !ok {
		return math.Point3{}, 0, fmt.Errorf("fillet: full-round corner planes are degenerate (normal equations singular)")
	}
	s := math.P3(x[0], x[1], x[2])
	return s, worstPlaneResidual(faces, s, r), nil
}

// worstPlaneResidual is the largest |distance(s, plane_i) − r| over the corner's faces — the
// common-sphere certificate the weld tolerance gates.
func worstPlaneResidual(faces []*topo.Face, s math.Point3, r float64) float64 {
	worst := 0.0
	for _, f := range faces {
		pl := f.Geometry().(geom.Plane)
		n := outwardPlaneNormal(f, pl)
		if res := stdmath.Abs(n.Dot(s.AsVector()) - (n.Dot(pl.Origin.AsVector()) - r)); res > worst {
			worst = res
		}
	}
	return worst
}

// tangentSphereBlend packages the solved common sphere as an ordinary cornerBlend: centre, sphere,
// and one tangent point per face (s pushed r along the outward normal) — the same shape
// solvePlanarBlend emits for a trihedral corner, so every downstream consumer (armCornerCentre,
// registerBlendArc, spherePatchFace, host rails meeting at the tangent point) works unchanged.
func tangentSphereBlend(v *topo.Vertex, faces []*topo.Face, s math.Point3, r float64) (*cornerBlend, error) {
	sph, err := geom.NewSphere(s, r)
	if err != nil {
		return nil, err
	}
	tan := make(map[uint64]math.Point3, len(faces))
	for _, f := range faces {
		tan[f.ID()] = s.TranslateBy(outwardPlaneNormal(f, f.Geometry().(geom.Plane)).Scale(r))
	}
	return &cornerBlend{vertex: v, center: s, sphere: sph, tan: tan}, nil
}
