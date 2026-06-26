// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"

	"oblikovati.org/api/contract"
	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
)

// The drawing drafting-standard surface (M14-F01 PBI-138, #385): read the active drawing's
// standards and the active style preset, and switch the standard (which re-points the preset
// so every annotation re-renders to it).

// registerDrawingStyleHandlers wires the drawingStyles.* methods.
func (r *Router) registerDrawingStyleHandlers() {
	r.readOnly(wire.MethodDrawingStylesListStandards, drawingStylesListStandards)
	r.readOnly(wire.MethodDrawingStylesGetActiveStyle, drawingStylesGetActiveStyle)
	r.mutating(wire.MethodDrawingStylesSetStandard, "Set Drawing Standard", drawingStylesSetStandard)
}

func drawingStylesListStandards(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	c, err := activeDrawing(s)
	if err != nil {
		return nil, err
	}
	out := wire.ListStandardsResult{Active: c.Styles().ActiveStandard().String()}
	for _, std := range c.Styles().Standards() {
		out.Standards = append(out.Standards, std.String())
	}
	return json.Marshal(out)
}

func drawingStylesGetActiveStyle(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	c, err := activeDrawing(s)
	if err != nil {
		return nil, err
	}
	return json.Marshal(wire.StandardStyleResult{Style: standardStyleInfo(c.Styles().ActiveStyle())})
}

func drawingStylesSetStandard(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	c, err := activeDrawing(s)
	if err != nil {
		return nil, err
	}
	var in wire.SetStandardArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	std, ok := types.ParseDraftingStandard(in.Standard)
	if !ok {
		return nil, fmt.Errorf("drawing: unknown drafting standard %q", in.Standard)
	}
	c.Styles().SetActiveStandard(std)
	s.ActiveDocument().MarkDirty()
	return json.Marshal(wire.StandardStyleResult{Style: standardStyleInfo(c.Styles().ActiveStyle())})
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
