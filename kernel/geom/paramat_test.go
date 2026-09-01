// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"testing"

	"oblikovati.org/math"
)

// TestParamAtInvertsPointAt checks ParamAt is a right inverse of PointAt in point
// space: PointAt(ParamAt(PointAt(u,v))) == PointAt(u,v). Comparing points (not
// parameters) sidesteps periodic-wrap/branch ambiguity. Reuses the shared
// sampleSurfaces fixture (one interior (u,v) per surface type).
func TestParamAtInvertsPointAt(t *testing.T) {
	t.Parallel()
	for _, c := range sampleSurfaces(t) {
		want := c.s.PointAt(c.u, c.v)
		ru, rv := c.s.ParamAt(want)
		if got := c.s.PointAt(ru, rv); got.DistanceTo(want) > 1e-6 {
			t.Errorf("%s ParamAt round-trip at (%g,%g): got %v, want %v", c.name, c.u, c.v, got, want)
		}
	}
}

// TestParamAtFootOfOffSurfacePoint checks the metric-foot property where it holds
// exactly — plane/cylinder/sphere, whose normal is a parameter-frame direction: a
// point pushed off along the normal inverts to the foot directly beneath it. (Cone
// and torus give a frame projection that is not the exact metric foot off-surface;
// see the interface doc — irrelevant to tessellation, which inverts on-surface points.)
func TestParamAtFootOfOffSurfacePoint(t *testing.T) {
	t.Parallel()
	metricFoot := map[string]bool{"plane": true, "cylinder": true, "sphere": true}
	for _, c := range sampleSurfaces(t) {
		if !metricFoot[c.name] {
			continue
		}
		foot := c.s.PointAt(c.u, c.v)
		n := c.s.NormalAt(c.u, c.v)
		off := foot.TranslateBy(n.Scale(0.25))
		ru, rv := c.s.ParamAt(off)
		if got := c.s.PointAt(ru, rv); got.DistanceTo(foot) > 1e-6 {
			t.Errorf("%s ParamAt foot at (%g,%g): got %v, want %v", c.name, c.u, c.v, got, foot)
		}
	}
}

// TestParamAtAnalyticAngles checks the closed-form angular inversion is exact (not
// just point-consistent) for the analytic surfaces away from the seam.
func TestParamAtAnalyticAngles(t *testing.T) {
	t.Parallel()
	cyl, err := NewCylinder(math.P3(1, 2, 3), math.V3(0, 0, 1), 2)
	must(t, err)
	u, v := cyl.ParamAt(cyl.PointAt(1.0, 4.0))
	if !nearAngle(u, 1.0) || !near(v, 4.0) {
		t.Errorf("cylinder ParamAt = (%g,%g), want (1,4)", u, v)
	}
}

func near(a, b float64) bool      { return a-b < 1e-9 && b-a < 1e-9 }
func nearAngle(a, b float64) bool { return near(a, b) }

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
