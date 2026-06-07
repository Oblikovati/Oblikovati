// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati/api/types"
	"oblikovati/api/wire"
)

// TestViewsLifecycleOverTheAPI drives the views.* surface against a live session: a fresh
// document has one active view; add/activate/rename/close and layout all round-trip, and
// closing the last view is refused.
func TestViewsLifecycleOverTheAPI(t *testing.T) {
	r, s := seededSession(t)

	var list wire.ListViewsResult
	call(t, r, s, "views.list", "{}", &list)
	if len(list.Views) != 1 || !list.Views[0].Active || list.Layout != types.LayoutSingle {
		t.Fatalf("fresh document views = %+v, want one active view, single layout", list)
	}

	var added wire.ViewInfo
	call(t, r, s, "views.add", `{"name":"Iso","copyActiveCamera":true}`, &added)
	if added.Index != 1 || added.Name != "Iso" || !added.Active {
		t.Fatalf("add = %+v, want index 1 'Iso' active", added)
	}

	call(t, r, s, "views.list", "{}", &list)
	if len(list.Views) != 2 || list.ActiveIndex != 1 {
		t.Fatalf("after add views=%d active=%d, want 2 active=1", len(list.Views), list.ActiveIndex)
	}

	call(t, r, s, "views.activate", `{"index":0}`, &list)
	if list.ActiveIndex != 0 {
		t.Fatalf("after activate(0) active=%d, want 0", list.ActiveIndex)
	}

	var ri wire.ViewInfo
	call(t, r, s, "views.rename", `{"index":0,"name":"Front"}`, &ri)
	if ri.Name != "Front" {
		t.Errorf("rename = %q, want Front", ri.Name)
	}

	var lay wire.LayoutResult
	call(t, r, s, "views.setLayout", `{"layout":4}`, &lay)
	if lay.Layout != types.LayoutFour {
		t.Errorf("setLayout = %v, want four", lay.Layout)
	}
	call(t, r, s, "views.getLayout", "{}", &lay)
	if lay.Layout != types.LayoutFour {
		t.Errorf("getLayout = %v, want four", lay.Layout)
	}

	call(t, r, s, "views.close", `{"index":1}`, &list)
	if len(list.Views) != 1 {
		t.Fatalf("after close views=%d, want 1", len(list.Views))
	}
	if _, err := r.Handle(s, "views.close", []byte(`{"index":0}`)); err == nil {
		t.Fatal("closing the last view should error")
	}
}

// TestCameraReflectedInActiveViewListing checks set_camera updates the active view's camera
// as reported by views.list — proving camera is per-view.
func TestCameraReflectedInActiveViewListing(t *testing.T) {
	r, s := seededSession(t)
	call(t, r, s, "view.setCamera", `{"eye":[9,8,7],"target":[1,2,3],"up":[0,1,0],"fov":0.7}`, nil)

	var list wire.ListViewsResult
	call(t, r, s, "views.list", "{}", &list)
	got := list.Views[list.ActiveIndex].Camera
	if got.Eye != [3]float64{9, 8, 7} || got.Target != [3]float64{1, 2, 3} {
		t.Fatalf("active view camera = %+v, want eye[9 8 7] target[1 2 3]", got)
	}
}
