// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// --- synthetic high-aspect panel: an EXACT cylindrical strip as a BSplineSurface -----------------
//
// #2009's motivating bug needs a panel that is BOTH high-aspect (a long straight rail starved to 2
// points by discretizeEdge's sagitta test) AND genuinely curved (so the interior grid refines and
// the CDT is forced to fan against the starved boundary — a FLAT panel never saturates the interior
// grid and never triggers the bug: recon's §2 negative control, plain-(u,v) == metric, interior
// saturated). A cylindrical strip is the simplest surface with both properties at once: its axial
// rails (fixed angle) are EXACTLY straight lines, its angular rails (fixed height) are EXACTLY
// circular arcs, and its area has a textbook closed form (arc length × height) — matching the
// brief's "developable panel; cylinder-segment area is closed-form" gate.

// cylindricalStripSurface builds the EXACT NURBS representation of a cylinder segment: radius r,
// total angular sweep (centered on the +X axis), and axial height h. u parametrizes height (the
// straight ruling direction), v parametrizes angle via a single-span RATIONAL quadratic Bezier — the
// classical construction that reproduces a circular arc exactly for sweep < π (Piegl & Tiller §7.5):
// weights (1, cos(sweep/2), 1) with the middle control point at the tangent intersection
// r/cos(sweep/2).
func cylindricalStripSurface(t testing.TB, r, sweep, h float64) geom.BSplineSurface {
	return cylindricalStripSurfaceBunched(t, r, sweep, h, 1)
}

// cylindricalStripSurfaceBunched is cylindricalStripSurface with the SAME exact circular-arc shape
// (bunch=1 reproduces it exactly) but a non-uniform v-speed near v=0 when bunch>1 — reproducing the
// recon's real-fixture pathology ("coons parameterization bunches hard toward the rail", §2: local
// |∂P/∂u| ≈ 100× the domain mean) WITHOUT changing the surface's geometric shape or its closed-form
// area (area is parameterization-invariant). This exploits the classical fact that a conic's
// rational quadratic Bezier representation is non-unique: for control points (P0,P1,P2) and weights
// (w0,w1,w2), only the projective ratio w1²/(w0·w2) fixes the conic's SHAPE — asymmetrically
// rescaling w0→w0/bunch, w2→w2·bunch preserves that ratio (same shape, verified exactly on-circle by
// the caller) while concentrating parameter range away from v=0 (so a small Δv near the v=0 rail —
// where the interior grid's first off-boundary column lands — covers an anomalously LARGE 3D
// distance: exactly the local-metric spike that forces discretizeEdge's starved 2-point rail against
// a saturated interior grid into one giant off-chord triangle).
func cylindricalStripSurfaceBunched(t testing.TB, r, sweep, h, bunch float64) geom.BSplineSurface {
	t.Helper()
	if sweep <= 0 || sweep >= stdmath.Pi {
		t.Fatalf("cylindricalStripSurfaceBunched: sweep=%g must be in (0, pi) for a single-span exact arc", sweep)
	}
	alpha := sweep / 2
	cosA, sinA := stdmath.Cos(alpha), stdmath.Sin(alpha)
	p0 := math.P3(r*cosA, -r*sinA, 0)
	p1 := math.P3(r/cosA, 0, 0)
	p2 := math.P3(r*cosA, r*sinA, 0)
	ctrl := [][]math.Point3{
		{p0, p1, p2},
		{math.P3(p0.X, p0.Y, h), math.P3(p1.X, p1.Y, h), math.P3(p2.X, p2.Y, h)},
	}
	w0, w2 := 1/bunch, bunch // ratio w1^2/(w0*w2) = cos^2(alpha) unchanged: same conic, different speed
	w := [][]float64{{w0, cosA, w2}, {w0, cosA, w2}}
	s, err := geom.NewBSplineSurface(1, 2, ctrl, w, []float64{0, 0, 1, 1}, []float64{0, 0, 0, 1, 1, 1})
	if err != nil {
		t.Fatalf("NewBSplineSurface: %v", err)
	}
	return s
}

// cylindricalStripArea is the exact (closed-form) lateral area of the strip: arc length × height —
// the truth reference for both the convergence and over-enclosure-guard gates.
func cylindricalStripArea(r, sweep, h float64) float64 { return sweep * r * h }

// cylindricalStripFace wraps s in a single-face body. The two axial (u) edges are its EXACT straight
// rails (geom.LineSegment — the #2009 starvation trigger, since discretizeEdge gives a straight edge
// just 2 points regardless of length); the two angular (v) edges are the surface's EXACT circular
// arcs (geom.Arc3d — already sagitta-adaptive, never a densification candidate). Returns the face and
// its two straight-rail edges (railV0 at v=0, railV1 at v=1) for the direct h-sweep test.
func cylindricalStripFace(t testing.TB, s geom.BSplineSurface, r, sweep, h float64) (face *topo.Face, rails [2]*topo.Edge) {
	body, rails := cylindricalStripBody(t, s, r, sweep, h)
	return body.Faces()[0], rails
}

// cylindricalStripBody builds the single-face strip body and returns it whole (plus its two straight
// rails), the shared plumbing behind cylindricalStripFace. Split out so the #2010 pick benchmark can
// drive body-level queries (ops.LocateUsingPoint → closerEdge → discretizeEdge → starvedEdgeTarget)
// on the exact same fixture without duplicating the builder.
func cylindricalStripBody(t testing.TB, s geom.BSplineSurface, r, sweep, h float64) (body *topo.Body, rails [2]*topo.Edge) {
	t.Helper()
	lin := topo.NewLineage(topo.Tok("test", "cylstrip", 0))
	bld := topo.NewBuilder(false, lin)
	c00, c10, c11, c01 := s.PointAt(0, 0), s.PointAt(1, 0), s.PointAt(1, 1), s.PointAt(0, 1)
	v00 := bld.AddVertex(c00, topo.NewLineage(topo.Tok("test", "v", 0)))
	v10 := bld.AddVertex(c10, topo.NewLineage(topo.Tok("test", "v", 1)))
	v11 := bld.AddVertex(c11, topo.NewLineage(topo.Tok("test", "v", 2)))
	v01 := bld.AddVertex(c01, topo.NewLineage(topo.Tok("test", "v", 3)))

	alpha := sweep / 2
	arcLo, err := geom.NewArc3d(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), r, -alpha, sweep)
	if err != nil {
		t.Fatalf("NewArc3d (v=0 side): %v", err)
	}
	arcHi, err := geom.NewArc3d(math.P3(0, 0, h), math.V3(0, 0, 1), math.V3(1, 0, 0), r, -alpha, sweep)
	if err != nil {
		t.Fatalf("NewArc3d (v=1 side): %v", err)
	}
	railV0 := bld.AddEdge(geom.NewLineSegment(c00, c10), v00, v10, topo.NewLineage(topo.Tok("test", "e", 0)))
	arcU1 := bld.AddEdge(arcHi, v10, v11, topo.NewLineage(topo.Tok("test", "e", 1)))
	railV1 := bld.AddEdge(geom.NewLineSegment(c11, c01), v11, v01, topo.NewLineage(topo.Tok("test", "e", 2)))
	arcU0 := bld.AddEdge(arcLo, v00, v01, topo.NewLineage(topo.Tok("test", "e", 3)))

	bld.AddFace(s, topo.NewLineage(topo.Tok("test", "face", 0)),
		topo.OuterLoop(topo.Fwd(railV0), topo.Fwd(arcU1), topo.Fwd(railV1), topo.Rev(arcU0)))
	return bld.Build(), [2]*topo.Edge{railV0, railV1}
}

// relErr is |got-want|/|want|, or |got| when want is 0.
func relErr(got, want float64) float64 {
	if want == 0 {
		return stdmath.Abs(got)
	}
	return stdmath.Abs(got-want) / stdmath.Abs(want)
}

// maxTriangleArea returns the largest single triangle's area in m — the "one giant fan triangle"
// over-enclosure signal the recon root-caused (§2: a single 4.743-area triangle, 1.5× the whole true
// surface).
func maxTriangleArea(m *Mesh) float64 {
	var max float64
	for t := 0; 3*t+2 < len(m.Indices); t++ {
		a := m.Positions[m.Indices[3*t]]
		b := m.Positions[m.Indices[3*t+1]]
		c := m.Positions[m.Indices[3*t+2]]
		if area := 0.5 * float64(a.VectorTo(b).Cross(a.VectorTo(c)).Length()); area > max {
			max = area
		}
	}
	return max
}

// bunchedPanel is the shared ~15:1 aspect, deliberately BUNCHED synthetic panel every test in this
// file drives: radius 2, sweep 1 rad (arc length 2), height 30 (aspect 15) — matching the brief's
// "~15:1 developable panel" — with bunch=10 concentrating the v-parameterization's speed near the
// v=0 rail (aspect measured via metricScale drops from 15.15 unbunched to 6.75 bunched, still
// comfortably > aspectDensifyThreshold=4, but now exhibits the recon's real local-metric-spike
// mechanism: TestStarvedRailFixShrinksOverEnclosureTriangle shows the UNFIXED 2-point rail produces
// a single triangle carrying 23% of the panel's whole area at this bunch level — an unbunched exact
// cylinder (bunch=1) is too well-behaved (near-isometric (u,v)) to reproduce the over-enclosure at
// all, see aspect-mesh-fix-report.md's honest note on this).
const (
	bunchedR     = 2.0
	bunchedSweep = 1.0
	bunchedH     = 30.0
	bunchedK     = 10.0
)

func bunchedPanel(t *testing.T) (geom.BSplineSurface, float64) {
	t.Helper()
	s := cylindricalStripSurfaceBunched(t, bunchedR, bunchedSweep, bunchedH, bunchedK)
	return s, cylindricalStripArea(bunchedR, bunchedSweep, bunchedH)
}

// TestNurbsPcurveMeshConvergesHighAspectPanel is hard gate 1 (the brief's "recon #1 test"), driven
// through the REAL production path: nurbsPcurveMesh on the bunched high-aspect panel must land
// within 1% of the closed-form area, fold-free, and pass the over-enclosure guard at every quality
// level — the boundary densification target h is metric/geometry-derived, not quality-derived
// (starvedRailTarget, nurbs_pcurve_mesh.go), so unlike the pre-fix bug (which got WORSE, not
// better, as interior density rose: recon §1, +210%→+216% as tol tightened 1e-2→1e-5, a genuine
// non-convergence) this fix's boundary resolution does not degrade as the interior grid refines.
func TestNurbsPcurveMeshConvergesHighAspectPanel(t *testing.T) {
	s, truth := bunchedPanel(t)
	face, _ := cylindricalStripFace(t, s, bunchedR, bunchedSweep, bunchedH)

	su, sv := metricScale(s)
	if a := faceAspect(s, su, sv); a <= aspectDensifyThreshold {
		t.Fatalf("synthetic panel aspect=%.2f, want > %.1f (test setup must exercise the gate)", a, aspectDensifyThreshold)
	}

	prevErr := stdmath.Inf(1)
	for _, tol := range []float64{0.05, 0.01, 0.001, 0.0001} {
		q := Quality{ChordTolerance: tol, AngleTolerance: DefaultQuality().AngleTolerance}
		m := nurbsPcurveMesh(face, q)
		if m == nil {
			t.Fatalf("nurbsPcurveMesh declined the synthetic high-aspect panel at tol=%g", tol)
		}
		area := MeshArea(m)
		rel := relErr(area, truth)
		mx := maxTriangleArea(m)
		t.Logf("tol=%g area=%.4f relErr=%.5f folds=%d maxTri=%.4f (%.2f%% of panel)",
			tol, area, rel, FoldEdgeCount(m), mx, 100*mx/truth)
		if rel > 0.01 {
			t.Errorf("tol=%g: mesh area %.4f vs analytic %.4f (rel %.4f > 1%%)", tol, area, truth, rel)
		}
		if folds := FoldEdgeCount(m); folds != 0 {
			t.Errorf("tol=%g: %d fold edges, want 0", tol, folds)
		}
		// Over-enclosure guard (recon's real invariant — the dihedral fold test provably misses
		// this, §2): the mesh must not exceed the true area by more than a generous slack, AND no
		// single triangle may dominate the panel (the "one giant fan triangle" failure mode —
		// TestStarvedRailFixShrinksOverEnclosureTriangle shows the UNFIXED bare rail hits 23%).
		if area > 1.01*truth {
			t.Errorf("tol=%g: mesh area %.4f exceeds (1+tol)*truth %.4f — over-enclosure", tol, area, 1.01*truth)
		}
		if mx > 0.15*truth {
			t.Errorf("tol=%g: max single triangle area %.4f exceeds 15%% of the whole panel (%.4f) — a giant fan triangle", tol, mx, 0.15*truth)
		}
		// Monotone-enough as tol tightens: a small FP/re-triangulation slack (0.002 abs), since the
		// fix is already deep in the converged regime (<1%) well before the tightest tol — unlike
		// the pre-fix bug's unambiguous, order-of-magnitude non-convergence (recon §1).
		if rel > prevErr+0.002 {
			t.Errorf("tol=%g: rel error %.5f increased from previous %.5f by more than the FP slack — not converging", tol, rel, prevErr)
		}
		prevErr = rel
	}
}

// TestStarvedRailFixShrinksOverEnclosureTriangle is the discriminative proof that the fix matters —
// not merely that the production path happens to look fine. It compares the FIXED path (production
// discretizeEdge, which densifies the starved rail) against the UNFIXED path (the literal pre-#2009
// behavior: the rail's bare 2-point sampleEdgeCurve output, bypassing densifyStarvedRail) on the
// IDENTICAL bunched panel + quality. Plain MeshArea alone does not discriminate here (both land
// within ~1% — the over-enclosed triangle's excess area is masked by under-enclosure elsewhere, an
// aggregate coincidence, recon's own warning about area-only gates) — maxTriangleArea does: the
// UNFIXED single worst triangle carries a large fraction of the whole panel, which the fix shrinks
// by an order of magnitude. This is the recon's "the dihedral fold test provably misses this"
// invariant made concrete.
func TestStarvedRailFixShrinksOverEnclosureTriangle(t *testing.T) {
	s, truth := bunchedPanel(t)
	face, rails := cylindricalStripFace(t, s, bunchedR, bunchedSweep, bunchedH)
	q := DefaultQuality()
	su, sv := metricScale(s)

	fixed := nurbsPcurveMesh(face, q)
	if fixed == nil {
		t.Fatal("nurbsPcurveMesh returned nil")
	}
	unfixedUV, unfixed3D := unfixedRailLoop(t, face, rails, q)
	unfixed, _ := metricCDTPatch(s, su, sv, q, unfixed3D, unfixedUV, nil, nil, 1)
	if unfixed == nil {
		t.Fatal("metricCDTPatch (unfixed) returned nil")
	}

	fixedMax, unfixedMax := maxTriangleArea(fixed), maxTriangleArea(unfixed)
	t.Logf("FIXED   area=%.4f maxTri=%.4f (%.1f%% of panel)", MeshArea(fixed), fixedMax, 100*fixedMax/truth)
	t.Logf("UNFIXED area=%.4f maxTri=%.4f (%.1f%% of panel)", MeshArea(unfixed), unfixedMax, 100*unfixedMax/truth)

	if unfixedMax < 0.15*truth {
		t.Fatalf("test setup: UNFIXED max triangle %.4f is not a giant fan (< 15%% of panel %.4f) — bunch/geometry no longer reproduces the bug", unfixedMax, truth)
	}
	if fixedMax >= unfixedMax {
		t.Errorf("FIXED max triangle %.4f did not shrink below UNFIXED %.4f — the densification is not helping", fixedMax, unfixedMax)
	}
	if fixedMax > 0.15*truth {
		t.Errorf("FIXED max triangle %.4f still exceeds 15%% of panel %.4f — over-enclosure guard would fail", fixedMax, 0.15*truth)
	}
}

// unfixedRailLoop reproduces concatLoopPcurve's boundary assembly but with the straight rails' RAW
// sampleEdgeCurve output (2 points, bypassing densifyStarvedRail) — the literal pre-#2009 behavior —
// so it can be compared directly against the production (fixed) mesh on the identical panel.
func unfixedRailLoop(t *testing.T, f *topo.Face, rails [2]*topo.Edge, q Quality) ([]math.Point2, []math.Point3) {
	t.Helper()
	isRail := func(e *topo.Edge) bool { return e == rails[0] || e == rails[1] }
	raw := func(e *topo.Edge) []math.Point3 {
		if isRail(e) {
			return sampleEdgeCurve(e, q) // bypasses densifyStarvedRail: the pre-#2009 2-point rail
		}
		return discretizeEdge(e, q)
	}
	return assembleLoop(t, f, raw)
}

// TestStarvedRailHSweepMonotoneConvergence is the direct, literal reproduction of the recon's own
// validated table (aspect-mesh-recon-report.md §3): driving densifyStraightEdgeCurve +
// metricCDTPatch at successively smaller boundary targets h (independent of kBoundaryCells/Quality)
// on the bunched panel and asserting the "giant fan triangle" signal — maxTriangleArea, the
// over-enclosure invariant the dihedral fold test misses — shrinks MONOTONICALLY (non-increasing,
// small slack) as h tightens, converging well below the panel's whole area; MeshArea and
// FoldEdgeCount stay bounded throughout. Proof of the underlying MECHANISM (recon's own metric),
// not just the one production density TestNurbsPcurveMeshConvergesHighAspectPanel checks.
func TestStarvedRailHSweepMonotoneConvergence(t *testing.T) {
	s, truth := bunchedPanel(t)
	face, rails := cylindricalStripFace(t, s, bunchedR, bunchedSweep, bunchedH)
	q := DefaultQuality()
	su, sv := metricScale(s)

	prevMax := stdmath.Inf(1)
	for _, target := range []float64{30.0, 15.0, 8.0, 4.0, 2.0, 1.0, 0.5, 0.25} { // coarse (~undensified) → fine
		outerUV, outer3D := starvedSweepLoop(t, face, rails, q, target)
		m, loops := metricCDTPatch(s, su, sv, q, outer3D, outerUV, nil, nil, 1)
		if m == nil {
			t.Fatalf("h=%g: metricCDTPatch returned nil", target)
		}
		if !patchIsManifold(m, loops) {
			t.Fatalf("h=%g: patch is not manifold", target)
		}
		area, mx := MeshArea(m), maxTriangleArea(m)
		t.Logf("h=%.3g boundaryPts=%d tris=%d area=%.4f relErr=%.5f folds=%d maxTri=%.4f",
			target, len(outer3D), len(m.Indices)/3, area, relErr(area, truth), FoldEdgeCount(m), mx)
		if FoldEdgeCount(m) != 0 {
			t.Errorf("h=%g: folds present, want 0", target)
		}
		if relErr(area, truth) > 0.05 {
			t.Errorf("h=%g: area %.4f vs truth %.4f (rel %.4f > 5%%)", target, area, truth, relErr(area, truth))
		}
		if mx > prevMax+0.25 { // monotone non-increasing (recon's #1 test), slack for re-triangulation noise
			t.Errorf("h=%g: max triangle area %.4f increased from previous %.4f — not converging monotonically", target, mx, prevMax)
		}
		prevMax = mx
	}
	if prevMax > 0.15*truth {
		t.Errorf("finest h: max triangle area %.4f still exceeds 15%% of panel %.4f", prevMax, 0.15*truth)
	}
}

// starvedSweepLoop reproduces concatLoopPcurve's boundary assembly for cylindricalStripFace, but
// densifies the two straight rails at an EXPLICIT target h (bypassing kBoundaryCells/aspect-gate) via
// the production densifyStraightEdgeCurve — driving the exact function the recon validated, at
// arbitrary densities, the way the recon's own throwaway probe did.
func starvedSweepLoop(t *testing.T, f *topo.Face, rails [2]*topo.Edge, q Quality, target float64) ([]math.Point2, []math.Point3) {
	t.Helper()
	isRail := func(e *topo.Edge) bool { return e == rails[0] || e == rails[1] }
	dense := func(e *topo.Edge) []math.Point3 {
		if isRail(e) {
			return densifyStraightEdgeCurve(e, target)
		}
		return discretizeEdge(e, q) // the curved arcs: sagitta-adaptive, untouched
	}
	return assembleLoop(t, f, dense)
}

// assembleLoop reproduces concatLoopPcurve's boundary assembly (drop the point shared with the
// previous edge, strip the closing duplicate, re-derive (u,v) by projection) but with a
// caller-supplied per-edge point source — the shared plumbing behind starvedSweepLoop (explicit h)
// and unfixedRailLoop (the pre-#2009 raw 2-point rail).
func assembleLoop(t *testing.T, f *topo.Face, pointsFor func(*topo.Edge) []math.Point3) ([]math.Point2, []math.Point3) {
	t.Helper()
	s := f.Geometry()
	var p3 []math.Point3
	for _, u := range f.Loops()[0].EdgeUses() {
		pts := pointsFor(u.Edge())
		if u.Reversed() {
			pts = reverse3(pts)
		}
		if len(p3) > 0 {
			pts = pts[1:]
		}
		p3 = append(p3, pts...)
	}
	if n := len(p3); n > 1 && p3[0].DistanceTo(p3[n-1]) < ResolutionForPoints(p3).Weld() {
		p3 = p3[:n-1]
	}
	return geom.ProjectCurveToSurface(s, p3), p3
}
