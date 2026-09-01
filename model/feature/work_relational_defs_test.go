// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"reflect"
	"testing"

	"oblikovati.org/math"
)

// TestRelationalWorkFeaturesRoundTrip marshals a part holding every new relational axis/point kind
// and restores it, asserting the recipe is stable — exercising each definition's kindName/refs and
// the .obk encode/decode codecs (#1840, #1842).
func TestRelationalWorkFeaturesRoundTrip(t *testing.T) {
	t.Parallel()
	g := NewWorkGeometry()
	pt := g.WorkPoints().AddByPosition(func() math.Point3 { return math.P3(0, 0, 5) })
	g.WorkAxes().AddByPointAndPlane(pt.Key(), OriginXYPlane)
	g.WorkAxes().AddByLineAndPoint(OriginXAxis, pt.Key())
	g.WorkAxes().AddByLineAndPlane(OriginXAxis, OriginXYPlane)
	g.WorkPoints().AddByPoint(pt.Key())
	g.WorkPoints().AddByTwoLines(OriginXAxis, OriginYAxis)
	g.WorkPoints().AddByThreePlanes(OriginXYPlane, OriginXZPlane, OriginYZPlane)

	data, err := MarshalWork(g)
	if err != nil {
		t.Fatalf("MarshalWork: %v", err)
	}
	restored := NewWorkGeometry()
	if err := ApplyWork(restored, data); err != nil {
		t.Fatalf("ApplyWork: %v", err)
	}
	again, err := MarshalWork(restored)
	if err != nil {
		t.Fatalf("re-MarshalWork: %v", err)
	}
	if !reflect.DeepEqual(data, again) {
		t.Errorf("relational work features did not round-trip:\n first=%+v\n again=%+v", data, again)
	}
	// The restored kinds and health match the originals.
	if a := restored.WorkAxes(); a.Item(a.Count()-1).Kind() != "line-and-plane" {
		t.Errorf("last restored axis kind = %q, want line-and-plane", a.Item(a.Count()-1).Kind())
	}
	if p := restored.WorkPoints(); p.Item(p.Count()-1).Kind() != "three-planes" || !p.Item(p.Count()-1).Health().OK() {
		t.Errorf("last restored point = kind %q healthy %v, want three-planes healthy", p.Item(p.Count()-1).Kind(), p.Item(p.Count()-1).Health().OK())
	}
}

// Relational datum-axis (#1840) and datum-point (#1842) constructors: analytic checks that each
// builds the expected geometry and reports degenerate input as unhealthy.

// TestRelationalUnresolvedRefsAreUnhealthy: every relational constructor goes unhealthy when a
// reference cannot resolve, exercising each definition's ref-resolution error path.
func TestRelationalUnresolvedRefsAreUnhealthy(t *testing.T) {
	t.Parallel()
	const bad = WorkRef("plane/999") // resolves as no axis / no plane / no point
	build := []struct {
		name    string
		healthy func(*WorkGeometry) bool
	}{
		{"axis point-and-plane", func(g *WorkGeometry) bool { return g.WorkAxes().AddByPointAndPlane(bad, OriginXYPlane).Health().OK() }},
		{"axis point-and-plane bad plane", func(g *WorkGeometry) bool {
			return g.WorkAxes().AddByPointAndPlane(OriginCenter, bad).Health().OK()
		}},
		{"axis line-and-point", func(g *WorkGeometry) bool { return g.WorkAxes().AddByLineAndPoint(bad, OriginCenter).Health().OK() }},
		{"axis line-and-point bad point", func(g *WorkGeometry) bool {
			return g.WorkAxes().AddByLineAndPoint(OriginXAxis, bad).Health().OK()
		}},
		{"axis line-and-plane", func(g *WorkGeometry) bool { return g.WorkAxes().AddByLineAndPlane(bad, OriginXYPlane).Health().OK() }},
		{"axis line-and-plane bad plane", func(g *WorkGeometry) bool {
			return g.WorkAxes().AddByLineAndPlane(OriginXAxis, bad).Health().OK()
		}},
		{"point on-point", func(g *WorkGeometry) bool { return g.WorkPoints().AddByPoint(bad).Health().OK() }},
		{"point two-lines", func(g *WorkGeometry) bool { return g.WorkPoints().AddByTwoLines(bad, OriginYAxis).Health().OK() }},
		{"point two-lines second", func(g *WorkGeometry) bool {
			return g.WorkPoints().AddByTwoLines(OriginXAxis, bad).Health().OK()
		}},
		{"point three-planes", func(g *WorkGeometry) bool {
			return g.WorkPoints().AddByThreePlanes(bad, OriginXZPlane, OriginYZPlane).Health().OK()
		}},
		{"point three-planes second", func(g *WorkGeometry) bool {
			return g.WorkPoints().AddByThreePlanes(OriginXYPlane, bad, OriginYZPlane).Health().OK()
		}},
		{"point three-planes third", func(g *WorkGeometry) bool {
			return g.WorkPoints().AddByThreePlanes(OriginXYPlane, OriginXZPlane, bad).Health().OK()
		}},
	}
	for _, b := range build {
		if b.healthy(NewWorkGeometry()) {
			t.Errorf("%s with an unresolvable reference should be unhealthy", b.name)
		}
	}
}

func TestAxisPointAndPlane(t *testing.T) {
	t.Parallel()
	g := NewWorkGeometry()
	pt := g.WorkPoints().AddByPosition(func() math.Point3 { return math.P3(0, 0, 5) })
	wa := g.WorkAxes().AddByPointAndPlane(pt.Key(), OriginXYPlane)
	if !wa.Health().OK() {
		t.Fatalf("point-and-plane axis sick: %+v", wa.Health())
	}
	if !wa.Origin().IsEqualTo(math.P3(0, 0, 5), wtol) {
		t.Errorf("origin = %v, want (0,0,5)", wa.Origin())
	}
	if !wa.Direction().AsVector().IsParallelTo(math.V3(0, 0, 1), wtol) {
		t.Errorf("direction = %v, want +Z (the XY normal)", wa.Direction())
	}
}

func TestAxisLineAndPoint(t *testing.T) {
	t.Parallel()
	g := NewWorkGeometry()
	pt := g.WorkPoints().AddByPosition(func() math.Point3 { return math.P3(0, 5, 0) })
	wa := g.WorkAxes().AddByLineAndPoint(OriginXAxis, pt.Key())
	if !wa.Health().OK() {
		t.Fatalf("line-and-point axis sick: %+v", wa.Health())
	}
	if !wa.Origin().IsEqualTo(math.P3(0, 5, 0), wtol) || !wa.Direction().AsVector().IsParallelTo(math.V3(1, 0, 0), wtol) {
		t.Errorf("axis = %v dir %v, want through (0,5,0) parallel to X", wa.Origin(), wa.Direction())
	}
}

func TestAxisLineAndPlane(t *testing.T) {
	t.Parallel()
	g := NewWorkGeometry()
	// A grounded line at 45° in the XZ plane, projected onto XY, becomes the X axis.
	diag, _ := math.NewUnitVector3(1, 0, 1)
	line := g.WorkAxes().AddByLine(math.P3(0, 0, 2), diag)
	wa := g.WorkAxes().AddByLineAndPlane(line.Key(), OriginXYPlane)
	if !wa.Health().OK() {
		t.Fatalf("line-and-plane axis sick: %+v", wa.Health())
	}
	if !wa.Direction().AsVector().IsParallelTo(math.V3(1, 0, 0), wtol) {
		t.Errorf("projected direction = %v, want +X", wa.Direction())
	}
	if o := wa.Origin(); o.Z > wtol || o.Z < -wtol {
		t.Errorf("projected origin Z = %v, want on the XY plane (0)", o.Z)
	}
}

func TestAxisLineAndPlanePerpendicularIsDegenerate(t *testing.T) {
	t.Parallel()
	g := NewWorkGeometry()
	line := g.WorkAxes().AddByLine(math.P3(0, 0, 3), mustUnit(0, 0, 1)) // ⟂ to XY
	wa := g.WorkAxes().AddByLineAndPlane(line.Key(), OriginXYPlane)
	if wa.Health().OK() {
		t.Error("a line perpendicular to the plane should be degenerate")
	}
}

func TestPointByPoint(t *testing.T) {
	t.Parallel()
	g := NewWorkGeometry()
	src := g.WorkPoints().AddByPosition(func() math.Point3 { return math.P3(1, 2, 3) })
	wp := g.WorkPoints().AddByPoint(src.Key())
	if !wp.Point().IsEqualTo(math.P3(1, 2, 3), wtol) {
		t.Errorf("point = %v, want (1,2,3)", wp.Point())
	}
}

func TestPointTwoLinesIntersectAtOrigin(t *testing.T) {
	t.Parallel()
	g := NewWorkGeometry()
	wp := g.WorkPoints().AddByTwoLines(OriginXAxis, OriginYAxis)
	if !wp.Health().OK() {
		t.Fatalf("two-lines point sick: %+v", wp.Health())
	}
	if !wp.Point().IsEqualTo(math.P3(0, 0, 0), wtol) {
		t.Errorf("X∩Y = %v, want the origin", wp.Point())
	}
}

func TestPointTwoLinesSkewIsDegenerate(t *testing.T) {
	t.Parallel()
	g := NewWorkGeometry()
	// The X axis and a Y-parallel line lifted to z=5 never meet.
	skew := g.WorkAxes().AddByLine(math.P3(0, 0, 5), mustUnit(0, 1, 0))
	wp := g.WorkPoints().AddByTwoLines(OriginXAxis, skew.Key())
	if wp.Health().OK() {
		t.Error("skew lines should be degenerate")
	}
}

func TestPointThreePlanesAtOrigin(t *testing.T) {
	t.Parallel()
	g := NewWorkGeometry()
	wp := g.WorkPoints().AddByThreePlanes(OriginXYPlane, OriginXZPlane, OriginYZPlane)
	if !wp.Health().OK() {
		t.Fatalf("three-planes point sick: %+v", wp.Health())
	}
	if !wp.Point().IsEqualTo(math.P3(0, 0, 0), wtol) {
		t.Errorf("XY∩XZ∩YZ = %v, want the origin", wp.Point())
	}
}

func TestPointThreePlanesParallelIsDegenerate(t *testing.T) {
	t.Parallel()
	g := NewWorkGeometry()
	off := g.WorkPlanes().AddByPlaneAndOffset(OriginXYPlane, func() float64 { return 5 }) // parallel to XY
	wp := g.WorkPoints().AddByThreePlanes(OriginXYPlane, off.Key(), OriginXZPlane)
	if wp.Health().OK() {
		t.Error("two parallel planes should make the three-plane point degenerate")
	}
}
