// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"oblikovati.org/addin/opregistry"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

// drawingViewSession seeds a boxed part "box.opd" and an active drawing that references it —
// the fixture for the drawingViews.* handlers.
func drawingViewSession(t *testing.T) (*Router, *app.Session) {
	t.Helper()
	r := New(opregistry.Default())
	s := app.NewSession()
	part, err := compdef.AddPart(s.Workspace(), "box.opd", true)
	if err != nil {
		t.Fatalf("AddPart: %v", err)
	}
	def := part.Content().(*compdef.PartComponentDefinition)
	sk := def.Sketches().Add(sketch.XYPlane())
	c0 := sk.Points().Add(math.P2(-2, -3))
	c1 := sk.Points().Add(math.P2(2, -3))
	c2 := sk.Points().Add(math.P2(2, 3))
	c3 := sk.Points().Add(math.P2(-2, 3))
	sk.Lines().Add(c0, c1)
	sk.Lines().Add(c1, c2)
	sk.Lines().Add(c2, c3)
	sk.Lines().Add(c3, c0)
	feature.NewExtrudeFeatures(def.Features()).AddByDistanceExtent(sk, 0, ops.NewBody, func() float64 { return 5 })
	def.Recompute()

	var info wire.DocumentInfo
	call(t, r, s, "documents.create", `{"type":"drawing","name":"box.odd"}`, &info)
	call(t, r, s, "drawing.setModelReference", `{"fullDocumentName":"box.opd"}`, nil)
	return r, s
}

// TestDrawingViewsLifecycleOverWire drives the whole drawing-view surface: a base view + a
// projected view off it, the curve readback (visible + hidden, keyed), and the cascading delete.
func TestDrawingViewsLifecycleOverWire(t *testing.T) {
	t.Parallel()
	r, s := drawingViewSession(t)

	var base wire.ViewResult
	call(t, r, s, "drawingViews.addBase", `{"name":"FRONT","orientation":"front","scale":2,"centerXmm":120,"centerYmm":100}`, &base)
	if base.View.Name != "FRONT" || base.View.Projected {
		t.Fatalf("base view = %+v, want a base view FRONT", base.View)
	}
	if base.View.VisibleCount == 0 {
		t.Error("base view has no visible curves")
	}

	var proj wire.ViewResult
	call(t, r, s, "drawingViews.addProjected", `{"name":"RIGHT","baseView":"FRONT","direction":"right","centerXmm":240,"centerYmm":100}`, &proj)
	if !proj.View.Projected || proj.View.BaseView != "FRONT" {
		t.Fatalf("projected view = %+v, want projected off FRONT", proj.View)
	}

	var aux wire.ViewResult
	call(t, r, s, "drawingViews.addAuxiliary", `{"name":"AUX","parentView":"FRONT","foldAngleDeg":0,"centerXmm":120,"centerYmm":240}`, &aux)
	if aux.View.Type != "auxiliary" || aux.View.BaseView != "FRONT" || aux.View.Projected {
		t.Fatalf("auxiliary view = %+v, want an auxiliary off FRONT", aux.View)
	}
	if aux.View.VisibleCount == 0 {
		t.Error("auxiliary view has no visible curves")
	}

	var det wire.ViewResult
	call(t, r, s, "drawingViews.addDetail", `{"name":"DET","parentView":"FRONT","boundaryXmm":120,"boundaryYmm":100,"radiusMm":40,"scale":4,"centerXmm":360,"centerYmm":100}`, &det)
	if det.View.Type != "detail" || det.View.BaseView != "FRONT" {
		t.Fatalf("detail view = %+v, want a detail off FRONT", det.View)
	}

	var brk wire.ViewResult
	call(t, r, s, "drawingViews.addBreak", `{"name":"BRK","parentView":"FRONT","orientation":"horizontal","gapStartMm":110,"gapEndMm":130,"centerXmm":120,"centerYmm":320}`, &brk)
	if brk.View.Type != "break" || brk.View.BaseView != "FRONT" {
		t.Fatalf("break view = %+v, want a break off FRONT", brk.View)
	}

	var list wire.ListDrawingViewsResult
	call(t, r, s, "drawingViews.list", "{}", &list)
	if len(list.Views) != 5 {
		t.Fatalf("views = %d, want 5 (base + projected + auxiliary + detail + break)", len(list.Views))
	}

	var curves wire.ViewCurvesResult
	call(t, r, s, "drawingViews.curves", `{"view":"FRONT"}`, &curves)
	if len(curves.Segments) == 0 {
		t.Fatal("FRONT view returned no drawing curves")
	}
	var hidden int
	for _, seg := range curves.Segments {
		if seg.EdgeKey == "" {
			t.Errorf("segment %+v has no edge key", seg)
		}
		if !seg.Visible {
			hidden++
		}
	}
	if hidden == 0 {
		t.Error("front view of a block should have hidden (back) edges")
	}

	// Deleting the base view cascades to the projected view.
	var after wire.ListDrawingViewsResult
	call(t, r, s, "drawingViews.delete", `{"name":"FRONT"}`, &after)
	if len(after.Views) != 0 {
		t.Errorf("views after deleting the base = %d, want 0 (projected cascades)", len(after.Views))
	}
}

// TestDrawingExportDXFOverWire checks the active sheet (with a view) exports to a DXF file.
func TestDrawingExportDXFOverWire(t *testing.T) {
	t.Parallel()
	r, s := drawingViewSession(t)
	call(t, r, s, "drawingViews.addBase", `{"name":"FRONT","orientation":"front","scale":2,"centerXmm":120,"centerYmm":100}`, nil)

	// Forward-slash the temp path so it is a valid JSON string on Windows (backslashes are
	// JSON escapes); Go's file ops accept forward slashes on every OS.
	path := filepath.ToSlash(t.TempDir()) + "/sheet.dxf"
	var res wire.ExportDrawingDXFResult
	call(t, r, s, "drawing.exportDXF", `{"path":"`+path+`","version":"r2018"}`, &res)
	if res.Entities == 0 || res.Path != path {
		t.Fatalf("export result = %+v, want entities written to %q", res, path)
	}
	data, err := os.ReadFile(path)
	if err != nil || !strings.Contains(string(data), "LINE") {
		t.Fatalf("exported DXF unreadable or empty: err=%v", err)
	}
}

// TestDrawingExtraViewsOverWire drives slice, breakout and draft over the wire.
func TestDrawingExtraViewsOverWire(t *testing.T) {
	t.Parallel()
	r, s := drawingViewSession(t)
	call(t, r, s, "drawingViews.addBase", `{"name":"FRONT","orientation":"front","scale":2,"centerXmm":120,"centerYmm":100}`, nil)

	var slice, breakout, draft wire.ViewResult
	call(t, r, s, "drawingViews.addSlice", `{"name":"SL","parentView":"FRONT","x1":80,"y1":100,"x2":160,"y2":100,"centerXmm":120,"centerYmm":240}`, &slice)
	if slice.View.Type != "slice" {
		t.Errorf("slice view type = %q, want slice", slice.View.Type)
	}
	call(t, r, s, "drawingViews.addBreakout", `{"name":"BO","parentView":"FRONT","boundaryXmm":120,"boundaryYmm":100,"radiusMm":60,"centerXmm":300,"centerYmm":100}`, &breakout)
	if breakout.View.Type != "breakout" || breakout.View.BaseView != "FRONT" {
		t.Errorf("breakout view = %+v, want breakout off FRONT", breakout.View)
	}
	call(t, r, s, "drawingViews.addDraft", `{"name":"DR","widthMm":80,"heightMm":50,"centerXmm":120,"centerYmm":340}`, &draft)
	if draft.View.Type != "draft" {
		t.Errorf("draft view type = %q, want draft", draft.View.Type)
	}
}

func TestDrawingViewsRejectBadArgs(t *testing.T) {
	t.Parallel()
	r, s := drawingViewSession(t)
	for method, args := range map[string]string{
		"drawingViews.addBase":      `{"orientation":"sideways"}`,
		"drawingViews.addProjected": `{"baseView":"NOPE","direction":"right"}`,
		"drawingViews.curves":       `{"view":"missing"}`,
	} {
		if _, err := r.Handle(s, method, []byte(args)); err == nil {
			t.Errorf("%s(%s) = ok, want error", method, args)
		}
	}
}
