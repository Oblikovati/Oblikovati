// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/drawing"
)

// The drawing-annotation surface (M14-F02 #813): a centre-of-gravity marker on a view (positioned
// from the referenced model's mass properties) and revision-cloud markup on the active sheet.

// registerDrawingAnnotationHandlers wires the drawingAnnotations.* methods.
func (r *Router) registerDrawingAnnotationHandlers() {
	r.readOnly(wire.MethodDrawingAnnotationsList, ctxQuery(activeSheetAnnotations, drawingAnnotationsList))
	r.mutating(wire.MethodDrawingAnnotationsAddCoG, "Add Annotation", typedCtx(activeSheetAnnotations, drawingAnnotationsAddCoG))
	r.mutating(wire.MethodDrawingAnnotationsAddRevisionCloud, "Add Annotation", typedCtx(activeSheetAnnotations, drawingAnnotationsAddRevisionCloud))
	r.mutating(wire.MethodDrawingAnnotationsAddCenterMarks, "Add Annotation", typedCtx(activeSheetAnnotations, drawingAnnotationsAddCenterMarks))
	r.mutating(wire.MethodDrawingAnnotationsAddCenterlines, "Add Annotation", typedCtx(activeSheetAnnotations, drawingAnnotationsAddCenterlines))
	r.mutating(wire.MethodDrawingAnnotationsAddFCF, "Add Annotation", typedCtx(activeSheetAnnotations, drawingAnnotationsAddFCF))
	r.mutating(wire.MethodDrawingAnnotationsAddDatum, "Add Annotation", typedCtx(activeSheetAnnotations, drawingAnnotationsAddDatum))
	r.mutating(wire.MethodDrawingAnnotationsAddSurfaceText, "Add Annotation", typedCtx(activeSheetAnnotations, drawingAnnotationsAddSurfaceText))
	r.mutating(wire.MethodDrawingAnnotationsAddPartsList, "Add Annotation", typedCtx(activeSheetAnnotations, drawingAnnotationsAddPartsList))
	r.mutating(wire.MethodDrawingAnnotationsAddBalloon, "Add Annotation", typedCtx(activeSheetAnnotations, drawingAnnotationsAddBalloon))
	r.mutating(wire.MethodDrawingAnnotationsAddHoleTable, "Add Annotation", typedCtx(activeSheetAnnotations, drawingAnnotationsAddHoleTable))
	r.mutating(wire.MethodDrawingAnnotationsAddRevTable, "Add Annotation", typedCtx(activeSheetAnnotations, drawingAnnotationsAddRevisionTable))
	r.mutating(wire.MethodDrawingAnnotationsAddRevTag, "Add Annotation", typedCtx(activeSheetAnnotations, drawingAnnotationsAddRevisionTag))
	r.mutating(wire.MethodDrawingAnnotationsAddNote, "Add Annotation", typedCtx(activeSheetAnnotations, drawingAnnotationsAddNote))
	r.mutating(wire.MethodDrawingAnnotationsAddCustomTable, "Add Annotation", typedCtx(activeSheetAnnotations, drawingAnnotationsAddCustomTable))
	r.mutating(wire.MethodDrawingAnnotationsAddHoleNotes, "Add Annotation", typedCtx(activeSheetAnnotations, drawingAnnotationsAddHoleNotes))
	r.mutating(wire.MethodDrawingAnnotationsDelete, "Delete Annotation", typedCtx(activeSheetAnnotations, drawingAnnotationsDelete))
}

// activeSheetAnnotations returns the active drawing's active-sheet annotation collection.
func activeSheetAnnotations(s *app.Session) (*drawing.DrawingAnnotations, error) {
	c, err := activeDrawing(s)
	if err != nil {
		return nil, err
	}
	sheet := c.Sheets().Active()
	if sheet == nil {
		return nil, fmt.Errorf("drawing: no active sheet")
	}
	return sheet.Annotations(), nil
}

func drawingAnnotationsList(_ *app.Session, an *drawing.DrawingAnnotations) (wire.ListDrawingAnnotationsResult, error) {
	return listAnnotationsResult(an), nil
}

// listAnnotationsResult flattens a sheet's annotation collection into its wire result (shared by
// the list and delete handlers). Kept as append-to-nil so an empty collection marshals to null.
func listAnnotationsResult(an *drawing.DrawingAnnotations) wire.ListDrawingAnnotationsResult {
	out := wire.ListDrawingAnnotationsResult{}
	for i := 0; i < an.Count(); i++ {
		out.Annotations = append(out.Annotations, drawingAnnotationInfo(an.Item(i)))
	}
	return out
}

func drawingAnnotationsAddCoG(s *app.Session, an *drawing.DrawingAnnotations, in wire.AddCoGMarkerArgs) (wire.AnnotationResult, error) {
	a, err := an.AddCoGMarker(in.Name, in.ViewName)
	if err != nil {
		return wire.AnnotationResult{}, err
	}
	s.ActiveDocument().MarkDirty()
	return wire.AnnotationResult{Annotation: drawingAnnotationInfo(a)}, nil
}

func drawingAnnotationsAddRevisionCloud(s *app.Session, an *drawing.DrawingAnnotations, in wire.AddRevisionCloudArgs) (wire.AnnotationResult, error) {
	a, err := an.AddRevisionCloud(in.Name, in.XMM, in.YMM, in.WidthMM, in.HeightMM, in.Tag)
	if err != nil {
		return wire.AnnotationResult{}, err
	}
	s.ActiveDocument().MarkDirty()
	return wire.AnnotationResult{Annotation: drawingAnnotationInfo(a)}, nil
}

func drawingAnnotationsAddCenterMarks(s *app.Session, an *drawing.DrawingAnnotations, in wire.AddCenterMarksArgs) (wire.CenterMarksResult, error) {
	marks, err := an.AddCenterMarks(in.ViewName)
	if err != nil {
		return wire.CenterMarksResult{}, err
	}
	s.ActiveDocument().MarkDirty()
	out := wire.CenterMarksResult{}
	for _, m := range marks {
		out.Annotations = append(out.Annotations, drawingAnnotationInfo(m))
	}
	return out, nil
}

func drawingAnnotationsAddCenterlines(s *app.Session, an *drawing.DrawingAnnotations, in wire.AddCenterlinesArgs) (wire.AnnotationResult, error) {
	a, err := an.AddCenterlines(in.Name, in.ViewName)
	if err != nil {
		return wire.AnnotationResult{}, err
	}
	s.ActiveDocument().MarkDirty()
	return wire.AnnotationResult{Annotation: drawingAnnotationInfo(a)}, nil
}

func drawingAnnotationsAddFCF(s *app.Session, an *drawing.DrawingAnnotations, in wire.AddFeatureControlFrameArgs) (wire.AnnotationResult, error) {
	characteristic, ok := types.ParseGeometricCharacteristic(in.Characteristic)
	if !ok {
		return wire.AnnotationResult{}, fmt.Errorf("drawing: unknown geometric characteristic %q", in.Characteristic)
	}
	a, err := an.AddFeatureControlFrame(in.Name, in.XMM, in.YMM, characteristic, in.Tolerance, in.Datums)
	if err != nil {
		return wire.AnnotationResult{}, err
	}
	s.ActiveDocument().MarkDirty()
	return wire.AnnotationResult{Annotation: drawingAnnotationInfo(a)}, nil
}

func drawingAnnotationsAddDatum(s *app.Session, an *drawing.DrawingAnnotations, in wire.AddDatumFeatureArgs) (wire.AnnotationResult, error) {
	a, err := an.AddDatumFeature(in.Name, in.XMM, in.YMM, in.Letter)
	if err != nil {
		return wire.AnnotationResult{}, err
	}
	s.ActiveDocument().MarkDirty()
	return wire.AnnotationResult{Annotation: drawingAnnotationInfo(a)}, nil
}

func drawingAnnotationsAddSurfaceText(s *app.Session, an *drawing.DrawingAnnotations, in wire.AddSurfaceTextureArgs) (wire.AnnotationResult, error) {
	variant := types.MaterialRemovalAny
	if in.MaterialRemoval != "" {
		v, ok := types.ParseMaterialRemoval(in.MaterialRemoval)
		if !ok {
			return wire.AnnotationResult{}, fmt.Errorf("drawing: unknown material-removal variant %q (want any|required|prohibited)", in.MaterialRemoval)
		}
		variant = v
	}
	a, err := an.AddSurfaceTexture(in.Name, in.XMM, in.YMM, in.Roughness, variant)
	if err != nil {
		return wire.AnnotationResult{}, err
	}
	s.ActiveDocument().MarkDirty()
	return wire.AnnotationResult{Annotation: drawingAnnotationInfo(a)}, nil
}

func drawingAnnotationsAddPartsList(s *app.Session, an *drawing.DrawingAnnotations, in wire.AddPartsListArgs) (wire.AnnotationResult, error) {
	a, err := an.AddPartsList(in.Name, in.XMM, in.YMM)
	if err != nil {
		return wire.AnnotationResult{}, err
	}
	s.ActiveDocument().MarkDirty()
	return wire.AnnotationResult{Annotation: drawingAnnotationInfo(a)}, nil
}

func drawingAnnotationsAddBalloon(s *app.Session, an *drawing.DrawingAnnotations, in wire.AddBalloonArgs) (wire.AnnotationResult, error) {
	a, err := an.AddBalloon(in.Name, in.XMM, in.YMM, in.Item, in.LeaderXMM, in.LeaderYMM)
	if err != nil {
		return wire.AnnotationResult{}, err
	}
	s.ActiveDocument().MarkDirty()
	return wire.AnnotationResult{Annotation: drawingAnnotationInfo(a)}, nil
}

func drawingAnnotationsAddHoleTable(s *app.Session, an *drawing.DrawingAnnotations, in wire.AddHoleTableArgs) (wire.AnnotationResult, error) {
	a, err := an.AddHoleTable(in.Name, in.ViewName, in.XMM, in.YMM)
	if err != nil {
		return wire.AnnotationResult{}, err
	}
	s.ActiveDocument().MarkDirty()
	return wire.AnnotationResult{Annotation: drawingAnnotationInfo(a)}, nil
}

func drawingAnnotationsAddRevisionTable(s *app.Session, an *drawing.DrawingAnnotations, in wire.AddRevisionTableArgs) (wire.AnnotationResult, error) {
	rows := make([]drawing.RevisionRow, len(in.Rows))
	for i, r := range in.Rows {
		rows[i] = drawing.RevisionRow{Revision: r.Revision, Date: r.Date, Description: r.Description}
	}
	a, err := an.AddRevisionTable(in.Name, in.XMM, in.YMM, rows)
	if err != nil {
		return wire.AnnotationResult{}, err
	}
	s.ActiveDocument().MarkDirty()
	return wire.AnnotationResult{Annotation: drawingAnnotationInfo(a)}, nil
}

func drawingAnnotationsAddRevisionTag(s *app.Session, an *drawing.DrawingAnnotations, in wire.AddRevisionTagArgs) (wire.AnnotationResult, error) {
	a, err := an.AddRevisionTag(in.Name, in.XMM, in.YMM, in.Revision)
	if err != nil {
		return wire.AnnotationResult{}, err
	}
	s.ActiveDocument().MarkDirty()
	return wire.AnnotationResult{Annotation: drawingAnnotationInfo(a)}, nil
}

func drawingAnnotationsAddNote(s *app.Session, an *drawing.DrawingAnnotations, in wire.AddDrawingNoteArgs) (wire.AnnotationResult, error) {
	a, err := an.AddNote(in.Name, in.XMM, in.YMM, in.Text, in.LeaderXMM, in.LeaderYMM)
	if err != nil {
		return wire.AnnotationResult{}, err
	}
	s.ActiveDocument().MarkDirty()
	return wire.AnnotationResult{Annotation: drawingAnnotationInfo(a)}, nil
}

func drawingAnnotationsAddCustomTable(s *app.Session, an *drawing.DrawingAnnotations, in wire.AddCustomTableArgs) (wire.AnnotationResult, error) {
	a, err := an.AddCustomTable(in.Name, in.XMM, in.YMM, in.Headers, in.Rows)
	if err != nil {
		return wire.AnnotationResult{}, err
	}
	s.ActiveDocument().MarkDirty()
	return wire.AnnotationResult{Annotation: drawingAnnotationInfo(a)}, nil
}

func drawingAnnotationsAddHoleNotes(s *app.Session, an *drawing.DrawingAnnotations, in wire.AddHoleNotesArgs) (wire.AnnotationResult, error) {
	quantity := types.HoleNotePerHole
	if in.Quantity != "" {
		q, ok := types.ParseHoleNoteQuantity(in.Quantity)
		if !ok {
			return wire.AnnotationResult{}, fmt.Errorf("drawing: unknown hole-note quantity %q (want perHole|combined)", in.Quantity)
		}
		quantity = q
	}
	a, err := an.AddHoleNotes(in.Name, in.ViewName, quantity, in.Format)
	if err != nil {
		return wire.AnnotationResult{}, err
	}
	s.ActiveDocument().MarkDirty()
	return wire.AnnotationResult{Annotation: drawingAnnotationInfo(a)}, nil
}

func drawingAnnotationsDelete(s *app.Session, an *drawing.DrawingAnnotations, in wire.DeleteAnnotationArgs) (wire.ListDrawingAnnotationsResult, error) {
	if err := an.Remove(in.Name); err != nil {
		return wire.ListDrawingAnnotationsResult{}, err
	}
	s.ActiveDocument().MarkDirty()
	return listAnnotationsResult(an), nil
}

// drawingAnnotationInfo flattens an annotation into its wire DTO.
func drawingAnnotationInfo(a *drawing.DrawingAnnotation) wire.DrawingAnnotationInfo {
	return wire.DrawingAnnotationInfo{
		Name: a.Name(), Kind: a.Kind().String(), ViewName: a.ViewName(), Tag: a.Tag(),
		CurveCount: a.CurveCount(), RowCount: a.RowCount(), ThreadCount: a.ThreadCount(),
	}
}
