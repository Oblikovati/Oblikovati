// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/wire"
)

// TestHighlightSetLifecycle (#157): create, add items, recolour, list, delete.
func TestHighlightSetLifecycle(t *testing.T) {
	t.Parallel()
	r, s, ref0, ref1 := boxFaceRefs(t)

	var info wire.HighlightSetInfo
	call(t, r, s, "model.highlightSets.create", `{"name":"guide","color":"#ff0000"}`, &info)
	if info.Name != "guide" || info.Color != "#ff0000" || info.Count != 0 {
		t.Fatalf("create = %+v, want guide #ff0000 count 0", info)
	}

	call(t, r, s, "model.highlightSets.addItems", mustJSON(t, wire.HighlightSetItemsArgs{Name: "guide", Refs: []string{ref0, ref1}}), &info)
	if info.Count != 2 {
		t.Errorf("after addItems count = %d, want 2", info.Count)
	}

	call(t, r, s, "model.highlightSets.setColor", `{"name":"guide","color":"#00ff00"}`, &info)
	if info.Color != "#00ff00" {
		t.Errorf("after setColor = %q, want #00ff00", info.Color)
	}

	var list wire.ListHighlightSetsResult
	call(t, r, s, "model.highlightSets.list", `{}`, &list)
	if len(list.Sets) != 1 || list.Sets[0].Count != 2 {
		t.Errorf("list = %+v, want one set with 2 items", list.Sets)
	}

	var ok wire.OKResult
	call(t, r, s, "model.highlightSets.delete", `{"name":"guide"}`, &ok)
	call(t, r, s, "model.highlightSets.list", `{}`, &list)
	if len(list.Sets) != 0 {
		t.Errorf("after delete, list = %+v, want empty", list.Sets)
	}
}

// TestHighlightSetCreateDuplicateAndBadColor (#157): a duplicate name and a malformed colour
// are rejections.
func TestHighlightSetCreateDuplicateAndBadColor(t *testing.T) {
	t.Parallel()
	r, s := emptyPartSession(t)
	call(t, r, s, "model.highlightSets.create", `{"name":"a","color":"#123456"}`, &wire.HighlightSetInfo{})
	if _, err := r.Handle(s, "model.highlightSets.create", []byte(`{"name":"a","color":"#123456"}`)); err == nil {
		t.Error("creating a duplicate-named highlight set should fail")
	}
	if _, err := r.Handle(s, "model.highlightSets.create", []byte(`{"name":"b","color":"not-a-color"}`)); err == nil {
		t.Error("a malformed colour should fail")
	}
}

// TestHighlightSetItemsResolveToGeometry (#157): a highlight set's stored references resolve to
// real selectables against the live bodies — the path the viewport overlay draws. Both ref forms
// an add-in encounters must work: the "face/…" form model.selection reports AND the raw key
// form model.referenceKeys reports (the documented highlight-set input), including edges.
func TestHighlightSetItemsResolveToGeometry(t *testing.T) {
	t.Parallel()
	r, s, ref0, _ := boxFaceRefs(t)
	if sel, ok := s.ResolveReference(ref0); !ok || sel == nil {
		t.Errorf("the selection-form face reference did not resolve (ok=%v)", ok)
	}

	var keys wire.ReferenceKeysResult
	call(t, r, s, "model.referenceKeys", `{}`, &keys)
	if sel, ok := s.ResolveReference(keys.Bodies[0].Faces[0].Key); !ok || sel == nil {
		t.Errorf("the raw model.referenceKeys face key did not resolve (ok=%v)", ok)
	}
	if sel, ok := s.ResolveReference(keys.Bodies[0].Edges[0].Key); !ok || sel == nil {
		t.Errorf("the raw model.referenceKeys edge key did not resolve (ok=%v)", ok)
	}
	if _, ok := s.ResolveReference("face/ZZZZ"); ok {
		t.Error("a bogus reference resolved unexpectedly")
	}
}
