// SPDX-License-Identifier: GPL-2.0-only

package events

import (
	"encoding/json"
	"sync"
	"testing"

	"oblikovati.org/api/wire"
	"oblikovati.org/math"
	"oblikovati.org/model/assembly"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
)

// jointRecorder collects the joint push events the relay forwards.
type jointRecorder struct {
	mu  sync.Mutex
	got []wire.JointEventPayload
}

func (r *jointRecorder) sink(b []byte) {
	var tag struct {
		Type string `json:"type"`
	}
	if json.Unmarshal(b, &tag) != nil {
		return
	}
	if tag.Type != wire.EventAssemblyJointAdded && tag.Type != wire.EventAssemblyJointDeleted {
		return
	}
	var p wire.JointEventPayload
	if json.Unmarshal(b, &p) == nil {
		r.mu.Lock()
		r.got = append(r.got, p)
		r.mu.Unlock()
	}
}

func (r *jointRecorder) ofType(eventType string) []wire.JointEventPayload {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []wire.JointEventPayload
	for _, p := range r.got {
		if p.Type == eventType {
			out = append(out, p)
		}
	}
	return out
}

// TestSubscribeAssemblyRelaysJoints checks adding and deleting a joint surfaces the joint
// events to an add-in sink, tagged with the document and kind (M12-F02).
func TestSubscribeAssemblyRelaysJoints(t *testing.T) {
	asm := compdef.NewAssemblyComponentDefinition()
	var rec jointRecorder
	subs := SubscribeAssembly(asm.Events().Bus(), doc.ID(9), rec.sink)
	defer cancelAll(subs)

	base := asm.Place("base:1", compdef.NewPartComponentDefinition(), math.Identity4())
	moving := asm.Place("moving:1", compdef.NewPartComponentDefinition(), math.Identity4())
	z, _ := math.NewUnitVector3(0, 0, 1)
	j := asm.Joints().AddRotational(
		assembly.Ref{Occurrence: base, Primitive: assembly.LinePrimitive(math.P3(0, 0, 0), z)},
		assembly.Ref{Occurrence: moving, Primitive: assembly.LinePrimitive(math.P3(0, 0, 0), z)})
	asm.Joints().Delete(j.ID())

	added := rec.ofType(wire.EventAssemblyJointAdded)
	if len(added) != 1 || added[0].Document != 9 || added[0].Joint != j.ID() || added[0].Kind != "rotational" {
		t.Fatalf("added events = %+v, want one rotational joint on document 9", added)
	}
	if got := rec.ofType(wire.EventAssemblyJointDeleted); len(got) != 1 || got[0].Joint != j.ID() {
		t.Errorf("deleted events = %+v, want one for the joint", got)
	}
}
