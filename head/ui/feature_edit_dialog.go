//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"github.com/Oblikovati/oblikovati/app"
	"github.com/Oblikovati/oblikovati/head/internal/native"
)

// The feature-edit flow in the head: double-clicking a feature in the model browser
// opens this dialog bound to the feature's parameters (today an extrude's distance). OK
// recomputes the part with the new values; Cancel restores the values captured when the
// edit opened. The distance is in the document's length unit (e.g. mm), matching the
// extrude tool's dialog.

// featureEditUI holds the dialog's distance field across frames and whether it was open
// last frame (so it seeds the field once when the edit opens).
var featureEditUI struct {
	distance float32
	open     bool
}

// drawFeatureEditDialog shows the parameter editor while a feature edit is open and keeps
// the feature's distance in sync with the field each frame; OK commits, Cancel reverts.
func drawFeatureEditDialog(s *app.Session) {
	if !s.IsEditingFeature() {
		featureEditUI.open = false
		return
	}
	if !featureEditUI.open { // edit just opened — seed the field from the feature's value
		featureEditUI.distance = float32(s.EditFeatureDistanceDisplay())
		featureEditUI.open = true
	}
	native.SetNextWindowSize(280, 132)
	if native.Begin("Edit Feature") {
		native.Text(s.EditingFeatureName())
		native.Text("Distance (" + s.LengthUnitName() + ")")
		native.InputFloat("##edit-feature-distance", &featureEditUI.distance)
		s.SetEditFeatureDistanceDisplay(float64(featureEditUI.distance)) // keep the feature in sync
		if native.Button("OK") {
			if err := s.CommitFeatureEdit(); err == nil { // a sick result keeps the dialog open
				featureEditUI.open = false
			}
		}
		native.SameLine()
		if native.Button("Cancel") {
			s.CancelFeatureEdit()
			featureEditUI.open = false
		}
	}
	native.End()
}
