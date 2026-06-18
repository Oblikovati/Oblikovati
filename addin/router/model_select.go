// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"

	"oblikovati.org/addin/modelaccess"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/event"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/model/feature"
)

// Selection mutation (#157): make the read-only selection writable. References are the face/vertex
// strings model.selection / model.referenceKeys report; these resolve them against the active part's
// bodies, mutate the session selection, announce the change (relayed to add-ins via #148), and
// return the new selection summary.

// registerSelectionMutationHandlers wires the model.select/deselect/clearSelection methods.
func (r *Router) registerSelectionMutationHandlers() {
	r.handlers[wire.MethodModelSelect] = modelSelect
	r.handlers[wire.MethodModelDeselect] = modelDeselect
	r.handlers[wire.MethodModelClearSelection] = modelClearSelection
}

// modelSelect selects the referenced entities, replacing the selection unless Mode is "add".
func modelSelect(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.SelectArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	bodies, err := activeBodies(s, wire.MethodModelSelect)
	if err != nil {
		return nil, err
	}
	if in.Mode != "add" {
		s.Selection().Clear()
	}
	for _, ref := range in.Refs {
		sel, ok := resolveSelectionRef(bodies, ref)
		if !ok {
			return nil, fmt.Errorf("%s: cannot resolve reference %q (not a face/vertex of the active part)", wire.MethodModelSelect, ref)
		}
		s.Selection().Add(sel)
	}
	return announceSelection(s)
}

// modelDeselect removes the referenced entities from the selection (unknown refs are ignored).
func modelDeselect(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.DeselectArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	bodies, err := activeBodies(s, wire.MethodModelDeselect)
	if err != nil {
		return nil, err
	}
	for _, ref := range in.Refs {
		if sel, ok := resolveSelectionRef(bodies, ref); ok {
			s.Selection().Remove(sel)
		}
	}
	return announceSelection(s)
}

// modelClearSelection clears the whole selection.
func modelClearSelection(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	s.Selection().Clear()
	return announceSelection(s)
}

// activeBodies returns the active part's surface bodies, erroring when there is no active part.
func activeBodies(s *app.Session, method string) ([]*topo.Body, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", method, err)
	}
	n := part.SurfaceBodies().Count()
	bodies := make([]*topo.Body, 0, n)
	for i := 0; i < n; i++ {
		bodies = append(bodies, part.SurfaceBodies().Item(i))
	}
	return bodies, nil
}

// resolveSelectionRef resolves a face/vertex reference string to a selectable handle on the bodies.
// Edges/work-features are not round-trippable through model.selection yet (it reports no ref for
// them), so they are not resolved here.
func resolveSelectionRef(bodies []*topo.Body, ref string) (app.Selectable, bool) {
	if key, ok := feature.FaceRefKey(feature.WorkRef(ref)); ok {
		for _, b := range bodies {
			if f, found := b.FindFaceByKey(key); found {
				return app.FaceHandle{Face: f, Body: b}, true
			}
		}
	}
	if key, ok := feature.VertexRefKey(feature.WorkRef(ref)); ok {
		for _, b := range bodies {
			if v, found := b.FindVertexByKey(key); found {
				return app.VertexHandle{Vertex: v}, true
			}
		}
	}
	return nil, false
}

// announceSelection emits the selection-changed event (relayed to add-ins, #148) and returns the
// new selection summary — the same shape model.selection reports.
func announceSelection(s *app.Session) (json.RawMessage, error) {
	event.Emit(s.Events(), event.After, app.SelectionChanged{Count: s.Selection().Count()})
	return json.Marshal(currentSelection(s))
}

// currentSelection builds the selection summary DTO from the session selection.
func currentSelection(s *app.Session) wire.SelectionResult {
	items := s.Selection().Items()
	kinds := make([]int, len(items))
	for i, it := range items {
		kinds[i] = int(it.SelectionKind())
	}
	return wire.SelectionResult{Count: len(items), Kinds: kinds, Refs: s.Selection().References()}
}
