// SPDX-License-Identifier: GPL-2.0-only
package ops

import (
	"math"
	"testing"

	m "oblikovati.org/math"
)

// sampleEllipse returns n points of the T6 base ellipse (a=15,b=10) in the z=0 host plane.
func sampleEllipse(n int) []m.Point2 {
	pts := make([]m.Point2, n)
	for i := 0; i < n; i++ {
		t := 2 * math.Pi * float64(i) / float64(n)
		pts[i] = m.P2(15*math.Cos(t), 10*math.Sin(t))
	}
	return pts
}

func TestRimCrossingsT6(t *testing.T) {
	rim := sampleEllipse(720)
	boundary := boundaryLine2{origin: m.P2(0, -7), dir: m.V2(1, 0)} // the receded boundary y=-7
	res := ResolutionForSize(50)
	cs := rimCrossings(rim, boundary, res)
	if len(cs) != 2 {
		t.Fatalf("ellipse ∩ y=-7 should have 2 crossings, got %d", len(cs))
	}
	xs := []float64{cs[0].P.X, cs[1].P.X}
	for _, x := range xs {
		if math.Abs(math.Abs(x)-10.712142) > 0.05 {
			t.Errorf("crossing x=%.4f, want ±10.712142", x)
		}
	}
}

func TestObstacleNodesTangentRejected(t *testing.T) {
	rim := sampleEllipse(720)
	boundary := boundaryLine2{origin: m.P2(0, -10), dir: m.V2(1, 0)} // tangent to the ellipse bottom
	res := ResolutionForSize(50)
	if _, ok := obstacleNodes(rim, boundary, res); ok {
		t.Errorf("a rim tangent to the boundary must be rejected (no dip, no patch)")
	}
}
