// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"strings"
	"testing"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
)

// viewport.click / viewport.key exist so a client can drive the INPUT path. Creating geometry
// through sketch.addEntity never takes it, so no client could reach — or test — what happens on
// the way in: inferred constraints, previews, multi-click chains (#2032).

// sketchClickSession returns a router and a session editing a sketch on the XY plane, framed so
// the plane's origin projects into the viewport.
func sketchClickSession(t *testing.T) (*Router, *app.Session) {
	t.Helper()
	r, s := seededSession(t)
	call(t, r, s, "documents.create", `{"type":"part","name":"click"}`, &wire.OKResult{})
	var sk wire.CreateSketchResult
	call(t, r, s, "sketch.create", `{"plane":"xy"}`, &sk)
	call(t, r, s, "sketch.edit", `{"sketchIndex":0}`, &wire.OKResult{})
	return r, s
}

// TestClickAtAModelPointDrivesTheActiveTool: the click has to reach the running command, and the
// caller has to be told the command is still collecting points.
func TestClickAtAModelPointDrivesTheActiveTool(t *testing.T) {
	r, s := sketchClickSession(t)
	call(t, r, s, "commands.execute", `{"id":"Sketch.Line"}`, &wire.OKResult{})

	var got wire.ClickViewportResult
	call(t, r, s, "viewport.click", `{"point":[0,0,0]}`, &got)

	if got.ActiveTool != "Line" {
		t.Errorf("after one click the active tool is %q, want the Line command still collecting", got.ActiveTool)
	}
}

// TestClickReportsTheProjectedPixel: a caller giving a model point gets back the pixel it became,
// which is the only way to tell WHERE the host decided to click.
func TestClickReportsTheProjectedPixel(t *testing.T) {
	r, s := sketchClickSession(t)

	var origin, offset wire.ClickViewportResult
	call(t, r, s, "viewport.click", `{"point":[0,0,0]}`, &origin)
	call(t, r, s, "viewport.click", `{"point":[5,0,0]}`, &offset)

	if origin.X == offset.X && origin.Y == offset.Y {
		t.Errorf("two different model points both clicked pixel (%v,%v) — the projection is not being applied", origin.X, origin.Y)
	}
}

// TestClickOffScreenIsAnError: a point behind the camera must be reported, not silently clicked at
// some meaningless pixel.
func TestClickOffScreenIsAnError(t *testing.T) {
	r, s := sketchClickSession(t)
	call(t, r, s, "view.setCamera", `{"eye":[0,0,10],"target":[0,0,0],"up":[0,1,0],"fov":0.8}`, &wire.CameraView{})

	_, err := r.Handle(s, "viewport.click", []byte(`{"point":[0,0,900]}`))
	if err == nil {
		t.Fatal("clicking a point behind the camera should be an error")
	}
	if !strings.Contains(err.Error(), "900") {
		t.Errorf("error %q should name the offending point", err)
	}
}

// TestUnknownButtonIsRejected: a typo must name what was given and what was expected, not fall
// back to the left button and look like it worked.
func TestUnknownButtonIsRejected(t *testing.T) {
	r, s := sketchClickSession(t)

	_, err := r.Handle(s, "viewport.click", []byte(`{"point":[0,0,0],"button":"clicky"}`))
	if err == nil {
		t.Fatal("an unknown button should be rejected")
	}
	if !strings.Contains(err.Error(), "clicky") || !strings.Contains(err.Error(), "left") {
		t.Errorf("error %q should name the offending value and the accepted ones", err)
	}
}

// TestEscapeFinishesAChain is why viewport.key exists: a continuous command has no click that
// ends it, so without a key a client could start a chain and never finish it.
func TestEscapeFinishesAChain(t *testing.T) {
	r, s := sketchClickSession(t)
	call(t, r, s, "commands.execute", `{"id":"Sketch.Line"}`, &wire.OKResult{})
	for _, p := range []string{`{"point":[0,0,0]}`, `{"point":[4,0,0]}`, `{"point":[4,3,0]}`} {
		call(t, r, s, "viewport.click", p, &wire.ClickViewportResult{})
	}

	var got wire.PressKeyResult
	call(t, r, s, "viewport.key", `{"key":"Escape"}`, &got)

	if got.ActiveTool != "" {
		t.Errorf("after Escape the active tool is %q, want the chain finished", got.ActiveTool)
	}
	var ents wire.EnumerateEntitiesResult
	call(t, r, s, "sketch.entities", `{"sketchIndex":0}`, &ents)
	if n := countLines(ents); n != 2 {
		t.Errorf("the finished chain left %d lines, want 2", n)
	}
}

// TestEmptyKeyIsRejected: an empty key would otherwise be delivered as a no-op that silently did
// nothing, which is the hardest kind of automation bug to see.
func TestEmptyKeyIsRejected(t *testing.T) {
	r, s := sketchClickSession(t)

	_, err := r.Handle(s, "viewport.key", []byte(`{"key":""}`))
	if err == nil {
		t.Fatal("an empty key should be rejected")
	}
	if !strings.Contains(err.Error(), "Escape") {
		t.Errorf("error %q should say what a key name looks like", err)
	}
}

// countLines is how many line entities a listing holds.
func countLines(r wire.EnumerateEntitiesResult) int {
	n := 0
	for _, e := range r.Entities {
		if e.Kind == "line" {
			n++
		}
	}
	return n
}
