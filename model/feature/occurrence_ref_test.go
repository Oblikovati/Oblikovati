// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// TestOccurrenceRefRoundTrip: an occurrence-qualified ref encodes an occurrence path (even with
// names containing slashes or spaces) and its native datum ref, and decodes back exactly (#1857).
func TestOccurrenceRefRoundTrip(t *testing.T) {
	path := []string{"gear:1", "sub/assembly:2", "shaft x:3"}
	ref := EncodeOccurrenceRef(path, "axis/0")
	gotPath, native, ok := ParseOccurrenceRef(ref)
	if !ok || native != "axis/0" || len(gotPath) != 3 {
		t.Fatalf("decode = path %v native %q ok %v", gotPath, native, ok)
	}
	for i := range path {
		if gotPath[i] != path[i] {
			t.Errorf("path[%d] = %q, want %q", i, gotPath[i], path[i])
		}
	}
	if _, _, ok := ParseOccurrenceRef("plane/3"); ok {
		t.Error("a plain datum ref must not decode as an occurrence ref")
	}
	if _, _, ok := ParseOccurrenceRef("occ/notbase64!/plane/3"); ok {
		t.Error("a bad base64 path must not decode")
	}
}

type fakeExternalResolver struct {
	pl sketch.Plane
	o  math.Point3
	d  math.UnitVector3
	pt math.Point3
}

func (f fakeExternalResolver) OccurrencePlane(WorkRef) (sketch.Plane, bool) { return f.pl, true }
func (f fakeExternalResolver) OccurrenceAxisLine(WorkRef) (math.Point3, math.UnitVector3, bool) {
	return f.o, f.d, true
}
func (f fakeExternalResolver) OccurrencePoint(WorkRef) (math.Point3, bool) { return f.pt, true }

// TestExternalResolverInterceptsOccurrenceRefs: with an external resolver installed, the plane/axis/
// point resolvers delegate occurrence-qualified refs to it; without one, an occ ref errors (#1857).
func TestExternalResolverInterceptsOccurrenceRefs(t *testing.T) {
	g := NewWorkGeometry()
	occ := EncodeOccurrenceRef([]string{"sub:1"}, "plane/3")
	if _, err := g.plane(occ); err == nil {
		t.Error("an occurrence ref without an external resolver should error")
	}

	g.SetExternalDatumResolver(fakeExternalResolver{pl: sketch.XYPlane(), o: math.P3(1, 2, 3), d: mustUnit(0, 0, 1), pt: math.P3(4, 5, 6)})
	if pl, err := g.plane(occ); err != nil || !pl.Origin().IsEqualTo(math.P3(0, 0, 0), wtol) {
		t.Errorf("plane occ ref = %v (err %v), want the XY plane", pl.Origin(), err)
	}
	ax, err := g.axis(EncodeOccurrenceRef([]string{"sub:1"}, "axis/0"))
	if err != nil || !ax.Origin().IsEqualTo(math.P3(1, 2, 3), wtol) {
		t.Errorf("axis occ ref = %v (err %v), want origin (1,2,3)", ax, err)
	}
	if !ax.Direction().AsVector().IsEqualTo(math.V3(0, 0, 1), wtol) {
		t.Errorf("axis dir = %v, want +Z", ax.Direction())
	}
	if p, err := g.point(EncodeOccurrenceRef([]string{"sub:1"}, "point/0")); err != nil || !p.IsEqualTo(math.P3(4, 5, 6), wtol) {
		t.Errorf("point occ ref = %v (err %v), want (4,5,6)", p, err)
	}
}
