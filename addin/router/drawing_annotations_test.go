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
