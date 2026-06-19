// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/hex"
	"encoding/json"
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
	r.handlers[wire.MethodFlatPatternListOrientations] = flatPatternListOrientations
	r.handlers[wire.MethodFlatPatternAddOrientation] = flatPatternAddOrientation
	r.handlers[wire.MethodFlatPatternActivateOrientation] = flatPatternActivateOrientation
	r.handlers[wire.MethodFlatPatternDeleteOrientation] = flatPatternDeleteOrientation
	r.handlers[wire.MethodFlatPatternEdgesOfType] = flatPatternEdgesOfType
	r.handlers[wire.MethodFlatPatternFaces] = flatPatternFaces
	r.handlers[wire.MethodFlatPatternMapEntity] = flatPatternMapEntity
	r.handlers[wire.MethodFlatPatternListPlates] = flatPatternListPlates
	r.handlers[wire.MethodFlatPatternGetSettings] = flatPatternGetSettings
	r.handlers[wire.MethodFlatPatternSetSettings] = flatPatternSetSettings
	r.handlers[wire.MethodFlatPatternListBendOrder] = flatPatternListBendOrder
	r.handlers[wire.MethodFlatPatternSetBendOrder] = flatPatternSetBendOrder
	r.handlers[wire.MethodFlatPatternAddCenterline] = flatPatternAddCenterline
	r.handlers[wire.MethodFlatPatternListCenterlines] = flatPatternListCenterlines
	r.handlers[wire.MethodFlatPatternDeleteCenterline] = flatPatternDeleteCenterline
}

func flatPatternAddCenterline(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, _, err := activeSheetMetal(s)
	if err != nil {
		return nil, err
	}
	var in wire.AddCenterlineArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	part.AddCosmeticCenterline(gmath.P2(in.Start.X, in.Start.Y), gmath.P2(in.End.X, in.End.Y))
	return json.Marshal(centerlinesResult(part))
}

func flatPatternListCenterlines(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	part, _, err := activeSheetMetal(s)
	if err != nil {
		return nil, err
	}
	return json.Marshal(centerlinesResult(part))
}

func flatPatternDeleteCenterline(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, _, err := activeSheetMetal(s)
	if err != nil {
		return nil, err
	}
	var in wire.DeleteCenterlineArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	if err := part.DeleteCosmeticCenterline(in.Index); err != nil {
		return nil, err
	}
	return json.Marshal(centerlinesResult(part))
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

func flatPatternListBendOrder(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	part, _, err := activeSheetMetal(s)
	if err != nil {
		return nil, err
	}
	return json.Marshal(bendOrderResult(part))
}

func flatPatternSetBendOrder(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, _, err := activeSheetMetal(s)
	if err != nil {
		return nil, err
	}
	var in wire.SetBendOrderArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	if err := part.SetBendOrder(in.Order); err != nil {
		return nil, err
	}
	return json.Marshal(bendOrderResult(part))
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

func flatPatternListPlates(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	part, _, err := activeSheetMetal(s)
	if err != nil {
		return nil, err
	}
	plates, err := part.FlatPlates()
	if err != nil {
		return nil, err
	}
	out := wire.PlatesResult{Plates: make([]wire.PlateInfo, len(plates))}
	for i, p := range plates {
		out.Plates[i] = wire.PlateInfo{Index: i, Length: p.Length, Width: p.Width, Area: p.Area}
	}
	return json.Marshal(out)
}

func flatPatternGetSettings(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	part, _, err := activeSheetMetal(s)
	if err != nil {
		return nil, err
	}
	return json.Marshal(wire.SettingsResult{Settings: wire.FlatPatternSettings{DeferUpdate: part.FlatSettings().DeferUpdate}})
}

func flatPatternSetSettings(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, _, err := activeSheetMetal(s)
	if err != nil {
		return nil, err
	}
	var in wire.SetSettingsArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	part.SetFlatDeferUpdate(in.DeferUpdate)
	return json.Marshal(wire.SettingsResult{Settings: wire.FlatPatternSettings{DeferUpdate: part.FlatSettings().DeferUpdate}})
}

func flatPatternMapEntity(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, _, err := activeSheetMetal(s)
	if err != nil {
		return nil, err
	}
	var in wire.MapEntityArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	mapper := part.MapFlatToFolded
	if in.ToFlat {
		mapper = part.MapFoldedToFlat
	}
	// Keys are the raw topology reference keys model.referenceKeys reports, so a caller can
	// feed one straight back in.
	mapped, found, err := mapper([]byte(in.Key))
	if err != nil {
		return nil, err
	}
	out := wire.MapEntityResult{Found: found}
	if found {
		out.Key, out.Kind = string(mapped), "face"
	}
	return json.Marshal(out)
}

func flatPatternEdgesOfType(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, _, err := activeSheetMetal(s)
	if err != nil {
		return nil, err
	}
	var in wire.EdgesOfTypeArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	filter, err := edgeTypeFilter(in.Type)
	if err != nil {
		return nil, err
	}
	flat, err := part.Unfold()
	if err != nil {
		return nil, err
	}
	return json.Marshal(wire.EdgesResult{Edges: classifiedEdges(flat.Bends, filter)})
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

func flatPatternFaces(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	part, _, err := activeSheetMetal(s)
	if err != nil {
		return nil, err
	}
	flat, err := part.Unfold()
	if err != nil {
		return nil, err
	}
	// The flat is a constant-thickness plate: its front (top) and back (bottom) faces share
	// the developed footprint area (the body volume over the gauge).
	area := 0.0
	if flat.Thickness > 0 {
		area = ops.BodyGeometryProperties(flat.Body, ops.Quality{ChordTolerance: 1e-3}).Volume / flat.Thickness
	}
	return json.Marshal(wire.FacesResult{Faces: []wire.FlatFaceInfo{
		{Type: types.FrontFlatPatternFace.String(), Area: area},
		{Type: types.BackFlatPatternFace.String(), Area: area},
	}})
}

func flatPatternListOrientations(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	part, _, err := activeSheetMetal(s)
	if err != nil {
		return nil, err
	}
	return json.Marshal(orientationsResult(part))
}

func flatPatternAddOrientation(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, _, err := activeSheetMetal(s)
	if err != nil {
		return nil, err
	}
	var in wire.AddOrientationArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	or, err := orientationFromArgs(part, in)
	if err != nil {
		return nil, err
	}
	if err := part.FlatOrientations().Add(or); err != nil {
		return nil, err
	}
	if in.Activate {
		if err := part.FlatOrientations().Activate(or.Name); err != nil {
			return nil, err
		}
	}
	return json.Marshal(wire.OrientationResult{Orientation: orientationInfo(part, or)})
}

func flatPatternActivateOrientation(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, _, err := activeSheetMetal(s)
	if err != nil {
		return nil, err
	}
	var in wire.ActivateOrientationArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	if err := part.FlatOrientations().Activate(in.Name); err != nil {
		return nil, err
	}
	return json.Marshal(wire.OrientationResult{Orientation: orientationInfo(part, part.FlatOrientations().Active())})
}

func flatPatternDeleteOrientation(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, _, err := activeSheetMetal(s)
	if err != nil {
		return nil, err
	}
	var in wire.DeleteOrientationArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	if err := part.FlatOrientations().Delete(in.Name); err != nil {
		return nil, err
	}
	return json.Marshal(orientationsResult(part))
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
