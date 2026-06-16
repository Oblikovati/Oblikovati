// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/hex"
	"encoding/json"
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/compdef"
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
