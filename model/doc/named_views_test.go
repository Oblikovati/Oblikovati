// SPDX-License-Identifier: GPL-2.0-only

package doc

import (
	"testing"

	"oblikovati.org/math"
)

// TestNamedViewsStore covers the per-document named-view store: capture (replacing a same-named
// view), lookup, sorted enumeration, and delete (M16-F03 #404).
func TestNamedViewsStore(t *testing.T) {
	vs := newDocument(Part, "p.obk", nil, true).Views()
	if len(vs.NamedViews()) != 0 {
		t.Fatal("a fresh document should have no named views")
	}

	iso := ViewHome{Eye: math.P3(1, 1, 1), Target: math.P3(0, 0, 0), Up: math.V3(0, 1, 0), FOV: 1}
	vs.CaptureNamed("Iso", iso)
	vs.CaptureNamed("Top", ViewHome{Eye: math.P3(0, 5, 0)})
	vs.CaptureNamed("Iso", ViewHome{Eye: math.P3(2, 2, 2)}) // replaces the first Iso

	got, ok := vs.NamedView("Iso")
	if !ok || got.Eye.X != 2 {
		t.Errorf("NamedView(Iso) = (%+v, %v), want the replacement eye.x=2", got, ok)
	}
	if _, ok := vs.NamedView("missing"); ok {
		t.Error("an unknown named view should not be found")
	}

	all := vs.NamedViews()
	if len(all) != 2 || all[0].Name != "Iso" || all[1].Name != "Top" {
		t.Errorf("NamedViews = %+v, want [Iso, Top] sorted", all)
	}

	if !vs.DeleteNamed("Top") {
		t.Error("DeleteNamed(Top) should report it existed")
	}
	if vs.DeleteNamed("Top") {
		t.Error("DeleteNamed(Top) again should report it was absent")
	}
	if len(vs.NamedViews()) != 1 {
		t.Errorf("after delete, named views = %d, want 1", len(vs.NamedViews()))
	}
}
