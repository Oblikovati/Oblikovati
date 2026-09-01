// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"testing"

	"oblikovati.org/model/param"
)

// TestParameterIntrospectionRoundTrip: a renamed model parameter and a disabled-action
// mask survive the parameter recipe round-trip (#1853). Without the recipe fields a
// reopened model parameter would lose Renamed (re-added under its stored name) and its
// DisabledActionTypes would reset to none.
func TestParameterIntrospectionRoundTrip(t *testing.T) {
	t.Parallel()
	src := param.NewParameters()
	m, err := src.AddModelParameter("d0", "5 mm")
	if err != nil {
		t.Fatalf("AddModelParameter: %v", err)
	}
	if err := src.Rename(m.ID(), "thickness"); err != nil {
		t.Fatalf("Rename: %v", err)
	}
	m.SetDisabledActionTypes(param.ActionRename | param.ActionDelete)

	recs := parametersRecipeOf(src)
	dst := param.NewParameters()
	if err := applyParametersTo(dst, nil, recs); err != nil {
		t.Fatalf("applyParametersTo: %v", err)
	}

	got, ok := dst.ByName("thickness")
	if !ok {
		t.Fatalf("thickness missing after round-trip")
	}
	if !got.Renamed() {
		t.Error("restored parameter lost Renamed()")
	}
	if got.DisabledActionTypes() != param.ActionRename|param.ActionDelete {
		t.Errorf("restored mask = %v, want rename|delete", got.DisabledActionTypes())
	}
}
