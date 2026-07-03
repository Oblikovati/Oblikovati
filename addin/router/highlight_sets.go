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
	r.readOnly(wire.MethodModelHighlightSetCreate, typed(createHighlightSet))
	r.readOnly(wire.MethodModelHighlightSetDelete, typed(deleteHighlightSet))
	r.readOnly(wire.MethodModelHighlightSetAddItems, typed(addHighlightItems))
	r.readOnly(wire.MethodModelHighlightSetSetColor, typed(setHighlightSetColor))
	r.readOnly(wire.MethodModelHighlightSetList, listHighlightSets)
}

// createHighlightSet adds a named, colored emphasis group.
func createHighlightSet(s *app.Session, in wire.CreateHighlightSetArgs) (wire.HighlightSetInfo, error) {
	color, err := parseHexColor(in.Color)
	if err != nil {
		return wire.HighlightSetInfo{}, fmt.Errorf("model.highlightSets.create: %w", err)
	}
	h, err := s.HighlightSets().Create(in.Name, color)
	if err != nil {
		return wire.HighlightSetInfo{}, fmt.Errorf("model.highlightSets.create: %w", err)
	}
	return highlightSetInfo(h), nil
}

// addHighlightItems adds references to a named set.
func addHighlightItems(s *app.Session, in wire.HighlightSetItemsArgs) (wire.HighlightSetInfo, error) {
	h, ok := s.HighlightSets().ByName(in.Name)
	if !ok {
		return wire.HighlightSetInfo{}, fmt.Errorf("model.highlightSets.addItems: no highlight set %q", in.Name)
	}
	h.AddItems(in.Refs...)
	return highlightSetInfo(h), nil
}

// setHighlightSetColor re-colours a named set.
func setHighlightSetColor(s *app.Session, in wire.SetHighlightSetColorArgs) (wire.HighlightSetInfo, error) {
	h, ok := s.HighlightSets().ByName(in.Name)
	if !ok {
		return wire.HighlightSetInfo{}, fmt.Errorf("model.highlightSets.setColor: no highlight set %q", in.Name)
	}
	color, err := parseHexColor(in.Color)
	if err != nil {
		return wire.HighlightSetInfo{}, fmt.Errorf("model.highlightSets.setColor: %w", err)
	}
	h.SetColor(color)
	return highlightSetInfo(h), nil
}

// deleteHighlightSet removes a named set.
func deleteHighlightSet(s *app.Session, in wire.HighlightSetRefArgs) (wire.OKResult, error) {
	if !s.HighlightSets().Delete(in.Name) {
		return wire.OKResult{}, fmt.Errorf("model.highlightSets.delete: no highlight set %q", in.Name)
	}
	return wire.OKResult{OK: true}, nil
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
