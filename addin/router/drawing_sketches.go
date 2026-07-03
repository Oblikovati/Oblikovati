// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/drawing"
)

// Drawing sketches (M14-F08 #638): 2D geometry drawn directly in sheet space (millimetres) on the
// active sheet — linework and the boundaries hatch regions fill.

// registerDrawingSketchHandlers wires the drawingSketches.* methods.
func (r *Router) registerDrawingSketchHandlers() {
	r.readOnly(wire.MethodDrawingSketchesList, ctxQuery(activeSheetSketches, drawingSketchesList))
	r.mutating(wire.MethodDrawingSketchesAdd, "Create Sketch", typedCtx(activeSheetSketches, drawingSketchesAdd))
	r.mutating(wire.MethodDrawingSketchesAddEntity, "Add Sketch Geometry", typedCtx(activeSheetSketches, drawingSketchesAddEntity))
	r.mutating(wire.MethodDrawingSketchesAddHatch, "Add Hatch", typedCtx(activeSheetSketches, drawingSketchesAddHatch))
}

// activeSheetSketches returns the active drawing's active-sheet sketch collection.
func activeSheetSketches(s *app.Session) (*drawing.DrawingSketches, error) {
	c, err := activeDrawing(s)
	if err != nil {
		return nil, err
	}
	sheet := c.Sheets().Active()
	if sheet == nil {
		return nil, fmt.Errorf("drawing: no active sheet")
	}
	return sheet.Sketches(), nil
}

func drawingSketchesList(_ *app.Session, ss *drawing.DrawingSketches) (wire.ListDrawingSketchesResult, error) {
	out := wire.ListDrawingSketchesResult{}
	for i := 0; i < ss.Count(); i++ {
		out.Sketches = append(out.Sketches, drawingSketchInfo(ss.Item(i)))
	}
	return out, nil
}

func drawingSketchesAdd(s *app.Session, ss *drawing.DrawingSketches, in wire.AddDrawingSketchArgs) (wire.DrawingSketchResult, error) {
	sk := ss.Add(in.Name)
	s.ActiveDocument().MarkDirty()
	return wire.DrawingSketchResult{Sketch: drawingSketchInfo(sk)}, nil
}

func drawingSketchesAddEntity(s *app.Session, ss *drawing.DrawingSketches, in wire.AddDrawingSketchEntityArgs) (wire.DrawingSketchResult, error) {
	kind, ok := types.ParseDrawingSketchEntityKind(in.Kind)
	if !ok {
		return wire.DrawingSketchResult{}, fmt.Errorf("drawing: unknown sketch entity kind %q (want line|circle|rectangle)", in.Kind)
	}
	sk, err := ss.AddEntity(in.SketchName, kind, in.Points, in.Radius)
	if err != nil {
		return wire.DrawingSketchResult{}, err
	}
	s.ActiveDocument().MarkDirty()
	return wire.DrawingSketchResult{Sketch: drawingSketchInfo(sk)}, nil
}

func drawingSketchesAddHatch(s *app.Session, ss *drawing.DrawingSketches, in wire.AddHatchRegionArgs) (wire.DrawingSketchResult, error) {
	pattern := types.HatchGeneral
	if in.Pattern != "" {
		p, ok := types.ParseHatchPattern(in.Pattern)
		if !ok {
			return wire.DrawingSketchResult{}, fmt.Errorf("drawing: unknown hatch pattern %q (want general|cross|ansi31)", in.Pattern)
		}
		pattern = p
	}
	sk, err := ss.AddHatchRegion(in.SketchName, in.XMM, in.YMM, in.WidthMM, in.HeightMM, pattern, in.ScaleMM)
	if err != nil {
		return wire.DrawingSketchResult{}, err
	}
	s.ActiveDocument().MarkDirty()
	return wire.DrawingSketchResult{Sketch: drawingSketchInfo(sk)}, nil
}

// drawingSketchInfo flattens a drawing sketch into its wire DTO.
func drawingSketchInfo(s *drawing.DrawingSketch) wire.DrawingSketchInfo {
	return wire.DrawingSketchInfo{Name: s.Name(), EntityCount: s.EntityCount(), CurveCount: s.CurveCount()}
}
