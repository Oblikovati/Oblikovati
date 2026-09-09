// SPDX-License-Identifier: GPL-2.0-only

package query

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
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
