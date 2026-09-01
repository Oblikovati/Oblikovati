// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/math"
	"oblikovati.org/test-utilities/nopscad"
)

// TestNopWasherCSG re-models NopSCADlib's washer the OpenSCAD-CSG way — a solid
// cylinder (OD) minus a concentric cylinder (ID) — via the general boolean, and
// validates the result against the rendered golden. NopSCADlib draws an M3
// washer as OD 7, ID 3.1, and linear_extrude(thickness-0.05) = 0.45 thick.
//
// Note on kernel behaviour surfaced here: the exact planar drill
// (brep.CutCylindricalHole) only operates on planar-faceted slabs, so it cannot
// bore a curved cylinder. Cylinder−cylinder therefore goes through
// ops.Boolean's curved (faceted CSG) path, which yields a triangulated solid —
// correct volume but no analytic cylinder faces. The exact analytic annulus
// (true cylinder walls + planar caps) is built the Inventor way by the revolve
// integration test; this unit test pins the CSG idiom.
//
// Reference: NopSCADlib/vitamins/washer.scad + vitamins/washers.scad
// (M3_washer = ["M3",3,7,0.5,...]).
func TestNopWasherCSG(t *testing.T) {
	t.Parallel()
	const rOut, rIn, h = 3.5, 1.55, 0.45 // OD/2, (ID=size+0.1)/2, thickness-0.05

	outer, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), rOut, h)
	if err != nil {
		t.Fatalf("outer SolidCylinder: %v", err)
	}
	// Inner tool overshoots top and bottom (like OpenSCAD's epsilon) for a clean cut.
	inner, err := brep.SolidCylinder(math.P3(0, 0, -0.1), math.V3(0, 0, 1), rIn, h+0.2)
	if err != nil {
		t.Fatalf("inner SolidCylinder: %v", err)
	}
	body, err := ops.Boolean(ops.Cut, outer, inner)
	if err != nil {
		t.Fatalf("Boolean(Cut): %v", err)
	}

	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("washer not a valid solid: %+v", r)
	}
	if open := ops.BoundaryEdges(body); len(open) != 0 {
		t.Fatalf("washer has %d boundary edges, want 0 (watertight)", len(open))
	}

	got := query.BodyGeometryProperties(body, ops.DefaultQuality()).Volume

	// Analytic annulus volume — faceted CSG inscribes the circle, so the result
	// is slightly under; allow a faceting band.
	wantExact := stdmath.Pi * (rOut*rOut - rIn*rIn) * h
	if rel := stdmath.Abs(got-wantExact) / wantExact; rel > 0.03 {
		t.Errorf("washer volume = %.5f, want analytic %.5f (rel %.4f > 3%% faceting band)", got, wantExact, rel)
	}

	// Cross-check against the OpenSCAD golden (also faceted): they should agree closely.
	gold, err := nopscad.Golden("washer")
	if err != nil {
		t.Fatalf("load washer golden: %v", err)
	}
	if rel := stdmath.Abs(got-gold.Volume) / gold.Volume; rel > 0.03 {
		t.Errorf("washer volume = %.5f, golden %.5f (rel %.4f > 3%% band)", got, gold.Volume, rel)
	}
}
