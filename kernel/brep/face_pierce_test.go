// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// findSideFace returns the cylindrical side face of a cylinder body.
func findSideFace(t *testing.T, b *topo.Body) *topo.Face {
	t.Helper()
	for _, f := range b.Faces() {
		if _, ok := f.Geometry().(geom.Cylinder); ok {
			return f
		}
	}
	t.Fatal("no cylindrical side face")
	return nil
}

// findCapAt returns the planar cap face whose plane passes through z.
func findCapAt(t *testing.T, b *topo.Body, z float64) *topo.Face {
	t.Helper()
	for _, f := range b.Faces() {
		pl, ok := f.Geometry().(geom.Plane)
		if ok && stdmath.Abs(float64(pl.Origin.Z)-z) < 1e-9 {
			return f
		}
	}
	t.Fatalf("no planar cap at z=%v", z)
	return nil
}

func TestPointInFaceTrimCylinderSide(t *testing.T) {
	t.Parallel()
	const r, h = 3.0, 8.0
	body, err := SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), r, h)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	side := findSideFace(t, body)
	// A point ON the side, at mid height, is within the trim.
	inside := math.P3(r, 0, h/2)
	if !PointInFaceTrim(side, inside) {
		t.Errorf("mid-height side point %v reported OUTSIDE the side trim", inside)
	}
	// A point on the (infinite) cylinder surface but ABOVE the finite band is outside the trim.
	above := math.P3(r, 0, h+2)
	if PointInFaceTrim(side, above) {
		t.Errorf("point above the finite side band %v reported INSIDE the side trim", above)
	}
	below := math.P3(r, 0, -2)
	if PointInFaceTrim(side, below) {
		t.Errorf("point below the finite side band %v reported INSIDE the side trim", below)
	}
}

func TestPointInFaceTrimCap(t *testing.T) {
	t.Parallel()
	const r, h = 3.0, 8.0
	body, err := SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), r, h)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	top := findCapAt(t, body, h)
	// The cap centre is inside the disk trim.
	if !PointInFaceTrim(top, math.P3(0, 0, h)) {
		t.Error("cap centre reported OUTSIDE the cap trim")
	}
	// A point on the cap plane but beyond the disk radius is outside the trim.
	if PointInFaceTrim(top, math.P3(r+2, 0, h)) {
		t.Error("point beyond the cap radius reported INSIDE the cap trim")
	}
}
