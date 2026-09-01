// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	"fmt"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/internal/tol"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Curved-face miter seam (curved-miter-seam-derivation.md, families B, C & D). Two picked edges that
// SHARE a face meet at a valence-3 vertex; the third edge stays sharp. Each blends with the SAME
// rolling-ball radius r, so the two constant-r arm tubes cannot be joined by a corner sphere — they
// MUTUALLY TRIM along one seam curve, the equal-r bisector arm₁∩arm₂. This file owns the arm-pair
// RECOGNITION and top-level dispatch only:
//   - families B/C (one exact torus + one exact cylinder arm): buildCurvedMiterArms below, seam solved
//     by the torus∩cylinder sampler in fillet_miter_curved_seam.go (miterCornerBallCenter's spine
//     crossing in fillet_miter_curved_cornerball.go);
//   - family D (BOTH arms exact tori, e.g. simple/O9's second corner — two parallel-axis equal-radius
//     boss fillets sharing one cap plane): buildCurvedMiterTorusPair / solveCurvedMiterTorusPair in
//     fillet_miter_curved_torustorus.go, a closed-form sampler (no Newton/LM needed at all — see that
//     file's derivation note).
//
// sampleMiterSeam (the all-planar mirror-plane case, fillet_miter.go) is left byte-identical.

// curvedMiterArms is one miter corner's two equal-radius rolling-ball arms as the exact analytic
// surfaces whose intersection is the seam: exactly one torus (a circle-edge Cyl∧Plane arm) and one
// cylinder (a line-edge Plane∧Plane / equal-parallel Cyl∧Cyl / axis-∥ Cyl∧Plane arm). ok=false when
// the pair is anything else (sphere/torus/BSpline face, cone/BSpline outer, non-equal cyl∩cyl) — the
// honest-reject boundary the caller floors on (do-no-harm).
type curvedMiterArms struct {
	tor    geom.Torus
	cyl    geom.Cylinder
	torIdx int // index in ps of the torus arm's pick
	cylIdx int // index in ps of the cylinder arm's pick
}

// buildCurvedMiterArms builds both miter edges' exact rolling-ball arm surfaces and returns them as
// the torus∩cylinder pair the seam sampler needs — ok=false unless there is EXACTLY one torus and
// one cylinder arm (the equal-r bisector covers only that pairing; every sphere/torus/BSpline face
// and non-equal cyl∩cyl arm keeps flooring; a torus∧torus pair — family D — is caught upstream by
// buildCurvedMiterTorusPair before this is even tried). Each arm is convex-external only.
func buildCurvedMiterArms(ps []filletPick, r float64, res tol.Resolution) (curvedMiterArms, bool) {
	arms := curvedMiterArms{torIdx: -1, cylIdx: -1}
	for i, p := range ps {
		s, ok := miterEdgeArmSurface(p.edge, r, res)
		if !ok {
			return curvedMiterArms{}, false
		}
		switch a := s.(type) {
		case geom.Torus:
			arms.tor, arms.torIdx = a, i
		case geom.Cylinder:
			arms.cyl, arms.cylIdx = a, i
		}
	}
	return arms, arms.torIdx >= 0 && arms.cylIdx >= 0
}

// miterEdgeArmSurface builds one miter edge's exact rolling-ball arm surface (a geom.Torus for a
// circle-edge Cyl∧Plane rim, a geom.Cylinder for a Plane∧Plane line edge or an equal-radius
// parallel-axis Cyl∧Cyl edge). It is convexity-AWARE (ClassifyEdgeConvexity is the discriminator):
// a CONVEX edge takes the R−r arm, a CONCAVE (reentrant) edge the R+r/void-side arm
// (fillet_miter_concave.go — advances M3/M9/O2/P2/P3; NOT _arm.go, the GOARCH-suffix trap). ok=false for a tangent/smooth edge (no
// corner to round) or any unsupported host pair — the honest-reject boundary.
func miterEdgeArmSurface(e *topo.Edge, r float64, res tol.Resolution) (geom.Surface, bool) {
	conv := ClassifyEdgeConvexity(e)
	if conv != EdgeConvex && conv != EdgeConcave {
		return nil, false // a tangent/smooth edge has no corner to round
	}
	if cyl, pl, ok := cylinderPlaneEdge(e); ok {
		return cylinderPlaneMiterArm(e, cyl, pl, r, res, conv)
	}
	if a, b, nA, nB, err := edgePlanarFaces(e); err == nil {
		_, _ = a, b
		return planarMiterArmCylinder(e, nA, nB, r, conv)
	}
	return equalParallelCylMiterArm(e, r, res, conv)
}

// cylinderPlaneMiterArm builds the arm of a Cylinder∧Plane miter edge: an exact torus (axis ⊥ plane,
// circle edge) or an exact cylinder (axis ∥ plane, line edge), dispatched by convexity (conv) to the
// convex R−r or concave R+r arm builder via curvedMiterTorusArm / curvedMiterCylinderArm, using the
// plane host's material-outward normal.
func cylinderPlaneMiterArm(e *topo.Edge, cyl geom.Cylinder, pl geom.Plane, r float64, res tol.Resolution, conv EdgeConvexity) (geom.Surface, bool) {
	outwardN, ok := planeHostNormal(e, pl)
	if !ok {
		return nil, false
	}
	switch classifyCurvedArm(cyl, pl, res) {
	case armTorus:
		return curvedMiterTorusArm(e, cyl, pl, outwardN, r, res, conv)
	case armCylinder:
		return curvedMiterCylinderArm(e, cyl, pl, outwardN, r, res, conv)
	}
	return nil, false
}

// planarMiterArmCylinder builds the rolling-ball cylinder arm of a Plane∧Plane miter edge: the
// constant-r cylinder whose axis is the edge line offset by offDir·r (the same centre line the planar
// edge fillet's cyl uses), so the arm face and seam agree. The offset SIGN is convexity-aware: a CONVEX
// edge offsets INTO the material (−, along the interior bisector), a CONCAVE (reentrant) edge into the
// VOID (+) so the fillet ADDS the valley wedge (M3/M9's Plane∧Plane arms).
func planarMiterArmCylinder(e *topo.Edge, nA, nB math.Vector3, r float64, conv EdgeConvexity) (geom.Surface, bool) {
	sign := -1.0 // convex: ball centre offset into the material along the interior bisector
	if conv == EdgeConcave {
		sign = 1.0 // concave (reentrant valley): ball centre into the VOID
	}
	offDir := nA.Add(nB).Scale(math.Scalar(sign / (1 + float64(nA.Dot(nB)))))
	axis, err := math.UnitVector3FromVector(e.StartVertex().Point().VectorTo(e.EndVertex().Point()))
	if err != nil {
		return nil, false
	}
	base := e.StartVertex().Point().TranslateBy(offDir.Scale(math.Scalar(r)))
	arm, err := geom.NewCylinderWithRef(base, axis.AsVector(), nA.Negate(), r)
	return arm, err == nil
}

// miterHasCurvedContact reports whether a miter corner has a CYLINDER (non-planar) contact face — the
// shared face itself, or either edge's outer face — routing it to the curved seam path (families B/C/D).
// An all-planar miter returns false and keeps the byte-identical planar mirror-plane path.
func miterHasCurvedContact(ps []filletPick, shared *topo.Face) bool {
	if _, planar := shared.Geometry().(geom.Plane); !planar {
		return true
	}
	for _, p := range ps {
		if outer := otherFace(p.edge, shared); outer != nil {
			if _, planar := outer.Geometry().(geom.Plane); !planar {
				return true
			}
		}
	}
	return false
}

// solveCurvedMiter builds a curved-contact miter corner (families B/C/D): it constructs the two exact
// arm surfaces, the arm∩arm seam, and the corner-ball centre, and packs them into the cornerMiter's
// curved slot for curvedMiterBody to weld. It tries family D (torus∧torus, the closed-form pair) FIRST
// — buildCurvedMiterArms structurally cannot match two tori (its torIdx/cylIdx would leave cylIdx=−1),
// so trying either order is equivalent; family D first avoids that guaranteed-failing attempt. It
// floors (honest reject) when neither pairing matches or a seam cannot close — do-no-harm, never a
// partial body.
func solveCurvedMiter(v *topo.Vertex, ps []filletPick, shared *topo.Face, r float64) (*cornerMiter, error) {
	res := miterResolution(v, ps)
	if pair, ok := buildCurvedMiterTorusPair(ps, r, res); ok {
		return solveCurvedMiterTorusPair(v, shared, pair, r, res)
	}
	arms, ok := buildCurvedMiterArms(ps, r, res)
	if !ok {
		return nil, fmt.Errorf("fillet: curved miter arms unsupported at vertex %d (need one torus + one cylinder equal-r arm, or two coaxial tori; radius %g)", v.ID(), r)
	}
	torOuter := otherFace(ps[arms.torIdx].edge, shared)
	if torOuter == nil {
		return nil, fmt.Errorf("fillet: curved miter torus edge %d has no outer face opposite the shared face", ps[arms.torIdx].edge.ID())
	}
	seam, center, ok := sampleCurvedMiterSeam(arms, shared, torOuter, v, r, res)
	if !ok {
		return nil, fmt.Errorf("fillet: curved miter seam did not close at vertex %d (radius %g)", v.ID(), r)
	}
	curved := &curvedMiterCorner{arms: arms, torEdge: ps[arms.torIdx].edge, cylEdge: ps[arms.cylIdx].edge, shared: shared, center: center}
	return &cornerMiter{vertex: v, shared: shared, sBot: seam[len(seam)-1], seam: seam, curved: curved}, nil
}

// miterResolution builds the model-relative tolerance for a miter corner from its two edges' vertices
// and the corner vertex (ADR-0042) — the sampler/projector tolerances scale with the corner, never a
// bare epsilon.
func miterResolution(v *topo.Vertex, ps []filletPick) tol.Resolution {
	pts := []math.Point3{v.Point()}
	for _, p := range ps {
		pts = append(pts, p.edge.StartVertex().Point(), p.edge.EndVertex().Point())
	}
	return tol.ForPoints(pts)
}
