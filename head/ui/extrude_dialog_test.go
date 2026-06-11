//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/model/feature"
)

// TestExtrudeDirectionToggleRoundTrip locks the Direction row's mapping: applying
// toggle i reads back as active index i, and only the fourth toggle (asymmetric) turns
// the two-distance mode on.
func TestExtrudeDirectionToggleRoundTrip(t *testing.T) {
	ext := app.NewExtrudeTool()
	for i := 0; i < 4; i++ {
		applyExtrudeDirection(ext, i)
		if got := extrudeDirectionIndex(ext); got != i {
			t.Errorf("after applyExtrudeDirection(%d), index = %d, want %d", i, got, i)
		}
		if ext.Asymmetric() != (i == 3) {
			t.Errorf("after applyExtrudeDirection(%d), Asymmetric() = %v", i, ext.Asymmetric())
		}
	}
}

// TestExtrudeExtentIndexMapping locks the extent toggle order against the tool's
// extent types (Distance, Through All, To Next).
func TestExtrudeExtentIndexMapping(t *testing.T) {
	ext := app.NewExtrudeTool()
	for i, et := range extrudeExtentTypes {
		ext.SetExtentType(et)
		if got := extrudeExtentIndex(ext); got != i {
			t.Errorf("extent %v maps to toggle %d, want %d", et, got, i)
		}
	}
	if extrudeExtentTypes[0] != feature.DistanceExtent {
		t.Error("the first extent toggle must be Distance (the panel's editable-field case)")
	}
}

// TestBooleanToggleIndexMapping locks the shared Boolean toggle order (Join, Cut,
// Intersect, New Solid — the reference panel's order) against the feature operation.
func TestBooleanToggleIndexMapping(t *testing.T) {
	for i, op := range booleanToggleOps {
		if got := booleanToggleIndex(op); got != i {
			t.Errorf("operation %v maps to toggle %d, want %d", op, got, i)
		}
	}
}

// TestExtrudeProfileChipText locks the Profiles chip caption per pick count: the
// required prompt when empty, then a count.
func TestExtrudeProfileChipText(t *testing.T) {
	ext := app.NewExtrudeTool()
	if got := extrudeProfileChipText(ext); got != "Select Profile" {
		t.Errorf("empty chip text = %q, want \"Select Profile\"", got)
	}
	ext.Pick(nil, app.ProfileHandle{ProfileIndex: 0})
	if got := extrudeProfileChipText(ext); got != "1 Profile" {
		t.Errorf("one-pick chip text = %q, want \"1 Profile\"", got)
	}
	ext.PickWithMods(nil, app.ProfileHandle{ProfileIndex: 1}, app.CtrlMod)
	if got := extrudeProfileChipText(ext); got != "2 Profiles" {
		t.Errorf("two-pick chip text = %q, want \"2 Profiles\"", got)
	}
}
