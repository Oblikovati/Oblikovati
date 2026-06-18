// SPDX-License-Identifier: GPL-2.0-only

package doc

import (
	"testing"

	"oblikovati.org/model/attr"
	"oblikovati.org/model/identity"
)

// TestDocumentAttributesAccessor covers the lazy-seeded manager and the persistence-bytes helpers:
// an empty document encodes no attribute block; a populated one round-trips through the bytes.
func TestDocumentAttributesAccessor(t *testing.T) {
	d := NewReference(Part, "/w/x.obk")

	if got := d.Attributes(); got == nil {
		t.Fatal("Attributes() returned nil")
	}
	if b := d.AttributeBytes(); b != nil {
		t.Errorf("AttributeBytes() of an empty document = %v, want nil", b)
	}

	d.Attributes().AttributeSets(identity.DocumentKey()).Set("com.acme").Put("k", attr.StringValue("v"))
	bytes := d.AttributeBytes()
	if len(bytes) == 0 {
		t.Fatal("AttributeBytes() of an annotated document is empty")
	}

	// SetAttributeBytes restores them into a fresh document.
	d2 := NewReference(Part, "/w/x.obk")
	if err := d2.SetAttributeBytes(bytes); err != nil {
		t.Fatalf("SetAttributeBytes: %v", err)
	}
	set, ok := d2.Attributes().AttributeSets(identity.DocumentKey()).Lookup("com.acme")
	if !ok {
		t.Fatal("restored document missing the com.acme set")
	}
	if a, ok := set.Attribute("k"); !ok {
		t.Error("restored set missing attribute k")
	} else if v, _ := a.Value().Str(); v != "v" {
		t.Errorf("restored k = %q, want v", v)
	}
}

// TestSetAttributeBytesRejectsCorrupt: empty bytes are a no-op; corrupt bytes are a clean error.
func TestSetAttributeBytesRejectsCorrupt(t *testing.T) {
	d := NewReference(Part, "/w/x.obk")
	if err := d.SetAttributeBytes(nil); err != nil {
		t.Errorf("SetAttributeBytes(nil) = %v, want no-op", err)
	}
	if err := d.SetAttributeBytes([]byte{0xff, 0x01, 0x02}); err == nil {
		t.Error("SetAttributeBytes of corrupt bytes should error")
	}
}
