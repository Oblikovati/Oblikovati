// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"math"
	"testing"
)

// TestScaleEntities checks a uniform scale maps positions and radii together and is a
// no-op at factor 1.
func TestScaleEntities(t *testing.T) {
	in := []Entity{
		&Line{Start: [3]float64{1, 2, 0}, End: [3]float64{3, 4, 0}},
		&Circle{Center: [3]float64{2, 0, 0}, Radius: 5},
	}
	out := ScaleEntities(in, 10)
	if l := out[0].(*Line); l.Start != [3]float64{10, 20, 0} || l.End != [3]float64{30, 40, 0} {
		t.Errorf("line not scaled: %+v", l)
	}
	if c := out[1].(*Circle); c.Center != [3]float64{20, 0, 0} || c.Radius != 50 {
		t.Errorf("circle not scaled: %+v", c)
	}
	if same := ScaleEntities(in, 1); &same[0] != &in[0] {
		t.Error("factor 1 should return the input slice unchanged")
	}
}

// TestInsertAffinePlacesGeometry checks an insert transform applies scale, rotation, and
// translation in the right order (scale in block axes, then rotate, then translate).
func TestInsertAffinePlacesGeometry(t *testing.T) {
	in := &Insert{Insertion: [3]float64{10, 5, 0}, Scale: [3]float64{2, 2, 1}, Rotation: math.Pi / 2}
	m := InsertAffine(in)
	// Block point (1,0): scale → (2,0); rotate +90° → (0,2); translate → (10,7).
	got := m.point([3]float64{1, 0, 0})
	if math.Abs(got[0]-10) > 1e-9 || math.Abs(got[1]-7) > 1e-9 {
		t.Errorf("transformed (1,0,0) = %v, want (10,7,0)", got)
	}
}

// TestTransformArcMirror checks a negative (mirroring) scale reverses the CCW sweep so the
// arc still traces the same geometry — the endpoints swap.
func TestTransformArcMirror(t *testing.T) {
	a := &Arc{Center: [3]float64{0, 0, 0}, Radius: 1, StartAngle: 0, EndAngle: math.Pi / 2}
	mirror := Affine{{-1, 0, 0, 0}, {0, 1, 0, 0}, {0, 0, 1, 0}} // mirror across Y axis
	g := transformArc(a, mirror)
	// Start (1,0) → (-1,0) i.e. angle π; end (0,1) → (0,1) i.e. angle π/2. Mirror swaps them.
	if math.Abs(g.StartAngle-math.Pi/2) > 1e-9 || math.Abs(math.Abs(g.EndAngle)-math.Pi) > 1e-9 {
		t.Errorf("mirrored arc angles = (%g,%g), want start π/2, end ±π", g.StartAngle, g.EndAngle)
	}
}

// TestTransformEntityAllTypes covers the per-type affine transform for the curve types not
// already exercised (ellipse, polyline, spline) under a translate+scale.
func TestTransformEntityAllTypes(t *testing.T) {
	m := Affine{{2, 0, 0, 1}, {0, 2, 0, 1}, {0, 0, 2, 0}} // scale 2, translate (1,1)
	el := TransformEntity(&Ellipse{Center: [3]float64{1, 0, 0}, MajorAxis: [3]float64{1, 0, 0}, AxisRatio: 0.5}, m).(*Ellipse)
	if el.Center != [3]float64{3, 1, 0} || el.MajorAxis != [3]float64{2, 0, 0} {
		t.Errorf("ellipse transform: c=%v major=%v", el.Center, el.MajorAxis)
	}
	pl := TransformEntity(&LwPolyline{Points: [][2]float64{{0, 0}, {1, 1}}, Bulges: []float64{0.2, 0}}, m).(*LwPolyline)
	if pl.Points[1] != [2]float64{3, 3} || pl.Bulges[0] != 0.2 {
		t.Errorf("polyline transform: %v bulges %v", pl.Points, pl.Bulges)
	}
	sp := TransformEntity(&Spline{ControlPoints: [][3]float64{{0, 0, 0}, {1, 1, 0}}, FitPoints: [][3]float64{{2, 2, 0}}}, m).(*Spline)
	if sp.ControlPoints[1] != [3]float64{3, 3, 0} || sp.FitPoints[0] != [3]float64{5, 5, 0} {
		t.Errorf("spline transform: ctrl=%v fit=%v", sp.ControlPoints, sp.FitPoints)
	}
}
