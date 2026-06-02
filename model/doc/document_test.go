// SPDX-License-Identifier: GPL-2.0-only

package doc

import "testing"

func TestDocumentReportsIdentityAndDirtyState(t *testing.T) {
	d := NewPartDocument("/proj/bracket.obk")

	if d.ID() == 0 {
		t.Fatal("ID() = 0, want a nonzero session handle")
	}
	if got := d.FullDocumentName(); got != "/proj/bracket.obk" {
		t.Errorf("FullDocumentName() = %q, want %q", got, "/proj/bracket.obk")
	}
	if got := d.DisplayName(); got != "bracket" {
		t.Errorf("DisplayName() = %q, want %q", got, "bracket")
	}
	if d.Dirty() {
		t.Error("new document is Dirty, want clean")
	}

	d.MarkDirty()
	if !d.Dirty() {
		t.Error("after MarkDirty, Dirty() = false")
	}
	d.ClearDirty()
	if d.Dirty() {
		t.Error("after ClearDirty, Dirty() = true")
	}
}

func TestSessionIDsAreUniqueAndNonzero(t *testing.T) {
	a := NewPartDocument("a.obk")
	b := NewPartDocument("b.obk")
	if a.ID() == b.ID() {
		t.Errorf("two documents share ID %d, want distinct", a.ID())
	}
}

func TestReferenceStubExistsWithoutPagingContent(t *testing.T) {
	stub := NewReference(Part, "/lib/fastener.obk")

	if !stub.IsReferenceStub() {
		t.Error("NewReference is not reported as a reference stub")
	}
	if stub.Open() {
		t.Error("reference stub reports Open() = true, want not paged in")
	}
	if stub.Content() != nil {
		t.Error("reference stub has non-nil Content, want unloaded")
	}
	if got := stub.FullDocumentName(); got != "/lib/fastener.obk" {
		t.Errorf("stub FullDocumentName() = %q, want identity preserved", got)
	}
	if got := stub.DocumentType(); got != Part {
		t.Errorf("stub DocumentType() = %v, want Part", got)
	}
}

func TestOpenDocumentIsNotAStub(t *testing.T) {
	d := NewAssemblyDocument("top.obk")
	if d.IsReferenceStub() {
		t.Error("open document reports IsReferenceStub() = true")
	}
	if !d.Open() {
		t.Error("open document reports Open() = false")
	}
}

func TestDisplayNameOverride(t *testing.T) {
	d := NewPartDocument("/proj/part1.obk")
	d.SetDisplayName("Left Bracket")
	if got := d.DisplayName(); got != "Left Bracket" {
		t.Errorf("DisplayName() = %q, want override", got)
	}
	d.SetDisplayName("")
	if got := d.DisplayName(); got != "part1" {
		t.Errorf("DisplayName() after clearing override = %q, want derived %q", got, "part1")
	}
}

func TestFullFileNameTracksDocumentName(t *testing.T) {
	d := NewDrawingDocument("/proj/sheet.obk")
	if d.FullFileName() != d.FullDocumentName() {
		t.Errorf("FullFileName() = %q, FullDocumentName() = %q; want equal for an on-disk document",
			d.FullFileName(), d.FullDocumentName())
	}
}
