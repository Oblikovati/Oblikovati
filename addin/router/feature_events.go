// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"oblikovati.org/addin/modelaccess"
	"oblikovati.org/app"
	"oblikovati.org/model/feature"
)

// lastPartFeature returns the active part's most recently added feature, or nil — the feature a
// successful features.add just appended (every part-feature op appends one feature at the end).
// The features.add handler passes it to Session.EmitFeatureLifecycle so the add-in relay forwards
// feature.added for the programmatic path, alongside the UI path's tool-commit emit (#1085).
func lastPartFeature(s *app.Session) *feature.PartFeature {
	part, err := modelaccess.ActivePart(s)
	if err != nil || part == nil {
		return nil
	}
	feats := part.Features()
	if feats.Count() == 0 {
		return nil
	}
	return feats.Item(feats.Count() - 1)
}
