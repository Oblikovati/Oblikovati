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

// TestWriteFullGraphStructure checks that a written R2000 file carries the complete system
// object graph AutoCAD requires — the nine symbol-table control objects, the standard
// records, the block records and the named-object dictionary — alongside the model-space
// entities, and that the $INSUNITS code round-trips through the header. The whole graph
// decoding without warning is the in-CI proxy for the dwgread "SUCCESS" validation.
func TestWriteFullGraphStructure(t *testing.T) {
	in := []Entity{
		&Line{Start: [3]float64{0, 0, 0}, End: [3]float64{10, 5, 0}},
		&Circle{Center: [3]float64{3, 4, 0}, Radius: 2.5, Normal: [3]float64{0, 0, 1}},
	}
	data, err := Write(&Drawing{Entities: in, Units: 4}) // 4 = millimetres
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	dr, warns, err := Decode(data)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if len(warns) != 0 {
		t.Errorf("unexpected decode warnings: %v", warns)
	}
	if dr.Units != 4 {
		t.Errorf("units round-trip = %d, want 4", dr.Units)
	}
	if len(dr.Entities) != len(in) {
		t.Fatalf("entities round-trip = %d, want %d", len(dr.Entities), len(in))
	}

	counts := tallyObjectTypes(t, data)
	for _, w := range []struct {
		typ  ObjectType
		want int
	}{
		{typeBlockControl, 1}, {typeLayerControl, 1}, {typeStyleControl, 1}, {typeLtypeControl, 1},
		{typeViewControl, 1}, {typeUcsControl, 1}, {typeVportControl, 1}, {typeAppidControl, 1},
		{typeDimstyleControl, 1}, {TypeLayer, 1}, {TypeStyle, 1}, {TypeLtype, 3}, {TypeAppid, 1},
		{TypeVport, 1}, {TypeDimstyle, 1}, {TypeBlockHeader, 2}, {TypeBlock, 2}, {TypeEndblk, 2},
		// NOD + ACAD_GROUP/MLINESTYLE/PLOTSETTINGS/LAYOUT sub-dictionaries.
		{TypeDictionary, 5},
		// Named-object-dictionary chain objects (MLINESTYLE is fixed 0x49; the rest are
		// class-resolved at 500/501/502 in the writer's class order).
		{0x49, 1}, {classDictWDflt, 1}, {classPlaceholder, 1}, {classLayout, 2},
	} {
		if counts[w.typ] != w.want {
			t.Errorf("object type %#x count = %d, want %d", int(w.typ), counts[w.typ], w.want)
		}
	}
}

// tallyObjectTypes walks the written file's object map and counts each object by type.
func tallyObjectTypes(t *testing.T, data []byte) map[ObjectType]int {
	t.Helper()
	h, err := ParseFileHeader(data)
	if err != nil {
		t.Fatalf("ParseFileHeader: %v", err)
	}
	omb, _ := h.ObjectMapBytes(data)
	refs, err := parseObjectMap(omb)
	if err != nil {
		t.Fatalf("parseObjectMap: %v", err)
	}
	counts := map[ObjectType]int{}
	for _, ref := range refs {
		hdr, err := decodeObjectHeader(data, ref, h.Version)
		if err != nil {
			t.Fatalf("object %d header: %v", ref.Handle, err)
		}
		counts[hdr.Type]++
	}
	return counts
}
