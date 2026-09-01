// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// TestRevolvedFaceAxisFromCylinder: the axis of a cylindrical face is its axis of revolution (#1840).
func TestRevolvedFaceAxisFromCylinder(t *testing.T) {
	t.Parallel()
	g := NewWorkGeometry()
	body, key := faceBody(t, mustCylinder(t)) // axis +Z at origin, radius 2
	wa := g.WorkAxes().AddByRevolvedFace(FaceRef(key))
	g.Recompute([]*topo.Body{body})
	if !wa.Health().OK() {
		t.Fatalf("revolved-face axis sick: %+v", wa.Health())
	}
	if !wa.Direction().AsVector().IsParallelTo(math.V3(0, 0, 1), wtol) {
		t.Errorf("axis direction = %v, want +Z", wa.Direction())
	}
	if !wa.Origin().IsEqualTo(math.P3(0, 0, 0), wtol) {
		t.Errorf("axis origin = %v, want the origin", wa.Origin())
	}
	if wa.Kind() != "revolved-face" {
		t.Errorf("kind = %q, want revolved-face", wa.Kind())
	}
}

// TestRevolvedFaceAxisConeAndTorus: cone and torus faces also yield their axis of revolution (#1840).
func TestRevolvedFaceAxisConeAndTorus(t *testing.T) {
	t.Parallel()
	cone, err := geom.NewCone(math.P3(0, 0, 1), math.V3(0, 0, 1), 0.5)
	if err != nil {
		t.Fatal(err)
	}
	o, d, err := revolvedFaceAxis(cone)
	if err != nil || !o.IsEqualTo(math.P3(0, 0, 1), wtol) || !d.AsVector().IsParallelTo(math.V3(0, 0, 1), wtol) {
		t.Errorf("cone axis = %v %v err %v, want apex (0,0,1) dir +Z", o, d, err)
	}
	tor, err := geom.NewTorus(math.P3(1, 2, 3), math.V3(0, 1, 0), 5, 1)
	if err != nil {
		t.Fatal(err)
	}
	o, d, err = revolvedFaceAxis(tor)
	if err != nil || !o.IsEqualTo(math.P3(1, 2, 3), wtol) || !d.AsVector().IsParallelTo(math.V3(0, 1, 0), wtol) {
		t.Errorf("torus axis = %v %v err %v, want centre (1,2,3) dir +Y", o, d, err)
	}
}

// TestRevolvedFaceAxisRejectsSphere: a sphere face has no axis of revolution (#1840).
func TestRevolvedFaceAxisRejectsSphere(t *testing.T) {
	t.Parallel()
	sph, _ := geom.NewSphere(math.P3(0, 0, 0), 2)
	if _, _, err := revolvedFaceAxis(sph); err == nil {
		t.Error("a sphere face should have no axis of revolution")
	}
}

// TestFaceCenterFromSphere: the centre of a spherical face is the sphere centre (#1842).
func TestFaceCenterFromSphere(t *testing.T) {
	t.Parallel()
	g := NewWorkGeometry()
	sph, _ := geom.NewSphere(math.P3(1, 2, 3), 2)
	body, key := faceBody(t, sph)
	wp := g.WorkPoints().AddByFaceCenter(FaceRef(key))
	g.Recompute([]*topo.Body{body})
	if !wp.Health().OK() {
		t.Fatalf("face-center point sick: %+v", wp.Health())
	}
	if !wp.Point().IsEqualTo(math.P3(1, 2, 3), wtol) {
		t.Errorf("sphere centre = %v, want (1,2,3)", wp.Point())
	}
	if wp.Kind() != "face-center" {
		t.Errorf("kind = %q, want face-center", wp.Kind())
	}
}

// TestFaceCenterTorusAndRejectsCylinder: a torus face yields its centre; a cylinder has none (#1842).
func TestFaceCenterTorusAndRejectsCylinder(t *testing.T) {
	t.Parallel()
	tor, _ := geom.NewTorus(math.P3(4, 5, 6), math.V3(0, 0, 1), 5, 1)
	c, err := faceCenter(tor)
	if err != nil || !c.IsEqualTo(math.P3(4, 5, 6), wtol) {
		t.Errorf("torus centre = %v err %v, want (4,5,6)", c, err)
	}
	if _, err := faceCenter(mustCylinder(t)); err == nil {
		t.Error("a cylinder face should have no centre point")
	}
}

// TestRevolvedFaceAxisRoundTrips: the revolved-face axis restores from the recipe and re-resolves
// against the body (#1840).
func TestRevolvedFaceAxisRoundTrips(t *testing.T) {
	t.Parallel()
	g := NewWorkGeometry()
	body, key := faceBody(t, mustCylinder(t))
	g.WorkAxes().AddByRevolvedFace(FaceRef(key))
	g.Recompute([]*topo.Body{body})

	data, err := MarshalWork(g)
	if err != nil {
		t.Fatalf("MarshalWork: %v", err)
	}
	restored := NewWorkGeometry()
	if err := ApplyWork(restored, data); err != nil {
		t.Fatalf("ApplyWork: %v", err)
	}
	restored.Recompute([]*topo.Body{body})
	ra := restored.WorkAxes().Item(restored.WorkAxes().Count() - 1)
	if ra.Kind() != "revolved-face" || !ra.Health().OK() {
		t.Errorf("restored axis kind %q healthy %v, want revolved-face healthy", ra.Kind(), ra.Health().OK())
	}
	if !ra.Direction().AsVector().IsParallelTo(math.V3(0, 0, 1), wtol) {
		t.Errorf("restored axis direction = %v, want +Z", ra.Direction())
	}
}

// TestFaceCenterRoundTrips: the face-center point restores from the recipe and re-resolves (#1842).
func TestFaceCenterRoundTrips(t *testing.T) {
	t.Parallel()
	g := NewWorkGeometry()
	sph, _ := geom.NewSphere(math.P3(1, 2, 3), 2)
	body, key := faceBody(t, sph)
	g.WorkPoints().AddByFaceCenter(FaceRef(key))
	g.Recompute([]*topo.Body{body})

	data, err := MarshalWork(g)
	if err != nil {
		t.Fatalf("MarshalWork: %v", err)
	}
	restored := NewWorkGeometry()
	if err := ApplyWork(restored, data); err != nil {
		t.Fatalf("ApplyWork: %v", err)
	}
	restored.Recompute([]*topo.Body{body})
	rp := restored.WorkPoints().Item(restored.WorkPoints().Count() - 1)
	if rp.Kind() != "face-center" || !rp.Point().IsEqualTo(math.P3(1, 2, 3), wtol) {
		t.Errorf("restored point kind %q at %v, want face-center at (1,2,3)", rp.Kind(), rp.Point())
	}
}
