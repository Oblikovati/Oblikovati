// SPDX-License-Identifier: GPL-2.0-only

package events

import (
	"encoding/json"
	"sync"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/math"
	"oblikovati.org/model/assembly"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
)

// constraintRecorder collects the relationship push events the relay forwards.
type constraintRecorder struct {
	mu  sync.Mutex
	got []wire.ConstraintEventPayload
}

func (r *constraintRecorder) sink(b []byte) {
	var tag struct {
		Type string `json:"type"`
	}
	if json.Unmarshal(b, &tag) != nil {
		return
	}
	switch tag.Type {
	case wire.EventAssemblyConstraintAdded, wire.EventAssemblyConstraintDeleted, wire.EventAssemblyResolved:
	default:
		return
	}
	var p wire.ConstraintEventPayload
	if json.Unmarshal(b, &p) == nil {
		r.mu.Lock()
		r.got = append(r.got, p)
		r.mu.Unlock()
	}
}

func (r *constraintRecorder) ofType(eventType string) []wire.ConstraintEventPayload {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []wire.ConstraintEventPayload
	for _, p := range r.got {
		if p.Type == eventType {
			out = append(out, p)
		}
	}
	return out
}

// TestSubscribeAssemblyRelaysConstraints checks adding, solving, and deleting a constraint
// surfaces the relationship events to an add-in sink, tagged with the document and kind.
func TestSubscribeAssemblyRelaysConstraints(t *testing.T) {
	asm := compdef.NewAssemblyComponentDefinition()
	var rec constraintRecorder
	subs := SubscribeAssembly(asm.Events().Bus(), doc.ID(7), rec.sink)
	defer cancelAll(subs)

	base := asm.Place("base:1", compdef.NewPartComponentDefinition(), math.Identity4())
	base.SetGrounded(true)
	moving := asm.Place("moving:1", compdef.NewPartComponentDefinition(), math.Translation4(math.V3(0, 0, 6)))
	zUp, _ := math.NewUnitVector3(0, 0, 1)
	zDown, _ := math.NewUnitVector3(0, 0, -1)

	m := asm.Constraints().AddMate(
		base, assembly.PlanePrimitive(math.P3(0, 0, 0), zUp),
		moving, assembly.PlanePrimitive(math.P3(0, 0, 0), zDown),
		0, types.MateSolutionOpposed)
	asm.SolveConstraints()
	asm.Constraints().Delete(m.ID())

	added := rec.ofType(wire.EventAssemblyConstraintAdded)
	if len(added) != 1 || added[0].Document != 7 || added[0].Constraint != m.ID() || added[0].Kind != "mate" {
		t.Fatalf("added events = %+v, want one mate on document 7", added)
	}
	if got := rec.ofType(wire.EventAssemblyResolved); len(got) != 1 || got[0].Document != 7 {
		t.Errorf("resolved events = %+v, want one on document 7", got)
	}
	if got := rec.ofType(wire.EventAssemblyConstraintDeleted); len(got) != 1 || got[0].Constraint != m.ID() {
		t.Errorf("deleted events = %+v, want one for the mate", got)
	}
}
