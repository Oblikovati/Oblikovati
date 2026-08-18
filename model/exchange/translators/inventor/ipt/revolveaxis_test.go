// SPDX-License-Identifier: GPL-2.0-only

package ipt

import (
	"math"
	"testing"
)

// TestRevolveAxis2DDecodesFixtureAxis: the Revolution feature's axis SketchLine3D projects into the
// profile plane as an axis-aligned centreline. The generated revolve fixtures turn a profile about a
// horizontal axis, so dir is along X and its Y component is ~0.
func TestRevolveAxis2DDecodesFixtureAxis(t *testing.T) {
	for _, f := range []string{"16_revolve.ipt", "24_revolve_270.ipt"} {
		d := openDoc(t, f)
		_, _, dx, dy, ok := RevolveAxis2D(d)
		if !ok {
			t.Errorf("%s: expected a decodable 2D axis", f)
			continue
		}
		if math.Hypot(dx, dy) < 1e-6 {
			t.Errorf("%s: axis direction is degenerate", f)
		}
		if math.Abs(dy) > 1e-6 {
			t.Errorf("%s: fixture axis should be horizontal, got dir=(%.3f,%.3f)", f, dx, dy)
		}
	}
}
