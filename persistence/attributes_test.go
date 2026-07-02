// SPDX-License-Identifier: GPL-2.0-only

package persistence

import (
	"path/filepath"
	"testing"

	"oblikovati.org/model/attr"
	"oblikovati.org/model/contentset"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/identity"
)

// TestAttributesRoundTripThroughPackage: an add-in attribute attached to a document persists in the
// .obk and reloads with it (#155) — the durable storage that makes attributes useful.
func TestAttributesRoundTripThroughPackage(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tagged.obk")
	store := NewPackageStore()
	ws := doc.NewWorkspace(store, contentset.Default())
	d, err := ws.Add(doc.Part, path, true)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	d.Attributes().AttributeSets(identity.DocumentKey()).Set("com.acme.bom").Put("partNo", attr.StringValue("BRK-001"))
	d.Attributes().AttributeSets(identity.DocumentKey()).Set("com.acme.bom").Put("qty", attr.IntValue(4))
	if err := ws.Save(d); err != nil {
		t.Fatalf("Save: %v", err)
	}

	reopened, err := doc.NewWorkspace(store, contentset.Default()).Open(path, true)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	ss := reopened.Attributes().AttributeSets(identity.DocumentKey())
	set, ok := ss.Lookup("com.acme.bom")
	if !ok {
		t.Fatalf("reloaded document missing the com.acme.bom set; sets=%v", ss.Names())
	}
	a, ok := set.Attribute("partNo")
	if !ok {
		t.Fatalf("reloaded set missing partNo")
	}
	if v, ok := a.Value().Str(); !ok || v != "BRK-001" {
		t.Errorf("reloaded partNo = %v (ok=%v), want BRK-001", v, ok)
	}
	if q, ok := set.Attribute("qty"); !ok {
		t.Error("reloaded set missing qty")
	} else if iv, _ := q.Value().Int(); iv != 4 {
		t.Errorf("reloaded qty = %d, want 4", iv)
	}
}

// TestUnannotatedDocumentWritesNoAttributes: a document with no attributes round-trips with an
// empty attribute manager (the .obk carries no attribute block).
func TestUnannotatedDocumentWritesNoAttributes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "plain.obk")
	store := NewPackageStore()
	ws := doc.NewWorkspace(store, contentset.Default())
	d, err := ws.Add(doc.Part, path, true)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := ws.Save(d); err != nil {
		t.Fatalf("Save: %v", err)
	}
	reopened, err := doc.NewWorkspace(store, contentset.Default()).Open(path, true)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if n := reopened.Attributes().Count(); n != 0 {
		t.Errorf("reloaded plain document has %d anchored attribute objects, want 0", n)
	}
}
