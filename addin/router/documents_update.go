// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"

	"oblikovati.org/addin/modelaccess"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/health"
)

// Document update/rebuild (#139): drive and query the feature engine's recompute over the wire.
// update recomputes the out-of-date tail; rebuild marks every feature dirty first (a full
// parametric rebuild); requiresUpdate is the read-only stale-feature flag. The engine never
// aborts — a failed feature goes sick and poisons dependents — so update/rebuild report the sick
// features; with acceptErrorsAndContinue=false a sick feature instead fails the call.

// documentsUpdate recomputes the active part's out-of-date features.
func documentsUpdate(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	return recomputeActivePart(s, raw, false)
}

// documentsRebuild recomputes the active part's entire feature program (all features dirtied).
func documentsRebuild(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	return recomputeActivePart(s, raw, true)
}

// recomputeActivePart runs the engine (optionally dirtying everything first), then reports the
// resulting sick features — failing the call when any is sick and acceptErrorsAndContinue is off.
func recomputeActivePart(s *app.Session, raw json.RawMessage, rebuild bool) (json.RawMessage, error) {
	var in wire.UpdateDocumentArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	if rebuild {
		part.Features().MarkAllDirty()
	}
	part.Recompute()
	errs := sickFeatures(part)
	if len(errs) > 0 && !in.AcceptErrorsAndContinue {
		return nil, fmt.Errorf("documents.update: %d feature(s) are sick after recompute: %s "+
			"(pass acceptErrorsAndContinue to report them instead)", len(errs), errs[0].Name)
	}
	return json.Marshal(wire.UpdateDocumentResult{RequiresUpdate: part.Features().RequiresUpdate(), Errors: errs})
}

// documentsRequiresUpdate reports whether the active part has out-of-date features.
func documentsRequiresUpdate(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	return json.Marshal(wire.RequiresUpdateResult{RequiresUpdate: part.Features().RequiresUpdate()})
}

// sickFeatures collects the features that ended up sick after a recompute, in history order.
func sickFeatures(part *compdef.PartComponentDefinition) []wire.FeatureError {
	var out []wire.FeatureError
	feats := part.Features()
	for i := 0; i < feats.Count(); i++ {
		f := feats.Item(i)
		if h := f.Health(); h.Status == health.Sick {
			out = append(out, wire.FeatureError{ID: uint64(f.ID()), Name: f.Name(), Kind: f.Kind(), Message: h.Reason})
		}
	}
	return out
}
