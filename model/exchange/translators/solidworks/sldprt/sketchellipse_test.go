// SPDX-License-Identifier: GPL-2.0-only

package sldprt

import (
	"math"
	"testing"
)

// TestEllipseDecode checks an ellipse decodes to its centre, major-axis direction and semi-axes: the
// generated part is centred at the origin with a 30 mm horizontal major axis and a 20 mm vertical
// minor axis.
func TestEllipseDecode(t *testing.T) {
	d, err := Open(readTestdata(t, "ellipse_fmtb.sldprt"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	es := d.Sketches()[0].Ellipses
	if len(es) != 1 {
		t.Fatalf("got %d ellipses, want 1", len(es))
	}
	e := es[0]
	if math.Abs(e.Center.X) > 1e-9 || math.Abs(e.Center.Y) > 1e-9 {
		t.Errorf("centre = %+v, want (0,0)", e.Center)
	}
	if math.Abs(e.MajorRadius-0.03) > 1e-9 || math.Abs(e.MinorRadius-0.02) > 1e-9 {
		t.Errorf("radii = %g/%g, want 0.03/0.02", e.MajorRadius, e.MinorRadius)
	}
	if math.Abs(math.Abs(e.MajorX)-1) > 1e-9 || math.Abs(e.MajorY) > 1e-9 {
		t.Errorf("major-axis direction = (%g,%g), want (±1,0)", e.MajorX, e.MajorY)
	}
}

// TestEllipseFromPointsRejects verifies a non-ellipse point set (no symmetric perpendicular pairs) is
// rejected rather than mis-read as an ellipse.
func TestEllipseFromPointsRejects(t *testing.T) {
	pts := []Point{{0, 0}, {0.01, 0}, {0.02, 0.03}} // a triangle's corners, not an ellipse cross
	if _, ok := ellipseFromPoints(pts); ok {
		t.Error("ellipseFromPoints accepted a non-ellipse point set")
	}
}
