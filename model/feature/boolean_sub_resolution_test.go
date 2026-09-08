// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"strings"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/model/health"
)

// TestASubResolutionCutSickensItsFeature carries the kernel's size classification all the way to the
// browser. A four-metre plate pocketed by a 0.4 mm circle asks for a feature two millionths of the
// part: below the model's seam resolution, so the kernel refuses it by name before building anything.
// What the user must SEE is a sick feature naming the offending size and the remedy — not a healthy
// tick over a plate with no pocket in it (ADR-0061 stage 6).
func TestASubResolutionCutSickensItsFeature(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	extrudes := NewExtrudeFeatures(fs)
	const plate, bore = 400.0, 2e-4
	base := extrudes.AddByDistanceExtent(squareSketch(plate), 0, ops.NewBody, func() float64 { return plate / 2 })
	pocket := extrudes.AddByDistanceExtent(circleSketchAt(plate/2, plate/2, bore), 0, ops.Cut, func() float64 { return plate / 2 })

	fs.Recompute()
	if !base.Health().OK() {
		t.Fatalf("the base extrude must stay healthy: %+v", base.Health())
	}
	if pocket.Health().Status != health.Sick {
		t.Fatalf("a sub-resolution cut must sicken its feature; got %+v", pocket.Health())
	}
	if !hasDiagCode(pocket.Diagnostics(), ops.CodeBooleanSubResolutionTool) {
		t.Errorf("the kernel's named refusal must reach feature health as %q; got %v",
			ops.CodeBooleanSubResolutionTool, pocket.Diagnostics())
	}
	for _, want := range []string{"thinner than the model's seam resolution", "working unit"} {
		if !strings.Contains(pocket.Health().Reason, want) {
			t.Errorf("the sick reason must name %q so the user can act on it; got %q", want, pocket.Health().Reason)
		}
	}
	// Failure is LOCAL: the plate the refused pocket was cut from survives the recompute.
	if got := len(fs.Result()); got != 1 {
		t.Errorf("the refused cut must leave the base plate standing; the result holds %d bodies", got)
	}
}
