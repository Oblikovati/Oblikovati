// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
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
	return wire.FeatureDetail{
		FeatureInfo: featureInfo(f), Index: index,
		Scalars: featureScalars(part, f), Slots: featureSlots(f),
	}
}

// featureScalars renders the feature's editable scalars with their value in the
// document's preferred unit; nil when the definition exposes nothing editable.
func featureScalars(part *compdef.PartComponentDefinition, f *feature.PartFeature) []wire.FeatureScalar {
	return editableScalars(part.Units(), f.Definition())
}

// featureDetailReply marshals the refreshed-feature response shared by the raw
// assembly-derive / freeform edit handlers (which still return json.RawMessage).
func featureDetailReply(part *compdef.PartComponentDefinition, f *feature.PartFeature, index int) (json.RawMessage, error) {
	return json.Marshal(featureDetailResult(part, f, index))
}

// featureDetailResult renders the refreshed-feature response shared by the typed
// get/edit/rename/setSuppressed/reorder handlers.
func featureDetailResult(part *compdef.PartComponentDefinition, f *feature.PartFeature, index int) wire.FeatureDetailResult {
	return wire.FeatureDetailResult{Feature: featureDetail(part, f, index)}
}

// getFeature returns one feature's state and editable scalars by id.
func getFeature(_ *app.Session, part *compdef.PartComponentDefinition, in wire.FeatureRefArgs) (wire.FeatureDetailResult, error) {
	f, idx, err := partFeatureByID(part, in.ID, wire.MethodFeaturesGet)
	if err != nil {
		return wire.FeatureDetailResult{}, err
	}
	return featureDetailResult(part, f, idx), nil
}

// editFeature sets editable scalars of a placed feature in place. Every edit is
// validated before any is applied — a bad value mid-batch must not leave the
// definition half-edited — then the part recomputes once via CommitFeatureEdit.
func editFeature(s *app.Session, part *compdef.PartComponentDefinition, in wire.EditFeatureArgs) (wire.FeatureDetailResult, error) {
	f, idx, err := partFeatureByID(part, in.ID, wire.MethodFeaturesEdit)
	if err != nil {
		return wire.FeatureDetailResult{}, err
	}
	if err := planAndApplyFeatureEdits(part, f, in); err != nil {
		return wire.FeatureDetailResult{}, err
	}
	if err := s.CommitFeatureEdit(f); err != nil { // emits feature.edited (#1085)
		return wire.FeatureDetailResult{}, err
	}
	return featureDetailResult(part, f, idx), nil
}

// planAndApplyFeatureEdits validates the WHOLE batch — scalars and reference re-picks — before
// applying any, so a failure (bad value, missing profile, wrong slot kind) leaves the definition
// untouched (#163). Scalars are parsed only when present, so a repick-only edit works on a feature
// that exposes references but no editable scalars (e.g. a mirror's plane). The caller recomputes.
func planAndApplyFeatureEdits(part *compdef.PartComponentDefinition, f *feature.PartFeature, in wire.EditFeatureArgs) error {
	if len(in.Scalars) == 0 && len(in.Repick) == 0 {
		return fmt.Errorf("features.edit: feature %d has no scalar or repick edits to apply", in.ID)
	}
	applyScalars := func() { /* default: no scalar edits to apply */ }
	if len(in.Scalars) > 0 {
		apply, err := parseScalarEdits(part, f, in.Scalars)
		if err != nil {
			return err
		}
		applyScalars = apply
	}
	applyRepicks, err := planFeatureRepicks(part, f, in.Repick)
	if err != nil {
		return err
	}
	applyScalars()
	applyRepicks()
	return nil
}

// parseScalarEdits validates the whole batch — the feature must be editable, each
// index in range, each value parseable in the scalar's unit — and returns a single
// closure that applies every Set (Set itself cannot fail).
func parseScalarEdits(part *compdef.PartComponentDefinition, f *feature.PartFeature, edits []wire.ScalarEdit) (func(), error) {
	ed, ok := f.Definition().(feature.Editable)
	if !ok {
		return nil, fmt.Errorf("features.edit: feature %d (%s) has no editable scalars", uint64(f.ID()), f.Kind())
	}
	return planScalarEdits(part.Units(), ed, edits, wire.MethodFeaturesEdit)
}

// deleteFeature removes a feature from the history and recomputes.
func deleteFeature(s *app.Session, part *compdef.PartComponentDefinition, in wire.FeatureRefArgs) (wire.DeleteFeatureResult, error) {
	f, _, err := partFeatureByID(part, in.ID, wire.MethodFeaturesDelete)
	if err != nil {
		return wire.DeleteFeatureResult{}, err
	}
	id := uint64(f.ID())
	snapshot := snapshotConstructionConsumers(part) // construction datums with a consumer, before delete (#1849)
	if err := s.DeleteFeature(f); err != nil {      // emits feature.deleted (#1085)
		return wire.DeleteFeatureResult{}, err
	}
	pruneConstructionAfterDelete(part, snapshot) // auto-delete a construction datum this feature was the last consumer of
	return wire.DeleteFeatureResult{ID: id, Deleted: true}, nil
}

// renameFeature sets a feature's display name (the id stays stable).
func renameFeature(s *app.Session, part *compdef.PartComponentDefinition, in wire.RenameFeatureArgs) (wire.FeatureDetailResult, error) {
	f, idx, err := partFeatureByID(part, in.ID, wire.MethodFeaturesRename)
	if err != nil {
		return wire.FeatureDetailResult{}, err
	}
	if err := s.RenameFeature(f, in.Name); err != nil {
		return wire.FeatureDetailResult{}, err
	}
	return featureDetailResult(part, f, idx), nil
}

// setFeatureSuppressed sets explicit suppression (idempotent) and recomputes.
func setFeatureSuppressed(s *app.Session, part *compdef.PartComponentDefinition, in wire.SetFeatureSuppressedArgs) (wire.FeatureDetailResult, error) {
	f, idx, err := partFeatureByID(part, in.ID, wire.MethodFeaturesSetSuppressed)
	if err != nil {
		return wire.FeatureDetailResult{}, err
	}
	if err := s.SetFeatureSuppressed(f, in.Suppressed); err != nil {
		return wire.FeatureDetailResult{}, err
	}
	return featureDetailResult(part, f, idx), nil
}

// reorderFeature moves a feature to a new history index and recomputes. The detail
// is re-resolved after the move so the reply carries the feature's new index.
func reorderFeature(s *app.Session, part *compdef.PartComponentDefinition, in wire.ReorderFeatureArgs) (wire.FeatureDetailResult, error) {
	f, _, err := partFeatureByID(part, in.ID, wire.MethodFeaturesReorder)
	if err != nil {
		return wire.FeatureDetailResult{}, err
	}
	if err := s.ReorderFeature(f, in.NewIndex); err != nil {
		return wire.FeatureDetailResult{}, err
	}
	_, idx, err := partFeatureByID(part, in.ID, wire.MethodFeaturesReorder)
	if err != nil {
		return wire.FeatureDetailResult{}, err
	}
	return featureDetailResult(part, f, idx), nil
}

// registerFeatureHandlers wires the features.* methods: creation (list/add, backed by
// the operation registry) and the post-placement lifecycle (issue #140).
func (r *Router) registerFeatureHandlers() {
	r.readOnly(wire.MethodFeaturesList, r.listFeatureKinds)
	r.mutating(wire.MethodFeaturesAdd, "Add Feature", r.addFeature)
	r.readOnly(wire.MethodFeaturesGet, typedPart(getFeature))
	r.mutating(wire.MethodFeaturesEdit, "Edit Feature", typedPart(editFeature))
	r.mutating(wire.MethodFeaturesDelete, "Delete Feature", typedPart(deleteFeature))
	r.mutating(wire.MethodFeaturesRename, "Rename Feature", typedPart(renameFeature))
	r.mutating(wire.MethodFeaturesSetSuppressed, "Suppress Feature", typedPart(setFeatureSuppressed))
	r.mutating(wire.MethodFeaturesReorder, "Reorder Features", typedPart(reorderFeature))
}
