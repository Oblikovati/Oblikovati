// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/wire"
)

// TestSketchTextAddEditGet drives the by-reference text methods end to end: add a styled
// text entity, read it back, edit a subset of fields, and confirm the edit took.
func TestSketchTextAddEditGet(t *testing.T) {
	t.Parallel()
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})

	var added wire.AddEntityIDResult
	call(t, r, s, "sketch.addText",
		`{"sketchIndex":0,"anchor":[1,1],"text":"PART A","height":"5 mm","justify":"center","vJustify":"middle","font":"Liberation Sans"}`,
		&added)
	if added.EntityID == 0 {
		t.Fatal("addText returned no id")
	}

	var got wire.SketchTextResult
	call(t, r, s, "sketch.getText", `{"sketchIndex":0,"entityId":`+itoa(added.EntityID)+`}`, &got)
	if got.Style.Content != "PART A" || got.Style.HAlign != "center" || got.Style.VAlign != "middle" {
		t.Fatalf("getText style = %+v, want PART A/center/middle", got.Style)
	}
	if got.Style.Family != "Liberation Sans" {
		t.Errorf("font family = %q, want Liberation Sans", got.Style.Family)
	}

	var edited wire.SketchTextResult
	call(t, r, s, "sketch.editText",
		`{"sketchIndex":0,"entityId":`+itoa(added.EntityID)+`,"text":"NEW","justify":"right"}`,
		&edited)
	if edited.Style.Content != "NEW" || edited.Style.HAlign != "right" {
		t.Errorf("after edit style = %+v, want NEW/right", edited.Style)
	}
	if edited.Style.VAlign != "middle" {
		t.Errorf("unspecified vAlign changed to %q, want middle (partial edit preserves it)", edited.Style.VAlign)
	}
}

// TestTextEmbossByReferenceThroughRouter drives the bug-fix path through the public
// features.add API: a base block, then an emboss that references a sketch TEXT entity
// (textEntity), producing a raised solid without baking glyph geometry.
func TestTextEmbossByReferenceThroughRouter(t *testing.T) {
	if testing.Short() {
		t.Skip("corpus tier (~6s): `make test-corpus`")
	}
	t.Parallel()
	r, s := emptyPartSession(t)

	// Base 50×50 block on XY.
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"rectangle","points":[[0,0],[50,50]]}`, &wire.AddSketchEntityResult{})
	call(t, r, s, "features.add", `{"kind":"extrude","args":{"sketchIndex":0,"profileIndex":0,"distance":"10 mm"}}`, nil)

	// A second sketch carrying a text entity, then emboss it by reference.
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	var txt wire.AddEntityIDResult
	call(t, r, s, "sketch.addText", `{"sketchIndex":1,"anchor":[5,5],"text":"OK","height":"10 mm"}`, &txt)

	var emb struct {
		Bodies int `json:"bodies"`
	}
	call(t, r, s, "features.add",
		`{"kind":"emboss","args":{"sketchIndex":1,"textEntity":`+itoa(txt.EntityID)+`,"depth":"2 mm"}}`,
		&emb)
	if emb.Bodies != 1 {
		t.Fatalf("text emboss bodies = %d, want 1", emb.Bodies)
	}
}

// itoa renders a uint64 without importing strconv at call sites in the table above.
func itoa(v uint64) string {
	if v == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for v > 0 {
		i--
		b[i] = byte('0' + v%10)
		v /= 10
	}
	return string(b[i:])
}
