// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/param"
)

func TestDimension3DDistance(t *testing.T) {
	ps := param.NewParameters()
	dc := NewDimensionConstraints3D(ps)
	a := NewPoint3D(math.P3(0, 0, 0))
	b := NewPoint3D(math.P3(0, 0, 4)) // distance 4

	d, err := dc.AddDistance(a, b, "5 cm")
	if err != nil {
		t.Fatalf("AddDistance: %v", err)
	}
	if r := d.Residuals(); len(r) != 1 || !approx(r[0], -1) { // 4 - 5
		t.Errorf("residual = %v, want [-1]", r)
	}
	if len(d.Variables()) != 6 {
		t.Errorf("3D distance has %d vars, want 6", len(d.Variables()))
	}
	b.SetPosition(math.P3(0, 0, 5))
	if !approx(d.Residuals()[0], 0) || !approx(d.Measured(), 5) {
		t.Errorf("after leveling: residual=%v measured=%v", d.Residuals()[0], d.Measured())
	}
	if d.Parameter() == nil || dc.Count() != 1 || dc.Item(0) != d || len(dc.All()) != 1 {
		t.Error("3D dimension collection tracking wrong")
	}

	d.SetDriven(true)
	if d.Residuals() != nil || d.Variables() != nil || !d.Driven() {
		t.Error("driven 3D dimension still constrains")
	}
}
