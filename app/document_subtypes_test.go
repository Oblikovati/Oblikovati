// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/model/doc"
)

func TestStampDocumentSubTypeValidates(t *testing.T) {
	t.Parallel()
	s := sessionWithPart(t)
	d := s.ActiveDocument()
	if err := s.StampDocumentSubType(d, "ghost"); err == nil {
		t.Error("an unregistered subtype must fail")
	}
	if err := s.RegisterDocumentSubType(DocumentSubType{ID: "com.x.study", BaseType: doc.Drawing}); err != nil {
		t.Fatalf("Register: %v", err)
	}
	if err := s.StampDocumentSubType(d, "com.x.study"); err == nil {
		t.Error("a base-type mismatch must fail")
	}
	if err := s.RegisterDocumentSubType(DocumentSubType{ID: "com.x.part", BaseType: doc.Part}); err != nil {
		t.Fatalf("Register: %v", err)
	}
	if err := s.StampDocumentSubType(d, "com.x.part"); err != nil {
		t.Fatalf("Stamp: %v", err)
	}
	if d.SubType() != "com.x.part" {
		t.Errorf("subType = %q, want com.x.part", d.SubType())
	}
}

func TestRegisterDocumentSubTypeNeedsID(t *testing.T) {
	t.Parallel()
	if err := NewSession().RegisterDocumentSubType(DocumentSubType{BaseType: doc.Part}); err == nil {
		t.Error("a subtype without an id must fail")
	}
}

// TestBuiltInSubTypesSeededAndReserved pins the M03-F11 invariants: the
// sheet-metal flavor exists from session start (so .obk files persist the
// discriminator before M20 ships its environment), and clients cannot claim
// ids under the reserved prefix.
func TestBuiltInSubTypesSeededAndReserved(t *testing.T) {
	t.Parallel()
	s := NewSession()
	flavors := s.DocumentSubTypes()
	if len(flavors) == 0 || flavors[0].ID != "org.oblikovati.part.sheetMetal" {
		t.Fatalf("flavors = %+v, want the built-in sheet-metal flavor seeded first", flavors)
	}
	err := s.RegisterDocumentSubType(DocumentSubType{ID: "org.oblikovati.part.weldment", BaseType: doc.Part})
	if err == nil {
		t.Error("registering under the reserved org.oblikovati. prefix must fail")
	}
}
