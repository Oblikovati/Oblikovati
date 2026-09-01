// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"
	"strings"
	"testing"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
)

// extrudedPartViaAPI drives the wire surface end-to-end — sketch.create,
// sketch.rectangle, features.add(extrude 50mm) — and returns the extrude's stable id
// from model.tree, the fixture every features.* lifecycle test starts from.
func extrudedPartViaAPI(t *testing.T) (*Router, *app.Session, uint64) {
	t.Helper()
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &struct{}{})
	call(t, r, s, "sketch.rectangle", `{"sketchIndex":0,"width":"40 mm","height":"30 mm"}`, &struct{}{})
	call(t, r, s, "features.add", `{"kind":"extrude","args":{"sketchIndex":0,"distance":"50 mm"}}`, &struct{}{})
	tree := modelTreeOf(t, r, s)
	if len(tree.Features) != 1 {
		t.Fatalf("fixture: model.tree reports %d features, want 1", len(tree.Features))
	}
	return r, s, tree.Features[0].ID
}

func modelTreeOf(t *testing.T, r *Router, s *app.Session) wire.ModelTreeResult {
	t.Helper()
	var tree wire.ModelTreeResult
	call(t, r, s, "model.tree", `{}`, &tree)
	return tree
}

func TestFeaturesGetReportsEditableScalars(t *testing.T) {
	t.Parallel()
	r, s, id := extrudedPartViaAPI(t)
	var got wire.FeatureDetailResult
	call(t, r, s, "features.get", fmt.Sprintf(`{"id":%d}`, id), &got)
	f := got.Feature
	if f.Kind != "extrude" || f.Index != 0 || f.Suppressed {
		t.Errorf("detail = kind %q index %d suppressed %v, want extrude/0/false", f.Kind, f.Index, f.Suppressed)
	}
	if len(f.Scalars) == 0 {
		t.Fatal("extrude detail reports no editable scalars")
	}
	if sc := f.Scalars[0]; sc.Label != "Distance" || sc.Value != 50 || sc.Unit != "mm" {
		t.Errorf("scalar 0 = %q %v %s, want Distance 50 mm", sc.Label, sc.Value, sc.Unit)
	}
}

func TestFeaturesGetUnknownIDFails(t *testing.T) {
	t.Parallel()
	r, s, _ := extrudedPartViaAPI(t)
	if _, err := r.Handle(s, "features.get", []byte(`{"id":999999}`)); err == nil {
		t.Error("expected an error for an unknown feature id")
	}
}

func TestFeaturesEditChangesScalarAndRecomputes(t *testing.T) {
	t.Parallel()
	r, s, id := extrudedPartViaAPI(t)
	var got wire.FeatureDetailResult
	call(t, r, s, "features.edit",
		fmt.Sprintf(`{"id":%d,"scalars":[{"index":0,"value":"60 mm"}]}`, id), &got)
	if v := got.Feature.Scalars[0].Value; v != 60 {
		t.Errorf("distance after edit = %v mm, want 60", v)
	}
	if tree := modelTreeOf(t, r, s); tree.Bodies != 1 {
		t.Errorf("bodies after edit = %d, want 1", tree.Bodies)
	}
}

func TestFeaturesEditValidatesBeforeApplying(t *testing.T) {
	t.Parallel()
	r, s, id := extrudedPartViaAPI(t)
	// Batch with a valid first edit and a broken second one: nothing may be applied.
	_, err := r.Handle(s, "features.edit",
		[]byte(fmt.Sprintf(`{"id":%d,"scalars":[{"index":0,"value":"60 mm"},{"index":9,"value":"1 mm"}]}`, id)))
	if err == nil || !strings.Contains(err.Error(), "out of range") {
		t.Fatalf("expected an out-of-range error, got %v", err)
	}
	var got wire.FeatureDetailResult
	call(t, r, s, "features.get", fmt.Sprintf(`{"id":%d}`, id), &got)
	if v := got.Feature.Scalars[0].Value; v != 50 {
		t.Errorf("distance after failed batch = %v mm, want 50 (untouched)", v)
	}
	if _, err := r.Handle(s, "features.edit",
		[]byte(fmt.Sprintf(`{"id":%d,"scalars":[]}`, id))); err == nil {
		t.Error("expected an error for an empty scalars batch")
	}
	if _, err := r.Handle(s, "features.edit",
		[]byte(fmt.Sprintf(`{"id":%d,"scalars":[{"index":0,"value":"bogus"}]}`, id))); err == nil {
		t.Error("expected an error for an unparseable value")
	}
}

func TestFeaturesRenameViaAPI(t *testing.T) {
	t.Parallel()
	r, s, id := extrudedPartViaAPI(t)
	var got wire.FeatureDetailResult
	call(t, r, s, "features.rename", fmt.Sprintf(`{"id":%d,"name":"Base Boss"}`, id), &got)
	if got.Feature.Name != "Base Boss" || got.Feature.ID != id {
		t.Errorf("rename reply = %q id %d, want \"Base Boss\" id %d", got.Feature.Name, got.Feature.ID, id)
	}
	if tree := modelTreeOf(t, r, s); tree.Features[0].Name != "Base Boss" {
		t.Errorf("model.tree name = %q, want \"Base Boss\"", tree.Features[0].Name)
	}
	if _, err := r.Handle(s, "features.rename", []byte(fmt.Sprintf(`{"id":%d,"name":""}`, id))); err == nil {
		t.Error("expected an error renaming to the empty name")
	}
}

func TestFeaturesSetSuppressedViaAPI(t *testing.T) {
	t.Parallel()
	r, s, id := extrudedPartViaAPI(t)
	var got wire.FeatureDetailResult
	call(t, r, s, "features.setSuppressed", fmt.Sprintf(`{"id":%d,"suppressed":true}`, id), &got)
	if !got.Feature.Suppressed {
		t.Error("reply does not report the feature suppressed")
	}
	if tree := modelTreeOf(t, r, s); tree.Bodies != 0 {
		t.Errorf("bodies with the only extrude suppressed = %d, want 0", tree.Bodies)
	}
	call(t, r, s, "features.setSuppressed", fmt.Sprintf(`{"id":%d,"suppressed":false}`, id), &got)
	if tree := modelTreeOf(t, r, s); tree.Bodies != 1 {
		t.Errorf("bodies after unsuppress = %d, want 1", tree.Bodies)
	}
}

func TestFeaturesDeleteViaAPI(t *testing.T) {
	t.Parallel()
	r, s, id := extrudedPartViaAPI(t)
	var got wire.DeleteFeatureResult
	call(t, r, s, "features.delete", fmt.Sprintf(`{"id":%d}`, id), &got)
	if !got.Deleted || got.ID != id {
		t.Errorf("delete reply = %+v, want deleted id %d", got, id)
	}
	tree := modelTreeOf(t, r, s)
	if len(tree.Features) != 0 || tree.Bodies != 0 {
		t.Errorf("after delete: %d features, %d bodies; want 0/0", len(tree.Features), tree.Bodies)
	}
	if _, err := r.Handle(s, "features.delete", []byte(fmt.Sprintf(`{"id":%d}`, id))); err == nil {
		t.Error("expected an error deleting an already-deleted id")
	}
}

func TestFeaturesReorderViaAPI(t *testing.T) {
	t.Parallel()
	r, s, _ := extrudedPartViaAPI(t)
	// A second, independent extrude on its own sketch; it lands at history index 1.
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &struct{}{})
	call(t, r, s, "sketch.rectangle", `{"sketchIndex":1,"width":"10 mm","height":"10 mm"}`, &struct{}{})
	call(t, r, s, "features.add", `{"kind":"extrude","args":{"sketchIndex":1,"distance":"5 mm","op":"newBody"}}`, &struct{}{})
	tree := modelTreeOf(t, r, s)
	if len(tree.Features) != 2 {
		t.Fatalf("fixture: %d features, want 2", len(tree.Features))
	}
	second := tree.Features[1].ID

	var got wire.FeatureDetailResult
	call(t, r, s, "features.reorder", fmt.Sprintf(`{"id":%d,"newIndex":0}`, second), &got)
	if got.Feature.Index != 0 {
		t.Errorf("reorder reply index = %d, want 0", got.Feature.Index)
	}
	if tree := modelTreeOf(t, r, s); tree.Features[0].ID != second {
		t.Errorf("model.tree first feature id = %d, want %d", tree.Features[0].ID, second)
	}
	if _, err := r.Handle(s, "features.reorder", []byte(fmt.Sprintf(`{"id":%d,"newIndex":42}`, second))); err == nil {
		t.Error("expected an error for an out-of-range index")
	}
}
