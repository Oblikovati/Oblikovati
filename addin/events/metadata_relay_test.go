// SPDX-License-Identifier: GPL-2.0-only

package events

import (
	"encoding/json"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/event"
)

// TestForwardsObjectRenamed: a metadata rename raised on the session bus is relayed to the add-in
// sink as an object.renamed wire event carrying the object kind, key, and old/new names (#1644).
func TestForwardsObjectRenamed(t *testing.T) {
	s := app.NewSession()
	var rec rawRecorder
	if subs := Subscribe(s, rec.sink); len(subs) == 0 {
		t.Fatal("Subscribe returned no subscriptions")
	}
	event.Emit(s.Events(), event.After, app.ObjectRenamed{
		Document: 5, Kind: types.ObjectKindFeature, Key: "7", OldName: "Extrusion1", NewName: "Base",
	})
	if !hasObjectRenamed(rec.got) {
		t.Errorf("no object.renamed relayed; got %d payloads", len(rec.got))
	}
}

func hasObjectRenamed(got []string) bool {
	for _, raw := range got {
		var e wire.ObjectRenamedEvent
		if json.Unmarshal([]byte(raw), &e) == nil && e.Type == wire.EventObjectRenamed &&
			e.Document == 5 && e.Kind == types.ObjectKindFeature && e.Key == "7" &&
			e.OldName == "Extrusion1" && e.NewName == "Base" {
			return true
		}
	}
	return false
}

// TestForwardsPropertyChanged: a suppression change raised on the session bus is relayed as a
// property.changed wire event carrying the property name and old/new values (#1644).
func TestForwardsPropertyChanged(t *testing.T) {
	s := app.NewSession()
	var rec rawRecorder
	if subs := Subscribe(s, rec.sink); len(subs) == 0 {
		t.Fatal("Subscribe returned no subscriptions")
	}
	event.Emit(s.Events(), event.After, app.PropertyChanged{
		Document: 5, Kind: types.ObjectKindFeature, Key: "7", Property: "suppressed", OldValue: "false", NewValue: "true",
	})
	if !hasPropertyChanged(rec.got) {
		t.Errorf("no property.changed relayed; got %d payloads", len(rec.got))
	}
}

func hasPropertyChanged(got []string) bool {
	for _, raw := range got {
		var e wire.PropertyChangedEvent
		if json.Unmarshal([]byte(raw), &e) == nil && e.Type == wire.EventPropertyChanged &&
			e.Property == "suppressed" && e.OldValue == "false" && e.NewValue == "true" {
			return true
		}
	}
	return false
}
