// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/hex"
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/kernel/ops"
	gmath "oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/param"
	"oblikovati.org/model/sheetmetal"
)

// The flat-pattern orientation surface (M13-F05, #635): a sheet-metal part's developed flat
// carries named orientations — saved alignment states that frame it for drawing views and
// export. The active orientation drives the flat's reported length/width. Orientations persist
// in the document.

// registerFlatPatternHandlers wires the flatPattern.* methods.
func (r *Router) registerFlatPatternHandlers() {
	r.readOnly(wire.MethodFlatPatternListOrientations, ctxQuery(resolveSheetMetalPart, flatPatternListOrientations))
	r.mutating(wire.MethodFlatPatternAddOrientation, "Edit Flat Pattern", typedCtx(resolveSheetMetalPart, flatPatternAddOrientation))
	r.readOnly(wire.MethodFlatPatternActivateOrientation, typedCtx(resolveSheetMetalPart, flatPatternActivateOrientation))
	r.mutating(wire.MethodFlatPatternDeleteOrientation, "Edit Flat Pattern", typedCtx(resolveSheetMetalPart, flatPatternDeleteOrientation))
	r.readOnly(wire.MethodFlatPatternEdgesOfType, typedCtx(resolveSheetMetalPart, flatPatternEdgesOfType))
	r.readOnly(wire.MethodFlatPatternFaces, ctxQuery(resolveSheetMetalPart, flatPatternFaces))
	r.readOnly(wire.MethodFlatPatternMapEntity, typedCtx(resolveSheetMetalPart, flatPatternMapEntity))
	r.readOnly(wire.MethodFlatPatternListPlates, ctxQuery(resolveSheetMetalPart, flatPatternListPlates))
	r.readOnly(wire.MethodFlatPatternGetSettings, ctxQuery(resolveSheetMetalPart, flatPatternGetSettings))
	r.mutating(wire.MethodFlatPatternSetSettings, "Edit Flat Pattern Settings", typedCtx(resolveSheetMetalPart, flatPatternSetSettings))
	r.readOnly(wire.MethodFlatPatternListBendOrder, ctxQuery(resolveSheetMetalPart, flatPatternListBendOrder))
	r.mutating(wire.MethodFlatPatternSetBendOrder, "Edit Flat Pattern", typedCtx(resolveSheetMetalPart, flatPatternSetBendOrder))
	r.mutating(wire.MethodFlatPatternAddCenterline, "Edit Flat Pattern", typedCtx(resolveSheetMetalPart, flatPatternAddCenterline))
	r.readOnly(wire.MethodFlatPatternListCenterlines, ctxQuery(resolveSheetMetalPart, flatPatternListCenterlines))
	r.mutating(wire.MethodFlatPatternDeleteCenterline, "Edit Flat Pattern", typedCtx(resolveSheetMetalPart, flatPatternDeleteCenterline))
}

func flatPatternAddCenterline(_ *app.Session, ctx sheetMetalPart, in wire.AddCenterlineArgs) (wire.CenterlinesResult, error) {
	ctx.part.AddCosmeticCenterline(gmath.P2(in.Start.X, in.Start.Y), gmath.P2(in.End.X, in.End.Y))
	return centerlinesResult(ctx.part), nil
}

func flatPatternListCenterlines(_ *app.Session, ctx sheetMetalPart) (wire.CenterlinesResult, error) {
	return centerlinesResult(ctx.part), nil
}

func flatPatternDeleteCenterline(_ *app.Session, ctx sheetMetalPart, in wire.DeleteCenterlineArgs) (wire.CenterlinesResult, error) {
	if err := ctx.part.DeleteCosmeticCenterline(in.Index); err != nil {
		return wire.CenterlinesResult{}, err
	}
	return centerlinesResult(ctx.part), nil
}

// centerlinesResult renders the part's cosmetic centerlines as wire.
func centerlinesResult(part *compdef.PartComponentDefinition) wire.CenterlinesResult {
	lines := part.CosmeticCenterlines()
	out := wire.CenterlinesResult{Centerlines: make([]wire.CenterlineInfo, len(lines))}
	for i, c := range lines {
		out.Centerlines[i] = wire.CenterlineInfo{Index: i, Start: point2d(c.Start), End: point2d(c.End)}
	}
	return out
}

func flatPatternListBendOrder(_ *app.Session, ctx sheetMetalPart) (wire.BendOrderResult, error) {
	return bendOrderResult(ctx.part), nil
}

func flatPatternSetBendOrder(_ *app.Session, ctx sheetMetalPart, in wire.SetBendOrderArgs) (wire.BendOrderResult, error) {
	if err := ctx.part.SetBendOrder(in.Order); err != nil {
		return wire.BendOrderResult{}, err
	}
	return bendOrderResult(ctx.part), nil
}

// bendOrderResult renders the part's bends in their press-brake sequence (1-based order).
func bendOrderResult(part *compdef.PartComponentDefinition) wire.BendOrderResult {
	bends := part.OrderedBends()
	out := wire.BendOrderResult{Bends: make([]wire.BendOrderInfo, len(bends))}
	for i, b := range bends {
		out.Bends[i] = wire.BendOrderInfo{
			Feature: b.Feature, Order: i + 1, Angle: b.Angle * degPerRad, Radius: b.Radius,
		}
	}
	return out
}

func flatPatternListPlates(_ *app.Session, ctx sheetMetalPart) (wire.PlatesResult, error) {
	plates, err := ctx.part.FlatPlates()
	if err != nil {
		return wire.PlatesResult{}, err
	}
	out := wire.PlatesResult{Plates: make([]wire.PlateInfo, len(plates))}
	for i, p := range plates {
		out.Plates[i] = wire.PlateInfo{Index: i, Length: p.Length, Width: p.Width, Area: p.Area}
	}
	return out, nil
}

func flatPatternGetSettings(_ *app.Session, ctx sheetMetalPart) (wire.SettingsResult, error) {
	return wire.SettingsResult{Settings: wire.FlatPatternSettings{DeferUpdate: ctx.part.FlatSettings().DeferUpdate}}, nil
}

func flatPatternSetSettings(_ *app.Session, ctx sheetMetalPart, in wire.SetSettingsArgs) (wire.SettingsResult, error) {
	ctx.part.SetFlatDeferUpdate(in.DeferUpdate)
	return wire.SettingsResult{Settings: wire.FlatPatternSettings{DeferUpdate: ctx.part.FlatSettings().DeferUpdate}}, nil
}

func flatPatternMapEntity(_ *app.Session, ctx sheetMetalPart, in wire.MapEntityArgs) (wire.MapEntityResult, error) {
	mapper := ctx.part.MapFlatToFolded
	if in.ToFlat {
		mapper = ctx.part.MapFoldedToFlat
	}
	// Keys are the raw topology reference keys model.referenceKeys reports, so a caller can
	// feed one straight back in.
	mapped, found, err := mapper([]byte(in.Key))
	if err != nil {
		return wire.MapEntityResult{}, err
	}
	out := wire.MapEntityResult{Found: found}
	if found {
		out.Key, out.Kind = string(mapped), "face"
	}
	return out, nil
}

func flatPatternEdgesOfType(_ *app.Session, ctx sheetMetalPart, in wire.EdgesOfTypeArgs) (wire.EdgesResult, error) {
	filter, err := edgeTypeFilter(in.Type)
	if err != nil {
		return wire.EdgesResult{}, err
	}
	flat, err := ctx.part.Unfold()
	if err != nil {
		return wire.EdgesResult{}, err
	}
	return wire.EdgesResult{Edges: classifiedEdges(flat.Bends, filter)}, nil
}

// classifiedEdges turns the flat's fold lines into classified wire edges (bend-up unless the
// bend folds toward the back), keeping only those matching filter (nil ⇒ all).
func classifiedEdges(bends []feature.FlatBendLine, filter *types.FlatPatternEdgeType) []wire.FlatEdgeInfo {
	out := make([]wire.FlatEdgeInfo, 0, len(bends))
	for _, b := range bends {
		et := types.BendUpFlatPatternEdge
		if b.FoldDown {
			et = types.BendDownFlatPatternEdge
		}
		if filter != nil && *filter != et {
			continue
		}
		out = append(out, wire.FlatEdgeInfo{
			Start: point2d(b.A), End: point2d(b.B), Type: et.String(), Angle: b.Angle * degPerRad,
		})
	}
	return out
}

// edgeTypeFilter parses an optional edge-type filter; an empty string means "all types".
func edgeTypeFilter(s string) (*types.FlatPatternEdgeType, error) {
	if s == "" {
		return nil, nil
	}
	et, ok := types.ParseFlatPatternEdgeType(s)
	if !ok {
		return nil, fmt.Errorf("flatPattern: edge type %q: want bendUp|bendDown|tangent", s)
	}
	return &et, nil
}

func flatPatternFaces(_ *app.Session, ctx sheetMetalPart) (wire.FacesResult, error) {
	flat, err := ctx.part.Unfold()
	if err != nil {
		return wire.FacesResult{}, err
	}
	// The flat is a constant-thickness plate: its front (top) and back (bottom) faces share
	// the developed footprint area (the body volume over the gauge).
	area := 0.0
	if flat.Thickness > 0 {
		area = ops.BodyGeometryProperties(flat.Body, ops.Quality{ChordTolerance: 1e-3}).Volume / flat.Thickness
	}
	return wire.FacesResult{Faces: []wire.FlatFaceInfo{
		{Type: types.FrontFlatPatternFace.String(), Area: area},
		{Type: types.BackFlatPatternFace.String(), Area: area},
	}}, nil
}

func flatPatternListOrientations(_ *app.Session, ctx sheetMetalPart) (wire.OrientationsResult, error) {
	return orientationsResult(ctx.part), nil
}

func flatPatternAddOrientation(_ *app.Session, ctx sheetMetalPart, in wire.AddOrientationArgs) (wire.OrientationResult, error) {
	or, err := orientationFromArgs(ctx.part, in)
	if err != nil {
		return wire.OrientationResult{}, err
	}
	if err := ctx.part.FlatOrientations().Add(or); err != nil {
		return wire.OrientationResult{}, err
	}
	if in.Activate {
		if err := ctx.part.FlatOrientations().Activate(or.Name); err != nil {
			return wire.OrientationResult{}, err
		}
	}
	return wire.OrientationResult{Orientation: orientationInfo(ctx.part, or)}, nil
}

func flatPatternActivateOrientation(_ *app.Session, ctx sheetMetalPart, in wire.ActivateOrientationArgs) (wire.OrientationResult, error) {
	if err := ctx.part.FlatOrientations().Activate(in.Name); err != nil {
		return wire.OrientationResult{}, err
	}
	return wire.OrientationResult{Orientation: orientationInfo(ctx.part, ctx.part.FlatOrientations().Active())}, nil
}

func flatPatternDeleteOrientation(_ *app.Session, ctx sheetMetalPart, in wire.DeleteOrientationArgs) (wire.OrientationsResult, error) {
	if err := ctx.part.FlatOrientations().Delete(in.Name); err != nil {
		return wire.OrientationsResult{}, err
	}
	return orientationsResult(ctx.part), nil
}

// orientationFromArgs builds an orientation from the wire args: alignment type (default
// horizontal), rotation in degrees, and an optional hex alignment-axis reference key.
func orientationFromArgs(part *compdef.PartComponentDefinition, in wire.AddOrientationArgs) (*sheetmetal.FlatPatternOrientation, error) {
	at := types.HorizontalAlignment
	if in.AlignmentType != "" {
		parsed, ok := types.ParseAlignmentType(in.AlignmentType)
		if !ok {
			return nil, fmt.Errorf("flatPattern: alignmentType %q: want horizontal|vertical", in.AlignmentType)
		}
		at = parsed
	}
	rotation, err := resolveQuantity(part, fmt.Sprintf("%g deg", in.AlignmentRotation), param.Angle)
	if err != nil {
		return nil, fmt.Errorf("flatPattern: alignmentRotation %g: %w", in.AlignmentRotation, err)
	}
	key, err := hex.DecodeString(in.AlignmentAxis)
	if err != nil {
		return nil, fmt.Errorf("flatPattern: alignmentAxis %q is not a hex reference key: %w", in.AlignmentAxis, err)
	}
	return &sheetmetal.FlatPatternOrientation{
		Name: in.Name, AlignmentType: at, AlignmentRotation: rotation.Value,
		AlignmentAxisKey: key, FlipAlignmentAxis: in.FlipAlignmentAxis, FlipBaseFace: in.FlipBaseFace,
	}, nil
}

// orientationsResult renders all orientations as wire.
func orientationsResult(part *compdef.PartComponentDefinition) wire.OrientationsResult {
	set := part.FlatOrientations()
	out := wire.OrientationsResult{Orientations: make([]wire.FlatPatternOrientationInfo, 0, len(set.List()))}
	for _, or := range set.List() {
		out.Orientations = append(out.Orientations, orientationInfo(part, or))
	}
	return out
}

// orientationInfo renders one orientation as wire, including the flat's length/width under it
// (zero when the part has no developable flat yet). AlignmentRotation is reported in degrees.
func orientationInfo(part *compdef.PartComponentDefinition, or *sheetmetal.FlatPatternOrientation) wire.FlatPatternOrientationInfo {
	length, width, err := part.FlatLengthWidth(or)
	if err != nil {
		length, width = 0, 0
	}
	return wire.FlatPatternOrientationInfo{
		Name:              or.Name,
		AlignmentType:     or.AlignmentType.String(),
		AlignmentRotation: or.AlignmentRotation * degPerRad,
		AlignmentAxis:     hex.EncodeToString(or.AlignmentAxisKey),
		FlipAlignmentAxis: or.FlipAlignmentAxis,
		FlipBaseFace:      or.FlipBaseFace,
		Active:            part.FlatOrientations().IsActive(or),
		Length:            length,
		Width:             width,
	}
}
