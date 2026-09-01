// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// wingTestArc is a quarter arc of radius 8 in the x=2 plane (the shape of an arm cross-section at a
// wing cut station), centred so its two endpoints sit where a wing's nodeTa/nodeTb would.
func wingTestArc(t *testing.T) geom.Arc3d {
	t.Helper()
	arc, err := geom.Arc3dByThreePoints(
		math.P3(2, 8, 0), math.P3(2, 8/stdmath.Sqrt2, 8/stdmath.Sqrt2), math.P3(2, 0, 8))
	if err != nil {
		t.Fatalf("wingTestArc: %v", err)
	}
	return arc
}

// TestSampledArcSegsCarryExactSubSpans is the wing-side cure of the last nil-offer family
// (wing-arm-arcs-report.md): the wing's cut cross-section used to tile ringSegSamples straight
// chords with NO curve, so the flank patch's exact arm-arc sub-spans always met a nil on the shared
// edges (12 nil-vs-curve records per setback case, repaired only by catalog adoption). Each segment
// must now carry the arm arc RESTRICTED to its own sub-span — the same value the patch offers — so
// the weld becomes a two-sided value agreement and the wing's own loop model bounds the true arc.
func TestSampledArcSegsCarryExactSubSpans(t *testing.T) {
	t.Parallel()
	arc := wingTestArc(t)
	lo, hi := arc.Domain()
	for _, tc := range []struct {
		name       string
		start, end math.Point3
	}{
		{"forward", arc.PointAt(lo), arc.PointAt(hi)},
		{"reversed", arc.PointAt(hi), arc.PointAt(lo)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			segs := sampledArcSegs(arc, tc.start, tc.end)
			assertWingSegsOnArc(t, arc, segs)
		})
	}
}

// assertWingSegsOnArc requires every wing cut segment to carry a curve that spans its own endpoints
// and stays on the arc's circle — the "exact sub-span, not chord" contract. The mid-station radius
// check is what fails when the curve is nil (no sub-span at all) or a straight chord (mid-point
// sagitta 2R sin²(θ/4) ≈ 0.0684 at R=8, four decades above the 1e-9 gate).
func assertWingSegsOnArc(t *testing.T, arc geom.Arc3d, segs []endSeg) {
	t.Helper()
	if len(segs) != ringSegSamples {
		t.Fatalf("got %d segments, want ringSegSamples=%d", len(segs), ringSegSamples)
	}
	for i, s := range segs {
		if s.curve == nil {
			t.Fatalf("seg %d carries no curve — the wing still declines its own boundary (chord)", i)
		}
		clo, chi := s.curve.Domain()
		assertWingCurveEnd(t, i, "start", s.curve.PointAt(clo), s.from)
		assertWingCurveEnd(t, i, "end", s.curve.PointAt(chi), s.to)
		mid := s.curve.PointAt(clo + 0.5*(chi-clo))
		if r := float64(arc.Center.DistanceTo(mid)); stdmath.Abs(r-arc.Radius) > 1e-9 {
			t.Errorf("seg %d mid-station radius %.12f, want %.12f — the carried curve is not the arc's own sub-span", i, r, arc.Radius)
		}
	}
}

// assertWingCurveEnd fails when p is not within 1e-9 of want (the wing pins its two node endpoints
// exactly, interior stations are byte-identical arc samples).
func assertWingCurveEnd(t *testing.T, seg int, which string, p, want math.Point3) {
	t.Helper()
	if float64(p.DistanceTo(want)) > 1e-9 {
		t.Errorf("seg %d %s: curve endpoint %v, want the segment's own %v", seg, which, p, want)
	}
}

// TestSampledArcSegsPointsUnchanged pins the weld contract: the fix must not move a single boundary
// point — pts[0]/pts[n] pinned to the wing's node points, interior points the arc's own i/n samples
// (byte-identical to sampleCurveN's, the points the flank patch welds to). A moved point would open
// the wing↔patch weld as a T-junction.
func TestSampledArcSegsPointsUnchanged(t *testing.T) {
	t.Parallel()
	arc := wingTestArc(t)
	lo, hi := arc.Domain()
	start, end := arc.PointAt(lo), arc.PointAt(hi)
	segs := sampledArcSegs(arc, start, end)
	if segs[0].from != start || segs[len(segs)-1].to != end {
		t.Errorf("node pinning lost: got %v..%v, want %v..%v", segs[0].from, segs[len(segs)-1].to, start, end)
	}
	n := ringSegSamples
	for i := 1; i < n; i++ {
		want := arc.PointAt(lo + float64(i)/float64(n)*(hi-lo))
		if segs[i].from != want {
			t.Errorf("interior point %d moved: %v, want the arc's own sample %v", i, segs[i].from, want)
		}
	}
}
