// SPDX-License-Identifier: GPL-2.0-only

package feature

import "fmt"

// ReplaceFrom rebuilds the feature program from index prefix onward from the given recipe data,
// keeping features [0,prefix) — and the cached bodies they already computed — untouched. It is
// the engine half of incremental undo/redo (Oblikovati#1424): when a snapshot restore changed
// only a feature tail (every earlier feature and every non-feature recipe section is identical),
// reusing the clean prefix means the next Recompute re-evaluates just the rebuilt tail instead
// of the whole program — undoing one fillet on a long part costs one fillet, not N features.
//
// The kept prefix seeds pattern/mirror source resolution (sources are recorded as program
// indices), so a rebuilt tail feature that replicates an earlier one still binds correctly. The
// rebuilt tail features are added dirty, so Recompute's earliest-dirty scan starts at prefix.
//
// Callers must establish the precondition (identical prefix + non-feature sections) themselves;
// ReplaceFrom does not verify it. data is the tail only — the features at [prefix:].
func (fs *PartFeatures) ReplaceFrom(prefix int, data []FeatureData, sk SketchIndexer, work *WorkGeometry) error {
	if prefix < 0 || prefix > len(fs.items) {
		return fmt.Errorf("feature: ReplaceFrom prefix %d out of range [0,%d]", prefix, len(fs.items))
	}
	for _, pf := range fs.items[prefix:] {
		delete(fs.byID, pf.id) // drop the old tail's id bindings before rebuilding
	}
	// Cap the kept slice so the first rebuilt feature's append allocates a fresh backing array
	// rather than overwriting the old tail in place (the old tail may still be referenced).
	fs.items = fs.items[:prefix:prefix]
	restored := append([]*PartFeature(nil), fs.items...) // source resolution sees the kept prefix
	for i, fd := range data {
		pf, err := buildFeature(fs, fd, sk, restored, work) // Add-appends to fs.items
		if err != nil {
			return fmt.Errorf("feature %d (%s): %w", prefix+i, fd.Kind, err)
		}
		applyFeatureState(pf, fd)
		restored = append(restored, pf)
	}
	return nil
}
