// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/wire"
)

// TestDrawingAnnotationsOverWire drives the CoG-marker and revision-cloud surface: add both off a
// base view, list them, and delete one.
func TestDrawingAnnotationsOverWire(t *testing.T) {
	r, s := drawingViewSession(t)
	call(t, r, s, "drawingViews.addBase", `{"name":"FRONT","orientation":"front","scale":2,"centerXmm":120,"centerYmm":100}`, nil)

	var cog, cloud wire.AnnotationResult
	call(t, r, s, "drawingAnnotations.addCoG", `{"name":"CG","viewName":"FRONT"}`, &cog)
	if cog.Annotation.Kind != "cog" || cog.Annotation.ViewName != "FRONT" || cog.Annotation.CurveCount == 0 {
		t.Fatalf("CoG marker = %+v, want a cog on FRONT with glyph curves", cog.Annotation)
	}
	call(t, r, s, "drawingAnnotations.addRevisionCloud", `{"name":"REV","xmm":40,"ymm":40,"widthMm":60,"heightMm":40,"tag":"A"}`, &cloud)
	if cloud.Annotation.Kind != "revisionCloud" || cloud.Annotation.Tag != "A" || cloud.Annotation.CurveCount == 0 {
		t.Fatalf("revision cloud = %+v, want a tagged scalloped cloud", cloud.Annotation)
	}

	var list wire.ListDrawingAnnotationsResult
	call(t, r, s, "drawingAnnotations.list", "{}", &list)
	if len(list.Annotations) != 2 {
		t.Fatalf("annotations = %d, want 2", len(list.Annotations))
	}
	var after wire.ListDrawingAnnotationsResult
	call(t, r, s, "drawingAnnotations.delete", `{"name":"CG"}`, &after)
	if len(after.Annotations) != 1 {
		t.Errorf("annotations after delete = %d, want 1", len(after.Annotations))
	}
}

// TestDrawingCenterMarksOverWire drives the centre-mark surface: auto-marking a cylinder's TOP view
// places one crosshair (its two coincident rims dedup) through the live stack.
func TestDrawingCenterMarksOverWire(t *testing.T) {
	r, s := drawingCylinderSession(t)
	call(t, r, s, "drawingViews.addBase", `{"name":"TOP","orientation":"top","scale":1,"centerXmm":100,"centerYmm":100}`, nil)

	var marks struct {
		Annotations []struct {
			Kind       string `json:"kind"`
			CurveCount int    `json:"curveCount"`
		} `json:"annotations"`
	}
	call(t, r, s, "drawingAnnotations.addCenterMarks", `{"viewName":"TOP"}`, &marks)
	if len(marks.Annotations) != 1 {
		t.Fatalf("centre marks = %d, want 1 (dedup coincident rims)", len(marks.Annotations))
	}
	if marks.Annotations[0].Kind != "centerMark" || marks.Annotations[0].CurveCount == 0 {
		t.Errorf("centre mark = %+v, want a centerMark with a crosshair glyph", marks.Annotations[0])
	}
	if _, err := r.Handle(s, "drawingAnnotations.addCenterMarks", []byte(`{"viewName":"NOPE"}`)); err == nil {
		t.Error("addCenterMarks on a missing view = ok, want error")
	}
}

// TestDrawingCenterlinesOverWire drives the centerline surface: adding centerlines to a base view
// produces one dash-dot annotation through the live stack.
func TestDrawingCenterlinesOverWire(t *testing.T) {
	r, s := drawingViewSession(t)
	call(t, r, s, "drawingViews.addBase", `{"name":"FRONT","orientation":"front","scale":2,"centerXmm":120,"centerYmm":100}`, nil)

	var cl wire.AnnotationResult
	call(t, r, s, "drawingAnnotations.addCenterlines", `{"name":"CL","viewName":"FRONT"}`, &cl)
	if cl.Annotation.Kind != "centerline" || cl.Annotation.CurveCount < 4 {
		t.Fatalf("centerlines = %+v, want a centerline with a dash-dot cross", cl.Annotation)
	}
	if _, err := r.Handle(s, "drawingAnnotations.addCenterlines", []byte(`{"viewName":"NOPE"}`)); err == nil {
		t.Error("addCenterlines on a missing view = ok, want error")
	}
}

// TestDrawingFeatureControlFrameOverWire drives the GD&T surface: a position FCF with two datums
// produces a frame annotation (with frame + symbol curves) through the live stack.
func TestDrawingFeatureControlFrameOverWire(t *testing.T) {
	r, s := drawingViewSession(t)
	var fcf wire.AnnotationResult
	call(t, r, s, "drawingAnnotations.addFeatureControlFrame",
		`{"name":"FCF","xmm":60,"ymm":60,"characteristic":"position","tolerance":"0.5","datums":["A","B"]}`, &fcf)
	if fcf.Annotation.Kind != "featureControlFrame" || fcf.Annotation.CurveCount == 0 {
		t.Fatalf("FCF = %+v, want a featureControlFrame with frame geometry", fcf.Annotation)
	}
	if _, err := r.Handle(s, "drawingAnnotations.addFeatureControlFrame", []byte(`{"xmm":0,"ymm":0,"characteristic":"bogus","tolerance":"1"}`)); err == nil {
		t.Error("addFeatureControlFrame with a bad characteristic = ok, want error")
	}
}

func TestDrawingAnnotationsRejectBadArgs(t *testing.T) {
	r, s := drawingViewSession(t)
	for method, args := range map[string]string{
		"drawingAnnotations.addCoG":           `{"viewName":"NOPE"}`,
		"drawingAnnotations.addRevisionCloud": `{"widthMm":0,"heightMm":0}`,
		"drawingAnnotations.delete":           `{"name":"missing"}`,
	} {
		if _, err := r.Handle(s, method, []byte(args)); err == nil {
			t.Errorf("%s(%s) = ok, want error", method, args)
		}
	}
}
