// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Cylinder-side split topology (M2 Phase 1, Oblikovati/Oblikovati#1334). HalfSpaceCut by a plane parallel
// to the cylinder axis must rebuild a clean, seam-free manifold: the arc band, the box-wall lid and the
// trimmed caps, every edge shared by exactly two faces. Volume is checked through ops_test (brep cannot
// import ops); here the concern is watertight topology and analytic surfaces preserved.

// assertClosedManifold checks every edge of a solid is used exactly twice and the surfaces stayed
// analytic (cylinder bands + planar caps/lids), the watertight invariant the split must keep.
func assertClosedManifold(t *testing.T, body *topo.Body, wantFaces int) {
	t.Helper()
	if body == nil || !body.IsSolid() {
		t.Fatalf("result is not a solid: %+v", body)
	}
	if n := len(body.Faces()); n != wantFaces {
		t.Errorf("result has %d faces, want %d", n, wantFaces)
	}
	for _, e := range body.Edges() {
		if uses := len(e.Uses()); uses != 2 {
			t.Errorf("edge %v has %d uses, want 2 (a closed manifold)", e.Lineage(), uses)
		}
	}
	for _, f := range body.Faces() {
		switch f.Geometry().(type) {
		case geom.Cylinder, geom.Plane:
		default:
			t.Errorf("face surface %T is not analytic", f.Geometry())
		}
	}
}

// TestHalfSpaceCutCylinderSegmentTopology: an off-centre axis-parallel cut leaves an arc-band side, two
// half-disk caps and the wall lid — four faces, watertight.
func TestHalfSpaceCutCylinderSegmentTopology(t *testing.T) {
	t.Parallel()
	cyl, _ := SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3, 10)
	plane, _ := geom.NewPlane(math.P3(1.5, 0, 0), math.V3(1, 0, 0)) // keep x ≤ 1.5
	res, err := HalfSpaceCut(cyl, plane)
	if err != nil {
		t.Fatalf("HalfSpaceCut: %v", err)
	}
	assertClosedManifold(t, res, 4)
}

// TestHalfSpaceCutCylinderSeamOnCutTopology is a regression for the closed-edge seam split: when the
// cut keeps the circle's seam vertex (here the symmetric x ≤ 0 plane), the kept arc wraps the seam and
// must stay ONE edge that welds with the side band, not fragment into two open edges.
func TestHalfSpaceCutCylinderSeamOnCutTopology(t *testing.T) {
	t.Parallel()
	cyl, _ := SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3, 10)
	plane, _ := geom.NewPlane(math.P3(0, 0, 0), math.V3(1, 0, 0)) // keep x ≤ 0
	res, err := HalfSpaceCut(cyl, plane)
	if err != nil {
		t.Fatalf("HalfSpaceCut: %v", err)
	}
	assertClosedManifold(t, res, 4)
}

// TestHalfSpaceCutCylinderSlabTopology composes two opposite walls into a |x| ≤ 1.5 slab. The kept
// cylinder surface is two disconnected strips, so the looped split must emit TWO arc-band faces (six
// faces total), each watertight.
func TestHalfSpaceCutCylinderSlabTopology(t *testing.T) {
	t.Parallel()
	cyl, _ := SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3, 10)
	pHi, _ := geom.NewPlane(math.P3(1.5, 0, 0), math.V3(1, 0, 0))
	pLo, _ := geom.NewPlane(math.P3(-1.5, 0, 0), math.V3(-1, 0, 0))
	band, err := HalfSpaceCut(cyl, pHi)
	if err != nil {
		t.Fatalf("first wall: %v", err)
	}
	slab, err := HalfSpaceCut(band, pLo)
	if err != nil {
		t.Fatalf("second wall: %v", err)
	}
	cylFaces := 0
	for _, f := range slab.Faces() {
		if _, ok := f.Geometry().(geom.Cylinder); ok {
			cylFaces++
		}
	}
	if cylFaces != 2 {
		t.Errorf("slab has %d cylinder faces, want 2 (two disconnected strips)", cylFaces)
	}
	assertClosedManifold(t, slab, 6)
}

// TestHalfSpaceCutCylinderClearsKeepsWhole: an axis-parallel plane clear of the cylinder keeps it whole.
func TestHalfSpaceCutCylinderClearsKeepsWhole(t *testing.T) {
	t.Parallel()
	cyl, _ := SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3, 10)
	plane, _ := geom.NewPlane(math.P3(5, 0, 0), math.V3(1, 0, 0)) // x ≤ 5: the whole cylinder
	res, err := HalfSpaceCut(cyl, plane)
	if err != nil {
		t.Fatalf("HalfSpaceCut: %v", err)
	}
	assertClosedManifold(t, res, 3) // untouched: side + two caps
}
