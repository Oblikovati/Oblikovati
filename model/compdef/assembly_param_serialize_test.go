// SPDX-License-Identifier: GPL-2.0-only

package compdef_test

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/param"
	"oblikovati.org/model/sketch"
)

// TestAssemblyUserParametersRoundTrip checks that an assembly is a first-class parameter
// holder: numeric, text, and boolean user parameters authored on an assembly survive a full
// .obk save/reopen through the store (M39-F01, #1557). Before the fix assemblyRecipe carried
// no parameters field, so every assembly user parameter was silently lost on save.
func TestAssemblyUserParametersRoundTrip(t *testing.T) {
	store, ws, dir := assemblyWorkspace(t)
	asm, asmDef := newAssembly(t, ws, dir, "params.obk")
	if _, err := asmDef.Parameters().AddUserParameter("plateWidth", "40 mm"); err != nil { // 4 cm
		t.Fatalf("AddUserParameter: %v", err)
	}
	if _, err := asmDef.Parameters().AddTextUserParameter("vendor", "Acme"); err != nil {
		t.Fatalf("AddTextUserParameter: %v", err)
	}
	if _, err := asmDef.Parameters().AddBooleanUserParameter("machined", true); err != nil {
		t.Fatalf("AddBooleanUserParameter: %v", err)
	}

	def := reopenAssembly(t, store, ws, asm)
	width, ok := def.Parameters().ByName("plateWidth")
	if !ok {
		t.Fatalf("plateWidth missing after reopen")
	}
	if width.Kind() != param.UserParam {
		t.Errorf("plateWidth kind = %v, want UserParam", width.Kind())
	}
	if v := width.Value().Value; v < 3.999 || v > 4.001 {
		t.Errorf("plateWidth value = %v db units, want ~4 (40 mm)", v)
	}
	if vendor, ok := def.Parameters().ByName("vendor"); !ok || !vendor.IsText() || vendor.Text() != "Acme" {
		t.Errorf("vendor text param did not round-trip: ok=%v", ok)
	}
	if machined, ok := def.Parameters().ByName("machined"); !ok || !machined.IsBoolean() || !machined.Bool() {
		t.Errorf("machined boolean param did not round-trip: ok=%v", ok)
	}
}

// TestAssemblyParameterDrivenDimensionResolvesAfterReload is the assembly counterpart of the
// part-side TestParameterDrivenDimensionResolvesAfterRestore: an assembly sketch dimension
// whose expression names an assembly parameter must still resolve after a save/reopen. With
// the parameter table now persisted and restored before the sketches, "width" rebinds by name
// and the dimension measures its 40 mm target; the pre-fix regression (table lost on save)
// left "width" unknown so the dimension collapsed toward 0 (M39-F01, #1557).
func TestAssemblyParameterDrivenDimensionResolvesAfterReload(t *testing.T) {
	store, ws, dir := assemblyWorkspace(t)
	asm, asmDef := newAssembly(t, ws, dir, "driven.obk")
	if _, err := asmDef.Parameters().AddUserParameter("width", "40 mm"); err != nil {
		t.Fatalf("AddUserParameter: %v", err)
	}
	sk := asmDef.AddSketch(sketch.XYPlane(), nil)
	a := sk.Points().Add(math.P2(0, 0))
	b := sk.Points().Add(math.P2(4, 0)) // 4 cm == 40 mm
	if _, err := sk.DimensionConstraints().AddDistance(a, b, "width"); err != nil {
		t.Fatalf("AddDistance: %v", err)
	}

	def := reopenAssembly(t, store, ws, asm)
	if def.Sketches().Count() != 1 {
		t.Fatalf("reopened assembly has %d sketches, want 1", def.Sketches().Count())
	}
	got := def.Sketches().Item(0).DimensionConstraints().Item(0)
	if measured := got.Measured(); measured < 3.999 || measured > 4.001 {
		t.Errorf("restored param-driven assembly dimension measured %v, want ~4 (40 mm)", measured)
	}
}

// TestAssemblyParametersResetOnSnapshotRestore guards the snapshot (undo/redo) path: because
// RestoreSnapshot is a full replace, resetOccurrences must clear the parameter table before
// re-applying, or undoing back to a state with fewer parameters would leave stale ones behind
// (or duplicate-name errors on re-add). M39-F01, #1557.
func TestAssemblyParametersResetOnSnapshotRestore(t *testing.T) {
	asmDef := compdef.NewAssemblyComponentDefinition()
	baseline, err := asmDef.MarshalSnapshot() // zero parameters
	if err != nil {
		t.Fatalf("MarshalSnapshot: %v", err)
	}
	if _, err := asmDef.Parameters().AddUserParameter("temp", "10 mm"); err != nil {
		t.Fatalf("AddUserParameter: %v", err)
	}
	if err := asmDef.RestoreSnapshot(baseline); err != nil { // undo the add
		t.Fatalf("RestoreSnapshot: %v", err)
	}
	if _, ok := asmDef.Parameters().ByName("temp"); ok {
		t.Error("parameter 'temp' survived a restore to the pre-add snapshot (table not reset)")
	}
	// Re-adding the same name must not collide with a stale entry.
	if _, err := asmDef.Parameters().AddUserParameter("temp", "20 mm"); err != nil {
		t.Errorf("re-add after restore failed (stale parameter left behind?): %v", err)
	}
}
