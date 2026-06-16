// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/wire"
)

// TestFlatPatternOrientationsOverWire drives the M13-F05 orientation surface: a flanged sheet
// reports the default orientation with a positive length/width; adding a vertical orientation
// swaps them; activate/delete behave; the default is undeletable.
func TestFlatPatternOrientationsOverWire(t *testing.T) {
	r, s := flangedSheet(t)

	var list wire.OrientationsResult
	call(t, r, s, wire.MethodFlatPatternListOrientations, "{}", &list)
	if len(list.Orientations) != 1 || !list.Orientations[0].Active {
		t.Fatalf("default list = %+v, want one active orientation", list.Orientations)
	}
	def := list.Orientations[0]
	if def.Length <= 0 || def.Width <= 0 {
		t.Errorf("default length/width = %g×%g, want positive", def.Length, def.Width)
	}

	var added wire.OrientationResult
	call(t, r, s, wire.MethodFlatPatternAddOrientation, `{"name":"Vertical","alignmentType":"vertical","activate":true}`, &added)
	if !added.Orientation.Active || added.Orientation.AlignmentType != "vertical" {
		t.Errorf("added = %+v, want active vertical", added.Orientation)
	}
	if added.Orientation.Length != def.Width || added.Orientation.Width != def.Length {
		t.Errorf("vertical (%g×%g) should swap default (%g×%g)", added.Orientation.Length, added.Orientation.Width, def.Length, def.Width)
	}

	// Activate the default again, then delete the custom one.
	var act wire.OrientationResult
	call(t, r, s, wire.MethodFlatPatternActivateOrientation, `{"name":"Flat Pattern Default"}`, &act)
	if !act.Orientation.Active || act.Orientation.Name != "Flat Pattern Default" {
		t.Errorf("activated = %+v, want the default active", act.Orientation)
	}
	var afterDelete wire.OrientationsResult
	call(t, r, s, wire.MethodFlatPatternDeleteOrientation, `{"name":"Vertical"}`, &afterDelete)
	if len(afterDelete.Orientations) != 1 {
		t.Errorf("after delete = %d orientations, want 1", len(afterDelete.Orientations))
	}

	// Error paths.
	if _, err := r.Handle(s, wire.MethodFlatPatternDeleteOrientation, []byte(`{"name":"Flat Pattern Default"}`)); err == nil {
		t.Error("deleting the default orientation must error")
	}
	if _, err := r.Handle(s, wire.MethodFlatPatternActivateOrientation, []byte(`{"name":"Nope"}`)); err == nil {
		t.Error("activating an unknown orientation must error")
	}
}

// TestFlatPatternRejectsPlainPart the flat-pattern surface errors on an ordinary part.
func TestFlatPatternRejectsPlainPart(t *testing.T) {
	r, s := seededSession(t)
	if _, err := r.Handle(s, wire.MethodFlatPatternListOrientations, []byte("{}")); err == nil {
		t.Fatal("flatPattern on a plain part must error")
	}
}
