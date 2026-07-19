// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
)

// M7 whole-body gate — OCCT blend/simple/M7, the single-arm curved RUNOUT whose host B (plane x=60) FLUSH-
// CUTS the arm cylinder through its axis and whose z=100 cap carries one unrelated footprint hole (the
// forensic's 2-wire survivor). The runout retrim receded the bitten INNER footprint loop (not the outer box
// square), which the earlier outer-loop-only retrim declined. This pins, WITHOUT loosening the corpus gate,
// that the weld is a watertight 11-face solid, that OCCT's three named faces land at their DRAWEXE oracle
// areas (fillet CylindricalSurface r=10 = 1150.26; receded host Cyl r=25 = 4978.33; footprint-hole cap =
// 8584.49, STILL a 2-wire face), and that every face meshes fold-free — tessellation correctness being the
// repo's highest priority (CLAUDE.md). Oracle areas: .superpowers/sdd/curved-runout-forensic.md §Appendix.

const m7AreaRelTol = 0.01 // matches the corpus deps gate (1%)

// TestM7WholeBodyWatertight asserts the single-arm runout welds a watertight 11-face manifold solid — every
// edge 2-incident, valid + closed + holes-contained + IsSolid — the flush-cut-cap retrim's crux (a cracked
// or mis-classified inner-loop retrim fails here loud).
func TestM7WholeBodyWatertight(t *testing.T) {
	assertWatertight(t, "M7", caseResultBody(t, "M7"), 11)
}

// TestM7FilletFaceArea pins the ONE added fillet face — the exact CylindricalSurface r=10 — to OCCT's
// area 1150.26. A wrong-region or folded fillet strip reads far off.
func TestM7FilletFaceArea(t *testing.T) {
	f := m7FaceByCylinderRadius(t, caseResultBody(t, "M7"), 10)
	assertM7FaceArea(t, "fillet cylinder r=10", faceMeshArea2(f), 1150.26)
}

// TestM7HostCylinderReceded pins the receded host wall (Cyl r=25) to OCCT's 4978.33 — the runout removed
// ~half the wall, so a wrong-side retrim (the "smaller corner" heuristic) would read the complement.
func TestM7HostCylinderReceded(t *testing.T) {
	f := m7FaceByCylinderRadius(t, caseResultBody(t, "M7"), 25)
	assertM7FaceArea(t, "receded host cylinder r=25", faceMeshArea2(f), 4978.33)
}

// TestM7FootprintHoleSurvives asserts the unrelated footprint hole on the flush-cut cap SURVIVES the runout
// as a still-2-wire planar face at OCCT's area 8584.49 (forensic §2: "one unrelated 2-wire face survives").
// The earlier retrim declined here; a retrim that erased or mis-wound the hole fails this assertion.
func TestM7FootprintHoleSurvives(t *testing.T) {
	f := m7TwoWirePlanarFace(t, caseResultBody(t, "M7"))
	assertM7FaceArea(t, "footprint-hole cap (2 wires)", faceMeshArea2(f), 8584.49)
}

// TestM7TessellationFoldGate meshes every M7 face and asserts each is fold-free with a finite positive area,
// and that the summed mesh area equals OCCT's whole-result area (corpus 67937.6) within the deps tol.
func TestM7TessellationFoldGate(t *testing.T) {
	body := caseResultBody(t, "M7")
	total := 0.0
	for _, f := range body.Faces() {
		m := ops.TessellateFace(f, ops.PropertyQuality())
		area := ops.MeshArea(m)
		total += area
		assertM7FaceSane(t, f, m, area)
	}
	const want = 67937.6
	if rel := stdmath.Abs(total-want) / want; rel > m7AreaRelTol {
		t.Fatalf("M7 total mesh area %.2f, want OCCT %.1f within deps %.2f (rel %.4f)", total, want, m7AreaRelTol, rel)
	}
}

// assertM7FaceSane fails on a non-finite/non-positive area or ANY fold edge — M7 has no known residual fold.
func assertM7FaceSane(t *testing.T, f *topo.Face, m *ops.Mesh, area float64) {
	t.Helper()
	if area <= 0 || stdmath.IsInf(area, 0) || stdmath.IsNaN(area) {
		t.Fatalf("M7 %T face meshed to %.4f, want a finite positive area", f.Geometry(), area)
	}
	if n := ops.FoldEdgeCount(m); n != 0 {
		t.Fatalf("M7 %T face (area %.2f) has %d fold edges — a tessellation defect", f.Geometry(), area, n)
	}
}

// assertM7FaceArea fails unless got matches want within m7AreaRelTol (relative).
func assertM7FaceArea(t *testing.T, label string, got, want float64) {
	t.Helper()
	if rel := stdmath.Abs(got-want) / want; rel > m7AreaRelTol {
		t.Fatalf("M7 %s mesh area %.2f, want OCCT %.2f within %.0f%% (rel %.4f)", label, got, want, m7AreaRelTol*100, rel)
	}
}

// m7FaceByCylinderRadius returns the single cylinder face of the given radius (10 = fillet arm, 25 = host
// wall), failing when none or more than one matches (the radii are distinct in M7).
func m7FaceByCylinderRadius(t *testing.T, body *topo.Body, radius float64) *topo.Face {
	t.Helper()
	var found *topo.Face
	for _, f := range body.Faces() {
		cyl, ok := f.Geometry().(geom.Cylinder)
		if ok && stdmath.Abs(cyl.Radius-radius) < 1e-6*radius {
			if found != nil {
				t.Fatalf("M7 has more than one cylinder face of radius %g", radius)
			}
			found = f
		}
	}
	if found == nil {
		t.Fatalf("M7 carries no cylinder face of radius %g", radius)
	}
	return found
}

// m7TwoWirePlanarFace returns the unique planar face carrying an inner (hole) loop — the flush-cut cap whose
// footprint hole must survive. Fails when none or more than one such face exists.
func m7TwoWirePlanarFace(t *testing.T, body *topo.Body) *topo.Face {
	t.Helper()
	var found *topo.Face
	for _, f := range body.Faces() {
		if _, ok := f.Geometry().(geom.Plane); ok && len(f.Loops()) >= 2 {
			if found != nil {
				t.Fatalf("M7 has more than one multi-wire planar face")
			}
			found = f
		}
	}
	if found == nil {
		t.Fatalf("M7 carries no 2-wire planar face — the footprint hole did not survive the runout")
	}
	return found
}
