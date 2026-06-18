// SPDX-License-Identifier: GPL-2.0-only

package events

import (
	"encoding/json"
	"testing"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/event"
)

// TestForwardsParameterChanged: the granular parameter-change event emitted by parameters.set is
// relayed to the add-in sink with the parameter's new state (#148).
func TestForwardsParameterChanged(t *testing.T) {
	s := app.NewSession()
	var rec rawRecorder
	if subs := Subscribe(s, rec.sink); len(subs) == 0 {
		t.Fatal("Subscribe returned no subscriptions")
	}

	event.Emit(s.Events(), event.After, app.ParameterChanged{
		Document: 7, Name: "od", Kind: "model", Expression: "20 mm", Value: "20 mm",
	})

	for _, raw := range rec.got {
		var e wire.ParameterChangedEvent
		if json.Unmarshal([]byte(raw), &e) == nil && e.Type == wire.EventParameterChanged {
			if e.Document != 7 || e.Parameter.Name != "od" || e.Parameter.Expression != "20 mm" {
				t.Errorf("relayed parameter event = %+v, want od=20 mm on document 7", e)
			}
			return
		}
	}
	t.Errorf("no parameters.changed event relayed; got %d payloads", len(rec.got))
}
