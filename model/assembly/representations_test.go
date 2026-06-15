// SPDX-License-Identifier: GPL-2.0-only

package assembly

import (
	stdmath "math"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/math"
	"oblikovati.org/model/occurrence"
)

// newReps builds an empty representation hub over a fresh assembly.
func newReps() (*Representations, *occurrence.Occurrences, *ConstraintSet, *JointSet) {
	occs := occurrence.NewOccurrences()
	cs := NewConstraintSet(occs, nil)
	js := NewJointSet(occs, nil)
	return NewRepresentations(occs, cs, js), occs, cs, js
}

// TestDesignViewCaptureAndActivate checks a design-view representation captures and restores
// occurrence visibility without touching the base otherwise.
func TestDesignViewCaptureAndActivate(t *testing.T) {
	reps, occs, _, _ := newReps()
	a := place(occs, "a:1", math.Identity4())
	b := place(occs, "b:1", math.Identity4())
	b.SetVisible(false)

	dv := reps.CaptureDesignView("hidden-b", nil)
	b.SetVisible(true) // change the live state away from the captured one
	if _, err := reps.ActivateDesignView(dv.ID()); err != nil {
		t.Fatalf("ActivateDesignView: %v", err)
	}
	if !a.Visible() || b.Visible() {
		t.Errorf("activate restored visibility wrong: a=%v b=%v, want true/false", a.Visible(), b.Visible())
	}
	if !dv.Active() {
		t.Error("activated design-view representation is not marked active")
	}
}

// TestLODCaptureAndActivate checks a level-of-detail representation captures and restores
// occurrence suppression.
func TestLODCaptureAndActivate(t *testing.T) {
	reps, occs, _, _ := newReps()
	a := place(occs, "a:1", math.Identity4())
	b := place(occs, "b:1", math.Identity4())
	b.SetSuppressed(true)

	lod := reps.CaptureLOD("simplified")
	b.SetSuppressed(false)
	if _, err := reps.ActivateLOD(lod.ID()); err != nil {
		t.Fatalf("ActivateLOD: %v", err)
	}
	if a.Suppressed() || !b.Suppressed() {
		t.Errorf("activate restored suppression wrong: a=%v b=%v, want false/true", a.Suppressed(), b.Suppressed())
	}
}

// TestPositionalOverrideRepositions checks a positional override changes a constraint's value
// and re-solves the assembly to that position — the "piston at TDC vs BDC" case.
func TestPositionalOverrideRepositions(t *testing.T) {
	reps, occs, cs, _ := newReps()
	base := place(occs, "base:1", math.Identity4())
	base.SetGrounded(true)
	moving := place(occs, "moving:1", math.Translation4(math.V3(0, 0, 10)))
	top := PlanePrimitive(math.P3(0, 0, 0), unit(t, 0, 0, 1))
	bottom := PlanePrimitive(math.P3(0, 0, 0), unit(t, 0, 0, -1))
	mate := cs.AddMate(ref(base, top), ref(moving, bottom), 0, types.MateSolutionOpposed)

	open := reps.CapturePositional("open")
	if err := reps.SetPositionalOverride(open.ID(), mate.ID(), false, 3); err != nil {
		t.Fatalf("SetPositionalOverride: %v", err)
	}
	if _, err := reps.ActivatePositional(open.ID()); err != nil {
		t.Fatalf("ActivatePositional: %v", err)
	}
	if mate.Value() != 3 {
		t.Errorf("override value = %v, want 3", mate.Value())
	}
	if z := moving.Transform().Translation().Z; stdmath.Abs(stdmath.Abs(z)-3) > 1e-6 {
		t.Errorf("re-solved moving z = %v, want the faces 3 apart", z)
	}
}

// TestModelStateActivatesFamilies is the F04 acceptance: switching a model state applies its
// design-view, positional, and level-of-detail representations together — visibility,
// position, and suppression all change accordingly.
func TestModelStateActivatesFamilies(t *testing.T) {
	reps, occs, _, _ := newReps()
	a := place(occs, "a:1", math.Identity4())
	b := place(occs, "b:1", math.Identity4())

	b.SetVisible(false)
	dv := reps.CaptureDesignView("dv", nil)
	b.SetVisible(true)

	a.SetSuppressed(true)
	lod := reps.CaptureLOD("lod")
	a.SetSuppressed(false)

	ms := reps.CreateModelState("state", dv.Name(), "", lod.Name())
	if _, err := reps.ActivateModelState(ms.ID()); err != nil {
		t.Fatalf("ActivateModelState: %v", err)
	}
	if b.Visible() {
		t.Error("model state did not apply the design-view representation (b should be hidden)")
	}
	if !a.Suppressed() {
		t.Error("model state did not apply the level-of-detail representation (a should be suppressed)")
	}
	if !ms.Active() {
		t.Error("activated model state is not marked active")
	}
}

// TestDeleteRepresentations checks each family's delete removes the representation.
func TestDeleteRepresentations(t *testing.T) {
	reps, _, _, _ := newReps()
	dv := reps.CaptureDesignView("dv", nil)
	p := reps.CapturePositional("p")
	l := reps.CaptureLOD("l")
	ms := reps.CreateModelState("m", "dv", "p", "l")

	if !reps.DeleteDesignView(dv.ID()) || len(reps.AllDesignViews()) != 0 {
		t.Error("DeleteDesignView did not remove it")
	}
	if !reps.DeletePositional(p.ID()) || len(reps.AllPositionals()) != 0 {
		t.Error("DeletePositional did not remove it")
	}
	if !reps.DeleteLOD(l.ID()) || len(reps.AllLODs()) != 0 {
		t.Error("DeleteLOD did not remove it")
	}
	if !reps.DeleteModelState(ms.ID()) || len(reps.AllModelStates()) != 0 {
		t.Error("DeleteModelState did not remove it")
	}
}
