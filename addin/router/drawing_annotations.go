// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"

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
		Name: a.Name(), Kind: a.Kind().String(), ViewName: a.ViewName(), Tag: a.Tag(), CurveCount: a.CurveCount(),
	}
}
