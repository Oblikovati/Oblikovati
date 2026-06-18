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

// The drawing-dimension surface (M14-F03 PBI-141, #388): associative linear dimensions on a view,
// snapped to projected model vertices so the measured value tracks the model.

// registerDrawingDimensionHandlers wires the drawingDimensions.* methods.
func (r *Router) registerDrawingDimensionHandlers() {
	r.handlers[wire.MethodDrawingDimensionsList] = drawingDimensionsList
	r.handlers[wire.MethodDrawingDimensionsAddLinear] = drawingDimensionsAddLinear
	r.handlers[wire.MethodDrawingDimensionsAddRadial] = drawingDimensionsAddRadial
	r.handlers[wire.MethodDrawingDimensionsAddAngular] = drawingDimensionsAddAngular
	r.handlers[wire.MethodDrawingDimensionsAddBaseline] = drawingDimensionsAddBaseline
	r.handlers[wire.MethodDrawingDimensionsAddChain] = drawingDimensionsAddChain
	r.handlers[wire.MethodDrawingDimensionsDelete] = drawingDimensionsDelete
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

func drawingDimensionsList(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	ds, err := activeSheetDimensions(s)
	if err != nil {
		return nil, err
	}
	return json.Marshal(listDimensions(ds))
}

func drawingDimensionsAddLinear(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	ds, err := activeSheetDimensions(s)
	if err != nil {
		return nil, err
	}
	var in wire.AddLinearDimensionArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	dimType := types.AlignedDimension
	if in.Type != "" {
		t, ok := types.ParseDrawingDimensionType(in.Type)
		if !ok {
			return nil, fmt.Errorf("drawing: unknown dimension type %q (want aligned|horizontal|vertical)", in.Type)
		}
		dimType = t
	}
	d, err := ds.AddLinear(in.Name, in.ViewName, dimType, in.X1, in.Y1, in.X2, in.Y2, in.OffsetMM)
	if err != nil {
		return nil, err
	}
	s.ActiveDocument().MarkDirty()
	return json.Marshal(wire.DimensionResult{Dimension: drawingDimensionInfo(d)})
}

func drawingDimensionsAddRadial(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	ds, err := activeSheetDimensions(s)
	if err != nil {
		return nil, err
	}
	var in wire.AddRadialDimensionArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	dimType := types.RadiusDimension
	if in.Type != "" {
		t, ok := types.ParseDrawingDimensionType(in.Type)
		if !ok || (t != types.RadiusDimension && t != types.DiameterDimension) {
			return nil, fmt.Errorf("drawing: radial dimension type %q must be radius|diameter", in.Type)
		}
		dimType = t
	}
	d, err := ds.AddRadial(in.Name, in.ViewName, dimType, in.PickXMM, in.PickYMM)
	if err != nil {
		return nil, err
	}
	s.ActiveDocument().MarkDirty()
	return json.Marshal(wire.DimensionResult{Dimension: drawingDimensionInfo(d)})
}

func drawingDimensionsAddAngular(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	ds, err := activeSheetDimensions(s)
	if err != nil {
		return nil, err
	}
	var in wire.AddAngularDimensionArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	d, err := ds.AddAngular(in.Name, in.ViewName, in.X1, in.Y1, in.X2, in.Y2)
	if err != nil {
		return nil, err
	}
	s.ActiveDocument().MarkDirty()
	return json.Marshal(wire.DimensionResult{Dimension: drawingDimensionInfo(d)})
}

func drawingDimensionsAddBaseline(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	return drawingDimensionSet(s, raw, (*drawing.DrawingDimensions).AddBaselineSet)
}

func drawingDimensionsAddChain(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	return drawingDimensionSet(s, raw, (*drawing.DrawingDimensions).AddChainSet)
}

// drawingDimensionSet decodes an AddDimensionSetArgs and runs add (AddBaselineSet/AddChainSet),
// returning the created dimensions.
func drawingDimensionSet(s *app.Session, raw json.RawMessage,
	add func(*drawing.DrawingDimensions, string, types.DrawingDimensionType, [][2]float64) ([]*drawing.DrawingDimension, error),
) (json.RawMessage, error) {
	ds, err := activeSheetDimensions(s)
	if err != nil {
		return nil, err
	}
	var in wire.AddDimensionSetArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	dimType, pts, err := parseDimensionSetArgs(in)
	if err != nil {
		return nil, err
	}
	dims, err := add(ds, in.ViewName, dimType, pts)
	if err != nil {
		return nil, err
	}
	s.ActiveDocument().MarkDirty()
	out := wire.DimensionSetResult{}
	for _, d := range dims {
		out.Dimensions = append(out.Dimensions, drawingDimensionInfo(d))
	}
	return json.Marshal(out)
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

func drawingDimensionsDelete(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	ds, err := activeSheetDimensions(s)
	if err != nil {
		return nil, err
	}
	var in wire.DeleteDimensionArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	if err := ds.Remove(in.Name); err != nil {
		return nil, err
	}
	s.ActiveDocument().MarkDirty()
	return json.Marshal(listDimensions(ds))
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
	}
}
