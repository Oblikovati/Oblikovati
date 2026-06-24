// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"errors"
	"testing"

	"oblikovati.org/math"
)

// Axial prism subtract topology (M2 Phase 1, Oblikovati/Oblikovati#1334). SubtractAxialPrism must rebuild
// a watertight cylinder-with-tunnel: the side untouched, two holed caps, one wall per cross-section edge,
// every edge shared by exactly two faces. Volume is checked through ops_test.

// TestSubtractAxialPrismSquareHole: a 2×2 square tunnel → side + 2 caps + 4 walls = 7 faces, watertight.
func TestSubtractAxialPrismSquareHole(t *testing.T) {
	cyl, _ := SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3, 10)
	sq := []math.Point3{math.P3(1, 1, 0), math.P3(-1, 1, 0), math.P3(-1, -1, 0), math.P3(1, -1, 0)}
	res, err := SubtractAxialPrism(cyl, sq)
	if err != nil {
		t.Fatalf("SubtractAxialPrism: %v", err)
	}
	assertClosedManifold(t, res, 7)
}

// TestSubtractAxialPrismTriangleHole proves the polygon path is not rectangle-only: a triangular tunnel
// gives side + 2 caps + 3 walls = 6 faces, watertight.
func TestSubtractAxialPrismTriangleHole(t *testing.T) {
	cyl, _ := SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3, 10)
	tri := []math.Point3{math.P3(1.5, 0, 0), math.P3(-1, 1.2, 0), math.P3(-1, -1.2, 0)}
	res, err := SubtractAxialPrism(cyl, tri)
	if err != nil {
		t.Fatalf("SubtractAxialPrism: %v", err)
	}
	assertClosedManifold(t, res, 6)
}

// TestSubtractAxialPrismCornerOutsideDefers: a cross-section corner beyond the radius would breach the
// side — defer (ErrUnsupportedHalfSpace) so the caller keeps the CSG fallback.
func TestSubtractAxialPrismCornerOutsideDefers(t *testing.T) {
	cyl, _ := SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3, 10)
	wide := []math.Point3{math.P3(4, 1, 0), math.P3(-1, 1, 0), math.P3(-1, -1, 0), math.P3(4, -1, 0)}
	if _, err := SubtractAxialPrism(cyl, wide); !errors.Is(err, ErrUnsupportedHalfSpace) {
		t.Errorf("corner at radius 4 (> 3) should defer, got err=%v", err)
	}
}

// TestSubtractAxialPrismNotACylinderDefers: only a bare cylinder is handled.
func TestSubtractAxialPrismNotACylinderDefers(t *testing.T) {
	block, _ := SolidBlock(math.P3(0, 0, 0), math.P3(5, 5, 5), "b")
	sq := []math.Point3{math.P3(1, 1, 0), math.P3(2, 1, 0), math.P3(2, 2, 0), math.P3(1, 2, 0)}
	if _, err := SubtractAxialPrism(block, sq); !errors.Is(err, ErrUnsupportedHalfSpace) {
		t.Errorf("a non-cylinder target should defer, got err=%v", err)
	}
}
