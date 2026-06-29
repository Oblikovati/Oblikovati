// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"bytes"
	"encoding/json"
	"fmt"

	"oblikovati.org/model/depend"
	"oblikovati.org/model/feature"
)

// restoreFeatureTail applies a snapshot that changed only a feature tail — every parameter,
// sketch, work feature, and other non-feature section is identical to the live model, and the
// feature program agrees up to prefix. It keeps the live engine and its cached prefix bodies and
// rebuilds only features[prefix:], then recomputes from the dirty index (Oblikovati#1424). This
// is the incremental undo/redo path: where a full RestoreSnapshot re-evaluates every feature
// from an empty engine, this re-evaluates just the tail the edit actually touched.
//
// Correctness rests on the precondition fastRestorePrefix verifies: identical non-feature
// sections and an identical feature prefix mean the kept prefix bodies are exactly what the
// snapshot implies, so reusing them is sound. The prior wholesale-parameter set is merged back
// (never shrunk) because the partial recompute does not re-read the prefix's parameters, and a
// shrunk wholesale set could let a later parameter edit silently skip a feature (#1414).
func (d *PartComponentDefinition) restoreFeatureTail(r partRecipe, prefix int) error {
	if err := d.features.ReplaceFrom(prefix, r.Features[prefix:], sketchIndex{d.sketches}, d.work); err != nil {
		return fmt.Errorf("compdef: restore feature tail: %w", err)
	}
	d.rebindSketchProjections()
	prior := d.wholesaleParams
	d.Recompute()
	d.mergeWholesaleParams(prior)
	return nil
}

// fastRestorePrefix reports whether a snapshot restore can reuse the live engine's cached prefix
// (and the length of that reusable feature prefix). It is the eligibility check for the
// incremental path: every non-feature recipe section must be byte-identical to the live model and
// the feature programs must share a prefix. ok=false forces the full reset+rebuild — the always-
// correct fallback, so a missed fast path only costs speed, never correctness.
//
// Sections are compared as marshalled JSON rather than by reflect.DeepEqual so that a live recipe
// (nil slices) and a snapshot-decoded recipe (empty slices) normalise identically — the
// comparison reflects semantic equality, not in-memory representation.
func (d *PartComponentDefinition) fastRestorePrefix(target partRecipe) (int, bool) {
	cur, err := d.buildRecipe()
	if err != nil {
		return 0, false
	}
	if !sameNonFeatureSections(cur, target) {
		return 0, false
	}
	return commonFeaturePrefix(cur.Features, target.Features), true
}

// sameNonFeatureSections reports whether two recipes agree on everything except their feature
// programs, comparing the marshalled JSON of each with its Features field blanked.
func sameNonFeatureSections(a, b partRecipe) bool {
	a.Features, b.Features = nil, nil
	aj, err1 := json.Marshal(a)
	bj, err2 := json.Marshal(b)
	return err1 == nil && err2 == nil && bytes.Equal(aj, bj)
}

// commonFeaturePrefix returns the number of leading features two programs share (compared as
// marshalled JSON), so the restore knows the first feature index that actually changed.
func commonFeaturePrefix(a, b []feature.FeatureData) int {
	n := min(len(a), len(b))
	for i := 0; i < n; i++ {
		if !sameFeatureData(a[i], b[i]) {
			return i
		}
	}
	return n
}

// sameFeatureData reports whether two serialized features are identical, comparing marshalled
// JSON (robust to nil-vs-empty differences between a live and a decoded feature).
func sameFeatureData(a, b feature.FeatureData) bool {
	aj, err1 := json.Marshal(a)
	bj, err2 := json.Marshal(b)
	return err1 == nil && err2 == nil && bytes.Equal(aj, bj)
}

// mergeWholesaleParams unions prior into the current wholesale set, so an incremental restore
// never shrinks it (the partial recompute did not re-read the kept prefix's parameters, so its
// freshly-derived set is incomplete; a superset is conservative and safe — it can only force an
// unnecessary full rebuild on a later parameter edit, never permit a stale one — #1414).
func (d *PartComponentDefinition) mergeWholesaleParams(prior map[depend.Key]bool) {
	if len(prior) == 0 {
		return
	}
	if d.wholesaleParams == nil {
		d.wholesaleParams = map[depend.Key]bool{}
	}
	for k := range prior {
		d.wholesaleParams[k] = true
	}
}
