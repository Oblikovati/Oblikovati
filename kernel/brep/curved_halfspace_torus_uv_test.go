// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// TestTorusUVParamOfInvertsPoint: torusUV.paramOf inverts the torus surface point (seam-shifted) back to its
// (u,v) — the round-trip the spiric imprint sampling relies on (Oblikovati#1406).
func TestTorusUVParamOfInvertsPoint(t *testing.T) {
	t.Parallel()
	tor, _ := geom.NewTorus(math.P3(1, -2, 3), math.V3(0, 0, 1), 5, 2)
	c := torusUV{torus: tor, seamU: 0.7, seamV: 2.1}
	for i := range 8 {
		u := 2 * stdmath.Pi * float64(i) / 8
		for j := range 8 {
			v := 2 * stdmath.Pi * float64(j) / 8
			// absolute surface point, then invert: paramOf reports seam-relative, so add the seams back.
			p := tor.PointAt(u, v)
			uv := c.paramOf(p)
			gu, gv := wrapAngle(float64(uv.X)+c.seamU), wrapAngle(float64(uv.Y)+c.seamV)
			if angGap(gu, u) > 1e-9 || angGap(gv, v) > 1e-9 {
				t.Errorf("paramOf(PointAt(%.3f,%.3f)) → abs(%.6f,%.6f)", u, v, gu, gv)
			}
		}
	}
}

// angGap is the wrapped absolute difference between two angles.
func angGap(a, b float64) float64 {
	d := stdmath.Abs(a - b)
	if d > stdmath.Pi {
		d = 2*stdmath.Pi - d
	}
	return d
}

// TestTorusSpiricSectionCap: an axis-parallel plane between the inner and outer tube radii cuts ONE oval —
// torusSpiricSection returns its two ±1 branches over a centred [v0,v1] (v0<v1, not 2π-shifted), each lying
// on both the torus and the plane (Oblikovati#1406).
func TestTorusSpiricSectionCap(t *testing.T) {
	t.Parallel()
	tor, _ := geom.NewTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 2)
	plane, _ := geom.NewPlane(math.P3(0, 6, 0), math.V3(0, -1, 0)) // |K|/M = 6, between R−r=3 and R+r=7
	section, ok := torusSpiricSection(tor, plane)
	if !ok || len(section) != 2 {
		t.Fatalf("section ok=%v n=%d, want ok + 2 branches", ok, len(section))
	}
	for i, cv := range section {
		sa := cv.(geom.SpiricArc)
		if sa.V0 >= sa.V1 {
			t.Errorf("branch %d range [%.3f,%.3f] not native (V0<V1)", i, sa.V0, sa.V1)
		}
		if stdmath.Abs(sa.V0) > stdmath.Pi {
			t.Errorf("branch %d V0=%.3f not centred in [−π,π]", i, sa.V0)
		}
		for _, tt := range []float64{0, 0.5, 1} {
			p := cv.PointAt(tt)
			if d := stdmath.Abs(float64(plane.Origin.VectorTo(p).Dot(plane.Normal()))); d > 1e-6 {
				t.Errorf("branch %d t=%.1f off the cut plane by %.2e", i, tt, d)
			}
		}
	}
}

// TestTorusSpiricSectionPerpendicularDefers: a plane perpendicular to the axis has no spiric section (it cuts
// two circles, handled analytically), so torusSpiricSection reports ok=false (Oblikovati#1406).
func TestTorusSpiricSectionPerpendicularDefers(t *testing.T) {
	t.Parallel()
	tor, _ := geom.NewTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 2)
	plane, _ := geom.NewPlane(math.P3(0, 0, 1), math.V3(0, 0, 1))
	if _, ok := torusSpiricSection(tor, plane); ok {
		t.Error("perpendicular cut should defer (ok=false), it has no spiric section")
	}
}

// TestSpiricArcOfNativeOrdering: spiricArcOf stores a branch in native tube-angle order (V0<V1) even for a
// reversed loop traversal (t0>t1), so the direction-sensitive spiric mesher charts the same patch either way
// — the fix that made the unified cap match the analytic cap's mesh exactly (Oblikovati#1406).
func TestSpiricArcOfNativeOrdering(t *testing.T) {
	t.Parallel()
	base := geom.SpiricArc{Phi: 0.3, M: 1, K: -6, C: 0, Branch: 1, V0: -1, V1: 1}
	fwd := spiricArcOf(base, 0, 1).(geom.SpiricArc)
	rev := spiricArcOf(base, 1, 0).(geom.SpiricArc) // reversed traversal
	for _, sa := range []geom.SpiricArc{fwd, rev} {
		if sa.V0 >= sa.V1 {
			t.Errorf("spiricArcOf range [%.3f,%.3f] not native (V0<V1)", sa.V0, sa.V1)
		}
	}
	if fwd.V0 != rev.V0 || fwd.V1 != rev.V1 {
		t.Errorf("forward %v and reversed %v sub-ranges differ — both should be the native [−1,1]", fwd, rev)
	}
}
