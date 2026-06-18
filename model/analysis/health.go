// SPDX-License-Identifier: GPL-2.0-only

package analysis

import (
	"oblikovati.org/model/feature"
	"oblikovati.org/model/health"
)

// Model health aggregation (M18-F02 #430): the design-doctor roll-up of a part's feature health.
// Each feature already carries its own health.Health (a recompute or bind failure becomes Sick on
// the feature, never a panic); this enumerates the unhealthy ones and reports the worst status so
// the UI can list everything that needs repair.

// FeatureHealthItem is one feature's aggregated health: its name, status and the reason it is unwell.
type FeatureHealthItem struct {
	Name   string
	Status health.Status
	Reason string
}

// ModelHealth is a part's aggregated feature health.
type ModelHealth struct {
	Overall   health.Status // the most severe status across the part's features (suppressed ⇒ OK)
	SickCount int
	Unhealthy []FeatureHealthItem // every feature that is not OK, for repair
}

// ModelHealthOf aggregates the health of a part's features.
func ModelHealthOf(features *feature.PartFeatures) ModelHealth {
	mh := ModelHealth{Overall: health.OK}
	for i := 0; i < features.Count(); i++ {
		f := features.Item(i)
		status, reason := featureStatus(f)
		if status == health.OK {
			continue
		}
		mh.Unhealthy = append(mh.Unhealthy, FeatureHealthItem{Name: f.Name(), Status: status, Reason: reason})
		if status == health.Sick {
			mh.SickCount++
		}
		if severity(status) > severity(mh.Overall) {
			mh.Overall = status
		}
	}
	return mh
}

// featureStatus reads a feature's status, treating a suppressed-but-otherwise-OK feature as
// Suppressed (the suppression is the salient state).
func featureStatus(f *feature.PartFeature) (health.Status, string) {
	h := f.Health()
	if f.Suppressed() && h.Status == health.OK {
		return health.Suppressed, ""
	}
	return h.Status, h.Reason
}

// severity orders statuses for the overall roll-up: OK < Warning < Sick, with Suppressed neutral
// (an intentionally excluded feature does not make the model worse).
func severity(s health.Status) int {
	switch s {
	case health.Warning:
		return 1
	case health.Sick:
		return 2
	default: // OK, Suppressed
		return 0
	}
}
