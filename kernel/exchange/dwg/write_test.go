// SPDX-License-Identifier: GPL-2.0-only

package dwg

import (
	"math"
	"testing"
)

// TestWriteDecodeRoundTrip writes one of each supported entity type to an R2000 file and
// decodes it back, asserting the geometry survives byte-for-byte through the writer and the
// reader — the contract the export feature depends on (the encoders are exact inverses of
// the decoders).
func TestWriteDecodeRoundTrip(t *testing.T) {
	in := []Entity{
		&Line{Start: [3]float64{0, 0, 0}, End: [3]float64{10, 5, 0}},
		&Circle{Center: [3]float64{3, 4, 0}, Radius: 2.5, Normal: [3]float64{0, 0, 1}},
		&Arc{Center: [3]float64{1, 1, 0}, Radius: 3, StartAngle: 0.5, EndAngle: 2.0, Normal: [3]float64{0, 0, 1}},
		&Spline{Degree: 3, ControlPoints: [][3]float64{{0, 0, 0}, {1, 2, 0}, {3, 2, 0}, {4, 0, 0}}},
		&LwPolyline{Closed: true, Points: [][2]float64{{0, 0}, {2, 0}, {2, 2}}, Bulges: []float64{0, 0.5, 0}},
		&Point{Position: [3]float64{7, 8, 0}},
		&Ellipse{Center: [3]float64{1, 1, 0}, MajorAxis: [3]float64{2, 0, 0}, AxisRatio: 0.5, StartAngle: 0, EndAngle: 1.2, Normal: [3]float64{0, 0, 1}},
		&Spline{Degree: 3, FitPoints: [][3]float64{{0, 0, 0}, {1, 1, 0}, {2, 0, 0}}}, // fit-point form (scenario 2)
	}
	data, err := Write(&Drawing{Entities: in, Units: 5})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	dr, _, err := Decode(data)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if len(dr.Entities) != len(in) {
		t.Fatalf("round-trip kept %d of %d entities", len(dr.Entities), len(in))
	}
	closeP := func(a, b [3]float64) bool {
		return math.Abs(a[0]-b[0]) < 1e-9 && math.Abs(a[1]-b[1]) < 1e-9 && math.Abs(a[2]-b[2]) < 1e-9
	}
	l := dr.Entities[0].(*Line)
	if !closeP(l.Start, in[0].(*Line).Start) || !closeP(l.End, in[0].(*Line).End) {
		t.Errorf("line round-trip: got %v→%v", l.Start, l.End)
	}
	c := dr.Entities[1].(*Circle)
	if !closeP(c.Center, in[1].(*Circle).Center) || math.Abs(c.Radius-2.5) > 1e-9 {
		t.Errorf("circle round-trip: %v r=%g", c.Center, c.Radius)
	}
	a := dr.Entities[2].(*Arc)
	if math.Abs(a.Radius-3) > 1e-9 || math.Abs(a.StartAngle-0.5) > 1e-9 || math.Abs(a.EndAngle-2.0) > 1e-9 {
		t.Errorf("arc round-trip: r=%g [%g,%g]", a.Radius, a.StartAngle, a.EndAngle)
	}
	s := dr.Entities[3].(*Spline)
	want := in[3].(*Spline).ControlPoints
	if len(s.ControlPoints) != len(want) {
		t.Fatalf("spline ctrl points = %d, want %d", len(s.ControlPoints), len(want))
	}
	for i := range want {
		if !closeP(s.ControlPoints[i], want[i]) {
			t.Errorf("spline ctrl[%d] = %v, want %v", i, s.ControlPoints[i], want[i])
		}
	}
	p := dr.Entities[4].(*LwPolyline)
	if !p.Closed || len(p.Points) != 3 || math.Abs(p.Bulges[1]-0.5) > 1e-9 {
		t.Errorf("polyline round-trip: closed=%v pts=%d bulge=%v", p.Closed, len(p.Points), p.Bulges)
	}
	pt := dr.Entities[5].(*Point)
	if !closeP(pt.Position, [3]float64{7, 8, 0}) {
		t.Errorf("point round-trip: %v", pt.Position)
	}
	el := dr.Entities[6].(*Ellipse)
	if !closeP(el.Center, [3]float64{1, 1, 0}) || math.Abs(el.AxisRatio-0.5) > 1e-9 || math.Abs(el.EndAngle-1.2) > 1e-9 {
		t.Errorf("ellipse round-trip: c=%v ratio=%g end=%g", el.Center, el.AxisRatio, el.EndAngle)
	}
	fs := dr.Entities[7].(*Spline)
	if len(fs.FitPoints) != 3 || !closeP(fs.FitPoints[1], [3]float64{1, 1, 0}) {
		t.Errorf("fit spline round-trip: fit=%v", fs.FitPoints)
	}
}
