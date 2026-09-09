// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"strings"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/health"
)

// TestASubResolutionCutSickensItsFeature carries the kernel's size classification all the way to the
// browser. It drives the EXACT configuration the kernel sweep measured — the RING corpus body cut by
// an axial drill of radius 1e-10, which the sweep
// (kernel/ops/boolean/boolean_drill_sweep_test.go) shows the pipeline answered SILENTLY: the ring
// came back untouched, err=nil, nothing recorded.
//
// It combines two BUILT bodies rather than extruding a sketched circle, deliberately. The earlier
// version of this row pinned a 0.4 mm bore in a 400-unit plate, which is ~1.5x below the floor it was
// written against and was never shown to be unbuildable at all (finding 4 of the stage-6 review);
// worse, driving it through a sketch confounds the kernel's resolution floor with the sketch
// solver's own smallest workable circle. The corpus configuration has neither problem.
func TestASubResolutionCutSickensItsFeature(t *testing.T) {
	t.Parallel()
	ring, err := brep.SolidTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 1.5, "ring")
	if err != nil {
		t.Fatalf("ring: %v", err)
	}
	drill, err := brep.SolidCylinder(math.P3(5, 0, -4), math.V3(0, 0, 1), 1e-10, 8)
	if err != nil {
		t.Fatalf("drill: %v", err)
	}
	fs := NewPartFeatures(nil)
	base := NewBaseFeatures(fs).AddBase(ring, drill)
	cut := NewModifyFeatures(fs).AddCombine(0, 1, ops.Cut)

	fs.Recompute()
	if !base.Health().OK() {
		t.Fatalf("the base bodies are valid and must stay healthy: %+v", base.Health())
	}
	if cut.Health().Status != health.Sick {
		t.Fatalf("a sub-resolution cut must sicken its feature; got %+v", cut.Health())
	}
	if !hasDiagCode(cut.Diagnostics(), ops.CodeBooleanSubResolutionTool) {
		t.Errorf("the kernel's named refusal must reach feature health as %q; got %v",
			ops.CodeBooleanSubResolutionTool, cut.Diagnostics())
	}
	for _, want := range []string{"below this model's resolution", "working unit"} {
		if !strings.Contains(cut.Health().Reason, want) {
			t.Errorf("the sick reason must name %q so the user can act on it; got %q", want, cut.Health().Reason)
		}
	}
	// Failure is LOCAL: the two bodies the refused cut was to combine survive the recompute.
	if got := len(fs.Result()); got != 2 {
		t.Errorf("the refused cut must leave the operands standing; the result holds %d bodies", got)
	}
}

// TestAnOrdinaryBoreIsNotRefusedOnSize is the counterpart the floor needs: the RD- corpus row's own
// 0.8 bore goes through the same feature path and must NOT meet the size classification. A floor that
// refuses ordinary geometry would be worse than no floor at all.
func TestAnOrdinaryBoreIsNotRefusedOnSize(t *testing.T) {
	t.Parallel()
	ring, err := brep.SolidTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 1.5, "ring")
	if err != nil {
		t.Fatalf("ring: %v", err)
	}
	drill, err := brep.SolidCylinder(math.P3(5, 0, -4), math.V3(0, 0, 1), 0.8, 8)
	if err != nil {
		t.Fatalf("drill: %v", err)
	}
	fs := NewPartFeatures(nil)
	NewBaseFeatures(fs).AddBase(ring, drill)
	cut := NewModifyFeatures(fs).AddCombine(0, 1, ops.Cut)
	fs.Recompute()

	if !cut.Health().OK() {
		t.Fatalf("the RD- corpus bore must build: %+v", cut.Health())
	}
	if hasDiagCode(cut.Diagnostics(), ops.CodeBooleanSubResolutionTool) {
		t.Errorf("an ordinary bore must not meet the size classification; got %v", cut.Diagnostics())
	}
}
