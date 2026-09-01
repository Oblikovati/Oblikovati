// SPDX-License-Identifier: GPL-2.0-only
package blend

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	opstol "oblikovati.org/kernel/ops/internal/tol"
	"oblikovati.org/math"
)

// tri3Provider is the 3-sided fill tier of the corner-blend engine (ADR-0051, Port Contract 1). A
// trihedral corner is bounded by 3 rails; a polynomial DEGENERATE-4 Coons patch fills it by collapsing
// ONE corner to a POLE (its v=1 edge is a single point P), so geom.FillSurface's 4-sided machinery
// applies unchanged. A single tensor patch suffices — not a Gregory split — because the two arms
// meeting at each patch corner already share the host tangent plane, so the G1 ribbons are
// twist-COMPATIBLE. The pole is a GENUINE geometric corner (a real tangent point of the blend), so its
// tangent cone there is correct, not a defect; the anti-fold scan must therefore EXCLUDE the
// degenerate v=1 pole row where |S_u×S_v|→0 by construction (tri3NoFold). Any failed rail/ribbon/fill
// ⇒ honest-reject (ADR-3). It reuses the Task-3 obstacle/coons4 rail+ribbon+certify machinery whole.
type tri3Provider struct{}

var _ railProvider = tri3Provider{}

// Name reports the provider's telemetry kind (never read by assembly; ADR-2 lineage invariance).
func (tri3Provider) Name() CornerBlendKind { return BlendKindTri3 }

// Fits claims any 3-sided loop; Build's certificate is the real admissibility gate.
func (tri3Provider) Fits(loop RailLoop) bool { return loop.Valence() == 3 }

// Build collapses one corner to a pole (choosePole), fills the degenerate-4 patch, and certifies it,
// or declines (ok=false) so a later tier / honest-reject handles it. The RailLoop-path sibling of
// coons4Provider.Build for the 3-valence case.
func (tri3Provider) Build(loop RailLoop, scale opstol.Resolution) (CornerBlendPatch, Certificate, bool) {
	if loop.Valence() != 3 {
		return CornerBlendPatch{}, Certificate{}, false
	}
	fill, rails, sides, ok := tri3Fill(loop, choosePole(loop))
	if !ok {
		return CornerBlendPatch{}, Certificate{}, false
	}
	cert := certifyTri3Patch(fill, rails, sides, loop, scale)
	patch := CornerBlendPatch{Surface: fill, Loops: railLoopToFilletLoops(loop), Kind: BlendKindTri3}
	return patch, cert, true
}

// choosePole returns the corner index (0,1,2; corner k = curveStart(loop.Sides[k].Curve)) to collapse
// to the pole: the corner whose two incident sides' Adjacent normals agree best (max dot of the two
// adjacent surface normals AT that corner). This minimizes ribbon strain / parameter crowding (Port
// Contract 1). It is a heuristic — the tier walk tries analyticSphere/Torus first anyway. If any
// incident Adjacent is nil the agreement is undefined, so fall back to the deterministic default 0.
func choosePole(loop RailLoop) int {
	best, bestDot := 0, -2.0
	for k := range 3 {
		d, ok := cornerNormalAgreement(loop, k)
		if !ok {
			return 0
		}
		if d > bestDot {
			best, bestDot = k, d
		}
	}
	return best
}

// cornerNormalAgreement returns the dot of the two Adjacent surface normals at corner k — sampled from
// the sides incident there: side k (which starts at corner k) and side (k+2)%3 (which ends at it).
// ok=false when either incident side has a nil Adjacent (agreement undefined).
func cornerNormalAgreement(loop RailLoop, k int) (float64, bool) {
	p := curveStart(loop.Sides[k].Curve)
	n0, ok0 := adjNormalAt(loop.Sides[k].Adjacent, p)
	n1, ok1 := adjNormalAt(loop.Sides[(k+2)%3].Adjacent, p)
	if !ok0 || !ok1 {
		return 0, false
	}
	return n0.Dot(n1), true
}

// adjNormalAt returns adj's unit normal at p (via adj.ParamAt inverse), or ok=false for a nil adj.
func adjNormalAt(adj geom.Surface, p math.Point3) (math.Vector3, bool) {
	if adj == nil {
		return math.Vector3{}, false
	}
	u, v := adj.ParamAt(p)
	return adj.NormalAt(u, v), true
}

// tri3Fill builds the degenerate-4 rails, the base Coons fill, the three G1 ribbons, and the matched,
// boundary-pinned FillSurface. It returns the refined rails and assembled sides so certify measures the
// exact same geometry (no recomputation). ok=false on any failure (honest-reject, ADR-3).
func tri3Fill(loop RailLoop, pole int) (geom.BSplineSurface, [4]geom.BSplineCurve, [4]geom.FillSide, bool) {
	var noRails [4]geom.BSplineCurve
	c0, c1, d0, d1, s0, dA, dB, ok := tri3Rails(loop, pole)
	if !ok {
		return geom.BSplineSurface{}, noRails, [4]geom.FillSide{}, false
	}
	base, err := geom.CoonsFill(c0, c1, d0, d1)
	if err != nil {
		return geom.BSplineSurface{}, noRails, [4]geom.FillSide{}, false
	}
	return assembleTri3(loop, [4]geom.BSplineCurve{c0, c1, d0, d1}, base, s0, dA, dB)
}

// tri3Rails builds the four degenerate-4 boundary rails from the 3-sided loop with corner `pole`
// collapsed. Corners A,B,C = curveStart of the 3 sides (corner[k]); sides chain s0=A→B, s1=B→C,
// s2=C→A. For pole index p (P=corner[p]):
//
//	rail | fill edge | source side  | oriented  | note
//	c0   | VMinEdge  | s[(p+1)%3]   | x→y       | the base (the side NOT touching the apex)
//	d0   | UMinEdge  | s[p]         | x→apex    | leg (s[p] starts at apex; pinnedRail reverses)
//	d1   | UMaxEdge  | s[(p+2)%3]   | y→apex    | leg (s[(p+2)%3] ends at apex)
//	c1   | VMaxEdge  | —            | apex/pole | degenerate: every control point = apex
//
// apex=corner[p], x=corner[(p+1)%3], y=corner[(p+2)%3]. The sides touching the apex are the two whose
// endpoints include it; the third is the base. Returns the three real Sides (s0,dA,dB) so tri3Sides
// can pull Cont/Adjacent.
func tri3Rails(loop RailLoop, pole int) (c0, c1, d0, d1 geom.BSplineCurve, s0, dA, dB Side, ok bool) {
	corner := func(k int) math.Point3 { return curveStart(loop.Sides[k].Curve) }
	apex, x, y := corner(pole), corner((pole+1)%3), corner((pole+2)%3)
	s0, dA, dB = loop.Sides[(pole+1)%3], loop.Sides[pole], loop.Sides[(pole+2)%3]
	tol := opstol.ForPoints([]math.Point3{x, y, apex}).Weld()
	rc0, ok0 := sideRail(s0, x, y, tol)    // base VMin, x→y
	rd0, ok1 := sideRail(dA, x, apex, tol) // leg UMin, x→apex
	rd1, ok2 := sideRail(dB, y, apex, tol) // leg UMax, y→apex
	if !ok0 || !ok1 || !ok2 {
		return c0, c1, d0, d1, s0, dA, dB, false
	}
	c0, c1, d0, d1, ok = tri3Refine(rc0, rd0, rd1, x, y, apex)
	return c0, c1, d0, d1, s0, dA, dB, ok
}

// tri3Refine makes the two legs (d0/d1) knot-compatible, builds the degenerate pole c1 to match c0's
// degree+knots, refines all four for G1 (the SAME u-knots into c0 & c1 keeps c1 all-P and degenerate;
// v-knots into d0 & d1), then REBUILDS c1 against the refined c0 so c0/c1 stay knot-compatible for
// CoonsFill, and pins every corner. Ordering matters: c1 must be rebuilt AFTER refineForG1 grew c0.
func tri3Refine(c0, d0, d1 geom.BSplineCurve, x, y, apex math.Point3) (rc0, c1, rd0, rd1 geom.BSplineCurve, ok bool) {
	d0, d1, ok = makeRailPair(d0, d1)
	if !ok {
		return rc0, c1, rd0, rd1, false
	}
	c1, ok = degeneratePole(c0, apex)
	if !ok {
		return rc0, c1, rd0, rd1, false
	}
	c0, c1, d0, d1, ok = refineForG1(c0, c1, d0, d1)
	if !ok {
		return rc0, c1, rd0, rd1, false
	}
	if c1, ok = degeneratePole(c0, apex); !ok {
		return rc0, c1, rd0, rd1, false
	}
	pinEnds(&c0, x, y)
	pinEnds(&d0, x, apex)
	pinEnds(&d1, y, apex)
	return c0, c1, d0, d1, true
}

// degeneratePole builds the collapsed VMaxEdge rail: a B-spline matching c0's degree+knots with EVERY
// control point = P (unit weights). It is a real geom.BSplineCurve that evaluates to P everywhere, so
// CoonsFill accepts it — its two endpoints (which its consistentCorners check reads) are both exactly
// the apex, and it shares c0's u-direction degree+knots. Knots are copied so the result never aliases c0.
func degeneratePole(c0 geom.BSplineCurve, apex math.Point3) (geom.BSplineCurve, bool) {
	n := len(c0.Ctrl)
	ctrl, w := make([]math.Point3, n), make([]float64, n)
	for i := range ctrl {
		ctrl[i], w[i] = apex, 1.0
	}
	c1, err := geom.NewBSplineCurve(c0.Degree, ctrl, w, append([]float64(nil), c0.Knots...))
	return c1, err == nil
}

// assembleTri3 turns the refined rails into a matched, boundary-pinned FillSurface. The pole side (c1)
// gets NO ribbon (always G0); the three real sides get an outward cross-tangent ribbon (ribbonSide).
// Split from tri3Fill to keep both bodies within the function-length budget.
func assembleTri3(loop RailLoop, rails [4]geom.BSplineCurve, base geom.BSplineSurface, s0, dA, dB Side) (geom.BSplineSurface, [4]geom.BSplineCurve, [4]geom.FillSide, bool) {
	sides, ok := tri3Sides(rails, base, s0, dA, dB, loopRibLen(loop))
	if !ok {
		return geom.BSplineSurface{}, rails, [4]geom.FillSide{}, false
	}
	fill, err := geom.FillSurface(rails[0], rails[1], rails[2], rails[3], sides)
	if err != nil {
		return geom.BSplineSurface{}, rails, [4]geom.FillSide{}, false
	}
	fill, err = pinFillBoundary(fill, rails[0], rails[1], rails[2], rails[3])
	return fill, rails, sides, err == nil
}

// tri3Sides builds the four FillSides for the degenerate-4 patch. The three real sides get an
// adjacentRibbon whose reference is the OUTWARD cross-derivative (the negated plain-Coons inward
// cross-derivative) — load-bearing for NoFold: MatchSurface glues the ribbon on the OPPOSITE side of
// the seam, so an outward ribbon lands the fill's cross-derivative back INSIDE the patch (the Task-3
// fold-fix). The pole side (c1 / VMaxEdge) carries NO ribbon and is always G0 (a single point).
func tri3Sides(rails [4]geom.BSplineCurve, base geom.BSplineSurface, s0, dA, dB Side, ribLen float64) ([4]geom.FillSide, bool) {
	fs0, ok0 := ribbonSide(rails[0], s0, inwardCrossV(base, false).Scale(-1), ribLen) // c0 VMin (base)
	fs2, ok2 := ribbonSide(rails[2], dA, inwardCrossU(base, false).Scale(-1), ribLen) // d0 UMin (leg)
	fs3, ok3 := ribbonSide(rails[3], dB, inwardCrossU(base, true).Scale(-1), ribLen)  // d1 UMax (leg)
	if !ok0 || !ok2 || !ok3 {
		return [4]geom.FillSide{}, false
	}
	return [4]geom.FillSide{fs0, {Order: 0}, fs2, fs3}, true // c1 (pole) = G0, no ribbon
}

// poleExcl is the parametric window excluded BELOW the degenerate v=1 pole row in the anti-fold scan.
// At the pole the whole VMax edge collapses to the single point P, so |S_u×S_v|→0 there BY
// CONSTRUCTION — scanning to v=1 would false-flag a fold at a genuine (correct) geometric corner. 0.1
// caps the scan just short of the collapse while still guarding the whole real interior.
const poleExcl = 0.1

// tri3NoFold sweeps the interior u-columns (excluding the two corner columns via obstacleCornerExcl,
// like obstacleNoFold) but caps the v-range BELOW the pole row (vMax = v1 − poleExcl·(v1−v0)); the
// degenerate v=1 row is excluded because its Jacobian vanishes by construction. Any interior column
// that folds ⇒ false. It shares obstacleNoFold's u-sweep via noFoldOverColumns; the ONLY change is
// vMax instead of v1 (F3 de-dup).
func tri3NoFold(fill geom.BSplineSurface, scale opstol.Resolution) bool {
	v0, v1 := fill.VDomain()
	vMax := v1 - poleExcl*(v1-v0)
	return noFoldOverColumns(fill, v0, vMax, scale)
}

// certifyTri3Patch proves the degenerate-4 patch (ADR-3), reusing the obstacle certify generics:
// Closed from the loop; WeldsArms structural (the 3 real sides are spanned, the pole is the shared
// apex); NoFold from the pole-EXCLUDING sweep; MaxDev the G0 residual over the 3 REAL rails (the
// degenerate pole rail[1] is skipped); MaxAngleDev the G1 crease over only the G1 real sides.
func certifyTri3Patch(fill geom.BSplineSurface, rails [4]geom.BSplineCurve, sides [4]geom.FillSide, loop RailLoop, scale opstol.Resolution) Certificate {
	return Certificate{
		Closed:      loop.Closed(scale.Weld()),
		WeldsArms:   true,
		NoFold:      tri3NoFold(fill, scale),
		MaxDev:      tri3MaxDev(fill, rails),
		MaxAngleDev: tri3MaxAngleDev(fill, sides),
	}
}

// tri3MaxDev is the max G0 positional residual from the three REAL rails to the fill's matching edge
// (c0→VMin, d0→UMin, d1→UMax). The degenerate pole rail (rails[1]/VMax) is a single point and is
// skipped — measuring a rail deviation there is meaningless. ~0 after pinFillBoundary.
func tri3MaxDev(fill geom.BSplineSurface, rails [4]geom.BSplineCurve) float64 {
	m := railDev(fill, rails[0], edgeVMin)
	m = stdmath.Max(m, railDev(fill, rails[2], edgeUMin))
	return stdmath.Max(m, railDev(fill, rails[3], edgeUMax))
}

// tri3MaxAngleDev is the max G1 crease angle across ONLY the G1 real sides (Order>0). The pole (side 1)
// and any G0 side are skipped — measuring continuity at the collapsed apex would falsely reject a
// correct patch (the same rim rule as the obstacle/coons4 certify).
func tri3MaxAngleDev(fill geom.BSplineSurface, sides [4]geom.FillSide) float64 {
	edges := [4]fillEdge{edgeVMin, edgeVMax, edgeUMin, edgeUMax}
	m := 0.0
	for _, i := range []int{0, 2, 3} {
		if sides[i].Order > 0 {
			m = stdmath.Max(m, seamCrease(fill, sides[i].Adjacent, edges[i]))
		}
	}
	return m
}
