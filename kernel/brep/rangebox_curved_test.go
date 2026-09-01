// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/math"
)

// TestSolidCylinderRangeBoxSpansRadius is a regression for the vertex-only
// RangeBox bug: a body's bounding box was built from its vertices alone, so a
// cylinder (whose only vertices are the two seam points) reported a degenerate
// box like {0,3.5,0}..{0,3.5,4} instead of spanning ±radius. That made boolean
// classify() see a clearly-overlapping cylinder as disjoint and skip the cut.
// RangeBox now samples curved edges, so the box must span the full ±r footprint.
func TestSolidCylinderRangeBoxSpansRadius(t *testing.T) {
	t.Parallel()
	const r, h = 3.5, 4.0
	cyl, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), r, h)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	box := cyl.RangeBox()
	if stdmath.Abs(float64(box.Min.X)+r) > 1e-6 || stdmath.Abs(float64(box.Max.X)-r) > 1e-6 ||
		stdmath.Abs(float64(box.Min.Y)+r) > 1e-6 || stdmath.Abs(float64(box.Max.Y)-r) > 1e-6 {
		t.Errorf("cylinder RangeBox X/Y = [%v,%v]x[%v,%v], want ±%g", box.Min.X, box.Max.X, box.Min.Y, box.Max.Y, r)
	}
	if stdmath.Abs(float64(box.Min.Z)) > 1e-6 || stdmath.Abs(float64(box.Max.Z)-h) > 1e-6 {
		t.Errorf("cylinder RangeBox Z = [%v,%v], want [0,%g]", box.Min.Z, box.Max.Z, h)
	}
}
