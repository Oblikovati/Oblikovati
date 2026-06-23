// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// TestPlaneIntersectionLineExported: the exported helper returns the meeting line of two planes
// and errors when they are parallel (#1262).
func TestPlaneIntersectionLineExported(t *testing.T) {
	at, dir, err := PlaneIntersectionLine(sketch.XZPlane(), sketch.XYPlane())
	if err != nil {
		t.Fatalf("XZ∩XY: %v", err)
	}
	// The XZ and XY planes meet along the X axis: every point has y=z=0 and the direction is ±X.
	if !math.IsNearZero(at.Y, 1e-9) || !math.IsNearZero(at.Z, 1e-9) {
		t.Errorf("intersection anchor %v not on the X axis", at)
	}
	if d := dir.AsVector(); math.IsNearZero(d.X, 1e-9) {
		t.Errorf("intersection direction %v not along X", d)
	}
	if _, _, err := PlaneIntersectionLine(sketch.XYPlane(), sketch.XYPlane()); err == nil {
		t.Error("parallel planes should error (no intersection line)")
	}
}

// TestWorkPointByRef: the exported lookup resolves the origin centre and user work points, and
// reports false for an unknown reference.
func TestWorkPointByRef(t *testing.T) {
	g := NewWorkGeometry()
	if w, ok := g.WorkPointByRef(OriginCenter); !ok || w.Name() != "Center Point" {
		t.Fatalf("WorkPointByRef(OriginCenter) = %v, ok=%v; want the Center Point", w, ok)
	}
	if _, ok := g.WorkPointByRef("origin/point/bogus"); ok {
		t.Error("an unknown datum reference should report ok=false")
	}

	// A user work point resolves by its "point/N" reference; an out-of-range index reports false.
	up := g.WorkPoints().AddByPosition(func() math.Point3 { return math.P3(1, 2, 3) })
	if w, ok := g.WorkPointByRef(up.Key()); !ok || w != up {
		t.Errorf("WorkPointByRef(%q) = %v, ok=%v; want the user point", up.Key(), w, ok)
	}
	if _, ok := g.WorkPointByRef("point/999"); ok {
		t.Error("an out-of-range user point index should report ok=false")
	}
}
