// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// cylinderRuledUV builds a minimal radius-R cylinder ruledUV about the z-axis (base at the origin) for the
// (u,v)-arrangement unit tests: point3(u,v) = (R·cos u, R·sin u, v), so paramOf must invert it exactly.
func cylinderRuledUV(r, vMin, vMax float64) ruledUV {
	axis, ref := math.V3(0, 0, 1), math.V3(1, 0, 0)
	return ruledUV{
		base: math.P3(0, 0, 0), axis: axis, ref: ref, binor: axis.Cross(ref),
		radSlope: 0, radConst: r,
		band: coneSideBand_{vMin: vMin, vMax: vMax},
	}
}

// TestRuledParamOfInvertsPoint3: paramOf is the exact inverse of point3 over the band — a sampled (u,v)
// round-trips through the surface point and back to itself (Oblikovati#1405).
func TestRuledParamOfInvertsPoint3(t *testing.T) {
	c := cylinderRuledUV(3, -5, 5)
	for i := 0; i < 16; i++ {
		u := 2 * stdmath.Pi * float64(i) / 16
		for _, v := range []float64{-4, -1, 0, 2.5, 4} {
			uv := c.paramOf(c.point3(u, v))
			du := stdmath.Abs(uv.X - u)
			if du > stdmath.Pi { // both near the seam (0 vs 2π) — compare wrapped
				du = 2*stdmath.Pi - du
			}
			if du > 1e-9 || stdmath.Abs(uv.Y-v) > 1e-9 {
				t.Errorf("paramOf(point3(%.4f,%.4f)) = (%.6f,%.6f), want (%.4f,%.4f)", u, v, uv.X, uv.Y, u, v)
			}
		}
	}
}

// TestSampleImprintUVOnRimCircle: sampling the bottom-rim circle (v=vMin) as an imprint yields tagged
// (u,v) segments that all sit at v=vMin, carry the circle as their source curve, and whose endpoint
// parameters map back (via the circle) to the segment's (u,v) — the round-trip the boundary re-emission
// relies on (Oblikovati#1405).
func TestSampleImprintUVOnRimCircle(t *testing.T) {
	const r, vMin = 3.0, -5.0
	c := cylinderRuledUV(r, vMin, 5)
	rim, _ := geom.NewCircle(math.P3(0, 0, vMin), math.V3(0, 0, 1), r) // the bottom rim, z=vMin
	segs := c.sampleImprintUV(rim, 0, 1)
	if len(segs) != imprintSampleCount {
		t.Fatalf("sampled %d segments, want %d", len(segs), imprintSampleCount)
	}
	for i, s := range segs {
		if stdmath.Abs(s.a.Y-vMin) > 1e-9 || stdmath.Abs(s.b.Y-vMin) > 1e-9 {
			t.Errorf("segment %d not on the rim v=%.1f: a.v=%.6f b.v=%.6f", i, vMin, s.a.Y, s.b.Y)
		}
		if s.curve != rim {
			t.Errorf("segment %d lost its source curve tag", i)
		}
		// The tagged parameter at b must reproduce the segment's (u,v) endpoint through the surface.
		if got := c.paramOf(s.curve.PointAt(s.tB)); stdmath.Abs(got.Y-s.b.Y) > 1e-9 {
			t.Errorf("segment %d param tB=%.4f maps to v=%.6f, want %.6f", i, s.tB, got.Y, s.b.Y)
		}
	}
}
