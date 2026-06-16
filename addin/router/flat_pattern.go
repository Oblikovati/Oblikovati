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
	rotation, err := part.Units().Parse(fmt.Sprintf("%g deg", in.AlignmentRotation), param.Angle)
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
