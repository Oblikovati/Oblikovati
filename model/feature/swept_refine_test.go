// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
)

// Regression for Oblikovati#2078. A twisted swept side is a RULED surface that sideQuad stands in
// for with two triangles. Over a strongly twisted span those triangles are nowhere near it: for the
// blade below they miss it by 0.13 on a body only 0.16 thick, so opposite sides of the blade cut
// through each other. Picking the other diagonal is no answer — measured, the two choices bracket
// the true volume near-symmetrically (0.173 and 0.599 against 0.386). The span has to be subdivided.

// twistedBladeSections is the #2078 fixture: a thin rectangle whose far section is rotated about
// the section's own centre. 0.3 rad is the twist at which the unrefined body self-intersected.
func twistedBladeSections(twist float64) [][]math.Point3 {
	cx := 1.475
	root := []math.Point3{{X: 0.6, Y: -0.08, Z: 0}, {X: 2.35, Y: -0.08, Z: 0}, {X: 2.35, Y: 0.08, Z: 0}, {X: 0.6, Y: 0.08, Z: 0}}
	c, s := stdmath.Cos(twist), stdmath.Sin(twist)
	tip := make([]math.Point3, 4)
	for i, p := range root {
		dx, dy := float64(p.X)-cx, float64(p.Y)
		tip[i] = math.P3(cx+dx*c-dy*s, dx*s+dy*c, 1.4)
	}
	return [][]math.Point3{root, tip}
}

// ruledLoftVolume is the exact volume of the ruled solid the two sections span. The section at
// parameter t is the original acted on by M(t) = (1−t)I + tR(θ), so its area is |det M(t)| times
// the original's, and the volume is the height times the integral of that determinant.
func ruledLoftVolume(twist, area, height float64) float64 {
	c, s := stdmath.Cos(twist), stdmath.Sin(twist)
	const steps = 20000
	sum := 0.0
	for k := range steps {
		t := (float64(k) + 0.5) / steps
		m00 := (1 - t) + t*c
		sum += (m00*m00 + (t*s)*(t*s)) / steps
	}
	return sum * area * height
}

// TestTwistedSweepDoesNotSelfIntersect is the #2078 gate: the delivered body must not have faces
// passing through each other. ops.Validate cannot see this — it is topology only — which is why the
// original guard test passed on a body that was both self-intersecting and 55% under volume.
func TestTwistedSweepDoesNotSelfIntersect(t *testing.T) {
	t.Parallel()
	for _, twist := range []float64{0, 0.001, 0.05, 0.3} {
		blade, err := sweptSolid(twistedBladeSections(twist), false, "blade")
		if err != nil {
			t.Fatalf("twist=%.3f sweptSolid: %v", twist, err)
		}
		if hits := ops.SelfIntersections(blade, ops.DefaultQuality()); len(hits) > 0 {
			t.Errorf("twist=%.3f: the swept blade interpenetrates itself (%d pairs), first at %v",
				twist, len(hits), hits[0].Witness)
		}
	}
}

// TestTwistedSweepVolumeIsNotWildlyWrong pins the volume the refinement recovers. The unrefined
// body measured 0.1732 against a true ruled volume of 0.3862 — 55% low. The bound here is the
// faceting budget maxFacetWarpRatio buys, not the exact answer: a faceted loft is an approximation
// by design, and closing the rest of the gap costs mesh everywhere (#2081).
func TestTwistedSweepVolumeIsNotWildlyWrong(t *testing.T) {
	t.Parallel()
	const twist = 0.3
	blade, err := sweptSolid(twistedBladeSections(twist), false, "blade")
	if err != nil {
		t.Fatalf("sweptSolid: %v", err)
	}
	want := ruledLoftVolume(twist, 1.75*0.16, 1.4)
	got := ops.BodyGeometryProperties(blade, ops.DefaultQuality()).Volume
	if rel := stdmath.Abs(got-want) / want; rel > 0.25 {
		t.Errorf("twisted blade volume = %g, want ~%g (off by %.1f%%; measured 18.4%%, unrefined was 55%% low)",
			got, want, rel*100)
	}
	if got < 0.30 {
		t.Errorf("volume %g is no better than the unrefined 0.1732 — the span was not refined", got)
	}
}

// TestUntwistedSweepIsUntouched: a span whose sides are planar is already exact, so refinement must
// not add a single section. Otherwise every ordinary extrude/revolve pays for this.
func TestUntwistedSweepIsUntouched(t *testing.T) {
	t.Parallel()
	secs := twistedBladeSections(0)
	if got := refineWarpedSpans(secs, false); len(got) != len(secs) {
		t.Errorf("an unwarped span was subdivided into %d sections, want %d", len(got), len(secs))
	}
}

// TestSpanWarpRatioIsScaleFree: the ratio divides a deviation by a length, so the same shape built
// a thousand times bigger must be refined the same way (ADR-0042). A bare deviation would refine
// large models to death and leave small ones unrefined.
func TestSpanWarpRatioIsScaleFree(t *testing.T) {
	t.Parallel()
	base := twistedBladeSections(0.3)
	ref := spanWarpRatio(base[0], base[1])
	for _, k := range []float64{1e-3, 1, 1e3} {
		scaled := make([][]math.Point3, 2)
		for s := range base {
			scaled[s] = make([]math.Point3, len(base[s]))
			for i, p := range base[s] {
				scaled[s][i] = math.P3(float64(p.X)*k, float64(p.Y)*k, float64(p.Z)*k)
			}
		}
		if got := spanWarpRatio(scaled[0], scaled[1]); stdmath.Abs(got-ref) > 1e-9 {
			t.Errorf("at scale %g the warp ratio is %g, want %g — the measure carries model scale", k, got, ref)
		}
	}
}

// TestSpanWarpRatioCannotExceedAHalf is why refinement needs no explicit subdivision cap. Both
// a_i-a_j and b_i-b_j are SIDES of the quad, so neither exceeds quadScale; the warp is their
// difference, so |W| <= 2*quadScale and the ratio |W|/4/quadScale <= 1/2. A cap would be
// unreachable code. The fixture is a span folded back on itself — as violent as a span gets.
func TestSpanWarpRatioCannotExceedAHalf(t *testing.T) {
	t.Parallel()
	folded := [][2][]math.Point3{
		{{math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(1, 1, 0), math.P3(0, 1, 0)},
			{math.P3(0, 0, 1), math.P3(-1e4, 0, 1), math.P3(-1e4, 1, 1), math.P3(0, 1, 1)}},
		{{math.P3(0, 0, 0), math.P3(5, 0, 0), math.P3(5, 1, 0), math.P3(0, 1, 0)},
			{math.P3(0, 0, 1e-9), math.P3(-5, 0, 1e-9), math.P3(-5, 1, 1e-9), math.P3(0, 1, 1e-9)}},
	}
	for i, f := range folded {
		if got := spanWarpRatio(f[0], f[1]); got > 0.5+1e-12 {
			t.Errorf("fixture %d warps by %g, above the proven bound of 1/2", i, got)
		}
	}
	// The bound makes the subdivision count finite by construction, at the current budget 17.
	if got := spanSubdivision(folded[1][0], folded[1][1]); got > 17 {
		t.Errorf("a folded span asked for %d sub-spans, above the ceil(0.5/%g) = 17 the bound allows",
			got, maxFacetWarpRatio)
	}
}

// TestClosedLoopsAreNotRefined: the wrap span carries the correspondence offset closureShift
// resolved, and a section part-way through that offset is not defined. Refining it would pair the
// seam by the monodromy again — the very thing wrapShift exists to prevent.
func TestClosedLoopsAreNotRefined(t *testing.T) {
	t.Parallel()
	secs := [][]math.Point3{twistedBladeSections(0.3)[0], twistedBladeSections(0.3)[1], twistedBladeSections(0.6)[1]}
	if got := refineWarpedSpans(secs, true); len(got) != len(secs) {
		t.Errorf("a closed loop was refined to %d sections, want %d untouched", len(got), len(secs))
	}
}

// TestLerpSectionStaysOnTheRuledSurface: the inserted sections must lie on the surface the span
// already spanned, so refinement sharpens the approximation without moving the shape.
func TestLerpSectionStaysOnTheRuledSurface(t *testing.T) {
	t.Parallel()
	// Every coordinate must move, and by t — not by a half, and not only on some axes.
	a := []math.Point3{math.P3(1, 0, 0), math.P3(2, 0, 0)}
	b := []math.Point3{math.P3(9, 4, 8), math.P3(2, 4, 8)}
	mid := lerpSection(a, b, 0.25)
	if want := math.P3(3, 1, 2); float64(mid[0].DistanceTo(want)) > 1e-12 {
		t.Errorf("lerpSection at t=0.25 gave %v, want %v", mid[0], want)
	}
}

// TestGeneratorDensitySpansAreNotRefined pins the LOWER bound on maxFacetWarpRatio. A helical coil
// emits spans at a warp ratio of 0.0245 — the sampling its own generator already judged adequate.
// Refining those doubles the coil's mesh for no gain and reinstates #879, where a fine-pitch
// coil-join shredded into thousands of unpaired open edges because coincident vertices landed on
// opposite sides of the weld grid's cells. So the constant is bounded below, not just above.
func TestGeneratorDensitySpansAreNotRefined(t *testing.T) {
	t.Parallel()
	// A ring of 24 points rotated about a distant axis by one coil step — the shape a coil emits.
	const meanR, wireR, step = 0.35, 0.1, 2 * stdmath.Pi / 32
	ring := func(rot float64) []math.Point3 {
		out := make([]math.Point3, 24)
		for i := range out {
			a := 2 * stdmath.Pi * float64(i) / 24
			x, y := meanR+wireR*stdmath.Cos(a), wireR*stdmath.Sin(a)
			out[i] = math.P3(x*stdmath.Cos(rot)-y*stdmath.Sin(rot), x*stdmath.Sin(rot)+y*stdmath.Cos(rot), rot*0.3/(2*stdmath.Pi))
		}
		return out
	}
	a, b := ring(0), ring(step)
	if got := spanWarpRatio(a, b); got >= maxFacetWarpRatio {
		t.Fatalf("a coil-density span warps by %g, at or above the %g budget — every coil would be "+
			"subdivided, which is the #879 fine-pitch weld failure", got, maxFacetWarpRatio)
	}
	if got := len(refineWarpedSpans([][]math.Point3{a, b}, false)); got != 2 {
		t.Errorf("a coil-density span was refined to %d sections, want 2 untouched", got)
	}
}

// TestSpanWarpRatioScansEveryQuad: the span's verdict is the WORST facet in it, so a warp on any
// cross-section edge must count. A fixture whose warp sits on the first edge cannot show this —
// every other fixture here has its worst quad first — so this one deliberately leaves the first
// edge unwarped and bends the third.
func TestSpanWarpRatioScansEveryQuad(t *testing.T) {
	t.Parallel()
	a := []math.Point3{math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(1, 1, 0), math.P3(0, 1, 0)}
	// Points 0 and 1 translate straight up, so the first quad is planar; 2 and 3 splay apart.
	b := []math.Point3{math.P3(0, 0, 1), math.P3(1, 0, 1), math.P3(1, 1, 1), math.P3(0, 3, 1)}
	if first := spanWarpRatio(a[:2], b[:2]); first > 1e-12 {
		t.Fatalf("the fixture's first edge warps by %g — it cannot prove later edges are scanned", first)
	}
	if got := spanWarpRatio(a, b); got <= 1e-12 {
		t.Errorf("spanWarpRatio = %g: a warp on a later cross-section edge went unseen", got)
	}
	if got := spanSubdivision(a, b); got < 2 {
		t.Errorf("a span warped only on a later edge asked for %d sub-spans, want at least 2", got)
	}
}
