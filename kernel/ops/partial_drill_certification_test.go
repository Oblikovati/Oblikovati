// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Partial curved-on-planar drill (edge scallop) certification (#1591, ADR-0049 Slice A'). A cylinder drilled
// through a plate whose base circle CLIPS the plate edge cuts through the planeUV operand — both cap faces are
// trimmed by the circle, the breached side face is split in two, and a partial cylinder wall closes the notch.
// Certified by an INDEPENDENT analytic mass and the top-priority tessellation-watertightness gate.

// scallopPlate builds the fixture: a 10×10×2 plate drilled by a vertical cylinder (r=2) centered at (4,0), so
// its circle (x∈[2,6]) clips the plate's +x edge at x=5 — an edge scallop.
func scallopPlate(t *testing.T) *topo.Body {
	t.Helper()
	plate, err := brep.SolidBlock(math.P3(-5, -5, 0), math.P3(5, 5, 2), "plate")
	if err != nil {
		t.Fatalf("plate: %v", err)
	}
	drill, err := brep.SolidCylinder(math.P3(4, 0, -1), math.V3(0, 0, 1), 2, 4)
	if err != nil {
		t.Fatalf("drill: %v", err)
	}
	res, ok := brep.CutEdgeScallop(plate, drill)
	if !ok {
		t.Fatal("CutEdgeScallop declined the edge-clipping drill")
	}
	return res
}

// scallopAnalyticVolume is the exact removed-then-subtracted Volume: plate 200 minus the inside-plate disk
// area A_in = r²(π−arccos(d/r)) + d·√(r²−d²) (d=5−cx=1, r=2) times the thickness h=2.
func scallopAnalyticVolume() float64 {
	d, r, h := 1.0, 2.0, 2.0
	aIn := r*r*(stdmath.Pi-stdmath.Acos(d/r)) + d*stdmath.Sqrt(r*r-d*d)
	return 200 - aIn*h
}

func TestScallopIsWatertightManifold(t *testing.T) {
	t.Parallel()
	res := scallopPlate(t)
	if !res.IsSolid() {
		t.Fatal("scallop is not a solid")
	}
	if n := len(res.Shells()); n != 1 {
		t.Errorf("scallop has %d shells, want 1", n)
	}
	free := 0
	for _, e := range res.Edges() {
		if len(e.Uses()) != 2 {
			free++
		}
	}
	if free != 0 {
		t.Errorf("scallop B-rep has %d free edges, want 0", free)
	}
}

func TestScallopVolumeMatchesAnalytic(t *testing.T) {
	t.Parallel()
	res := scallopPlate(t)
	want := scallopAnalyticVolume()
	got := query.BodyGeometryProperties(res, DefaultQuality()).Volume
	if rel := stdmath.Abs(got-want) / want; rel > 0.01 {
		t.Errorf("scallop volume %.4f vs analytic %.4f; rel %.4f > 0.01", got, want, rel)
	}
}

// TestScallopTessellationIsWatertight is the top-priority mesh gate (CLAUDE.md): the trimmed caps, split side
// faces and partial wall must weld crack-free so the rendered notch shows no tear at the clipped edge.
func TestScallopTessellationIsWatertight(t *testing.T) {
	t.Parallel()
	res := scallopPlate(t)
	for _, gq := range gateQualities() {
		mesh, _ := tessellate.TessellateBody(res, gq.q)
		if free := cornerMeshFreeEdges(mesh); free != 0 {
			t.Errorf("%s quality: scallop tessellation has %d free edges (want 0) — a visible crack at the clipped edge",
				gq.name, free)
		}
	}
}

// TestScallopCutViaAnalyticDispatch is the reachability gate: ops.Boolean(Cut) must route the edge-clipping
// drill to the analytic assembler (curvedEdgeScallopCut), keeping the cylinder wall as one analytic face, not
// fall through to CSG triangle-soup (which is watertight and the right volume too, so only the analytic
// cylinder face distinguishes them).
func TestScallopCutViaAnalyticDispatch(t *testing.T) {
	t.Parallel()
	plate, _ := brep.SolidBlock(math.P3(-5, -5, 0), math.P3(5, 5, 2), "plate")
	drill, _ := brep.SolidCylinder(math.P3(4, 0, -1), math.V3(0, 0, 1), 2, 4)
	res, err := Boolean(Cut, plate, drill)
	if err != nil {
		t.Fatalf("Boolean(Cut): %v", err)
	}
	if n := len(res.Faces()); n > 20 {
		t.Errorf("cut has %d faces; want the analytic assembler (~8), not CSG triangle-soup", n)
	}
	hasCyl := false
	for _, f := range res.Faces() {
		if _, ok := f.Geometry().(geom.Cylinder); ok {
			hasCyl = true
			break
		}
	}
	if !hasCyl {
		t.Error("cut kept no analytic cylinder face — the scallop wall was not preserved (CSG fallback)")
	}
	for _, gq := range gateQualities() {
		mesh, _ := tessellate.TessellateBody(res, gq.q)
		if free := cornerMeshFreeEdges(mesh); free != 0 {
			t.Errorf("%s quality: dispatched scallop tessellation has %d free edges (want 0)", gq.name, free)
		}
	}
}
