// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/sketch"
)

// The sketch Format panel over the wire (#2015): per-entity line type, colour and stroke width,
// and the panel's armed creation modes. These are the values a DWG or DXF import carries in from
// the file's layer table, so an add-in driving an exchange workflow needs them.

// registerSketchFormatHandlers wires the Format-panel methods.
func (r *Router) registerSketchFormatHandlers() {
	r.readOnly(wire.MethodSketchGetEntityFormat, getSketchEntityFormat)
	r.mutating(wire.MethodSketchSetEntityFormat, "Set Sketch Format", typed(setSketchEntityFormat))
	r.readOnly(wire.MethodSketchGetFormatModes, getSketchFormatModes)
	r.mutating(wire.MethodSketchSetFormatModes, "", typed(setSketchFormatModes))
}

// getSketchEntityFormat serves wire.MethodSketchGetEntityFormat.
func getSketchEntityFormat(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.SketchEntityFormatArgs
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, fmt.Errorf("sketch.getEntityFormat: %w", err)
	}
	f, ok := s.EntityFormatOf(sketch.ID(in.EntityID))
	return json.Marshal(wire.SketchEntityFormatView{EntityID: in.EntityID, HasFormat: ok, Format: f})
}

// setSketchEntityFormat serves wire.MethodSketchSetEntityFormat. A format that overrides nothing
// clears the entity's overrides, which is how a caller returns it to the sketch's attributes.
func setSketchEntityFormat(s *app.Session, in wire.SetSketchEntityFormatArgs) (wire.SketchEntityFormatView, error) {
	if err := s.SetEntityFormatOf(sketch.ID(in.EntityID), in.Format); err != nil {
		return wire.SketchEntityFormatView{}, err
	}
	f, ok := s.EntityFormatOf(sketch.ID(in.EntityID))
	return wire.SketchEntityFormatView{EntityID: in.EntityID, HasFormat: ok, Format: f}, nil
}

// getSketchFormatModes serves wire.MethodSketchGetFormatModes.
func getSketchFormatModes(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	return json.Marshal(wire.SketchFormatModesView{Modes: s.FormatModes()})
}

// setSketchFormatModes serves wire.MethodSketchSetFormatModes. Every mode is replaced, so a
// caller changing one reads first.
func setSketchFormatModes(s *app.Session, in wire.SketchFormatModesView) (wire.SketchFormatModesView, error) {
	if err := s.SetFormatModes(in.Modes); err != nil {
		return wire.SketchFormatModesView{}, err
	}
	return wire.SketchFormatModesView{Modes: s.FormatModes()}, nil
}

// compile-time proof the view carries the contract's type, so the DTO and the model cannot drift.
var _ types.SketchEntityFormat = wire.SketchEntityFormatView{}.Format
