// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"oblikovati.org/model/health"
	"oblikovati.org/model/param"
)

// This file is the feature side of incremental parameter recompute (Oblikovati#1414):
// after a parameter edit, dirty only the features the edit can change instead of the
// whole program. The engine already rebuilds a CONTIGUOUS tail from the earliest dirty
// feature (see Recompute), so the only decision is the earliest feature an edit affects;
// everything before it is reused, everything from it rebuilds.

// MarkDirtyForParams marks dirty the earliest feature that depends on any of the changed
// parameters, and every feature after it. A feature depends on a changed parameter when it
// read the parameter directly (paramReads — a sheet-metal thickness, a suppression
// condition) or through a sketch it consumes (the sketch's captured dimension footprint). A
// sick feature is always re-evaluated, so fixing the offending parameter can recover it.
// Features before the earliest affected one are provably independent of the edit and keep
// their cached bodies.
//
// The caller must already have excluded parameters that reach geometry through paths the
// engine does not model per-feature (work-plane offsets, 3D-sketch dimensions); those force
// a full rebuild upstream (see compdef RecomputeAfterParameterEdit). With an empty change
// set this is a no-op — nothing dirties.
func (fs *PartFeatures) MarkDirtyForParams(changed []param.ID) {
	if len(changed) == 0 {
		return
	}
	set := idSetOf(changed)
	for i, pf := range fs.items {
		if fs.featureAffectedByParams(pf, set) {
			for _, tail := range fs.items[i:] {
				tail.dirty = true
			}
			return
		}
	}
}

// featureAffectedByParams reports whether a parameter in changed reaches pf — directly
// (its own reads) or through a sketch it consumes. A sick feature counts as affected so a
// parameter fix retries it.
func (fs *PartFeatures) featureAffectedByParams(pf *PartFeature, changed map[param.ID]bool) bool {
	if pf.health.Status == health.Sick {
		return true
	}
	if intersectsParams(pf.paramReads, changed) {
		return true
	}
	for _, sk := range pf.ConsumedSketches() {
		if intersectsParams(sk.ParameterFootprint(), changed) {
			return true
		}
	}
	return false
}

// ParameterReads returns every model parameter any feature read directly during the last
// recompute (the union of per-feature paramReads). The part recompute unions it with the
// consumed-sketch footprints to form the set it can target precisely; any parameter read
// elsewhere during recompute (a work-plane offset, a 3D-sketch dimension) falls outside it
// and forces a full rebuild — so an unmodelled path is never silently skipped.
func (fs *PartFeatures) ParameterReads() []param.ID {
	seen := map[param.ID]bool{}
	var out []param.ID
	for _, pf := range fs.items {
		for _, id := range pf.paramReads {
			if !seen[id] {
				seen[id] = true
				out = append(out, id)
			}
		}
	}
	return out
}

// idSetOf builds a lookup set from a parameter id slice.
func idSetOf(ids []param.ID) map[param.ID]bool {
	set := make(map[param.ID]bool, len(ids))
	for _, id := range ids {
		set[id] = true
	}
	return set
}

// intersectsParams reports whether any id in ids is in set.
func intersectsParams(ids []param.ID, set map[param.ID]bool) bool {
	for _, id := range ids {
		if set[id] {
			return true
		}
	}
	return false
}
