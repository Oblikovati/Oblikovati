// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/addin/opregistry"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
)

// TestSketchEntitiesReportReferenceKeys: sketch.entities now reports a persistent reference
// key per entity (#153), and the keys are non-empty and distinct.
func TestSketchEntitiesReportReferenceKeys(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	call(t, r, s, "sketch.rectangle", `{"sketchIndex":0,"width":"40 mm","height":"30 mm"}`, &wire.SketchRectangleResult{})

	var ents wire.EnumerateEntitiesResult
	call(t, r, s, "sketch.entities", `{"sketchIndex":0}`, &ents)
	if len(ents.Entities) == 0 {
		t.Fatal("no entities enumerated")
	}
	seen := map[string]bool{}
	for _, e := range ents.Entities {
		if e.ReferenceKey == "" {
			t.Errorf("entity %d (%s) has no reference key", e.ID, e.Kind)
		}
		if seen[e.ReferenceKey] {
			t.Errorf("duplicate reference key %q", e.ReferenceKey)
		}
		seen[e.ReferenceKey] = true
	}
}

// TestResolveSketchEntityReference: a key from sketch.entities resolves back to the same
// entity (kind=sketchEntity, right sketch index and session id).
func TestResolveSketchEntityReference(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	call(t, r, s, "sketch.rectangle", `{"sketchIndex":0,"width":"40 mm","height":"30 mm"}`, &wire.SketchRectangleResult{})

	var ents wire.EnumerateEntitiesResult
	call(t, r, s, "sketch.entities", `{"sketchIndex":0}`, &ents)
	want := ents.Entities[0]

	var res wire.ResolveSketchReferenceResult
	call(t, r, s, "sketch.resolveReference", mustJSON(t, wire.ResolveSketchReferenceArgs{ReferenceKey: want.ReferenceKey}), &res)
	if !res.Found || res.Kind != "sketchEntity" || res.SketchIndex != 0 || res.EntityID != want.ID {
		t.Errorf("resolve entity = %+v, want found sketchEntity sketch 0 id %d", res, want.ID)
	}
}

// TestSketchReferenceKeyResolvesToSketch: a sketch's own key (sketch.referenceKey) resolves
// back to that sketch (kind=sketch).
func TestSketchReferenceKeyResolvesToSketch(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})

	var key wire.SketchReferenceKeyResult
	call(t, r, s, "sketch.referenceKey", `{"sketchIndex":0}`, &key)
	k := key.ReferenceKey
	if k == "" {
		t.Fatal("sketch.referenceKey returned an empty key")
	}

	var res wire.ResolveSketchReferenceResult
	call(t, r, s, "sketch.resolveReference", mustJSON(t, wire.ResolveSketchReferenceArgs{ReferenceKey: k}), &res)
	if !res.Found || res.Kind != "sketch" || res.SketchIndex != 0 {
		t.Errorf("resolve sketch = %+v, want found sketch index 0", res)
	}
}

// TestReferenceKeyMethodsRejectNoActivePart: both methods error without an active part,
// rather than returning an empty/zero result.
func TestReferenceKeyMethodsRejectNoActivePart(t *testing.T) {
	r := New(opregistry.Default())
	s := app.NewSession()
	cases := map[string]string{
		"sketch.referenceKey":     `{"sketchIndex":0}`,
		"sketch.resolveReference": `{"referenceKey":"x"}`,
	}
	for method, args := range cases {
		if _, err := r.Handle(s, method, []byte(args)); err == nil {
			t.Errorf("%s with no active part should fail", method)
		}
	}
}

// TestSketchReferenceKeyBadIndexFails: an out-of-range sketch index is a rejection.
func TestSketchReferenceKeyBadIndexFails(t *testing.T) {
	r, s := emptyPartSession(t)
	if _, err := r.Handle(s, "sketch.referenceKey", []byte(`{"sketchIndex":7}`)); err == nil {
		t.Error("sketch.referenceKey with an out-of-range index should fail")
	}
}

// TestResolveUnknownReferenceNotFound: an unrecognized key resolves to found=false (the
// referent was deleted), not an error.
func TestResolveUnknownReferenceNotFound(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})

	var res wire.ResolveSketchReferenceResult
	call(t, r, s, "sketch.resolveReference", `{"referenceKey":"00000000-0000-5000-8000-000000000000"}`, &res)
	if res.Found {
		t.Errorf("unknown key resolved as found: %+v", res)
	}
}
