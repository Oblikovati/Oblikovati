// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"

	"oblikovati.org/api/contract"
	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/drawing"
)

// The drawing drafting-standard surface (M14-F01 PBI-138, #385): read the active drawing's
// standards and the active style preset, and switch the standard (which re-points the preset
// so every annotation re-renders to it).

// registerDrawingStyleHandlers wires the drawingStyles.* methods.
func (r *Router) registerDrawingStyleHandlers() {
	r.readOnly(wire.MethodDrawingStylesListStandards, ctxQuery(activeDrawing, drawingStylesListStandards))
	r.readOnly(wire.MethodDrawingStylesGetActiveStyle, ctxQuery(activeDrawing, drawingStylesGetActiveStyle))
	r.mutating(wire.MethodDrawingStylesSetStandard, "Set Drawing Standard", typedCtx(activeDrawing, drawingStylesSetStandard))
}

func drawingStylesListStandards(_ *app.Session, c *drawing.Content) (wire.ListStandardsResult, error) {
	out := wire.ListStandardsResult{Active: c.Styles().ActiveStandard().String()}
	for _, std := range c.Styles().Standards() {
		out.Standards = append(out.Standards, std.String())
	}
	return out, nil
}

func drawingStylesGetActiveStyle(_ *app.Session, c *drawing.Content) (wire.StandardStyleResult, error) {
	return wire.StandardStyleResult{Style: standardStyleInfo(c.Styles().ActiveStyle())}, nil
}

func drawingStylesSetStandard(s *app.Session, c *drawing.Content, in wire.SetStandardArgs) (wire.StandardStyleResult, error) {
	std, ok := types.ParseDraftingStandard(in.Standard)
	if !ok {
		return wire.StandardStyleResult{}, fmt.Errorf("drawing: unknown drafting standard %q", in.Standard)
	}
	c.Styles().SetActiveStandard(std)
	s.ActiveDocument().MarkDirty()
	return wire.StandardStyleResult{Style: standardStyleInfo(c.Styles().ActiveStyle())}, nil
}

// standardStyleInfo flattens a standard's style preset into its wire DTO.
func standardStyleInfo(ss contract.DrawingStandardStyle) wire.StandardStyleInfo {
	d, t, l := ss.DimensionStyle(), ss.TextStyle(), ss.LineStyle()
	return wire.StandardStyleInfo{
		Standard: ss.Standard().String(),
		Dimension: wire.DimensionStyleInfo{
			Name: d.Name(), TextHeightMM: d.TextHeightMM(), ArrowSizeMM: d.ArrowSizeMM(),
			DecimalPlaces: d.DecimalPlaces(), Unit: d.Unit().String(), LineWeightMM: d.LineWeightMM(),
		},
		Text: wire.TextStyleInfo{Name: t.Name(), FontName: t.FontName(), HeightMM: t.HeightMM()},
		Line: wire.LineStyleInfo{Name: l.Name(), WeightMM: l.WeightMM()},
	}
}
