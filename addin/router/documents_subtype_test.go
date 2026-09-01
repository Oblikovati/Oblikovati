// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/wire"
)

func TestDocumentSubTypesOverWire(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	call(t, r, s, "documents.registerSubType",
		`{"id":"com.x.sim.study","baseType":"part","displayName":"Simulation Study"}`, nil)

	// The built-in sheet-metal flavor is seeded at session start (M03-F11),
	// so the registry lists it ahead of the add-in's registration.
	var lst wire.ListDocumentSubTypesResult
	call(t, r, s, "documents.listSubTypes", "{}", &lst)
	if len(lst.SubTypes) != 2 || lst.SubTypes[1].ID != "com.x.sim.study" || lst.SubTypes[1].BaseType != "part" {
		t.Fatalf("subtypes = %+v, want the built-in sheet metal then the part-based study", lst.SubTypes)
	}
	if lst.SubTypes[0].ID != "org.oblikovati.part.sheetMetal" {
		t.Fatalf("subtypes[0] = %+v, want the built-in sheet-metal flavor", lst.SubTypes[0])
	}

	var info wire.DocumentInfo
	call(t, r, s, "documents.create",
		`{"type":"part","name":"study1.obk","subType":"com.x.sim.study"}`, &info)
	if info.SubType != "com.x.sim.study" {
		t.Fatalf("created info = %+v, want the flavor stamped", info)
	}

	if _, err := r.Handle(s, "documents.create",
		[]byte(`{"type":"part","name":"x.obk","subType":"ghost"}`)); err == nil {
		t.Error("an unregistered subtype should fail")
	}
	if _, err := r.Handle(s, "documents.registerSubType",
		[]byte(`{"id":"bad","baseType":"spaceship"}`)); err == nil {
		t.Error("an unknown base type should fail")
	}
	// Built-in ids are host-reserved: a client cannot claim the prefix, but
	// creating a document with the seeded flavor works (M03-F11, #612).
	if _, err := r.Handle(s, "documents.registerSubType",
		[]byte(`{"id":"org.oblikovati.part.weldment","baseType":"part"}`)); err == nil {
		t.Error("registering under the reserved org.oblikovati. prefix should fail")
	}
	var sheet wire.DocumentInfo
	call(t, r, s, "documents.create",
		`{"type":"part","name":"flange.obk","subType":"org.oblikovati.part.sheetMetal"}`, &sheet)
	if sheet.SubType != "org.oblikovati.part.sheetMetal" {
		t.Fatalf("created info = %+v, want the built-in sheet-metal flavor stamped", sheet)
	}
}
