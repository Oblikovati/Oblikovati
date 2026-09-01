// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// TestSampleCurve3OpenTrimmedPtsUnchanged is the WATERTIGHTNESS invariant for the N7 tessellation
// fix: sampleCurve3OpenTrimmed MUST return the exact same points sampleCurve3Open returns (the canal
// weld shares vertices by point, ADR-C4-2/F3 — a perturbed point cracks the seam). Only the per-sub-
// edge curves change (whole-curve → trimmed sub-span). Bytes, not tolerance, for both directions.
func TestSampleCurve3OpenTrimmedPtsUnchanged(t *testing.T) {
	t.Parallel()
	arc := trimTestArc(t)
	for _, rev := range []bool{false, true} {
		want := sampleCurve3Open(arc, rev)
		got, curves := sampleCurve3OpenTrimmed(arc, rev)
		if len(got) != len(want) || len(curves) != len(want) {
			t.Fatalf("rev=%v: len pts=%d curves=%d, want %d", rev, len(got), len(curves), len(want))
		}
		for i := range want {
			if got[i] != want[i] { // byte-identical, not within-tolerance
				t.Fatalf("rev=%v: pts[%d]=%v != sampleCurve3Open %v (watertight weld broken)", rev, i, got[i], want[i])
			}
		}
	}
}

// TestSampleCurve3OpenTrimmedSubSpans proves each sub-edge carries the curve TRIMMED to its own
// sub-span (curveSpan==vGap, the convention every normal kernel edge obeys): the curve's domain
// endpoints map to that sub-edge's own two vertices, and its interior lies ON the base arc (not on
// the chord) — the property that meshes the patch fold-free.
func TestSampleCurve3OpenTrimmedSubSpans(t *testing.T) {
	t.Parallel()
	arc := trimTestArc(t)
	lo, hi := arc.Domain()
	for _, rev := range []bool{false, true} {
		pts, curves := sampleCurve3OpenTrimmed(arc, rev)
		far := arc.PointAt(hi)
		if rev {
			far = arc.PointAt(lo)
		}
		for i, c := range curves {
			start, end := c.PointAt(0), c.PointAt(1)
			wantEnd := far
			if i+1 < len(pts) {
				wantEnd = pts[i+1]
			}
			if float64(start.DistanceTo(pts[i])) > 1e-9 {
				t.Fatalf("rev=%v sub-edge %d: curve start %v != vertex %v (curveSpan!=vGap)", rev, i, start, pts[i])
			}
			if float64(end.DistanceTo(wantEnd)) > 1e-9 {
				t.Fatalf("rev=%v sub-edge %d: curve end %v != next vertex %v (curveSpan!=vGap)", rev, i, end, wantEnd)
			}
			mid := c.PointAt(0.5)
			if r := float64(math.P3(0, 0, 0).DistanceTo(mid)); r < 0.999 || r > 1.001 {
				t.Fatalf("rev=%v sub-edge %d: midpoint radius %g, want ~1 (interior on the arc, not the chord)", rev, i, r)
			}
		}
	}
}

// trimTestArc is a unit semicircle in the XY plane — a curved base whose sub-span midpoints sit on
// the circle (radius 1), so a whole-curve-per-sub-edge regression is caught by the radius check.
func trimTestArc(t *testing.T) geom.Arc3d {
	t.Helper()
	arc, err := geom.Arc3dByThreePoints(math.P3(1, 0, 0), math.P3(0, 1, 0), math.P3(-1, 0, 0))
	if err != nil {
		t.Fatalf("Arc3dByThreePoints: %v", err)
	}
	return arc
}
