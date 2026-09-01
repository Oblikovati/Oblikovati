// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
)

// TestAttributesOverWire drives the #155 add-in attribute round trip over the router: set a typed
// value in a named set on the document, read it back, list it, enumerate the sets, find it, then
// delete it.
func TestAttributesOverWire(t *testing.T) {
	t.Parallel()
	r, s := emptyPartSession(t)
	id := uint64(s.ActiveDocument().ID())
	const set = "com.acme.bom"

	var setRes wire.AttributeResult
	call(t, r, s, "attributes.set",
		mustJSON(t, wire.SetAttributeArgs{Document: id, Set: set, Name: "partNo", Value: types.StringVariant("BRK-001")}), &setRes)
	if !setRes.Found {
		t.Fatalf("set reply not found: %+v", setRes)
	}
	if v, ok := setRes.Attribute.Value.Str(); !ok || v != "BRK-001" {
		t.Fatalf("set value = %v (ok=%v), want BRK-001", v, ok)
	}

	// a second typed attribute in the same set.
	call(t, r, s, "attributes.set",
		mustJSON(t, wire.SetAttributeArgs{Document: id, Set: set, Name: "qty", Value: types.IntegerVariant(4)}), &setRes)

	var getRes wire.AttributeResult
	call(t, r, s, "attributes.get", fmt.Sprintf(`{"document":%d,"set":%q,"name":"partNo"}`, id, set), &getRes)
	if !getRes.Found {
		t.Errorf("get partNo not found")
	}

	var miss wire.AttributeResult
	call(t, r, s, "attributes.get", fmt.Sprintf(`{"document":%d,"set":%q,"name":"nope"}`, id, set), &miss)
	if miss.Found {
		t.Errorf("get of an absent attribute should report Found=false")
	}

	var list wire.ListAttributesResult
	call(t, r, s, "attributes.list", fmt.Sprintf(`{"document":%d,"set":%q}`, id, set), &list)
	if len(list.Attributes) != 2 {
		t.Errorf("list = %+v, want 2 attributes", list.Attributes)
	}

	var sets wire.ListAttributeSetsResult
	call(t, r, s, "attributes.listSets", fmt.Sprintf(`{"document":%d}`, id), &sets)
	if len(sets.Sets) != 1 || sets.Sets[0] != set {
		t.Errorf("listSets = %+v, want [%s]", sets.Sets, set)
	}

	var found wire.FindByAttributeResult
	call(t, r, s, "attributes.find", fmt.Sprintf(`{"set":%q,"name":"partNo"}`, set), &found)
	if len(found.Matches) != 1 || found.Matches[0].Document != id {
		t.Errorf("find = %+v, want one match on document %d", found.Matches, id)
	}

	var del wire.DeleteAttributeResult
	call(t, r, s, "attributes.delete", fmt.Sprintf(`{"document":%d,"set":%q,"name":"partNo"}`, id, set), &del)
	if del.Removed != 1 {
		t.Errorf("delete Removed = %d, want 1", del.Removed)
	}
	// deleting the whole set removes the remaining attribute too.
	call(t, r, s, "attributes.delete", fmt.Sprintf(`{"document":%d,"set":%q}`, id, set), &del)
	if del.Removed != 1 {
		t.Errorf("delete-set Removed = %d, want 1 (the qty attribute)", del.Removed)
	}
}

// TestSetAttributeRequiresSetAndName: a blank set or name is a rejection, not a silent no-op.
func TestSetAttributeRequiresSetAndName(t *testing.T) {
	t.Parallel()
	r, s := emptyPartSession(t)
	id := uint64(s.ActiveDocument().ID())
	if _, err := r.Handle(s, "attributes.set", []byte(fmt.Sprintf(`{"document":%d,"set":"","name":"x"}`, id))); err == nil {
		t.Error("set with a blank set name should fail")
	}
}

// TestAttributeHandlersRejectBadInput: every attribute handler rejects malformed JSON and (for the
// document-targeted methods) an unknown document id, rather than panicking or silently succeeding.
func TestAttributeHandlersRejectBadInput(t *testing.T) {
	t.Parallel()
	r, s := emptyPartSession(t)
	docMethods := []string{"attributes.set", "attributes.get", "attributes.list", "attributes.listSets", "attributes.delete"}
	for _, m := range append([]string{"attributes.find"}, docMethods...) {
		if _, err := r.Handle(s, m, []byte(`{ not json`)); err == nil {
			t.Errorf("%s accepted malformed JSON", m)
		}
	}
	for _, m := range docMethods {
		args := []byte(`{"document":999999,"set":"s","name":"n"}`) // no such open document
		if _, err := r.Handle(s, m, args); err == nil {
			t.Errorf("%s accepted an unknown document id", m)
		}
	}
	// find with a blank set is a rejection.
	if _, err := r.Handle(s, "attributes.find", []byte(`{"set":""}`)); err == nil {
		t.Error("find with a blank set should fail")
	}
}

// TestAttributesPerTarget drives the per-entity (target) attribute round trip: tags anchored to
// two different reference keys are independent of each other and of the document, each get/list
// echoes its target, allTargets enumerates them, and the same target re-resolves the same anchor.
func TestAttributesPerTarget(t *testing.T) {
	t.Parallel()
	r, s := emptyPartSession(t)
	id := uint64(s.ActiveDocument().ID())
	const set = "com.oblikovati.traceon"
	const bodyA, bodyB = "body-ref-A", "body-ref-B"

	// Two electrodes get different voltages; the document-scoped attribute is independent.
	var res wire.AttributeResult
	call(t, r, s, "attributes.set", mustJSON(t, wire.SetAttributeArgs{Document: id, Set: set, Name: "voltage", Value: types.DoubleVariant(1000), Target: bodyA}), &res)
	if res.Attribute.Target != bodyA {
		t.Errorf("set echoed target %q, want %q", res.Attribute.Target, bodyA)
	}
	call(t, r, s, "attributes.set", mustJSON(t, wire.SetAttributeArgs{Document: id, Set: set, Name: "voltage", Value: types.DoubleVariant(-500), Target: bodyB}), &res)
	call(t, r, s, "attributes.set", mustJSON(t, wire.SetAttributeArgs{Document: id, Set: set, Name: "voltage", Value: types.DoubleVariant(0)}), &res) // document-scoped

	// Each target reads back its own value (the same target re-resolves the same anchor).
	expect := map[string]float64{bodyA: 1000, bodyB: -500}
	for target, want := range expect {
		call(t, r, s, "attributes.get", mustJSON(t, wire.GetAttributeArgs{Document: id, Set: set, Name: "voltage", Target: target}), &res)
		if !res.Found || res.Attribute.Target != target {
			t.Fatalf("get %s: found=%v target=%q", target, res.Found, res.Attribute.Target)
		}
		if v, _ := res.Attribute.Value.Double(); v != want {
			t.Errorf("get %s voltage = %g, want %g", target, v, want)
		}
	}

	// allTargets lists every anchor's attribute, each carrying its target (document + 2 bodies).
	var list wire.ListAttributesResult
	call(t, r, s, "attributes.list", mustJSON(t, wire.ListAttributesArgs{Document: id, Set: set, AllTargets: true}), &list)
	seen := map[string]bool{}
	for _, a := range list.Attributes {
		seen[a.Target] = true
	}
	if !seen[bodyA] || !seen[bodyB] || !seen[""] {
		t.Errorf("allTargets saw targets %v, want bodyA, bodyB and the document", seen)
	}

	// Deleting one target leaves the other intact.
	var del wire.DeleteAttributeResult
	call(t, r, s, "attributes.delete", mustJSON(t, wire.DeleteAttributeArgs{Document: id, Set: set, Name: "voltage", Target: bodyA}), &del)
	if del.Removed != 1 {
		t.Errorf("delete bodyA removed %d, want 1", del.Removed)
	}
	call(t, r, s, "attributes.get", mustJSON(t, wire.GetAttributeArgs{Document: id, Set: set, Name: "voltage", Target: bodyA}), &res)
	if res.Found {
		t.Error("bodyA attribute still present after delete")
	}
	call(t, r, s, "attributes.get", mustJSON(t, wire.GetAttributeArgs{Document: id, Set: set, Name: "voltage", Target: bodyB}), &res)
	if !res.Found {
		t.Error("bodyB attribute wrongly removed")
	}
}
