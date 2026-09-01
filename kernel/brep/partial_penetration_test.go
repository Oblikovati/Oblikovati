// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
)

// TestPartialPenetrationBlindPocket is the minimal repro for M20-F01 PBI-199 (#470): a square
// bar that pokes part-way into the top face of a box (its bottom cap lies INSIDE the box) cut
// from the box must leave a watertight blind pocket with the right volume. The planar B-rep
// boolean is exercised directly so the BSP-CSG fallback cannot mask the arrangement defect.
func TestPartialPenetrationBlindPocket(t *testing.T) {
	t.Parallel()
	body := box(0, 0, 0, 10, 10, 10) // [0,10]^3, volume 1000
	tool := box(3, 3, 5, 4, 4, 7)    // [3,7]x[3,7]x[5,12]; bottom z=5 is inside the box
	res, err := brep.Boolean(brep.Difference, body, tool)
	if err != nil {
		t.Fatalf("Boolean: %v", err)
	}
	if open := ops.BoundaryEdges(res); len(open) != 0 {
		t.Errorf("blind pocket left %d boundary edges, want 0 (watertight)", len(open))
	}
	if r := ops.Validate(res); !r.Valid {
		t.Errorf("blind pocket body invalid: manifold=%v closed=%v orient=%v", r.Manifold, r.Closed, r.OrientationOK)
	}
	got := query.BodyGeometryProperties(res, ops.DefaultQuality()).Volume
	want := 1000.0 - 4*4*5 // overlap [3,7]x[3,7]x[5,10] = 80 removed
	if stdmath.Abs(got-want) > 1e-6 {
		t.Errorf("blind pocket volume = %g, want %g", got, want)
	}
}

// TestBladeCrossesConcaveFacetedWall is the live PBI-199 defect (#470): a radial bar joined to
// a tube crosses the tube's CONCAVE faceted inner wall, spanning several facets, and ends inside
// the wall material (partial penetration of a re-entrant faceted surface). The fan blade hits
// exactly this; the planar boolean must union it into a single valid manifold solid rather than
// leaving a coincident shell.
func TestBladeCrossesConcaveFacetedWall(t *testing.T) {
	t.Parallel()
	tube := annularPrism(t, 24, 20, 4, "tube") // z∈[0,4], inner 12-gon (concave wall), outer 64-gon
	bar := box(16, -6, 1, 8, 12, 2)            // x∈[16,24] y∈[-6,6] z∈[1,3]; crosses the inner wall over several facets
	joined, err := brep.Boolean(brep.Union, tube, bar)
	if err != nil {
		t.Fatalf("Boolean(Union): %v", err)
	}
	if open := ops.BoundaryEdges(joined); len(open) != 0 {
		t.Errorf("blade∪tube left %d boundary edges, want 0 (watertight)", len(open))
	}
	if r := ops.Validate(joined); !r.Valid {
		t.Errorf("blade∪tube invalid: manifold=%v closed=%v orient=%v", r.Manifold, r.Closed, r.OrientationOK)
	}
}

// TestBladeFlushAgainstFacetGrazes is the flush/grazing trigger: a blade whose outer end face
// is COPLANAR with one facet of the tube's inner wall while its body crosses the neighbouring
// facets (the "blade grazes the faceted wall" case the fan hits). The planar boolean must still
// union it into a single valid manifold rather than welding a coincident shell.
func TestBladeFlushAgainstFacetGrazes(t *testing.T) {
	t.Parallel()
	tube := annularPrism(t, 24, 20, 4, "tube") // inner 12-gon facet near +X is the vertical chord x≈19.32, y∈[-5.18,5.18]
	// Bar outer face at x=19.32 (flush with that facet), crossing it; inner end inside the bore.
	bar := box(12, -3, 1, 7.32, 6, 2) // x∈[12,19.32] y∈[-3,3] z∈[1,3]; +X face coplanar with the facet
	joined, err := brep.Boolean(brep.Union, tube, bar)
	if err != nil {
		t.Fatalf("Boolean(Union): %v", err)
	}
	if open := ops.BoundaryEdges(joined); len(open) != 0 {
		t.Errorf("flush blade left %d boundary edges, want 0 (watertight)", len(open))
	}
	if r := ops.Validate(joined); !r.Valid {
		t.Errorf("flush blade invalid: manifold=%v closed=%v orient=%v", r.Manifold, r.Closed, r.OrientationOK)
	}
}
