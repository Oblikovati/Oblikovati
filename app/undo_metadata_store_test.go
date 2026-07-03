// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/model/display"
	"oblikovati.org/model/doc"
)

// plainContent is document content that is NOT a recipe store (only [doc.Content]) — the fake that
// drives metaStoreFor's "no undo stream" branch without an inline stub.
type plainContent struct{}

func (plainContent) DocumentType() doc.DocumentType { return doc.Part }

// metaDoc returns a fresh part document with every metadata channel populated, so a marshal captures
// all four (body names, color styles, sketch settings, display settings).
func metaDoc(t *testing.T) *doc.Document {
	t.Helper()
	d := doc.NewPartDocument("/proj/meta.obk").Document
	d.SetBodyName("k1", "Housing")
	d.SetBodyColorStyle("k1", "Brass")
	d.SetSketchSettings(d.SketchSettings())
	d.SetDisplaySettings(display.DefaultSettings())
	return d
}

// TestMetaStoreSnapshotRoundTrip: MarshalSnapshot captures all metadata and RestoreSnapshot reinstalls
// it into a fresh document (the settings-present restore branches), leaving both docs equal.
func TestMetaStoreSnapshotRoundTrip(t *testing.T) {
	src := documentMetaStore{inner: &fakeRecipeStore{}, doc: metaDoc(t)}
	blob, err := src.MarshalSnapshot()
	if err != nil {
		t.Fatalf("MarshalSnapshot: %v", err)
	}
	dst := doc.NewPartDocument("/proj/dst.obk").Document
	if err := (documentMetaStore{inner: &fakeRecipeStore{}, doc: dst}).RestoreSnapshot(blob); err != nil {
		t.Fatalf("RestoreSnapshot: %v", err)
	}
	assertMetaRestored(t, dst)
}

// assertMetaRestored checks every metadata channel survived the round trip.
func assertMetaRestored(t *testing.T, d *doc.Document) {
	t.Helper()
	if name, ok := d.BodyName("k1"); !ok || name != "Housing" {
		t.Errorf("BodyName = (%q,%v), want (Housing,true)", name, ok)
	}
	if style, ok := d.BodyColorStyle("k1"); !ok || style != "Brass" {
		t.Errorf("BodyColorStyle = (%q,%v), want (Brass,true)", style, ok)
	}
	if !d.SketchSettingsSet() {
		t.Error("sketch settings were not restored")
	}
	if _, ok := d.DisplaySettings(); !ok {
		t.Error("display settings were not restored")
	}
}

// TestMetaStoreRestoreRejectsInvalidJSON: a malformed snapshot surfaces the unmarshal error rather
// than corrupting the document.
func TestMetaStoreRestoreRejectsInvalidJSON(t *testing.T) {
	m := documentMetaStore{inner: &fakeRecipeStore{}, doc: metaDoc(t)}
	if err := m.RestoreSnapshot([]byte("not json")); err == nil {
		t.Error("RestoreSnapshot should surface an unmarshal error for malformed input")
	}
}

// TestMetaStoreRestoreSurfacesInnerFailure: when the inner recipe restore fails, the metadata store
// propagates the error and does not touch the document metadata.
func TestMetaStoreRestoreSurfacesInnerFailure(t *testing.T) {
	inner := &fakeRecipeStore{restoreErr: errMarshalBoom}
	m := documentMetaStore{inner: inner, doc: metaDoc(t)}
	if err := m.RestoreSnapshot([]byte(`{"recipe":"x"}`)); err == nil {
		t.Error("RestoreSnapshot should propagate the inner recipe-restore failure")
	}
}

// TestMetaStoreForRejectsNonRecipeContent: a document whose content has no recipe (no undo stream)
// yields no metadata store.
func TestMetaStoreForRejectsNonRecipeContent(t *testing.T) {
	d := doc.NewPartDocument("/proj/plain.obk").Document
	d.SetContent(plainContent{})
	if _, ok := metaStoreFor(d); ok {
		t.Error("metaStoreFor should reject content that is not a recipe store")
	}
}

// TestRecordMetadataEditIgnoresInactiveDocument: recording against a nil/background document is a
// no-op (the per-document undo stream records only the active document).
func TestRecordMetadataEditIgnoresInactiveDocument(t *testing.T) {
	s := NewSession()
	s.recordMetadataEdit(nil, "ignored") // must not panic and must record nothing
}
