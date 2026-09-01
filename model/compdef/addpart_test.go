// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"testing"

	"oblikovati.org/model/doc"
)

func TestAddPartInstallsRealizedContentAndActivates(t *testing.T) {
	t.Parallel()
	ws := doc.NewWorkspace(nil, testContentFactories())
	d, err := AddPart(ws, "bracket.obk", true)
	if err != nil {
		t.Fatalf("AddPart: %v", err)
	}
	if _, ok := d.Content().(*PartComponentDefinition); !ok {
		t.Fatalf("content is %T, want *PartComponentDefinition", d.Content())
	}
	if ws.ActiveDocument() != d {
		t.Fatal("AddPart did not make the new document active")
	}
	if d.DocumentType() != doc.Part {
		t.Fatalf("document type = %v, want part", d.DocumentType())
	}
}

func TestAddPartDuplicateNameErrors(t *testing.T) {
	t.Parallel()
	ws := doc.NewWorkspace(nil, testContentFactories())
	if _, err := AddPart(ws, "dup.obk", true); err != nil {
		t.Fatalf("first AddPart: %v", err)
	}
	if _, err := AddPart(ws, "dup.obk", true); err == nil {
		t.Fatal("expected error adding a second document with the same name")
	}
}
