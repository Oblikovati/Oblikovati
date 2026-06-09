// SPDX-License-Identifier: GPL-2.0-only

package doc

import (
	"errors"
	"testing"

	"oblikovati.org/api/types"
)

func TestDocumentSeedsOneDefaultView(t *testing.T) {
	d := newDocument(Part, "t.obk", nil, true)
	vs := d.Views()
	if vs.Count() != 1 || vs.Active() == nil || vs.Active().Name != "View 1" {
		t.Fatalf("default views = %+v, want one 'View 1'", vs.All())
	}
	if vs.Layout() != types.LayoutSingle {
		t.Errorf("default layout = %v, want single", vs.Layout())
	}
}

func TestDocumentViewsAddActivateRenameClose(t *testing.T) {
	vs := newDocument(Part, "t.obk", nil, true).Views()

	i := vs.Add(DefaultView("Iso"))
	if i != 1 || vs.ActiveIndex() != 1 || vs.Active().Name != "Iso" {
		t.Fatalf("after Add active = %d (%q), want index 1 'Iso'", vs.ActiveIndex(), vs.Active().Name)
	}
	if err := vs.Rename(0, "Front"); err != nil || vs.All()[0].Name != "Front" {
		t.Fatalf("Rename(0): err=%v name=%q", err, vs.All()[0].Name)
	}
	if err := vs.Activate(0); err != nil || vs.ActiveIndex() != 0 {
		t.Fatalf("Activate(0): err=%v active=%d", err, vs.ActiveIndex())
	}
	if err := vs.Activate(9); err == nil {
		t.Error("Activate out of range should error")
	}
	// Close one of two; active index stays valid.
	if err := vs.Close(1); err != nil || vs.Count() != 1 {
		t.Fatalf("Close(1): err=%v count=%d", err, vs.Count())
	}
	if vs.ActiveIndex() != 0 {
		t.Errorf("active index after close = %d, want 0", vs.ActiveIndex())
	}
}

func TestDocumentViewsRefusesClosingLast(t *testing.T) {
	vs := newDocument(Part, "t.obk", nil, true).Views()
	if err := vs.Close(0); !errors.Is(err, ErrLastView) {
		t.Fatalf("Close last view err = %v, want ErrLastView", err)
	}
}
