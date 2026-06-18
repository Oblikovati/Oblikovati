// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
)

// keyed3DSketch creates a 3D sketch with a line and a circle, returning the router/session.
func keyed3DSketch(t *testing.T) (*Router, *app.Session) {
	t.Helper()
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch3d.create", `{}`, &wire.CreateSketch3DResult{})
	call(t, r, s, "sketch3d.addEntity", `{"sketchIndex":0,"kind":"line","points":[[0,0,0],[3,0,4]]}`, &wire.AddSketch3DEntityResult{})
	call(t, r, s, "sketch3d.addEntity", `{"sketchIndex":0,"kind":"circle","points":[[0,0,0]],"radius":"10 mm"}`, &wire.AddSketch3DEntityResult{})
	return r, s
}

// TestSketch3DEntitiesReportReferenceKeys: sketch3d.entities reports a persistent reference
// key per entity (#153), non-empty and distinct.
func TestSketch3DEntitiesReportReferenceKeys(t *testing.T) {
	r, s := keyed3DSketch(t)
	var ents wire.EnumerateEntities3DResult
	call(t, r, s, "sketch3d.entities", `{"sketchIndex":0}`, &ents)
	if len(ents.Entities) == 0 {
		t.Fatal("no 3D entities enumerated")
	}
	seen := map[string]bool{}
	for _, e := range ents.Entities {
		if e.ReferenceKey == "" {
			t.Errorf("3D entity %d (%s) has no reference key", e.ID, e.Kind)
		}
		if seen[e.ReferenceKey] {
			t.Errorf("duplicate 3D reference key %q", e.ReferenceKey)
		}
		seen[e.ReferenceKey] = true
	}
}

// TestResolveSketch3DEntityReference: a key from sketch3d.entities resolves back to the same
// entity with Kind "sketch3dEntity".
func TestResolveSketch3DEntityReference(t *testing.T) {
	r, s := keyed3DSketch(t)
	var ents wire.EnumerateEntities3DResult
	call(t, r, s, "sketch3d.entities", `{"sketchIndex":0}`, &ents)
	want := ents.Entities[0]

	var res wire.ResolveSketchReferenceResult
	call(t, r, s, "sketch.resolveReference", mustJSON(t, wire.ResolveSketchReferenceArgs{ReferenceKey: want.ReferenceKey}), &res)
	if !res.Found || res.Kind != "sketch3dEntity" || res.SketchIndex != 0 || res.EntityID != want.ID {
		t.Errorf("resolve 3D entity = %+v, want found sketch3dEntity sketch 0 id %d", res, want.ID)
	}
}

// TestSketch3DReferenceKeyResolvesToSketch: a 3D sketch's own key (sketch3d.referenceKey)
// resolves back to it with Kind "sketch3d".
func TestSketch3DReferenceKeyResolvesToSketch(t *testing.T) {
	r, s := keyed3DSketch(t)
	var key wire.SketchReferenceKeyResult
	call(t, r, s, "sketch3d.referenceKey", `{"sketchIndex":0}`, &key)
	k := key.ReferenceKey
	if k == "" {
		t.Fatal("sketch3d.referenceKey returned an empty key")
	}

	var res wire.ResolveSketchReferenceResult
	call(t, r, s, "sketch.resolveReference", mustJSON(t, wire.ResolveSketchReferenceArgs{ReferenceKey: k}), &res)
	if !res.Found || res.Kind != "sketch3d" || res.SketchIndex != 0 {
		t.Errorf("resolve 3D sketch = %+v, want found sketch3d index 0", res)
	}
}

// TestSketch3DReferenceKeyBadIndexFails: an out-of-range 3D sketch index is rejected.
func TestSketch3DReferenceKeyBadIndexFails(t *testing.T) {
	r, s := emptyPartSession(t)
	if _, err := r.Handle(s, "sketch3d.referenceKey", []byte(`{"sketchIndex":5}`)); err == nil {
		t.Error("sketch3d.referenceKey with an out-of-range index should fail")
	}
}
