// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
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
	r.handlers[wire.MethodDrawingAnnotationsList] = drawingAnnotationsList
	r.handlers[wire.MethodDrawingAnnotationsAddCoG] = drawingAnnotationsAddCoG
	r.handlers[wire.MethodDrawingAnnotationsAddRevisionCloud] = drawingAnnotationsAddRevisionCloud
	r.handlers[wire.MethodDrawingAnnotationsAddCenterMarks] = drawingAnnotationsAddCenterMarks
	r.handlers[wire.MethodDrawingAnnotationsAddCenterlines] = drawingAnnotationsAddCenterlines
	r.handlers[wire.MethodDrawingAnnotationsAddFCF] = drawingAnnotationsAddFCF
	r.handlers[wire.MethodDrawingAnnotationsAddDatum] = drawingAnnotationsAddDatum
	r.handlers[wire.MethodDrawingAnnotationsAddSurfaceText] = drawingAnnotationsAddSurfaceText
	r.handlers[wire.MethodDrawingAnnotationsAddPartsList] = drawingAnnotationsAddPartsList
	r.handlers[wire.MethodDrawingAnnotationsAddBalloon] = drawingAnnotationsAddBalloon
	r.handlers[wire.MethodDrawingAnnotationsAddHoleTable] = drawingAnnotationsAddHoleTable
	r.handlers[wire.MethodDrawingAnnotationsAddRevTable] = drawingAnnotationsAddRevisionTable
	r.handlers[wire.MethodDrawingAnnotationsAddRevTag] = drawingAnnotationsAddRevisionTag
	r.handlers[wire.MethodDrawingAnnotationsAddNote] = drawingAnnotationsAddNote
	r.handlers[wire.MethodDrawingAnnotationsAddCustomTable] = drawingAnnotationsAddCustomTable
	r.handlers[wire.MethodDrawingAnnotationsDelete] = drawingAnnotationsDelete
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

func drawingAnnotationsList(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	an, err := activeSheetAnnotations(s)
	if err != nil {
		return nil, err
	}
	out := wire.ListDrawingAnnotationsResult{}
	for i := 0; i < an.Count(); i++ {
		out.Annotations = append(out.Annotations, drawingAnnotationInfo(an.Item(i)))
	}
	return json.Marshal(out)
}

func drawingAnnotationsAddCoG(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	an, err := activeSheetAnnotations(s)
	if err != nil {
		return nil, err
	}
	var in wire.AddCoGMarkerArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	a, err := an.AddCoGMarker(in.Name, in.ViewName)
	if err != nil {
		return nil, err
	}
	s.ActiveDocument().MarkDirty()
	return json.Marshal(wire.AnnotationResult{Annotation: drawingAnnotationInfo(a)})
}

func drawingAnnotationsAddRevisionCloud(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	an, err := activeSheetAnnotations(s)
	if err != nil {
		return nil, err
	}
	var in wire.AddRevisionCloudArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	a, err := an.AddRevisionCloud(in.Name, in.XMM, in.YMM, in.WidthMM, in.HeightMM, in.Tag)
	if err != nil {
		return nil, err
	}
	s.ActiveDocument().MarkDirty()
	return json.Marshal(wire.AnnotationResult{Annotation: drawingAnnotationInfo(a)})
}

func drawingAnnotationsAddCenterMarks(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	an, err := activeSheetAnnotations(s)
	if err != nil {
		return nil, err
	}
	var in wire.AddCenterMarksArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	marks, err := an.AddCenterMarks(in.ViewName)
	if err != nil {
		return nil, err
	}
	s.ActiveDocument().MarkDirty()
	out := wire.CenterMarksResult{}
	for _, m := range marks {
		out.Annotations = append(out.Annotations, drawingAnnotationInfo(m))
	}
	return json.Marshal(out)
}

func drawingAnnotationsAddCenterlines(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	an, err := activeSheetAnnotations(s)
	if err != nil {
		return nil, err
	}
	var in wire.AddCenterlinesArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	a, err := an.AddCenterlines(in.Name, in.ViewName)
	if err != nil {
		return nil, err
	}
	s.ActiveDocument().MarkDirty()
	return json.Marshal(wire.AnnotationResult{Annotation: drawingAnnotationInfo(a)})
}

func drawingAnnotationsAddFCF(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	an, err := activeSheetAnnotations(s)
	if err != nil {
		return nil, err
	}
	var in wire.AddFeatureControlFrameArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	characteristic, ok := types.ParseGeometricCharacteristic(in.Characteristic)
	if !ok {
		return nil, fmt.Errorf("drawing: unknown geometric characteristic %q", in.Characteristic)
	}
	a, err := an.AddFeatureControlFrame(in.Name, in.XMM, in.YMM, characteristic, in.Tolerance, in.Datums)
	if err != nil {
		return nil, err
	}
	s.ActiveDocument().MarkDirty()
	return json.Marshal(wire.AnnotationResult{Annotation: drawingAnnotationInfo(a)})
}

func drawingAnnotationsAddDatum(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	an, err := activeSheetAnnotations(s)
	if err != nil {
		return nil, err
	}
	var in wire.AddDatumFeatureArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	a, err := an.AddDatumFeature(in.Name, in.XMM, in.YMM, in.Letter)
	if err != nil {
		return nil, err
	}
	s.ActiveDocument().MarkDirty()
	return json.Marshal(wire.AnnotationResult{Annotation: drawingAnnotationInfo(a)})
}

func drawingAnnotationsAddSurfaceText(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	an, err := activeSheetAnnotations(s)
	if err != nil {
		return nil, err
	}
	var in wire.AddSurfaceTextureArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	variant := types.MaterialRemovalAny
	if in.MaterialRemoval != "" {
		v, ok := types.ParseMaterialRemoval(in.MaterialRemoval)
		if !ok {
			return nil, fmt.Errorf("drawing: unknown material-removal variant %q (want any|required|prohibited)", in.MaterialRemoval)
		}
		variant = v
	}
	a, err := an.AddSurfaceTexture(in.Name, in.XMM, in.YMM, in.Roughness, variant)
	if err != nil {
		return nil, err
	}
	s.ActiveDocument().MarkDirty()
	return json.Marshal(wire.AnnotationResult{Annotation: drawingAnnotationInfo(a)})
}

func drawingAnnotationsAddPartsList(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	an, err := activeSheetAnnotations(s)
	if err != nil {
		return nil, err
	}
	var in wire.AddPartsListArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	a, err := an.AddPartsList(in.Name, in.XMM, in.YMM)
	if err != nil {
		return nil, err
	}
	s.ActiveDocument().MarkDirty()
	return json.Marshal(wire.AnnotationResult{Annotation: drawingAnnotationInfo(a)})
}

func drawingAnnotationsAddBalloon(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	an, err := activeSheetAnnotations(s)
	if err != nil {
		return nil, err
	}
	var in wire.AddBalloonArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	a, err := an.AddBalloon(in.Name, in.XMM, in.YMM, in.Item, in.LeaderXMM, in.LeaderYMM)
	if err != nil {
		return nil, err
	}
	s.ActiveDocument().MarkDirty()
	return json.Marshal(wire.AnnotationResult{Annotation: drawingAnnotationInfo(a)})
}

func drawingAnnotationsAddHoleTable(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	an, err := activeSheetAnnotations(s)
	if err != nil {
		return nil, err
	}
	var in wire.AddHoleTableArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	a, err := an.AddHoleTable(in.Name, in.ViewName, in.XMM, in.YMM)
	if err != nil {
		return nil, err
	}
	s.ActiveDocument().MarkDirty()
	return json.Marshal(wire.AnnotationResult{Annotation: drawingAnnotationInfo(a)})
}

func drawingAnnotationsAddRevisionTable(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	an, err := activeSheetAnnotations(s)
	if err != nil {
		return nil, err
	}
	var in wire.AddRevisionTableArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	rows := make([]drawing.RevisionRow, len(in.Rows))
	for i, r := range in.Rows {
		rows[i] = drawing.RevisionRow{Revision: r.Revision, Date: r.Date, Description: r.Description}
	}
	a, err := an.AddRevisionTable(in.Name, in.XMM, in.YMM, rows)
	if err != nil {
		return nil, err
	}
	s.ActiveDocument().MarkDirty()
	return json.Marshal(wire.AnnotationResult{Annotation: drawingAnnotationInfo(a)})
}

func drawingAnnotationsAddRevisionTag(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	an, err := activeSheetAnnotations(s)
	if err != nil {
		return nil, err
	}
	var in wire.AddRevisionTagArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	a, err := an.AddRevisionTag(in.Name, in.XMM, in.YMM, in.Revision)
	if err != nil {
		return nil, err
	}
	s.ActiveDocument().MarkDirty()
	return json.Marshal(wire.AnnotationResult{Annotation: drawingAnnotationInfo(a)})
}

func drawingAnnotationsAddNote(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	an, err := activeSheetAnnotations(s)
	if err != nil {
		return nil, err
	}
	var in wire.AddDrawingNoteArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	a, err := an.AddNote(in.Name, in.XMM, in.YMM, in.Text, in.LeaderXMM, in.LeaderYMM)
	if err != nil {
		return nil, err
	}
	s.ActiveDocument().MarkDirty()
	return json.Marshal(wire.AnnotationResult{Annotation: drawingAnnotationInfo(a)})
}

func drawingAnnotationsAddCustomTable(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	an, err := activeSheetAnnotations(s)
	if err != nil {
		return nil, err
	}
	var in wire.AddCustomTableArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	a, err := an.AddCustomTable(in.Name, in.XMM, in.YMM, in.Headers, in.Rows)
	if err != nil {
		return nil, err
	}
	s.ActiveDocument().MarkDirty()
	return json.Marshal(wire.AnnotationResult{Annotation: drawingAnnotationInfo(a)})
}

func drawingAnnotationsDelete(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	an, err := activeSheetAnnotations(s)
	if err != nil {
		return nil, err
	}
	var in wire.DeleteAnnotationArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	if err := an.Remove(in.Name); err != nil {
		return nil, err
	}
	s.ActiveDocument().MarkDirty()
	out := wire.ListDrawingAnnotationsResult{}
	for i := 0; i < an.Count(); i++ {
		out.Annotations = append(out.Annotations, drawingAnnotationInfo(an.Item(i)))
	}
	return json.Marshal(out)
}

// drawingAnnotationInfo flattens an annotation into its wire DTO.
func drawingAnnotationInfo(a *drawing.DrawingAnnotation) wire.DrawingAnnotationInfo {
	return wire.DrawingAnnotationInfo{
		Name: a.Name(), Kind: a.Kind().String(), ViewName: a.ViewName(), Tag: a.Tag(),
		CurveCount: a.CurveCount(), RowCount: a.RowCount(),
	}
}
