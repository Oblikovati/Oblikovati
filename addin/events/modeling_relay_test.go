// SPDX-License-Identifier: GPL-2.0-only

package events

import (
	"encoding/json"
	"testing"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/event"
)

// TestForwardsFeatureLifecycle: the granular feature-lifecycle events are relayed to the add-in
// sink with the affected feature's identity and the right type tag per op (#148).
func TestForwardsFeatureLifecycle(t *testing.T) {
	cases := []struct {
		op   app.FeatureOp
		want string
	}{
		{app.FeatureAdded, wire.EventFeatureAdded},
		{app.FeatureEdited, wire.EventFeatureEdited},
		{app.FeatureDeleted, wire.EventFeatureDeleted},
	}
	for _, tc := range cases {
		s := app.NewSession()
		var rec rawRecorder
		if subs := Subscribe(s, rec.sink); len(subs) == 0 {
			t.Fatal("Subscribe returned no subscriptions")
		}
		event.Emit(s.Events(), event.After, app.FeatureLifecycleChanged{
			Document: 5, Op: tc.op, Feature: 9, Name: "Extrusion1", Kind: "extrude",
		})
		if !findFeatureEvent(rec.got, tc.want) {
			t.Errorf("op %v: no %s event relayed with feature 9; got %d payloads", tc.op, tc.want, len(rec.got))
		}
	}
}

// findFeatureEvent reports whether the recorded payloads contain a feature-lifecycle event of the
// wanted type carrying feature 9 / Extrusion1 / extrude.
func findFeatureEvent(got []string, want string) bool {
	for _, raw := range got {
		var e wire.FeatureLifecycleEvent
		if json.Unmarshal([]byte(raw), &e) == nil && e.Type == want {
			if e.Document == 5 && e.Feature == 9 && e.Name == "Extrusion1" && e.Kind == "extrude" {
				return true
			}
		}
	}
	return false
}

// TestForwardsSketchEdit: entering and leaving a sketch relay the editEntered/editExited events
// with the sketch identity (#148).
func TestForwardsSketchEdit(t *testing.T) {
	for _, tc := range []struct {
		entered bool
		want    string
	}{
		{true, wire.EventSketchEditEntered},
		{false, wire.EventSketchEditExited},
	} {
		s := app.NewSession()
		var rec rawRecorder
		Subscribe(s, rec.sink)
		event.Emit(s.Events(), event.After, app.SketchEditChanged{
			Document: 5, Entered: tc.entered, Sketch: 3, Name: "Sketch1",
		})
		found := false
		for _, raw := range rec.got {
			var e wire.SketchEditEvent
			if json.Unmarshal([]byte(raw), &e) == nil && e.Type == tc.want {
				if e.Sketch != 3 || e.Name != "Sketch1" {
					t.Errorf("relayed sketch event = %+v, want Sketch1 (id 3)", e)
				}
				found = true
			}
		}
		if !found {
			t.Errorf("entered=%v: no %s event relayed; got %d payloads", tc.entered, tc.want, len(rec.got))
		}
	}
}
