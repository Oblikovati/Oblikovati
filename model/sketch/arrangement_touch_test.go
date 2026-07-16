// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// TestCurveEndingOnCircleSplitsIt pins #25: a curve that merely ENDS on a circle must still divide
// it, at ANY angle.
//
// Two concentric circles make an annulus; two radial lines joining them must cut it into two
// half-annuli. That worked only when the lines happened to land on one of the circle's 24 sample
// angles. At any other angle the endpoint lies on the TRUE circle while the circle is a chord
// polygon, so the two sat up to a sagitta apart (0.008565·r, far outside arrMergeTol), never shared
// a node, and the line dangled: the annulus stayed whole and the lines came back as OPEN chains.
// Every downstream extrude then found no profile to build (ReadWriteHead, #22).
//
// The angle is the whole point of this test — 7.5° is mid-facet of the 24-gon, the worst case. A
// version of this test written at angle 0 passes even with the bug.
func TestCurveEndingOnCircleSplitsIt(t *testing.T) {
	for _, tc := range []struct {
		name  string
		angle float64
	}{
		{"on a sample angle", 0},
		{"mid-facet (worst case)", stdmath.Pi / 24},
		{"an arbitrary angle", 0.31415},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := annulusCutBy(tc.angle)
			ps := s.Profiles()
			closed, open := 0, 0
			for i := 0; i < ps.Count(); i++ {
				if ps.Item(i).IsClosed() {
					closed++
				} else {
					open++
				}
			}
			// the inner disc + two half-annuli; the radial lines bound cells, so nothing dangles
			if closed != 3 || open != 0 {
				t.Errorf("annulus cut by two radial lines gave %d closed / %d open profiles, want 3 / 0"+
					" — a line ending ON the circle failed to divide it", closed, open)
			}
			for i := 0; i < ps.Count(); i++ {
				p := ps.Item(i)
				if !p.IsClosed() {
					continue
				}
				// each half-annulus is 3*pi/2; the disc is pi. Loose: the polygons are faceted.
				if a := p.Area(); a < 3.0 || a > 4.8 {
					t.Errorf("profile %d area %.4f is neither the disc (~%.3f) nor a half-annulus (~%.3f)",
						i, a, stdmath.Pi, 1.5*stdmath.Pi)
				}
			}
		})
	}
}

// annulusCutBy builds circles r=1 and r=2 with radial lines joining them at ±angle.
func annulusCutBy(angle float64) *Sketch {
	s := NewSketches().Add(XYPlane())
	s.Circles().AddByCenterRadius(math.P2(0, 0), math.Scalar(1))
	s.Circles().AddByCenterRadius(math.P2(0, 0), math.Scalar(2))
	ax, ay := stdmath.Cos(angle), stdmath.Sin(angle)
	a := s.Points().Add(math.P2(math.Scalar(ax), math.Scalar(ay)))
	b := s.Points().Add(math.P2(math.Scalar(2*ax), math.Scalar(2*ay)))
	s.Lines().Add(a, b)
	c := s.Points().Add(math.P2(math.Scalar(-ax), math.Scalar(-ay)))
	d := s.Points().Add(math.P2(math.Scalar(-2*ax), math.Scalar(-2*ay)))
	s.Lines().Add(c, d)
	return s
}
