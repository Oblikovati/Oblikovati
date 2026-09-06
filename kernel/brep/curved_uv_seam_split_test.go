// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Splitting a segment that straddles the azimuth seam must leave BOTH pieces inside the strip. The piece
// on the far side arrives measured from the side it left — a run climbing a hair past 2π ends at 6.2955 —
// and must be re-based onto its own end of the strip. Left unrebased that piece was one segment spanning
// the WHOLE chart at the crossing's v (ADR-0061 stage 4).
func TestSeamSplitKeepsBothPiecesInTheStrip(t *testing.T) {
	t.Parallel()
	twoPi := 2 * stdmath.Pi
	for _, tc := range []struct {
		name string
		a, b math.Point2
	}{
		{"climbs past 2pi", math.P2(6.2709, 5.7574), math.P2(6.2955, 5.7494)},
		{"drops below 0", math.P2(0.0123, 5.7494), math.P2(-0.0123, 5.7574)},
		{"crosses well past", math.P2(6.0, 1.0), math.P2(6.9, 2.0)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			out := splitSeamCrossing(uvSeg{a: tc.a, b: tc.b, curve: geom.NewLineSegment(math.P3(0, 0, 0), math.P3(1, 0, 0)), tB: 1})
			if len(out) != 2 {
				t.Fatalf("splitSeamCrossing(%v→%v) returned %d segments, want 2", tc.a, tc.b, len(out))
			}
			for i, s := range out {
				for _, u := range []float64{float64(s.a.X), float64(s.b.X)} {
					if u < -arrTol || u > twoPi+arrTol {
						t.Errorf("piece %d has u=%.6f outside [0, 2π] (segment %v→%v)", i, u, s.a, s.b)
					}
				}
				if span := stdmath.Abs(float64(s.b.X - s.a.X)); span > stdmath.Pi {
					t.Errorf("piece %d spans %.6f in u — it crosses the whole chart instead of ending at the seam", i, span)
				}
			}
		})
	}
}

// The corpus case the defect came from: a thin rod driven obliquely through a cylinder's WALL and out
// through its top CAP. The rod's entry crossing wraps the rod's azimuth and crosses the chart seam at a
// place the crossing solver does not see, so the split-by-interpolation fallback carries it — and carried
// it wrong, leaving three unpaired edges and an open body (ADR-0061 stage 4).
func TestObliqueTunnelThroughWallAndCapWelds(t *testing.T) {
	t.Parallel()
	s := math.Scalar(1 / stdmath.Sqrt2)
	target, err := SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3, 10)
	if err != nil {
		t.Fatalf("SolidCylinder target: %v", err)
	}
	tool, err := SolidCylinder(math.P3(-6.5, 0, 2), math.V3(s, 0, s), 0.9, 16)
	if err != nil {
		t.Fatalf("SolidCylinder tool: %v", err)
	}
	res, err := Boolean(Difference, target, tool)
	if err != nil {
		t.Fatalf("Boolean(Difference): %v", err)
	}
	assertWatertight(t, res)
}
