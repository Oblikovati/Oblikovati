// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/wire"
)

// TestDrawingSketchesOverWire drives the sketch surface: add a sketch, add a rectangle + circle,
// list them — producing the curves through the live stack.
func TestDrawingSketchesOverWire(t *testing.T) {
	r, s := drawingViewSession(t)

	var sk wire.DrawingSketchResult
	call(t, r, s, "drawingSketches.add", `{"name":"S1"}`, &sk)
	if sk.Sketch.Name != "S1" {
		t.Fatalf("add sketch = %+v, want S1", sk.Sketch)
	}
	call(t, r, s, "drawingSketches.addEntity", `{"sketchName":"S1","kind":"rectangle","points":[[10,20],[60,50]]}`, &sk)
	call(t, r, s, "drawingSketches.addEntity", `{"sketchName":"S1","kind":"circle","points":[[100,100]],"radiusMm":12}`, &sk)
	if sk.Sketch.EntityCount != 2 || sk.Sketch.CurveCount < 4+8 {
		t.Fatalf("sketch after 2 entities = %+v, want 2 entities + rect/circle curves", sk.Sketch)
	}

	var list wire.ListDrawingSketchesResult
	call(t, r, s, "drawingSketches.list", "{}", &list)
	if len(list.Sketches) != 1 {
		t.Fatalf("sketches = %d, want 1", len(list.Sketches))
	}
	if _, err := r.Handle(s, "drawingSketches.addEntity", []byte(`{"sketchName":"NOPE","kind":"line","points":[[0,0],[1,1]]}`)); err == nil {
		t.Error("addEntity on a missing sketch = ok, want error")
	}
	if _, err := r.Handle(s, "drawingSketches.addEntity", []byte(`{"sketchName":"S1","kind":"bogus","points":[[0,0]]}`)); err == nil {
		t.Error("addEntity with a bad kind = ok, want error")
	}
}
