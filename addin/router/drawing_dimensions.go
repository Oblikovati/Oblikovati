// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/drawing"
)

// The drawing-dimension surface (M14-F03 PBI-141, #388): associative linear dimensions on a view,
// snapped to projected model vertices so the measured value tracks the model.

// registerDrawingDimensionHandlers wires the drawingDimensions.* methods.
func (r *Router) registerDrawingDimensionHandlers() {
	r.readOnly(wire.MethodDrawingDimensionsList, ctxQuery(activeSheetDimensions, drawingDimensionsList))
	r.mutating(wire.MethodDrawingDimensionsAddLinear, "Add Dimension", typedCtx(activeSheetDimensions, drawingDimensionsAddLinear))
	r.mutating(wire.MethodDrawingDimensionsAddRadial, "Add Dimension", typedCtx(activeSheetDimensions, drawingDimensionsAddRadial))
	r.mutating(wire.MethodDrawingDimensionsAddAngular, "Add Dimension", typedCtx(activeSheetDimensions, drawingDimensionsAddAngular))
	r.mutating(wire.MethodDrawingDimensionsAddBaseline, "Add Dimension", typedCtx(activeSheetDimensions, drawingDimensionSet((*drawing.DrawingDimensions).AddBaselineSet)))
	r.mutating(wire.MethodDrawingDimensionsAddChain, "Add Dimension", typedCtx(activeSheetDimensions, drawingDimensionSet((*drawing.DrawingDimensions).AddChainSet)))
	r.mutating(wire.MethodDrawingDimensionsAddOrdinate, "Add Dimension", typedCtx(activeSheetDimensions, drawingDimensionsAddOrdinate))
	r.mutating(wire.MethodDrawingDimensionsAddArcLength, "Add Dimension", typedCtx(activeSheetDimensions, drawingDimensionsAddArcLength))
	r.mutating(wire.MethodDrawingDimensionsDelete, "Delete Dimension", typedCtx(activeSheetDimensions, drawingDimensionsDelete))
	r.mutating(wire.MethodDrawingDimensionsSetTextStyle, "Edit Dimension", typedCtx(activeSheetDimensions, drawingDimensionsSetTextStyle))
}

// activeSheetDimensions returns the active drawing's active-sheet dimension collection.
func activeSheetDimensions(s *app.Session) (*drawing.DrawingDimensions, error) {
	c, err := activeDrawing(s)
	if err != nil {
		return nil, err
	}
	sheet := c.Sheets().Active()
	if sheet == nil {
		return nil, fmt.Errorf("drawing: no active sheet")
	}
	return sheet.Dimensions(), nil
}

func drawingDimensionsList(_ *app.Session, ds *drawing.DrawingDimensions) (wire.ListDrawingDimensionsResult, error) {
	return listDimensions(ds), nil
}

func drawingDimensionsAddLinear(s *app.Session, ds *drawing.DrawingDimensions, in wire.AddLinearDimensionArgs) (wire.DimensionResult, error) {
	dimType := types.AlignedDimension
	if in.Type != "" {
		t, ok := types.ParseDrawingDimensionType(in.Type)
		if !ok {
			return wire.DimensionResult{}, fmt.Errorf("drawing: unknown dimension type %q (want aligned|horizontal|vertical)", in.Type)
		}
		dimType = t
	}
	d, err := ds.AddLinear(in.Name, in.ViewName, dimType, in.X1, in.Y1, in.X2, in.Y2, in.OffsetMM)
	if err != nil {
		return wire.DimensionResult{}, err
	}
	s.ActiveDocument().MarkDirty()
	return wire.DimensionResult{Dimension: drawingDimensionInfo(d)}, nil
}

func drawingDimensionsAddRadial(s *app.Session, ds *drawing.DrawingDimensions, in wire.AddRadialDimensionArgs) (wire.DimensionResult, error) {
	dimType := types.RadiusDimension
	if in.Type != "" {
		t, ok := types.ParseDrawingDimensionType(in.Type)
		if !ok || (t != types.RadiusDimension && t != types.DiameterDimension) {
			return wire.DimensionResult{}, fmt.Errorf("drawing: radial dimension type %q must be radius|diameter", in.Type)
		}
		dimType = t
	}
	d, err := ds.AddRadial(in.Name, in.ViewName, dimType, in.PickXMM, in.PickYMM)
	if err != nil {
		return wire.DimensionResult{}, err
	}
	s.ActiveDocument().MarkDirty()
	return wire.DimensionResult{Dimension: drawingDimensionInfo(d)}, nil
}

func drawingDimensionsAddAngular(s *app.Session, ds *drawing.DrawingDimensions, in wire.AddAngularDimensionArgs) (wire.DimensionResult, error) {
	d, err := ds.AddAngular(in.Name, in.ViewName, in.X1, in.Y1, in.X2, in.Y2)
	if err != nil {
		return wire.DimensionResult{}, err
	}
	s.ActiveDocument().MarkDirty()
	return wire.DimensionResult{Dimension: drawingDimensionInfo(d)}, nil
}

// drawingDimensionSet builds a typed handler that runs add (AddBaselineSet/AddChainSet) over a
// decoded AddDimensionSetArgs, returning the created dimensions.
func drawingDimensionSet(
	add func(*drawing.DrawingDimensions, string, types.DrawingDimensionType, [][2]float64) ([]*drawing.DrawingDimension, error),
) func(*app.Session, *drawing.DrawingDimensions, wire.AddDimensionSetArgs) (wire.DimensionSetResult, error) {
	return func(s *app.Session, ds *drawing.DrawingDimensions, in wire.AddDimensionSetArgs) (wire.DimensionSetResult, error) {
		dimType, pts, err := parseDimensionSetArgs(in)
		if err != nil {
			return wire.DimensionSetResult{}, err
		}
		dims, err := add(ds, in.ViewName, dimType, pts)
		if err != nil {
			return wire.DimensionSetResult{}, err
		}
		s.ActiveDocument().MarkDirty()
		out := wire.DimensionSetResult{}
		for _, d := range dims {
			out.Dimensions = append(out.Dimensions, drawingDimensionInfo(d))
		}
		return out, nil
	}
}

func drawingDimensionsAddArcLength(s *app.Session, ds *drawing.DrawingDimensions, in wire.AddArcLengthDimensionArgs) (wire.DimensionResult, error) {
	d, err := ds.AddArcLength(in.Name, in.ViewName, in.PickXMM, in.PickYMM)
	if err != nil {
		return wire.DimensionResult{}, err
	}
	s.ActiveDocument().MarkDirty()
	return wire.DimensionResult{Dimension: drawingDimensionInfo(d)}, nil
}

func drawingDimensionsAddOrdinate(s *app.Session, ds *drawing.DrawingDimensions, in wire.AddOrdinateDimensionsArgs) (wire.DimensionSetResult, error) {
	datum, pts, err := parseOrdinateArgs(in)
	if err != nil {
		return wire.DimensionSetResult{}, err
	}
	dims, err := ds.AddOrdinateSet(in.ViewName, in.Axis != "vertical", datum, pts)
	if err != nil {
		return wire.DimensionSetResult{}, err
	}
	s.ActiveDocument().MarkDirty()
	out := wire.DimensionSetResult{}
	for _, d := range dims {
		out.Dimensions = append(out.Dimensions, drawingDimensionInfo(d))
	}
	return out, nil
}

// parseOrdinateArgs resolves an ordinate request's datum and measured points (each [x,y] sheet mm).
func parseOrdinateArgs(in wire.AddOrdinateDimensionsArgs) (datum [2]float64, pts [][2]float64, err error) {
	if in.Axis != "" && in.Axis != "horizontal" && in.Axis != "vertical" {
		return datum, nil, fmt.Errorf("drawing: ordinate axis %q must be horizontal|vertical", in.Axis)
	}
	if len(in.Datum) != 2 {
		return datum, nil, fmt.Errorf("drawing: datum must be [x,y], got %v", in.Datum)
	}
	datum = [2]float64{in.Datum[0], in.Datum[1]}
	pts = make([][2]float64, len(in.Points))
	for i, p := range in.Points {
		if len(p) != 2 {
			return datum, nil, fmt.Errorf("drawing: point %d must be [x,y], got %v", i, p)
		}
		pts[i] = [2]float64{p[0], p[1]}
	}
	return datum, pts, nil
}

// parseDimensionSetArgs resolves a set request's measurement type and pick points.
func parseDimensionSetArgs(in wire.AddDimensionSetArgs) (types.DrawingDimensionType, [][2]float64, error) {
	dimType := types.AlignedDimension
	if in.Type != "" {
		t, ok := types.ParseDrawingDimensionType(in.Type)
		if !ok {
			return 0, nil, fmt.Errorf("drawing: unknown dimension type %q", in.Type)
		}
		dimType = t
	}
	pts := make([][2]float64, len(in.Points))
	for i, p := range in.Points {
		if len(p) != 2 {
			return 0, nil, fmt.Errorf("drawing: point %d must be [x,y], got %v", i, p)
		}
		pts[i] = [2]float64{p[0], p[1]}
	}
	return dimType, pts, nil
}

func drawingDimensionsDelete(s *app.Session, ds *drawing.DrawingDimensions, in wire.DeleteDimensionArgs) (wire.ListDrawingDimensionsResult, error) {
	if err := ds.Remove(in.Name); err != nil {
		return wire.ListDrawingDimensionsResult{}, err
	}
	s.ActiveDocument().MarkDirty()
	return listDimensions(ds), nil
}

// listDimensions flattens a sheet's dimension collection into its wire result.
func listDimensions(ds *drawing.DrawingDimensions) wire.ListDrawingDimensionsResult {
	out := wire.ListDrawingDimensionsResult{}
	for i := 0; i < ds.Count(); i++ {
		out.Dimensions = append(out.Dimensions, drawingDimensionInfo(ds.Item(i)))
	}
	return out
}

// drawingDimensionInfo flattens a dimension into its wire DTO.
func drawingDimensionInfo(d *drawing.DrawingDimension) wire.DrawingDimensionInfo {
	return wire.DrawingDimensionInfo{
		Name: d.Name(), Type: d.Type().String(), ViewName: d.ViewName(),
		ValueMM: d.ValueMM(), ValueDeg: d.ValueDeg(), Text: d.Text(), CurveCount: d.CurveCount(),
		Prefix: d.Prefix(), Suffix: d.Suffix(), OverrideText: d.OverrideText(),
		HideValue: d.HideValue(), DualUnit: d.DualUnit(),
	}
}

// drawingDimensionsSetTextStyle applies the named dimension's text overrides (#1992/#1993).
func drawingDimensionsSetTextStyle(s *app.Session, ds *drawing.DrawingDimensions, in wire.SetDimensionTextStyleArgs) (wire.ListDrawingDimensionsResult, error) {
	style := drawing.DimensionTextStyle{
		Prefix: in.Prefix, Suffix: in.Suffix, OverrideText: in.OverrideText,
		HideValue: in.HideValue, DualUnit: in.DualUnit,
	}
	if err := ds.SetTextStyle(in.Name, style); err != nil {
		return wire.ListDrawingDimensionsResult{}, err
	}
	s.ActiveDocument().MarkDirty()
	return listDimensions(ds), nil
}
