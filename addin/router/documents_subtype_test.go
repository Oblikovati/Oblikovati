// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/wire"
)

func TestDocumentSubTypesOverWire(t *testing.T) {
	r, s := seededSession(t)
	call(t, r, s, "documents.registerSubType",
		`{"id":"com.x.sim.study","baseType":"part","displayName":"Simulation Study"}`, nil)

	var lst wire.ListDocumentSubTypesResult
	call(t, r, s, "documents.listSubTypes", "{}", &lst)
	if len(lst.SubTypes) != 1 || lst.SubTypes[0].BaseType != "part" {
		t.Fatalf("subtypes = %+v, want the part-based study", lst.SubTypes)
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
}
