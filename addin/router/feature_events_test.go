// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/event"
)

// TestFeaturesAddEmitsFeatureAdded: a features.add over the router emits a FeatureLifecycleChanged
// (FeatureAdded) carrying the new feature's identity, so the add-in relay forwards feature.added
// (#148).
func TestFeaturesAddEmitsFeatureAdded(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t) // active part with one rectangle profile on sketch 0

	var got []app.FeatureLifecycleChanged
	event.Subscribe(s.Events(), event.After, func(_ event.Context, e app.FeatureLifecycleChanged) event.Outcome {
		got = append(got, e)
		return event.Continue()
	})

	var ext extrudeBodies
	call(t, r, s, "features.add", `{"kind":"extrude","args":{"sketchIndex":0,"distance":"5 mm"}}`, &ext)

	var added *app.FeatureLifecycleChanged
	for i := range got {
		if got[i].Op == app.FeatureAdded {
			added = &got[i]
		}
	}
	if added == nil {
		t.Fatalf("features.add emitted no FeatureAdded event; got %d events", len(got))
	}
	if added.Kind != "extrude" || added.Feature == 0 {
		t.Errorf("FeatureAdded = %+v, want kind extrude with a non-zero feature id", *added)
	}
}
