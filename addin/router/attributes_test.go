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
	r, s := emptyPartSession(t)
	id := uint64(s.ActiveDocument().ID())
	if _, err := r.Handle(s, "attributes.set", []byte(fmt.Sprintf(`{"document":%d,"set":"","name":"x"}`, id))); err == nil {
		t.Error("set with a blank set name should fail")
	}
}
