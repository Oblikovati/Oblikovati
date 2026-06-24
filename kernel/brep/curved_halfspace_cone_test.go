// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"errors"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Cone half-space topology (M2 Phase 1, Oblikovati/Oblikovati#1334). A perpendicular cut rebuilds an
// exact cone (apex side) or frustum (base side), watertight; an oblique cut defers. Volume is checked in
// ops_test.

// assertConeManifold checks a solid is watertight (every edge used twice) with only cone/planar faces.
func assertConeManifold(t *testing.T, body *topo.Body, wantFaces int) {
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
		case geom.Cone, geom.Plane:
		default:
			t.Errorf("face surface %T is not analytic", f.Geometry())
		}
	}
}

// TestHalfSpaceCutConeBaseFrustumTopology: keeping the base side leaves a frustum (cone + two caps).
func TestHalfSpaceCutConeBaseFrustumTopology(t *testing.T) {
	cone, _ := SolidCylinderCone(math.P3(0, 0, 0), math.P3(0, 0, 10), 3, 0, "cone")
	plane, _ := geom.NewPlane(math.P3(0, 0, 5), math.V3(0, 0, 1)) // keep z ≤ 5
	res, err := HalfSpaceCut(cone, plane)
	if err != nil {
		t.Fatalf("cone ⟂ cut: %v", err)
	}
	assertConeManifold(t, res, 3)
}

// TestHalfSpaceCutConeApexConeTopology: keeping the apex side leaves a smaller cone (cone + one cap).
func TestHalfSpaceCutConeApexConeTopology(t *testing.T) {
	cone, _ := SolidCylinderCone(math.P3(0, 0, 0), math.P3(0, 0, 10), 3, 0, "cone")
	plane, _ := geom.NewPlane(math.P3(0, 0, 5), math.V3(0, 0, -1)) // keep z ≥ 5 (the tip)
	res, err := HalfSpaceCut(cone, plane)
	if err != nil {
		t.Fatalf("cone ⟂ cut: %v", err)
	}
	assertConeManifold(t, res, 2)
}

// TestHalfSpaceCutFrustumReCut: a frustum (both radii non-zero) is recognized and re-cut to a shorter
// frustum, so perpendicular cuts compose.
func TestHalfSpaceCutFrustumReCut(t *testing.T) {
	frustum, _ := SolidCylinderCone(math.P3(0, 0, 0), math.P3(0, 0, 10), 4, 2, "frustum")
	plane, _ := geom.NewPlane(math.P3(0, 0, 6), math.V3(0, 0, 1)) // keep z ≤ 6
	res, err := HalfSpaceCut(frustum, plane)
	if err != nil {
		t.Fatalf("frustum ⟂ cut: %v", err)
	}
	assertConeManifold(t, res, 3)
}

// TestHalfSpaceCutConeObliqueDefers: an oblique plane cuts an ellipse/hyperbola section that the analytic
// imprint does not solve, so the cut defers and the caller keeps the CSG fallback.
func TestHalfSpaceCutConeObliqueDefers(t *testing.T) {
	cone, _ := SolidCylinderCone(math.P3(0, 0, 0), math.P3(0, 0, 10), 3, 0, "cone")
	plane, _ := geom.NewPlane(math.P3(0, 0, 5), math.V3(1, 0, 1)) // tilted: not perpendicular to the axis
	if _, err := HalfSpaceCut(cone, plane); !errors.Is(err, ErrUnsupportedHalfSpace) {
		t.Errorf("oblique cone cut should defer, got err=%v", err)
	}
}
