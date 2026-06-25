// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
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

// TestHalfSpaceCutConeObliqueApexExact: an oblique plane cuts a FULL cone (apex at z=10) in a conic
// section the (u,v) ruled split now builds exactly — the full-cone-to-apex case (Oblikovati#1375). The
// plane drops the apex, so the kept part is a frustum-like band (the base cap, the cone band, and the
// section lid), watertight, no CSG fallback.
func TestHalfSpaceCutConeObliqueApexExact(t *testing.T) {
	cone, _ := SolidCylinderCone(math.P3(0, 0, 0), math.P3(0, 0, 10), 3, 0, "cone")
	plane, _ := geom.NewPlane(math.P3(0, 0, 4), math.V3(0.5, 0, 0.866)) // steep oblique → ellipse, apex dropped
	res, err := HalfSpaceCut(cone, plane)
	if err != nil {
		t.Fatalf("oblique full-cone cut should now be exact, got %v", err)
	}
	assertWatertight(t, res)
	if cones, _, _ := faceTypeCounts(t, res); cones != 1 {
		t.Errorf("result has %d cone faces, want exactly 1 (the band stays analytic)", cones)
	}
}

// TestHalfSpaceCutConeObliqueApexKept keeps the apex side of an oblique full-cone cut: the kept solid is
// the cone TIP closing to its apex pole, a single-loop cone face capped by the elliptical lid — watertight,
// one analytic cone face, no CSG (the apex-kept branch of the full-cone-to-apex split, Oblikovati#1375).
func TestHalfSpaceCutConeObliqueApexKept(t *testing.T) {
	cone, _ := SolidCylinderCone(math.P3(0, 0, 0), math.P3(0, 0, 10), 3, 0, "cone")
	plane, _ := geom.NewPlane(math.P3(0, 0, 4), math.V3(-0.5, 0, -0.866)) // keep the apex tip above the ellipse
	res, err := HalfSpaceCut(cone, plane)
	if err != nil {
		t.Fatalf("apex-kept oblique cut should be exact, got %v", err)
	}
	assertWatertight(t, res)
	if cones, _, _ := faceTypeCounts(t, res); cones != 1 {
		t.Errorf("result has %d cone faces, want exactly 1 (the cone tip stays analytic)", cones)
	}
}
