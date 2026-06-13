// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	gmath "oblikovati.org/math"
)

// unitBlock builds the solid box [0,1]³ (volume 1) used as an assembly-cut target.
func unitBlock(t *testing.T) *topo.Body {
	t.Helper()
	b, err := brep.SolidBlock(gmath.P3(0, 0, 0), gmath.P3(1, 1, 1), "target")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	return b
}

// TestAssemblyCutRemovesToolVolume gates the cut against the analytic value: a tool
// covering the top half of a unit box removes exactly 0.5 of its volume.
func TestAssemblyCutRemovesToolVolume(t *testing.T) {
	tool, err := brep.SolidBlock(gmath.P3(-1, -1, 0.5), gmath.P3(2, 2, 2), "tool")
	if err != nil {
		t.Fatalf("SolidBlock tool: %v", err)
	}
	f := NewAssemblyCutFeature(tool, ops.Cut)

	out, err := f.Recompute(Input{Bodies: []*topo.Body{unitBlock(t)}})
	if err != nil {
		t.Fatalf("Recompute: %v", err)
	}
	if len(out.Bodies) != 1 {
		t.Fatalf("result bodies = %d, want 1", len(out.Bodies))
	}
	if got := bodyVolume(out.Bodies[0]); math.Abs(got-0.5) > 1e-6 {
		t.Errorf("cut volume = %g, want 0.5 (top half removed)", got)
	}
}

// TestAssemblyCutAppliesToEveryBody confirms the tool machines each running body, so
// applying it across N targets cuts all N (the assembly host relies on this to machine
// a participant's several bodies in one Recompute).
func TestAssemblyCutAppliesToEveryBody(t *testing.T) {
	tool, _ := brep.SolidBlock(gmath.P3(-1, -1, 0.5), gmath.P3(2, 2, 2), "tool")
	f := NewAssemblyCutFeature(tool, ops.Cut)

	out, err := f.Recompute(Input{Bodies: []*topo.Body{unitBlock(t), unitBlock(t)}})
	if err != nil {
		t.Fatalf("Recompute: %v", err)
	}
	if len(out.Bodies) != 2 {
		t.Fatalf("result bodies = %d, want 2", len(out.Bodies))
	}
	for i, b := range out.Bodies {
		if got := bodyVolume(b); math.Abs(got-0.5) > 1e-6 {
			t.Errorf("body %d cut volume = %g, want 0.5", i, got)
		}
	}
}

// TestAssemblyCutDropsFullyConsumedBody: a tool that fully encloses the target removes
// the whole body, so the result drops it rather than carrying an empty body.
func TestAssemblyCutDropsFullyConsumedBody(t *testing.T) {
	tool, _ := brep.SolidBlock(gmath.P3(-1, -1, -1), gmath.P3(2, 2, 2), "tool")
	f := NewAssemblyCutFeature(tool, ops.Cut)

	out, err := f.Recompute(Input{Bodies: []*topo.Body{unitBlock(t)}})
	if err != nil {
		t.Fatalf("Recompute: %v", err)
	}
	if len(out.Bodies) != 0 {
		t.Errorf("result bodies = %d, want 0 (target fully consumed)", len(out.Bodies))
	}
}

// TestAssemblyHoleRemovesCylinder gates the parametric assembly hole against the
// analytic value: a through-hole of radius 0.25 drilled along +z removes a faceted
// cylinder (its 32-gon cross-section × the unit box's height) from the box.
func TestAssemblyHoleRemovesCylinder(t *testing.T) {
	axis, _ := gmath.NewUnitVector3(0, 0, 1)
	f, err := NewAssemblyHoleFeature(gmath.P3(0.5, 0.5, 0), axis, 0.5, 1.5)
	if err != nil {
		t.Fatalf("NewAssemblyHoleFeature: %v", err)
	}
	if f.Kind() != "assemblyHole" {
		t.Errorf("kind = %q, want assemblyHole", f.Kind())
	}

	out, err := f.Recompute(Input{Bodies: []*topo.Body{unitBlock(t)}})
	if err != nil {
		t.Fatalf("Recompute: %v", err)
	}
	if len(out.Bodies) != 1 {
		t.Fatalf("result bodies = %d, want 1", len(out.Bodies))
	}
	// Expected: unit box minus the faceted bore (its 32-gon cross-section × box height 1).
	boreArea := 0.5 * float64(holeFacets) * 0.25 * 0.25 * math.Sin(2*math.Pi/float64(holeFacets))
	want := 1.0 - boreArea*1.0
	if got := bodyVolume(out.Bodies[0]); math.Abs(got-want) > 1e-6 {
		t.Errorf("holed volume = %g, want %g (box minus the faceted bore)", got, want)
	}
}

// TestAssemblyHoleRejectsBadDimensions: non-positive diameter or depth is an error.
func TestAssemblyHoleRejectsBadDimensions(t *testing.T) {
	axis, _ := gmath.NewUnitVector3(0, 0, 1)
	if _, err := NewAssemblyHoleFeature(gmath.P3(0, 0, 0), axis, 0, 1); err == nil {
		t.Error("zero diameter should be rejected")
	}
}

// TestAssemblyCutNilToolFails: a missing tool is a lost input the engine can turn into
// feature health, reported as an error rather than a panic.
func TestAssemblyCutNilToolFails(t *testing.T) {
	f := NewAssemblyCutFeature(nil, ops.Cut)
	if _, err := f.Recompute(Input{Bodies: []*topo.Body{unitBlock(t)}}); err == nil {
		t.Fatal("Recompute with nil tool returned nil error, want a failure")
	}
}
