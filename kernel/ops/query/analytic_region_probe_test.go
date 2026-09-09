// SPDX-License-Identifier: GPL-2.0-only

package query

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// uvBoxRing is a rectangular closed uv polyline centred at (u, v), used here as one small hole.
func uvBoxRing(u, v, halfU, halfV float64) []arcSample {
	return []arcSample{
		{u: u - halfU, v: v - halfV}, {u: u + halfU, v: v - halfV},
		{u: u + halfU, v: v + halfV}, {u: u - halfU, v: v + halfV},
	}
}

// TestTwoDisjointLoopsAreProbedAtTheirOwnScale pins the search window (Oblikovati/Oblikovati#3516).
// A single grid over the loops' SHARED box resolves a region only while it is bigger than one cell of
// that box; two small loops half a period apart make the shared box a hundred times either loop, and
// every grid point then lands outside both. The two mouths of an axial bore through a ring are
// exactly that pair. Each loop's own box is gridded too, so the scale of the search follows the
// region rather than the pair's spread.
func TestTwoDisjointLoopsAreProbedAtTheirOwnScale(t *testing.T) {
	t.Parallel()
	half := 1.0 / float64(regionProbeGrid) / 4 // a quarter of a shared-box cell: invisible to that grid
	polys := [][]arcSample{uvBoxRing(1, 1, half, half), uvBoxRing(1, 1+stdmath.Pi, half, half)}
	per := uvPeriod{u: 2 * stdmath.Pi, v: 2 * stdmath.Pi}

	shared := regionProbeWindows(polys)[0]
	if _, _, d := deepestInWindow(polys, per, shared, true); d > 0 {
		t.Fatalf("the premise is stale: the shared-box grid found a point at depth %g, so this pair no "+
			"longer exercises the per-loop window", d)
	}
	u, v, ok := deepestProbe(polys, per, regionProbeWindows(polys), true)
	if !ok {
		t.Fatalf("no probe found inside either loop; the per-loop windows are not being gridded")
	}
	if !uvCrossingsOdd(polys, u, v, per) {
		t.Errorf("probe (%g, %g) is not inside the region the loops enclose", u, v)
	}
}

// TestASingleLoopKeepsTheSharedWindow holds the other half of the change: a face whose loops nest
// under one outer loop must grid exactly the window it always did, so nothing about an ordinary
// trimmed face moves.
func TestASingleLoopKeepsTheSharedWindow(t *testing.T) {
	t.Parallel()
	polys := [][]arcSample{uvBoxRing(1, 1, 0.5, 0.5)}
	if got := len(regionProbeWindows(polys)); got != 1 {
		t.Errorf("a single-loop face grids %d windows, want the shared one only", got)
	}
}

// TestABoredTorusFaceIsIntegratedNotDeclined is the #3516 regression at the integrator: the torus of
// a drilled ring must integrate over its analytic B-rep at a bore radius far below the one the fixed
// shared-box grid could resolve (measured: it declined at and below 0.0631 and certified at 0.08).
// The area it must report is the whole torus minus the two bore mouths, each ~pi·bore^2.
func TestABoredTorusFaceIsIntegratedNotDeclined(t *testing.T) {
	t.Parallel()
	body := boredRing(t, 1e-3)
	area, ok := AnalyticFaceArea(body.Faces()[0])
	if !ok {
		t.Fatalf("the bored torus face declined analytic integration; the body then falls to its mesh, " +
			"which measures this torus 2.839 light")
	}
	full := 4 * stdmath.Pi * stdmath.Pi * 5 * 1.5
	want := full - 2*stdmath.Pi*1e-3*1e-3
	if rel := stdmath.Abs(area-want) / want; rel > 1e-9 { // tol:calibrated — measured 1.2e-11
		t.Errorf("bored torus face area %.12g, want %.12g (%.3g relative)", area, want, rel)
	}
}

// TestAComplementSideFaceHasAnInteriorPoint is the #3516 regression at the certificate's probe. A
// face on a CLOSED surface may hold either side of its loops, and FaceInteriorPoint only ever
// searched the ENCLOSED one: the bored torus — the face carrying all but 6.3e-6 of the result's area
// — therefore had no representative point at ANY bore radius, and every per-face gate that skips an
// unprobeable face skipped it.
func TestAComplementSideFaceHasAnInteriorPoint(t *testing.T) {
	t.Parallel()
	for _, bore := range []float64{1e-3, 0.1, 0.8} {
		f := boredRing(t, bore).Faces()[0]
		p, ok := FaceInteriorPoint(f)
		if !ok {
			t.Errorf("bore %g: the bored torus face has no interior point", bore)
			continue
		}
		if !brep.PointInFaceTrim(f, p) {
			t.Errorf("bore %g: FaceInteriorPoint returned %v, which is not on the face", bore, p)
		}
	}
}

// boredRing is the RING corpus torus with an axial drill of the given radius through its tube, the
// pair every row above measures. Face 0 is the torus, face 1 the bore wall.
func boredRing(t *testing.T, bore float64) *topo.Body {
	t.Helper()
	ring, err := brep.SolidTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 1.5, "ring")
	if err != nil {
		t.Fatalf("ring: %v", err)
	}
	drill, err := brep.SolidCylinder(math.P3(5, 0, -4), math.V3(0, 0, 1), bore, 8)
	if err != nil {
		t.Fatalf("drill r=%g: %v", bore, err)
	}
	body, err := brep.Boolean(brep.Difference, ring, drill)
	if err != nil {
		t.Fatalf("bore %g: %v", bore, err)
	}
	return body
}

// uvDenseRing is a rectangular closed uv polyline with n samples per side, so a probe's distance to
// "the nearest boundary sample" means distance to the boundary. uvBoxRing's four corners do not:
// the middle of a long thin rectangle is a corner's length away from all of them.
func uvDenseRing(uLo, uHi, vLo, vHi float64, n int) []arcSample {
	var out []arcSample
	for i := range n {
		f := float64(i) / float64(n)
		out = append(out, arcSample{u: uLo + (uHi-uLo)*f, v: vLo})
	}
	for i := range n {
		f := float64(i) / float64(n)
		out = append(out, arcSample{u: uHi, v: vLo + (vHi-vLo)*f})
	}
	for i := range n {
		f := float64(i) / float64(n)
		out = append(out, arcSample{u: uHi - (uHi-uLo)*f, v: vHi})
	}
	for i := range n {
		f := float64(i) / float64(n)
		out = append(out, arcSample{u: uLo, v: vHi - (vHi-vLo)*f})
	}
	return out
}

// TestTheDeepestProbeIsRankedRelativeToItsOwnWindow pins the comparison deepestProbe makes across
// windows. Acceptance is RELATIVE — deepestInWindow stops at regionProbeDeepEnough of the window's
// diagonal — so ranking by ABSOLUTE depth lets a probe 0.25% inside a huge shared box beat one 9.8%
// inside a small loop's own box, and 0.25% inside is the ambiguous point that rule exists to reject.
//
// The two loops here are both slender, so no window reaches deepEnough and every one is searched:
// loop A (100 x 1) admits probes up to 0.5 deep, which is 0.5% of its window; loop B (1 x 0.2)
// admits 0.1, which is 9.8% of its. Absolute ranking returns A's point, relative ranking B's.
func TestTheDeepestProbeIsRankedRelativeToItsOwnWindow(t *testing.T) {
	t.Parallel()
	polys := [][]arcSample{
		uvDenseRing(0, 100, 0, 1, 60),
		uvDenseRing(200, 201, 0, 0.2, 60),
	}
	per := uvPeriod{}
	u, v, ok := deepestProbe(polys, per, regionProbeWindows(polys), true)
	if !ok {
		t.Fatalf("no probe found inside either loop")
	}
	if !uvCrossingsOdd(polys, u, v, per) {
		t.Fatalf("probe (%g, %g) is not inside the region the loops enclose", u, v)
	}
	if u < 150 {
		t.Errorf("probe (%g, %g) is in the slender 100x1 loop, %g of its own window deep; the 1x0.2 "+
			"loop admits a probe ~0.098 of ITS window deep, which is the unambiguous one",
			u, v, uvDepthOn(polys, u, v, per, true)/regionProbeWindows(polys)[1].diagonal())
	}
}

// TestTheComplementProbeAnswersForASeamWrappingLoop pins the REVERT of a decline that cost probe
// coverage. A loop set wrapping the parameter seam has no meaningful crossing parity, so the depth
// this probe ranks by is meaningless there — but declining on that ground took the certificate from
// 4 unprobed faces to 26 across kernel/ops/boolean and from 0 to 5 on its drill/torus/ring rows, for
// a risk no measured case demonstrated. brep.PointInFaceTrim certifies the probe against the face
// either way, so the worst an unranked probe can be is near the trim rather than deep inside it.
func TestTheComplementProbeAnswersForASeamWrappingLoop(t *testing.T) {
	t.Parallel()
	tor, err := geom.NewTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 1.5)
	if err != nil {
		t.Fatalf("torus: %v", err)
	}
	band := faceLoop{
		edges: []loopEdge{{samples: uvDenseRing(0, 2*stdmath.Pi, 1, 1.2, 40), uPeriod: 2 * stdmath.Pi, vPeriod: 2 * stdmath.Pi}},
		netU:  2 * stdmath.Pi,
	}
	if !loopsWrapASeam([]faceLoop{band}) {
		t.Fatalf("the premise is stale: a loop with netU = one period no longer reads as seam-wrapping")
	}
	if _, _, ok := faceComplementUV(tor, []faceLoop{band}); !ok {
		t.Error("the complement probe declined a seam-wrapping loop; an unranked certified probe beats none")
	}
}

// TestEveryFaceOfTheCorpusPairsIsProbeable is the RATCHET on probe coverage, and it exists because
// coverage was won and then quietly given back. Round 0 of #3516 took the boolean certificate from
// 26 unexamined faces to 0 on this corpus; round 1's seam decline took it back to 5, in exchange for
// a robustness argument with no measured case behind it. A count nothing pins drifts.
//
// The faces at stake are the bored torus of a drilled ring — the one that holds the COMPLEMENT of
// what its loops enclose — swept across the radii the corpus uses. None of these radii produces a
// seam-wrapping loop set, so this row does NOT catch the decline that caused the regression: that is
// TestTheComplementProbeAnswersForASeamWrappingLoop's job, and it fails the moment the decline
// returns. This row holds the other half — that the complement probe keeps answering at all — and it
// is the one that would catch a probe lost to the window list or the ranking instead.
func TestEveryFaceOfTheCorpusPairsIsProbeable(t *testing.T) {
	t.Parallel()
	for _, bore := range []float64{1e-3, 1e-2, 0.05, 0.0631, 0.1, 0.2, 0.4, 0.631, 0.8, 1.2} {
		body := boredRing(t, bore)
		for _, f := range body.Faces() {
			if _, ok := FaceInteriorPoint(f); !ok {
				t.Errorf("bore %g: face %q (%T, %d loops) has no interior point, so every per-face gate "+
					"that skips an unprobeable face skips it", bore, f.ReferenceKey(), f.Geometry(), len(f.Loops()))
			}
		}
	}
}
