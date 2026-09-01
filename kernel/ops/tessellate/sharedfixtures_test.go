// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// weTestTorus builds the shared host torus for these tests (fatal on the impossible constructor error).
func weTestTorus(t *testing.T, major, minor float64) geom.Torus {
	t.Helper()
	tor, err := geom.NewTorusWithRef(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), major, minor)
	if err != nil {
		t.Fatalf("host torus (R=%g a=%g): %v", major, minor, err)
	}
	return tor
}

// bridgedWallLoop builds the seam-bridged outer loop of a full-wrap wall between v=vBot and v=vTop:
// the seam's bottom vertex, the top rim all the way round, then the bottom rim back — the shape the
// curved boolean hands the tessellator for a drilled cylinder wall.
func bridgedWallLoop(s geom.Surface, vBot, vTop float64, n int) []math.Point3 {
	step := 2 * stdmath.Pi / float64(n)
	out := []math.Point3{s.PointAt(0, vBot)}
	for k := 0; k <= n; k++ { // top rim, θ: 0 → −2π (closing on its own start)
		out = append(out, s.PointAt(-step*float64(k), vTop))
	}
	for k := n; k > 0; k-- { // bottom rim back, θ: −2π → 0 (exclusive: index 0 already holds it)
		out = append(out, s.PointAt(-step*float64(k), vBot))
	}
	return out
}
