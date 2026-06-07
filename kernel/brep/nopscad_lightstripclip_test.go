// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	stdmath "math"
	"oblikovati/math"
	"testing"
)

// TestNopLightStripClipCSG pins the raw CSG footprint behind light_strip_clip: a
// concave linear-extruded polygon equivalent to OpenSCAD's difference of squares.
func TestNopLightStripClipCSG(t *testing.T) {
	const wall = 0.18
	const slot = 1.02
	const aperture = 0.60
	const depth = 1.0
	clipLength := slot + 2*wall
	clipWidth := 0.30 + 2*wall
	innerTop := clipWidth - 2*wall
	points := []math.Point3{
		math.P3(-clipLength/2, -wall, 0), math.P3(clipLength/2, -wall, 0), math.P3(clipLength/2, clipWidth-wall, 0),
		math.P3(aperture/2, clipWidth-wall, 0), math.P3(aperture/2, innerTop, 0), math.P3(slot/2, innerTop, 0),
		math.P3(slot/2, 0, 0), math.P3(-slot/2, 0, 0), math.P3(-slot/2, innerTop, 0), math.P3(-aperture/2, innerTop, 0),
		math.P3(-aperture/2, clipWidth-wall, 0), math.P3(-clipLength/2, clipWidth-wall, 0),
	}

	body := prismBody(points, 0, depth, "light-strip-clip")
	requireValidNopSolid(t, "light_strip_clip", body)
	want := nopPolygonArea(points) * depth
	if got := vol(body); stdmath.Abs(got-want)/want > 1e-3 {
		t.Errorf("light_strip_clip volume = %.6f, want %.6f", got, want)
	}
}
