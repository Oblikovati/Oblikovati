// SPDX-License-Identifier: GPL-2.0-only

package drawing

import "testing"

// TestEntityIdentity covers the EntityHandle/Kind accessors on every entity type, and that
// Kind.String yields the canonical (DXF) entity name.
func TestEntityIdentity(t *testing.T) {
	cases := []struct {
		e    Entity
		kind Kind
		name string
	}{
		{&Line{Handle: 1}, KindLine, "LINE"},
		{&Circle{Handle: 2}, KindCircle, "CIRCLE"},
		{&Arc{Handle: 3}, KindArc, "ARC"},
		{&Point{Handle: 4}, KindPoint, "POINT"},
		{&Ellipse{Handle: 5}, KindEllipse, "ELLIPSE"},
		{&LwPolyline{Handle: 6}, KindLwPolyline, "LWPOLYLINE"},
		{&Spline{Handle: 7}, KindSpline, "SPLINE"},
		{&Insert{Handle: 8}, KindInsert, "INSERT"},
	}
	for i, c := range cases {
		if c.e.Kind() != c.kind {
			t.Errorf("case %d kind = %v, want %v", i, c.e.Kind(), c.kind)
		}
		if c.e.Kind().String() != c.name {
			t.Errorf("case %d name = %q, want %q", i, c.e.Kind().String(), c.name)
		}
		if c.e.EntityHandle() != uint64(i+1) {
			t.Errorf("case %d handle = %d, want %d", i, c.e.EntityHandle(), i+1)
		}
	}
}

// TestKindStringUnknown maps an out-of-range kind to "UNKNOWN".
func TestKindStringUnknown(t *testing.T) {
	if got := Kind(999).String(); got != "UNKNOWN" {
		t.Errorf("Kind(999).String() = %q, want UNKNOWN", got)
	}
}

// TestPlanar checks a flat drawing reports its elevation and a drawing with off-plane
// geometry is non-planar.
func TestPlanar(t *testing.T) {
	flat := &Drawing{Entities: []Entity{
		&Line{Start: [3]float64{0, 0, 2}, End: [3]float64{1, 1, 2}},
		&Circle{Center: [3]float64{3, 3, 2}},
	}}
	if z, ok := flat.Planar(1e-9); !ok || z != 2 {
		t.Errorf("flat.Planar = (%g, %v), want (2, true)", z, ok)
	}
	bent := &Drawing{Entities: []Entity{
		&Point{Position: [3]float64{0, 0, 0}},
		&Point{Position: [3]float64{0, 0, 5}},
	}}
	if _, ok := bent.Planar(1e-9); ok {
		t.Error("bent.Planar = true, want false")
	}
	if _, ok := (&Drawing{}).Planar(1e-9); !ok {
		t.Error("empty drawing should be planar")
	}
}

// TestPlanarTolueratesOutliers is the #1549 regression: a flat survey map carrying a few
// off-sheet/misdecoded entities (here 18 coplanar at Z=2, 2 strays at ±1e7) must still be
// classified planar at the bulk elevation, not flipped onto the 3D path by the outliers.
func TestPlanarTolueratesOutliers(t *testing.T) {
	ents := make([]Entity, 0, 20)
	for i := 0; i < 18; i++ {
		ents = append(ents, &Line{Start: [3]float64{0, 0, 2}, End: [3]float64{1, 1, 2}})
	}
	ents = append(ents, &Point{Position: [3]float64{0, 0, 1e7}}, &Point{Position: [3]float64{0, 0, -1e7}})
	d := &Drawing{Entities: ents}
	if z, ok := d.Planar(1e-6); !ok || z != 2 {
		t.Errorf("mostly-flat drawing with outliers: Planar = (%g, %v), want (2, true)", z, ok)
	}

	// A genuinely 3D drawing (Z varies across most entities) stays non-planar.
	d3 := &Drawing{Entities: []Entity{
		&Point{Position: [3]float64{0, 0, 0}},
		&Point{Position: [3]float64{0, 0, 10}},
		&Point{Position: [3]float64{0, 0, 20}},
		&Point{Position: [3]float64{0, 0, 30}},
	}}
	if _, ok := d3.Planar(1e-6); ok {
		t.Error("genuinely 3D drawing classified planar")
	}
}

// TestMetersPerUnit covers a known code, unitless, and an unsupported code.
func TestMetersPerUnit(t *testing.T) {
	if m, ok := MetersPerUnit(INSMillimetres); !ok || m != 0.001 {
		t.Errorf("MetersPerUnit(mm) = (%g, %v), want (0.001, true)", m, ok)
	}
	if _, ok := MetersPerUnit(INSUnitless); ok {
		t.Error("unitless should report ok=false")
	}
	if _, ok := MetersPerUnit(99); ok {
		t.Error("unsupported code should report ok=false")
	}
}
