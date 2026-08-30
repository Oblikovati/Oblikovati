// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"oblikovati.org/kernel/topo"
	"oblikovati.org/model/health"
)

// Feature recompute engine — the FEATURE GRAPH / ORDERING (M48 #2234 split of engine.go). The dirty
// tracking and the ordering queries the recompute loop drives from: which feature is the earliest dirty
// one, the effective end of the list, the prefix bodies before a restart point, and which upstream
// features are Sick so their dependents quarantine. The recompute loop lives in engine_recompute.go.

// MarkDirty flags a feature for re-evaluation on the next recompute.
func (fs *PartFeatures) MarkDirty(f *PartFeature) { f.dirty = true }

// RequiresUpdate reports whether any feature up to the end-of-part marker is dirty — i.e. a
// recompute would change the result. It is the read-only "needs update" flag the API exposes
// (Inventor PartDocument.RequiresUpdate), the predicate behind documents.update (#139).
func (fs *PartFeatures) RequiresUpdate() bool { return fs.earliestDirty(fs.effectiveEnd()) >= 0 }

// MarkAllDirty flags every feature for re-evaluation. A parameter edit can change
// any feature's inputs — a dimension expression a sketch profile is built from, or
// a feature's own value closure (extrude distance, revolve angle) — but those
// inputs are read live and are not tracked as feature dependencies, so the engine
// would otherwise see nothing dirty and return the cached bodies. Invalidating the
// whole program on a parameter change keeps the model correct (a full parametric
// rebuild, as Inventor does).
func (fs *PartFeatures) MarkAllDirty() {
	for _, pf := range fs.items {
		pf.dirty = true
	}
}

// effectiveEnd returns the evaluation cutoff (the EOP marker, or the full length).
func (fs *PartFeatures) effectiveEnd() int {
	if fs.eop == eopAll || fs.eop > len(fs.items) {
		return len(fs.items)
	}
	return fs.eop
}

// earliestDirty returns the index of the first dirty feature below end, or -1.
func (fs *PartFeatures) earliestDirty(end int) int {
	for i := range end {
		if fs.items[i].dirty {
			return i
		}
	}
	return -1
}

// prefixBodies returns the cached body state just before index start.
func (fs *PartFeatures) prefixBodies(start int) []*topo.Body {
	if start <= 0 {
		return nil
	}
	return fs.items[start-1].cached
}

// sickBefore collects the sick features in the reused clean prefix, so tail
// features depending on them are still poisoned.
func (fs *PartFeatures) sickBefore(start int) map[ID]bool {
	sick := map[ID]bool{}
	for i := range start {
		if fs.items[i].health.Status == health.Sick {
			sick[fs.items[i].id] = true
		}
	}
	return sick
}

// dependsOnSick reports whether any of pf's dependencies is currently sick.
func (fs *PartFeatures) dependsOnSick(pf *PartFeature, sick map[ID]bool) bool {
	for _, d := range pf.deps {
		if sick[d] {
			return true
		}
	}
	return false
}
