// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
)

// TestDocumentPropertiesOverWire sets a typed iProperty over the wire, reads it back, and finds it
// in the document's property list — the #156 round trip a BOM/title-block client relies on.
func TestDocumentPropertiesOverWire(t *testing.T) {
	t.Parallel()
	r, s := emptyPartSession(t)
	id := uint64(s.ActiveDocument().ID())
	const set = "Design Tracking Properties"

	var setRes wire.PropertyResult
	args := mustJSON(t, wire.SetPropertyArgs{Document: id, Set: set, Name: "Part Number", Value: types.StringVariant("BRK-001")})
	call(t, r, s, "documents.setProperty", args, &setRes)
	if v, ok := setRes.Property.Value.Str(); !ok || v != "BRK-001" {
		t.Fatalf("set reply value = %v (ok=%v), want BRK-001", v, ok)
	}

	var getRes wire.PropertyResult
	call(t, r, s, "documents.getProperty", fmt.Sprintf(`{"document":%d,"set":%q,"name":"Part Number"}`, id, set), &getRes)
	if v, _ := getRes.Property.Value.Str(); v != "BRK-001" {
		t.Errorf("get value = %q, want BRK-001", v)
	}

	var list wire.ListPropertiesResult
	call(t, r, s, "documents.listProperties", fmt.Sprintf(`{"document":%d}`, id), &list)
	if !hasProperty(list.Properties, set, "Part Number", "BRK-001") {
		t.Errorf("listProperties missing the Part Number: %+v", list.Properties)
	}
}

// TestGetUnknownDocumentPropertyFails: reading an absent property is a rejection, not an empty hit.
func TestGetUnknownDocumentPropertyFails(t *testing.T) {
	t.Parallel()
	r, s := emptyPartSession(t)
	id := uint64(s.ActiveDocument().ID())
	if _, err := r.Handle(s, "documents.getProperty", []byte(fmt.Sprintf(`{"document":%d,"set":"Design Tracking Properties","name":"Nope"}`, id))); err == nil {
		t.Error("getProperty of an unknown property should fail")
	}
}

// hasProperty reports whether infos contains a string property with the given set/name/value.
func hasProperty(infos []wire.PropertyInfo, set, name, value string) bool {
	for _, p := range infos {
		if p.Set == set && p.Name == name {
			if v, ok := p.Value.Str(); ok && v == value {
				return true
			}
		}
	}
	return false
}
