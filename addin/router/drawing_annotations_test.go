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

// TestDrawingDatumFeatureOverWire drives the GD&T datum surface: a datum feature symbol produces a
// framed annotation (box + triangle) through the live stack.
func TestDrawingDatumFeatureOverWire(t *testing.T) {
	r, s := drawingViewSession(t)
	var dat wire.AnnotationResult
	call(t, r, s, "drawingAnnotations.addDatumFeature", `{"name":"DAT","xmm":70,"ymm":70,"letter":"A"}`, &dat)
	if dat.Annotation.Kind != "datumFeature" || dat.Annotation.CurveCount == 0 {
		t.Fatalf("datum = %+v, want a datumFeature with box+triangle geometry", dat.Annotation)
	}
	if _, err := r.Handle(s, "drawingAnnotations.addDatumFeature", []byte(`{"xmm":0,"ymm":0,"letter":""}`)); err == nil {
		t.Error("addDatumFeature with no letter = ok, want error")
	}
}

// TestDrawingSurfaceTextureOverWire drives the surface-texture surface: a machined surface texture
// symbol with a roughness value produces a checkmark annotation through the live stack.
func TestDrawingSurfaceTextureOverWire(t *testing.T) {
	r, s := drawingViewSession(t)
	var st wire.AnnotationResult
	call(t, r, s, "drawingAnnotations.addSurfaceTexture",
		`{"name":"ST","xmm":80,"ymm":80,"roughness":"1.6","materialRemoval":"required"}`, &st)
	if st.Annotation.Kind != "surfaceTexture" || st.Annotation.CurveCount < 3 {
		t.Fatalf("surface texture = %+v, want a surfaceTexture with checkmark geometry", st.Annotation)
	}
	if _, err := r.Handle(s, "drawingAnnotations.addSurfaceTexture", []byte(`{"xmm":0,"ymm":0,"materialRemoval":"bogus"}`)); err == nil {
		t.Error("addSurfaceTexture with a bad variant = ok, want error")
	}
}

// TestDrawingPartsListOverWire drives the parts-list surface: a drawing referencing an assembly of
// two distinct parts gets a parts list whose row count reflects the parts-only BOM, through the
// live stack.
func TestDrawingPartsListOverWire(t *testing.T) {
	r, s, _, _ := assemblySessionWithBoxes(t, 0, 2) // assembly "asm.obk" with two distinct parts
	call(t, r, s, "documents.create", `{"type":"drawing","name":"asm.odd"}`, nil)
	call(t, r, s, "drawing.setModelReference", `{"fullDocumentName":"asm.obk"}`, nil)

	var pl wire.AnnotationResult
	call(t, r, s, "drawingAnnotations.addPartsList", `{"name":"PL","xmm":40,"ymm":260}`, &pl)
	if pl.Annotation.Kind != "partsList" || pl.Annotation.CurveCount == 0 {
		t.Fatalf("parts list = %+v, want a partsList with grid geometry", pl.Annotation)
	}
	if pl.Annotation.RowCount != 2 {
		t.Errorf("parts list rowCount = %d, want 2 (two distinct parts)", pl.Annotation.RowCount)
	}
}

// TestDrawingPartsListNeedsAssembly: a parts list on a drawing referencing a part (not an assembly)
// errors.
func TestDrawingPartsListNeedsAssembly(t *testing.T) {
	r, s := drawingViewSession(t) // references a box PART, not an assembly
	if _, err := r.Handle(s, "drawingAnnotations.addPartsList", []byte(`{"xmm":40,"ymm":260}`)); err == nil {
		t.Error("addPartsList on a part-referencing drawing = ok, want error")
	}
}

// TestDrawingBalloonOverWire drives the balloon surface: a balloon with a leader produces a circle
// + leader annotation carrying its item number through the live stack.
func TestDrawingBalloonOverWire(t *testing.T) {
	r, s := drawingViewSession(t)
	var b wire.AnnotationResult
	call(t, r, s, "drawingAnnotations.addBalloon", `{"name":"B","xmm":100,"ymm":200,"item":3,"leaderXmm":120,"leaderYmm":180}`, &b)
	if b.Annotation.Kind != "balloon" || b.Annotation.CurveCount == 0 || b.Annotation.Tag != "3" {
		t.Fatalf("balloon = %+v, want a balloon (circle+leader) tagged item 3", b.Annotation)
	}
	if _, err := r.Handle(s, "drawingAnnotations.addBalloon", []byte(`{"xmm":0,"ymm":0,"item":0}`)); err == nil {
		t.Error("addBalloon with item 0 = ok, want error")
	}
}

// TestDrawingHoleTableOverWire drives the hole-table surface: a cylinder's TOP view yields a
// one-row table (its two coincident rims dedup) carrying its grid geometry through the live stack.
func TestDrawingHoleTableOverWire(t *testing.T) {
	r, s := drawingCylinderSession(t)
	call(t, r, s, "drawingViews.addBase", `{"name":"TOP","orientation":"top","scale":1,"centerXmm":100,"centerYmm":100}`, nil)

	var ht wire.AnnotationResult
	call(t, r, s, "drawingAnnotations.addHoleTable", `{"name":"HT","viewName":"TOP","xmm":220,"ymm":240}`, &ht)
	if ht.Annotation.Kind != "holeTable" || ht.Annotation.CurveCount == 0 {
		t.Fatalf("hole table = %+v, want a holeTable with grid geometry", ht.Annotation)
	}
	if ht.Annotation.RowCount != 1 {
		t.Errorf("hole table rowCount = %d, want 1 (dedup coincident rims)", ht.Annotation.RowCount)
	}
	if _, err := r.Handle(s, "drawingAnnotations.addHoleTable", []byte(`{"viewName":"NOPE","xmm":0,"ymm":0}`)); err == nil {
		t.Error("addHoleTable on a missing view = ok, want error")
	}
}

// TestDrawingRevisionTableOverWire drives the revision-table surface: a table with two rows
// produces a grid annotation reporting the row count through the live stack.
func TestDrawingRevisionTableOverWire(t *testing.T) {
	r, s := drawingViewSession(t)
	var rt wire.AnnotationResult
	call(t, r, s, "drawingAnnotations.addRevisionTable",
		`{"name":"RT","xmm":250,"ymm":60,"rows":[{"revision":"A","date":"2026-06-01","description":"Initial release"},{"revision":"B","date":"2026-06-18","description":"Added holes"}]}`, &rt)
	if rt.Annotation.Kind != "revisionTable" || rt.Annotation.CurveCount == 0 || rt.Annotation.RowCount != 2 {
		t.Fatalf("revision table = %+v, want a revisionTable with 2 rows + grid geometry", rt.Annotation)
	}
	if _, err := r.Handle(s, "drawingAnnotations.addRevisionTable", []byte(`{"xmm":0,"ymm":0}`)); err == nil {
		t.Error("addRevisionTable with no rows = ok, want error")
	}
}

// TestDrawingRevisionTagOverWire drives the revision-tag surface: a tag produces a triangle
// annotation carrying its revision letter through the live stack.
func TestDrawingRevisionTagOverWire(t *testing.T) {
	r, s := drawingViewSession(t)
	var tag wire.AnnotationResult
	call(t, r, s, "drawingAnnotations.addRevisionTag", `{"name":"RT1","xmm":120,"ymm":90,"revision":"B"}`, &tag)
	if tag.Annotation.Kind != "revisionTag" || tag.Annotation.CurveCount == 0 || tag.Annotation.Tag != "B" {
		t.Fatalf("revision tag = %+v, want a revisionTag triangle tagged B", tag.Annotation)
	}
	if _, err := r.Handle(s, "drawingAnnotations.addRevisionTag", []byte(`{"xmm":0,"ymm":0,"revision":""}`)); err == nil {
		t.Error("addRevisionTag with no revision = ok, want error")
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
