// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
)

// Highlight sets (#157): named, colored emphasis groups the viewport outlines without selecting.
// The state lives on the session (app.HighlightSets); these handlers create/mutate/list it, and
// the head's viewport overlay resolves each set's refs against the live geometry and draws them.

// registerHighlightSetHandlers wires the model.highlightSets.* methods.
func (r *Router) registerHighlightSetHandlers() {
	r.readOnly(wire.MethodModelHighlightSetCreate, createHighlightSet)
	r.readOnly(wire.MethodModelHighlightSetDelete, deleteHighlightSet)
	r.readOnly(wire.MethodModelHighlightSetAddItems, addHighlightItems)
	r.readOnly(wire.MethodModelHighlightSetSetColor, setHighlightSetColor)
	r.readOnly(wire.MethodModelHighlightSetList, listHighlightSets)
}

// createHighlightSet adds a named, colored emphasis group.
func createHighlightSet(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.CreateHighlightSetArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	color, err := parseHexColor(in.Color)
	if err != nil {
		return nil, fmt.Errorf("model.highlightSets.create: %w", err)
	}
	h, err := s.HighlightSets().Create(in.Name, color)
	if err != nil {
		return nil, fmt.Errorf("model.highlightSets.create: %w", err)
	}
	return json.Marshal(highlightSetInfo(h))
}

// addHighlightItems adds references to a named set.
func addHighlightItems(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.HighlightSetItemsArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	h, ok := s.HighlightSets().ByName(in.Name)
	if !ok {
		return nil, fmt.Errorf("model.highlightSets.addItems: no highlight set %q", in.Name)
	}
	h.AddItems(in.Refs...)
	return json.Marshal(highlightSetInfo(h))
}

// setHighlightSetColor re-colours a named set.
func setHighlightSetColor(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.SetHighlightSetColorArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	h, ok := s.HighlightSets().ByName(in.Name)
	if !ok {
		return nil, fmt.Errorf("model.highlightSets.setColor: no highlight set %q", in.Name)
	}
	color, err := parseHexColor(in.Color)
	if err != nil {
		return nil, fmt.Errorf("model.highlightSets.setColor: %w", err)
	}
	h.SetColor(color)
	return json.Marshal(highlightSetInfo(h))
}

// deleteHighlightSet removes a named set.
func deleteHighlightSet(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.HighlightSetRefArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	if !s.HighlightSets().Delete(in.Name) {
		return nil, fmt.Errorf("model.highlightSets.delete: no highlight set %q", in.Name)
	}
	return json.Marshal(wire.OKResult{OK: true})
}

// listHighlightSets lists the session's highlight sets.
func listHighlightSets(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	sets := s.HighlightSets().All()
	out := make([]wire.HighlightSetInfo, len(sets))
	for i, h := range sets {
		out[i] = highlightSetInfo(h)
	}
	return json.Marshal(wire.ListHighlightSetsResult{Sets: out})
}

// highlightSetInfo renders a highlight set's wire summary.
func highlightSetInfo(h *app.HighlightSet) wire.HighlightSetInfo {
	return wire.HighlightSetInfo{Name: h.Name(), Color: h.Color().Hex(), Count: h.Count()}
}

// parseHexColor parses a "#rrggbb" string into a types.Color.
func parseHexColor(hex string) (types.Color, error) {
	rgba, err := types.ParseHex(hex)
	if err != nil {
		return types.Color{}, fmt.Errorf("color %q: %w", hex, err)
	}
	return types.NewColor(uint8(rgba.R*255+0.5), uint8(rgba.G*255+0.5), uint8(rgba.B*255+0.5)), nil
}
