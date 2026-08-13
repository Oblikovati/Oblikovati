// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/hex"
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
	r.readOnly(wire.MethodDrawingViewsList, ctxQuery(activeSheetViews, drawingViewsList))
	r.mutating(wire.MethodDrawingViewsAddBase, "Add View", typedCtx(activeSheetViews, drawingViewsAddBase))
	r.mutating(wire.MethodDrawingViewsAddProjected, "Add View", typedCtx(activeSheetViews, drawingViewsAddProjected))
	r.mutating(wire.MethodDrawingViewsAddAuxiliary, "Add View", typedCtx(activeSheetViews, drawingViewsAddAuxiliary))
	r.mutating(wire.MethodDrawingViewsAddSection, "Add View", typedCtx(activeSheetViews, drawingViewsAddSection))
	r.mutating(wire.MethodDrawingViewsAddDetail, "Add View", typedCtx(activeSheetViews, drawingViewsAddDetail))
	r.mutating(wire.MethodDrawingViewsAddBreak, "Add View", typedCtx(activeSheetViews, drawingViewsAddBreak))
	r.mutating(wire.MethodDrawingViewsAddSlice, "Add View", typedCtx(activeSheetViews, drawingViewsAddSlice))
	r.mutating(wire.MethodDrawingViewsAddBreakout, "Add View", typedCtx(activeSheetViews, drawingViewsAddBreakout))
	r.mutating(wire.MethodDrawingViewsAddDraft, "Add View", typedCtx(activeSheetViews, drawingViewsAddDraft))
	r.mutating(wire.MethodDrawingViewsDelete, "Delete View", typedCtx(activeSheetViews, drawingViewsDelete))
	r.mutating(wire.MethodDrawingViewsSetLabel, "Edit View Label", typedCtx(activeSheetViews, drawingViewsSetLabel))
	r.mutating(wire.MethodDrawingViewsSetDisplay, "Edit View Display", typedCtx(activeSheetViews, drawingViewsSetDisplay))
	r.mutating(wire.MethodDrawingViewsRotate, "Rotate View", typedCtx(activeSheetViews, drawingViewsRotate))
	r.mutating(wire.MethodDrawingViewsAlign, "Align View", typedCtx(activeSheetViews, drawingViewsAlign))
	r.mutating(wire.MethodDrawingViewsAddCrop, "Crop View", typedCtx(activeSheetViews, drawingViewsAddCrop))
	r.mutating(wire.MethodDrawingViewsRemoveCrop, "Remove Crop", typedCtx(activeSheetViews, drawingViewsRemoveCrop))
	r.readOnly(wire.MethodDrawingViewsCurves, typedCtx(activeSheetViews, drawingViewsCurves))
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

func drawingViewsList(_ *app.Session, views *drawing.DrawingViews) (wire.ListDrawingViewsResult, error) {
	views.Recompute() // reflect any model edit before reporting counts
	return listDrawingViewsResult(views), nil
}

// listDrawingViewsResult flattens a sheet's view collection into its wire result (shared by the
// list and delete handlers). Kept as append-to-nil so an empty collection marshals to null.
func listDrawingViewsResult(views *drawing.DrawingViews) wire.ListDrawingViewsResult {
	out := wire.ListDrawingViewsResult{}
	for i := 0; i < views.Count(); i++ {
		out.Views = append(out.Views, drawingViewInfo(views.Item(i)))
	}
	return out
}

func drawingViewsAddBase(s *app.Session, views *drawing.DrawingViews, in wire.AddBaseViewArgs) (wire.ViewResult, error) {
	orient, err := parseOrientation(in.Orientation)
	if err != nil {
		return wire.ViewResult{}, err
	}
	style, err := parseViewStyle(in.Style)
	if err != nil {
		return wire.ViewResult{}, err
	}
	v, err := views.AddBase(drawing.BaseViewSpec{
		Name: in.Name, Orientation: orient, Style: style, Scale: in.Scale,
		CenterX: in.CenterXMM, CenterY: in.CenterYMM,
	})
	if err != nil {
		return wire.ViewResult{}, err
	}
	s.ActiveDocument().MarkDirty()
	return wire.ViewResult{View: drawingViewInfo(v)}, nil
}

func drawingViewsAddProjected(s *app.Session, views *drawing.DrawingViews, in wire.AddProjectedViewArgs) (wire.ViewResult, error) {
	dir, ok := types.ParseProjectionDirection(in.Direction)
	if !ok {
		return wire.ViewResult{}, fmt.Errorf("drawing: unknown projection direction %q", in.Direction)
	}
	v, err := views.AddProjected(drawing.ProjectedViewSpec{
		Name: in.Name, BaseView: in.BaseView, Direction: dir, CenterX: in.CenterXMM, CenterY: in.CenterYMM,
	})
	if err != nil {
		return wire.ViewResult{}, err
	}
	s.ActiveDocument().MarkDirty()
	return wire.ViewResult{View: drawingViewInfo(v)}, nil
}

func drawingViewsAddAuxiliary(s *app.Session, views *drawing.DrawingViews, in wire.AddAuxiliaryViewArgs) (wire.ViewResult, error) {
	v, err := views.AddAuxiliary(drawing.AuxiliaryViewSpec{
		Name: in.Name, ParentView: in.ParentView, FoldAngleRad: in.FoldAngleDeg * math.Pi / 180,
		CenterX: in.CenterXMM, CenterY: in.CenterYMM,
	})
	if err != nil {
		return wire.ViewResult{}, err
	}
	s.ActiveDocument().MarkDirty()
	return wire.ViewResult{View: drawingViewInfo(v)}, nil
}

func drawingViewsAddSection(s *app.Session, views *drawing.DrawingViews, in wire.AddSectionViewArgs) (wire.ViewResult, error) {
	sectionType, ok := types.ParseSectionViewType(in.SectionType)
	if !ok {
		return wire.ViewResult{}, fmt.Errorf("drawing: unknown section type %q (want none|quarter|half|threeQuarter)", in.SectionType)
	}
	// SectionDepthMM only limits the cut when FullDepth is off; a full-depth (default) section
	// keeps everything behind the plane, matching Inventor's AddSectionView(FullDepth, SectionDepth)
	// where the depth is read only when FullDepth is false (#1982).
	depth := 0.0
	if !in.FullDepth && in.SectionDepthMM > 0 {
		depth = in.SectionDepthMM
	}
	v, err := views.AddSection(drawing.SectionViewSpec{
		Name: in.Name, ParentView: in.ParentView, X1: in.X1, Y1: in.Y1, X2: in.X2, Y2: in.Y2,
		CenterX: in.CenterXMM, CenterY: in.CenterYMM,
		Depth: depth, Reverse: in.Reverse, Type: sectionType,
	})
	if err != nil {
		return wire.ViewResult{}, err
	}
	s.ActiveDocument().MarkDirty()
	return wire.ViewResult{View: drawingViewInfo(v)}, nil
}

func drawingViewsAddDetail(s *app.Session, views *drawing.DrawingViews, in wire.AddDetailViewArgs) (wire.ViewResult, error) {
	v, err := views.AddDetail(drawing.DetailViewSpec{
		Name: in.Name, ParentView: in.ParentView, BoundaryX: in.BoundaryXMM, BoundaryY: in.BoundaryYMM,
		RadiusMM: in.RadiusMM, Scale: in.Scale, CenterX: in.CenterXMM, CenterY: in.CenterYMM,
	})
	if err != nil {
		return wire.ViewResult{}, err
	}
	s.ActiveDocument().MarkDirty()
	return wire.ViewResult{View: drawingViewInfo(v)}, nil
}

func drawingViewsAddBreak(s *app.Session, views *drawing.DrawingViews, in wire.AddBreakViewArgs) (wire.ViewResult, error) {
	orient := types.BreakHorizontal
	if in.Orientation != "" {
		o, ok := types.ParseBreakOrientation(in.Orientation)
		if !ok {
			return wire.ViewResult{}, fmt.Errorf("drawing: unknown break orientation %q", in.Orientation)
		}
		orient = o
	}
	v, err := views.AddBreak(drawing.BreakViewSpec{
		Name: in.Name, ParentView: in.ParentView, Orientation: orient,
		GapStart: in.GapStartMM, GapEnd: in.GapEndMM, CenterX: in.CenterXMM, CenterY: in.CenterYMM,
	})
	if err != nil {
		return wire.ViewResult{}, err
	}
	s.ActiveDocument().MarkDirty()
	return wire.ViewResult{View: drawingViewInfo(v)}, nil
}

func drawingViewsAddSlice(s *app.Session, views *drawing.DrawingViews, in wire.AddSliceViewArgs) (wire.ViewResult, error) {
	v, err := views.AddSlice(drawing.SliceViewSpec{
		Name: in.Name, ParentView: in.ParentView, X1: in.X1, Y1: in.Y1, X2: in.X2, Y2: in.Y2,
		CenterX: in.CenterXMM, CenterY: in.CenterYMM,
	})
	if err != nil {
		return wire.ViewResult{}, err
	}
	s.ActiveDocument().MarkDirty()
	return wire.ViewResult{View: drawingViewInfo(v)}, nil
}

func drawingViewsAddBreakout(s *app.Session, views *drawing.DrawingViews, in wire.AddBreakoutViewArgs) (wire.ViewResult, error) {
	v, err := views.AddBreakout(drawing.BreakoutViewSpec{
		Name: in.Name, ParentView: in.ParentView, BoundaryX: in.BoundaryXMM, BoundaryY: in.BoundaryYMM,
		RadiusMM: in.RadiusMM, CenterX: in.CenterXMM, CenterY: in.CenterYMM,
	})
	if err != nil {
		return wire.ViewResult{}, err
	}
	s.ActiveDocument().MarkDirty()
	return wire.ViewResult{View: drawingViewInfo(v)}, nil
}

func drawingViewsAddDraft(s *app.Session, views *drawing.DrawingViews, in wire.AddDraftViewArgs) (wire.ViewResult, error) {
	v, err := views.AddDraft(drawing.DraftViewSpec{
		Name: in.Name, WidthMM: in.WidthMM, HeightMM: in.HeightMM, CenterX: in.CenterXMM, CenterY: in.CenterYMM,
	})
	if err != nil {
		return wire.ViewResult{}, err
	}
	s.ActiveDocument().MarkDirty()
	return wire.ViewResult{View: drawingViewInfo(v)}, nil
}

// drawingViewsSetLabel applies the named view's label overrides (#1983).
func drawingViewsSetLabel(s *app.Session, views *drawing.DrawingViews, in wire.SetViewLabelArgs) (wire.ListDrawingViewsResult, error) {
	style := drawing.ViewLabelStyle{
		Text: in.Text, ShowLabel: in.ShowLabel, ShowName: in.ShowName, ShowScale: in.ShowScale,
		XMM: in.LabelXMM, YMM: in.LabelYMM,
	}
	if err := views.SetLabel(in.Name, style); err != nil {
		return wire.ListDrawingViewsResult{}, err
	}
	s.ActiveDocument().MarkDirty()
	return listDrawingViewsResult(views), nil
}

// drawingViewsSetDisplay toggles a view's edge-display options — currently tangent-edge display (#1984).
func drawingViewsSetDisplay(s *app.Session, views *drawing.DrawingViews, in wire.SetViewDisplayArgs) (wire.ListDrawingViewsResult, error) {
	if in.DisplayTangentEdges == nil {
		return listDrawingViewsResult(views), nil // nothing to change
	}
	if err := views.SetDisplayTangentEdges(in.Name, *in.DisplayTangentEdges); err != nil {
		return wire.ListDrawingViewsResult{}, err
	}
	s.ActiveDocument().MarkDirty()
	return listDrawingViewsResult(views), nil
}

// drawingViewsRotate sets a view's rotation about its centre (#1988).
func drawingViewsRotate(s *app.Session, views *drawing.DrawingViews, in wire.RotateViewArgs) (wire.ListDrawingViewsResult, error) {
	if err := views.Rotate(in.Name, in.AngleDeg); err != nil {
		return wire.ListDrawingViewsResult{}, err
	}
	s.ActiveDocument().MarkDirty()
	return listDrawingViewsResult(views), nil
}

// drawingViewsAlign locks a view to an anchor view on a shared axis, or frees it (#1988).
func drawingViewsAlign(s *app.Session, views *drawing.DrawingViews, in wire.AlignViewArgs) (wire.ListDrawingViewsResult, error) {
	alignment, ok := types.ParseDrawingViewAlignment(in.Alignment)
	if !ok {
		return wire.ListDrawingViewsResult{}, fmt.Errorf("drawing: unknown alignment %q (want horizontal|vertical|inPosition)", in.Alignment)
	}
	var just *types.ViewJustification
	if in.Justification != "" {
		j, ok := types.ParseViewJustification(in.Justification)
		if !ok {
			return wire.ListDrawingViewsResult{}, fmt.Errorf("drawing: unknown justification %q (want centered|fixed)", in.Justification)
		}
		just = &j
	}
	if err := views.Align(in.Name, in.AnchorView, alignment, just); err != nil {
		return wire.ListDrawingViewsResult{}, err
	}
	s.ActiveDocument().MarkDirty()
	return listDrawingViewsResult(views), nil
}

// drawingViewsAddCrop clips the named view to a rectangular or circular fence (#1987).
func drawingViewsAddCrop(s *app.Session, views *drawing.DrawingViews, in wire.AddViewCropArgs) (wire.ListDrawingViewsResult, error) {
	if in.Shape != "" && in.Shape != "rectangle" && in.Shape != "circle" {
		return wire.ListDrawingViewsResult{}, fmt.Errorf("drawing: crop shape %q must be rectangle|circle", in.Shape)
	}
	mark, ok := types.ParseCropBreakMarkLineType(in.BreakMark)
	if !ok {
		return wire.ListDrawingViewsResult{}, fmt.Errorf("drawing: unknown crop break mark %q (want none|continuous|zigzag)", in.BreakMark)
	}
	spec := drawing.CropSpec{
		View: in.View, Circle: in.Shape == "circle", BreakMark: mark,
		X0: in.X0, Y0: in.Y0, X1: in.X1, Y1: in.Y1,
		CircleX: in.CircleXMM, CircleY: in.CircleYMM, Radius: in.RadiusMM,
	}
	if _, err := views.AddCrop(spec); err != nil {
		return wire.ListDrawingViewsResult{}, err
	}
	s.ActiveDocument().MarkDirty()
	return listDrawingViewsResult(views), nil
}

// drawingViewsRemoveCrop drops every crop on the named view (#1987).
func drawingViewsRemoveCrop(s *app.Session, views *drawing.DrawingViews, in wire.RemoveViewCropArgs) (wire.ListDrawingViewsResult, error) {
	if err := views.RemoveCrop(in.View); err != nil {
		return wire.ListDrawingViewsResult{}, err
	}
	s.ActiveDocument().MarkDirty()
	return listDrawingViewsResult(views), nil
}

func drawingViewsDelete(s *app.Session, views *drawing.DrawingViews, in wire.DeleteViewArgs) (wire.ListDrawingViewsResult, error) {
	if err := views.Remove(in.Name); err != nil {
		return wire.ListDrawingViewsResult{}, err
	}
	s.ActiveDocument().MarkDirty()
	return listDrawingViewsResult(views), nil
}

func drawingViewsCurves(_ *app.Session, views *drawing.DrawingViews, in wire.ViewCurvesArgs) (wire.ViewCurvesResult, error) {
	views.Recompute() // re-project against the current model
	v, ok := views.ByName(in.View)
	if !ok {
		return wire.ViewCurvesResult{}, fmt.Errorf("drawing: no view named %q", in.View)
	}
	out := wire.ViewCurvesResult{}
	for _, c := range v.Curves() {
		out.Segments = append(out.Segments, wire.DrawingCurveSegment{
			AX: float64(c.Start().X), AY: float64(c.Start().Y),
			BX: float64(c.End().X), BY: float64(c.End().Y),
			Visible: c.IsVisible(), Kind: c.Kind().String(), EdgeKey: hex.EncodeToString(c.EdgeKey()),
			EdgeType: edgeTypeSpelling(c.EdgeType()),
		})
	}
	return out, nil
}

// edgeTypeSpelling is the wire spelling of a curve's edge role, or "" for an ordinary sharp edge so
// the field is omitted on the common case (#1984).
func edgeTypeSpelling(t types.DrawingEdgeType) string {
	if t == types.UnknownDrawingEdge {
		return ""
	}
	return t.String()
}

// alignmentSpelling is the wire spelling of a view's alignment lock, or "" when the view is free so
// the field is omitted (#1988).
func alignmentSpelling(v *drawing.DrawingView) string {
	if !v.IsAligned() {
		return ""
	}
	return v.Alignment().String()
}

// drawingViewInfo flattens a drawing view into its wire DTO.
func drawingViewInfo(v *drawing.DrawingView) wire.DrawingViewInfo {
	visible, hidden := v.VisibleHidden()
	x, y := v.CenterMM()
	info := wire.DrawingViewInfo{
		Name: v.Name(), Type: v.Type().String(), Projected: v.IsProjected(),
		Orientation: v.Orientation().String(), Scale: v.Scale(), Style: v.Style().String(),
		CenterXMM: x, CenterYMM: y, VisibleCount: visible, HiddenCount: hidden,
		BaseView:  v.BaseViewName(), // the parent of any derived view (projected/auxiliary/section)
		Label:     v.Label(),
		ShowLabel: v.ShowLabel(), ShowName: v.ShowName(), ShowScale: v.ShowScale(),
		CropCount: v.CropCount(), DisplayTangentEdges: v.DisplayTangentEdges(),
		RotationDeg: v.RotationDeg(), Aligned: v.IsAligned(), AlignedTo: v.AlignedTo(),
		Alignment: alignmentSpelling(v), Justification: v.Justification().String(),
	}
	info.LabelXMM, info.LabelYMM = v.LabelPositionMM()
	switch v.Type() {
	case types.DrawingViewProjected:
		info.Direction = v.Direction().String()
	case types.DrawingViewAuxiliary:
		info.FoldAngleDeg = v.FoldAngle() * 180 / math.Pi
	case types.DrawingViewSection:
		info.SectionDepthMM = v.SectionDepthMM()
		info.SectionReverse = v.SectionReverse()
		info.SectionType = v.SectionType().String()
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
