// SPDX-License-Identifier: GPL-2.0-only

package tessellate_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/math"
)

// TestFacetKeepsASmallCylindersVolume pins that ops.Facet stays faithful at small radii (#31).
//
// Facet used to mesh through bodyTriangles, which pins its quality to the BSP CSG's chord-only
// booleanInputQuality (AngleTolerance = pi). Chord-only faceting is RADIUS-BLIND: at the display chord
// tolerance (0.05) an r=0.15 cylinder admits 2*acos(1-0.05/0.15) = 1.68 rad per facet — FOUR facets, a
// square prism holding 2r^2h = 64% of the volume. But Facet's cage feeds the EXACT PLANAR boolean, not
// the BSP, so it never needed that trade; the angle bound gives ~32 facets at any radius.
//
// This is not a cosmetic bias. A screw's Ø3mm shaft lost 38.5 of its 106.03 mm³ here, BEFORE any boolean
// ran, and the cut that followed was exactly as wrong as its input (removing 81.8 mm³ where 43.7 was
// possible) — a silently wrong solid, the failure CLAUDE.md ranks above all feature work.
//
// The bound is 3%: the honest 32-gon inscribed bias is ~0.64%, while the square regression is 36% — so
// this discriminates by more than 10x and cannot be met by tightening tessellation noise.
func TestFacetKeepsASmallCylindersVolume(t *testing.T) {
	t.Parallel()
	const r, h = 0.15, 1.5 // the real screw shaft: Ø3mm x 15mm
	cyl, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), r, h)
	if err != nil {
		t.Fatalf("cylinder: %v", err)
	}
	faceted := ops.Facet(cyl, "test/facet")
	if faceted == nil {
		t.Fatal("Facet returned nil for a plain cylinder")
	}
	want := stdmath.Pi * r * r * h
	got := query.BodyGeometryProperties(faceted, ops.PropertyQuality()).Volume
	if stdmath.Abs(got-want)/want > 0.03 {
		square := 2 * r * r * h // what a 4-facet collapse yields
		t.Errorf("faceted cylinder volume = %g cm³, want %g +/-3%% (a 4-facet square prism would give %g)",
			got, want, square)
	}
}

// TestFacetKeepsASmallCylinderRound pins the same defect topologically, independently of any volume
// integration: the faceted wall must have far more than a square's four sides. A cylinder facets to its
// wall segments plus two caps, so the old chord-only collapse read 6 faces total (4 walls + 2 caps).
func TestFacetKeepsASmallCylinderRound(t *testing.T) {
	t.Parallel()
	cyl, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 0.15, 1.5)
	if err != nil {
		t.Fatalf("cylinder: %v", err)
	}
	faceted := ops.Facet(cyl, "test/facet")
	if faceted == nil {
		t.Fatal("Facet returned nil for a plain cylinder")
	}
	// 12 walls is still generous headroom under the ~32 the angle bound gives, so this pins the
	// radius-blindness rather than the exact facet count.
	if n := len(faceted.Faces()); n < 14 {
		t.Errorf("faceted Ø3mm cylinder has %d faces — too few to be round (a square prism is 6)", n)
	}
}
