// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/drawing"
)

// The drawing-view surface (M14-F02 PBI-139, #386): project the referenced model onto the
// active sheet as base and projected views, and read the hidden-line drawing curves. Views are
// re-projected on read so they track model edits (associativity).

// registerDrawingViewHandlers wires the drawingViews.* methods.
func (r *Router) registerDrawingViewHandlers() {
	r.handlers[wire.MethodDrawingViewsList] = drawingViewsList
	r.handlers[wire.MethodDrawingViewsAddBase] = drawingViewsAddBase
	r.handlers[wire.MethodDrawingViewsAddProjected] = drawingViewsAddProjected
	r.handlers[wire.MethodDrawingViewsAddAuxiliary] = drawingViewsAddAuxiliary
	r.handlers[wire.MethodDrawingViewsDelete] = drawingViewsDelete
	r.handlers[wire.MethodDrawingViewsCurves] = drawingViewsCurves
}

// activeSheetViews returns the active drawing's active-sheet view collection.
func activeSheetViews(s *app.Session) (*drawing.DrawingViews, error) {
	c, err := activeDrawing(s)
	if err != nil {
		return nil, err
	}
	sheet := c.Sheets().Active()
	if sheet == nil {
		return nil, fmt.Errorf("drawing: no active sheet")
	}
	return sheet.Views(), nil
}

func drawingViewsList(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	views, err := activeSheetViews(s)
	if err != nil {
		return nil, err
	}
	views.Recompute() // reflect any model edit before reporting counts
	out := wire.ListDrawingViewsResult{}
	for i := 0; i < views.Count(); i++ {
		out.Views = append(out.Views, drawingViewInfo(views.Item(i)))
	}
	return json.Marshal(out)
}

func drawingViewsAddBase(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	views, err := activeSheetViews(s)
	if err != nil {
		return nil, err
	}
	var in wire.AddBaseViewArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	orient, err := parseOrientation(in.Orientation)
	if err != nil {
		return nil, err
	}
	style, err := parseViewStyle(in.Style)
	if err != nil {
		return nil, err
	}
	v, err := views.AddBase(drawing.BaseViewSpec{
		Name: in.Name, Orientation: orient, Style: style, Scale: in.Scale,
		CenterX: in.CenterXMM, CenterY: in.CenterYMM,
	})
	if err != nil {
		return nil, err
	}
	s.ActiveDocument().MarkDirty()
	return json.Marshal(wire.ViewResult{View: drawingViewInfo(v)})
}

func drawingViewsAddProjected(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	views, err := activeSheetViews(s)
	if err != nil {
		return nil, err
	}
	var in wire.AddProjectedViewArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	dir, ok := types.ParseProjectionDirection(in.Direction)
	if !ok {
		return nil, fmt.Errorf("drawing: unknown projection direction %q", in.Direction)
	}
	v, err := views.AddProjected(drawing.ProjectedViewSpec{
		Name: in.Name, BaseView: in.BaseView, Direction: dir, CenterX: in.CenterXMM, CenterY: in.CenterYMM,
	})
	if err != nil {
		return nil, err
	}
	s.ActiveDocument().MarkDirty()
	return json.Marshal(wire.ViewResult{View: drawingViewInfo(v)})
}

func drawingViewsAddAuxiliary(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	views, err := activeSheetViews(s)
	if err != nil {
		return nil, err
	}
	var in wire.AddAuxiliaryViewArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	v, err := views.AddAuxiliary(drawing.AuxiliaryViewSpec{
		Name: in.Name, ParentView: in.ParentView, FoldAngleRad: in.FoldAngleDeg * math.Pi / 180,
		CenterX: in.CenterXMM, CenterY: in.CenterYMM,
	})
	if err != nil {
		return nil, err
	}
	s.ActiveDocument().MarkDirty()
	return json.Marshal(wire.ViewResult{View: drawingViewInfo(v)})
}

func drawingViewsDelete(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	views, err := activeSheetViews(s)
	if err != nil {
		return nil, err
	}
	var in wire.DeleteViewArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	if err := views.Remove(in.Name); err != nil {
		return nil, err
	}
	s.ActiveDocument().MarkDirty()
	out := wire.ListDrawingViewsResult{}
	for i := 0; i < views.Count(); i++ {
		out.Views = append(out.Views, drawingViewInfo(views.Item(i)))
	}
	return json.Marshal(out)
}

func drawingViewsCurves(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	views, err := activeSheetViews(s)
	if err != nil {
		return nil, err
	}
	var in wire.ViewCurvesArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	views.Recompute() // re-project against the current model
	v, ok := views.ByName(in.View)
	if !ok {
		return nil, fmt.Errorf("drawing: no view named %q", in.View)
	}
	out := wire.ViewCurvesResult{}
	for _, c := range v.Curves() {
		out.Segments = append(out.Segments, wire.DrawingCurveSegment{
			AX: float64(c.Start().X), AY: float64(c.Start().Y),
			BX: float64(c.End().X), BY: float64(c.End().Y),
			Visible: c.IsVisible(), Kind: c.Kind().String(), EdgeKey: hex.EncodeToString(c.EdgeKey()),
		})
	}
	return json.Marshal(out)
}

// drawingViewInfo flattens a drawing view into its wire DTO.
func drawingViewInfo(v *drawing.DrawingView) wire.DrawingViewInfo {
	visible, hidden := v.VisibleHidden()
	x, y := v.CenterMM()
	info := wire.DrawingViewInfo{
		Name: v.Name(), Type: v.Type().String(), Projected: v.IsProjected(),
		Orientation: v.Orientation().String(), Scale: v.Scale(), Style: v.Style().String(),
		CenterXMM: x, CenterYMM: y, VisibleCount: visible, HiddenCount: hidden,
	}
	switch v.Type() {
	case types.DrawingViewProjected:
		info.BaseView = v.BaseViewName()
		info.Direction = v.Direction().String()
	case types.DrawingViewAuxiliary:
		info.BaseView = v.BaseViewName()
		info.FoldAngleDeg = v.FoldAngle() * 180 / math.Pi
	}
	return info
}

func parseOrientation(spelling string) (types.BaseViewOrientation, error) {
	if spelling == "" {
		return types.BaseViewFront, nil
	}
	o, ok := types.ParseBaseViewOrientation(spelling)
	if !ok {
		return 0, fmt.Errorf("drawing: unknown base view orientation %q", spelling)
	}
	return o, nil
}

func parseViewStyle(spelling string) (types.DrawingViewStyle, error) {
	if spelling == "" {
		return types.HiddenLineViewStyle, nil
	}
	st, ok := types.ParseDrawingViewStyle(spelling)
	if !ok {
		return 0, fmt.Errorf("drawing: unknown view style %q", spelling)
	}
	return st, nil
}
