// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// curveAt belongs to the curved-assembly helpers, not the tessellator; it stayed in
// kernel/ops when the tessellation family moved out, so its coverage test follows it.
func TestCurveAtBounds(t *testing.T) {
	t.Parallel()
	if curveAt(nil, 0) != nil {
		t.Error("curveAt past the end should be nil")
	}
	c := geom.NewLineSegment(math.P3(0, 0, 0), math.P3(1, 0, 0))
	if curveAt([]geom.Curve3{c}, 0) == nil {
		t.Error("curveAt within range should return the curve")
	}
}
