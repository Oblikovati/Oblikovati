// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// rimCircleZ returns the target's top-cap rim circle: radius R at height h, axis +z through the origin.
func rimCircleZ(r, h float64) geom.Circle {
	c, _ := geom.NewCircle(math.P3(0, 0, math.Scalar(h)), math.V3(0, 0, 1), r)
	return c
}

func obliqueTool(baseX, r float64) geom.Cylinder {
	s := 1 / stdmath.Sqrt2
	c, _ := geom.NewCylinder(math.P3(math.Scalar(baseX), 0, 2), math.V3(math.Scalar(s), 0, math.Scalar(s)), r)
	return c
}

// TestCapRimCornersTwoOnCrossing: the base=-5.6 tool's exit ellipse crosses the rim, so there are exactly
// two corner points, each on the rim (r=3) AND on the tool wall (dist=0.9), symmetric about y=0.
func TestCapRimCornersTwoOnCrossing(t *testing.T) {
	tool := obliqueTool(-5.6, 0.9)
	rim := rimCircleZ(3, 10)
	corners := capRimCorners(tool, rim)
	if len(corners) != 2 {
		t.Fatalf("rim-crossing tool: got %d corners, want 2", len(corners))
	}
	for _, c := range corners {
		if r := stdmath.Hypot(float64(c.point.X), float64(c.point.Y)); stdmath.Abs(r-3) > 1e-9 {
			t.Errorf("corner not on rim: r=%.12f want 3 (point %v)", r, c.point)
		}
		if d := distToAxis(c.point, tool); stdmath.Abs(d-0.9) > 1e-7 {
			t.Errorf("corner not on tool wall: dist=%.9f want 0.9 (point %v)", d, c.point)
		}
		if stdmath.Abs(float64(c.point.Z)-10) > 1e-9 {
			t.Errorf("corner off cap plane: z=%.12f want 10", c.point.Z)
		}
	}
	// symmetric about y=0: the two corners share x and have opposite y
	if stdmath.Abs(float64(corners[0].point.X-corners[1].point.X)) > 1e-6 ||
		stdmath.Abs(float64(corners[0].point.Y+corners[1].point.Y)) > 1e-6 {
		t.Errorf("corners not symmetric about y=0: %v %v", corners[0].point, corners[1].point)
	}
}

// TestCapRimCornersNoneWhenInside: the base=-6.5 tool (slice 1) exits with the ellipse strictly inside the
// rim, so the tool wall never reaches the rim — zero corners.
func TestCapRimCornersNoneWhenInside(t *testing.T) {
	if n := len(capRimCorners(obliqueTool(-6.5, 0.9), rimCircleZ(3, 10))); n != 0 {
		t.Errorf("interior-exit tool: got %d corners, want 0", n)
	}
}

func distToAxis(p math.Point3, c geom.Cylinder) float64 {
	w := c.Origin.VectorTo(p)
	ax := w.Dot(c.AxisDir.AsVector())
	return float64(w.Sub(c.AxisDir.AsVector().Scale(ax)).Length())
}
