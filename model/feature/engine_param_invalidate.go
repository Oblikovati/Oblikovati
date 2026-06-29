// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"oblikovati.org/model/depend"
	"oblikovati.org/model/health"
)

// This file is the feature side of incremental recompute (Oblikovati#1414, ADR-0044):
// after a change, dirty only the features the change can reach instead of the whole
// program. The engine already rebuilds a CONTIGUOUS tail from the earliest dirty feature
// (see Recompute), so the only decision is the earliest feature a change affects;
// everything before it is reused, everything from it rebuilds. The change-set and the
// per-feature footprints are dependency keys (depend.Key), so the attribution is agnostic
// to whether the changed source is a parameter (today) or external geometry (a future
// adaptive reference) — the same intersection serves both.

// MarkDirtyForChange marks dirty the earliest feature that depends on any of the changed
// keys, and every feature after it. A feature depends on a changed key when it read the
// key directly (paramReads — a sheet-metal thickness, a suppression condition) or through
// a sketch it consumes (the sketch's captured footprint, which includes its host work
// plane's footprint). A sick feature is always re-evaluated, so fixing the offending
// source can recover it. Features before the earliest affected one are provably
// independent of the change and keep their cached bodies.
//
// The caller must already have excluded keys that reach geometry through paths the engine
// does not model per-feature; those force a full rebuild upstream (see compdef
// RecomputeAfterChange). With an empty change set this is a no-op — nothing dirties.
func (fs *PartFeatures) MarkDirtyForChange(changed []depend.Key) {
	if len(changed) == 0 {
		return
	}
	set := depend.NewSet(changed)
	for i, pf := range fs.items {
		if fs.featureAffectedByChange(pf, set) {
			for _, tail := range fs.items[i:] {
				tail.dirty = true
			}
			return
		}
	}
}

// featureAffectedByChange reports whether a changed key reaches pf — directly (its own
// reads) or through a sketch it consumes. A sick feature counts as affected so a fix
// retries it.
func (fs *PartFeatures) featureAffectedByChange(pf *PartFeature, changed depend.Set) bool {
	if pf.health.Status == health.Sick {
		return true
	}
	if depend.Intersects(pf.paramReads, changed) {
		return true
	}
	for _, sk := range pf.ConsumedSketches() {
		if depend.Intersects(sk.ParameterFootprint(), changed) {
			return true
		}
	}
	return false
}

// DependencyReads returns every key any feature read directly during the last recompute
// (the union of per-feature paramReads). The part recompute unions it with the
// consumed-sketch footprints to form the set it can target precisely; any read elsewhere
// during recompute (a 3D-sketch dimension, a host-plane closure not yet attributed) falls
// outside it and forces a full rebuild — so an unmodelled path is never silently skipped.
func (fs *PartFeatures) DependencyReads() []depend.Key {
	seen := map[depend.Key]bool{}
	var out []depend.Key
	for _, pf := range fs.items {
		for _, k := range pf.paramReads {
			if !seen[k] {
				seen[k] = true
				out = append(out, k)
			}
		}
	}
	return out
}
