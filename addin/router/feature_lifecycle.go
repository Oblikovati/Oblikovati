// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"

	"oblikovati.org/addin/modelaccess"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/param"
)

// Handlers for the features.* lifecycle methods (issue #140): get, edit (scalar
// inputs), delete, rename, setSuppressed, reorder. Features are addressed by the
// stable id model.tree reports — never by index, which reorder/delete invalidate.
// Mutations go through the app session so every change records an undo edit.

// featureInfo renders one feature as the model-tree wire DTO (shared with modelTree).
func featureInfo(f *feature.PartFeature) wire.FeatureInfo {
	fi := wire.FeatureInfo{ID: uint64(f.ID()), Name: f.Name(), Kind: f.Kind(), Suppressed: f.Suppressed()}
	if h := f.Health(); !h.OK() {
		fi.Health = h.Reason
	}
	return fi
}

// partFeatureByID resolves a wire feature id against the active part, also returning
// the feature's current history index, naming the wire method for errors.
func partFeatureByID(part *compdef.PartComponentDefinition, id uint64, method string) (*feature.PartFeature, int, error) {
	feats := part.Features()
	for i := 0; i < feats.Count(); i++ {
		if uint64(feats.Item(i).ID()) == id {
			return feats.Item(i), i, nil
		}
	}
	return nil, 0, fmt.Errorf("%s: no feature with id %d (ids come from model.tree)", method, id)
}

// featureDetail renders one feature with its history index and the editable scalars
// features.edit accepts, mirroring workPlaneInfo for datum planes.
func featureDetail(part *compdef.PartComponentDefinition, f *feature.PartFeature, index int) wire.FeatureDetail {
	return wire.FeatureDetail{FeatureInfo: featureInfo(f), Index: index, Scalars: featureScalars(part, f)}
}

// featureScalars renders the feature's editable scalars with their value in the
// document's preferred unit; nil when the definition exposes nothing editable.
func featureScalars(part *compdef.PartComponentDefinition, f *feature.PartFeature) []wire.FeatureScalar {
	ed, ok := f.Definition().(feature.Editable)
	if !ok {
		return nil
	}
	params := ed.EditableParams()
	out := make([]wire.FeatureScalar, len(params))
	for i, p := range params {
		out[i] = wire.FeatureScalar{
			Index:   i,
			Label:   p.Label,
			Unit:    part.Units().PreferredName(p.Unit),
			Value:   part.Units().ToPreferred(param.Q(p.Get(), p.Unit)),
			Integer: p.Integer,
		}
	}
	return out
}

// featureDetailReply marshals the refreshed-feature response shared by the
// get/edit/rename/setSuppressed/reorder handlers.
func featureDetailReply(part *compdef.PartComponentDefinition, f *feature.PartFeature, index int) (json.RawMessage, error) {
	return json.Marshal(wire.FeatureDetailResult{Feature: featureDetail(part, f, index)})
}

// getFeature returns one feature's state and editable scalars by id.
func getFeature(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, f, idx, err := resolveFeatureArgs(s, raw, wire.MethodFeaturesGet)
	if err != nil {
		return nil, err
	}
	return featureDetailReply(part, f, idx)
}

// resolveFeatureArgs decodes a FeatureRefArgs request and resolves its feature —
// the shared front half of the get/delete handlers.
func resolveFeatureArgs(s *app.Session, raw json.RawMessage, method string) (*compdef.PartComponentDefinition, *feature.PartFeature, int, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, nil, 0, err
	}
	var in wire.FeatureRefArgs
	if err := decode(raw, &in); err != nil {
		return nil, nil, 0, err
	}
	f, idx, err := partFeatureByID(part, in.ID, method)
	if err != nil {
		return nil, nil, 0, err
	}
	return part, f, idx, nil
}

// editFeature sets editable scalars of a placed feature in place. Every edit is
// validated before any is applied — a bad value mid-batch must not leave the
// definition half-edited — then the part recomputes once via CommitFeatureEdit.
func editFeature(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	var in wire.EditFeatureArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	f, idx, err := partFeatureByID(part, in.ID, wire.MethodFeaturesEdit)
	if err != nil {
		return nil, err
	}
	apply, err := parseScalarEdits(part, f, in.Scalars)
	if err != nil {
		return nil, err
	}
	apply()
	if err := s.CommitFeatureEdit(f); err != nil {
		return nil, err
	}
	return featureDetailReply(part, f, idx)
}

// parseScalarEdits validates the whole batch — the feature must be editable, each
// index in range, each value parseable in the scalar's unit — and returns a single
// closure that applies every Set (Set itself cannot fail).
func parseScalarEdits(part *compdef.PartComponentDefinition, f *feature.PartFeature, edits []wire.ScalarEdit) (func(), error) {
	if len(edits) == 0 {
		return nil, fmt.Errorf("features.edit: scalars is empty; expected at least one {index,value} edit")
	}
	ed, ok := f.Definition().(feature.Editable)
	if !ok {
		return nil, fmt.Errorf("features.edit: feature %d (%s) has no editable scalars", uint64(f.ID()), f.Kind())
	}
	params := ed.EditableParams()
	sets := make([]func(), len(edits))
	for i, e := range edits {
		if e.Index < 0 || e.Index >= len(params) {
			return nil, fmt.Errorf("features.edit: scalar index %d out of range (%d scalars, see features.get)", e.Index, len(params))
		}
		p := params[e.Index]
		q, err := part.Units().Parse(e.Value, p.Unit)
		if err != nil {
			return nil, fmt.Errorf("features.edit: scalar %d value %q: %w", e.Index, e.Value, err)
		}
		set, v := p.Set, q.Value
		sets[i] = func() { set(v) }
	}
	return func() {
		for _, set := range sets {
			set()
		}
	}, nil
}

// deleteFeature removes a feature from the history and recomputes.
func deleteFeature(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	_, f, _, err := resolveFeatureArgs(s, raw, wire.MethodFeaturesDelete)
	if err != nil {
		return nil, err
	}
	id := uint64(f.ID())
	if err := s.DeleteFeature(f); err != nil {
		return nil, err
	}
	return json.Marshal(wire.DeleteFeatureResult{ID: id, Deleted: true})
}

// renameFeature sets a feature's display name (the id stays stable).
func renameFeature(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	var in wire.RenameFeatureArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	f, idx, err := partFeatureByID(part, in.ID, wire.MethodFeaturesRename)
	if err != nil {
		return nil, err
	}
	if err := s.RenameFeature(f, in.Name); err != nil {
		return nil, err
	}
	return featureDetailReply(part, f, idx)
}

// setFeatureSuppressed sets explicit suppression (idempotent) and recomputes.
func setFeatureSuppressed(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	var in wire.SetFeatureSuppressedArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	f, idx, err := partFeatureByID(part, in.ID, wire.MethodFeaturesSetSuppressed)
	if err != nil {
		return nil, err
	}
	if err := s.SetFeatureSuppressed(f, in.Suppressed); err != nil {
		return nil, err
	}
	return featureDetailReply(part, f, idx)
}

// reorderFeature moves a feature to a new history index and recomputes. The detail
// is re-resolved after the move so the reply carries the feature's new index.
func reorderFeature(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	var in wire.ReorderFeatureArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	f, _, err := partFeatureByID(part, in.ID, wire.MethodFeaturesReorder)
	if err != nil {
		return nil, err
	}
	if err := s.ReorderFeature(f, in.NewIndex); err != nil {
		return nil, err
	}
	_, idx, err := partFeatureByID(part, in.ID, wire.MethodFeaturesReorder)
	if err != nil {
		return nil, err
	}
	return featureDetailReply(part, f, idx)
}

// registerFeatureHandlers wires the features.* methods: creation (list/add, backed by
// the operation registry) and the post-placement lifecycle (issue #140).
func (r *Router) registerFeatureHandlers() {
	r.handlers[wire.MethodFeaturesList] = r.listFeatureKinds
	r.handlers[wire.MethodFeaturesAdd] = r.addFeature
	r.handlers[wire.MethodFeaturesGet] = getFeature
	r.handlers[wire.MethodFeaturesEdit] = editFeature
	r.handlers[wire.MethodFeaturesDelete] = deleteFeature
	r.handlers[wire.MethodFeaturesRename] = renameFeature
	r.handlers[wire.MethodFeaturesSetSuppressed] = setFeatureSuppressed
	r.handlers[wire.MethodFeaturesReorder] = reorderFeature
}
