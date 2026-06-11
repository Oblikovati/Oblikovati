// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/model/doc"
)

func TestStampDocumentSubTypeValidates(t *testing.T) {
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
	if err := NewSession().RegisterDocumentSubType(DocumentSubType{BaseType: doc.Part}); err == nil {
		t.Error("a subtype without an id must fail")
	}
}
