// SPDX-License-Identifier: GPL-2.0-only

package param

import "oblikovati.org/model/depend"

// This file is the parameter side of incremental recompute (Oblikovati#1414): two
// small mechanisms that let the feature engine dirty only the features a parameter
// edit can change, instead of marking the whole program dirty and rebuilding every
// feature from index 0.
//
//   - Read tracking ([Track]) records which parameters a piece of work read through
//     [Parameter.ModelValue]. Capturing it around a sketch solve yields the dimension
//     parameters that position that sketch; around a feature recompute, the parameters
//     that feature consumes directly (a sheet-metal thickness, a suppression condition).
//   - Change tracking ([DrainChanged]) records which parameters an edit actually
//     re-evaluated (the edited parameter and its transitive dependents — recorded in
//     graph.recomputeFrom). The edit seam intersects the two: a feature is dirty only
//     when a parameter it reads is among the parameters the edit changed.
//
// Both are single-threaded by construction — the model recomputes on one goroutine —
// so neither needs synchronization.

// recordRead notes that parameter id was read, when a footprint capture is active.
// It is a no-op (one nil check) outside a [Track], so normal solves and renders pay
// nothing.
func (ps *Parameters) recordRead(id ID) {
	if ps.readSink != nil {
		ps.readSink[id] = true
	}
}

// Track runs fn while recording every parameter read through [Parameter.ModelValue],
// returning the ids read. Tracks compose: reads captured by an inner Track also bubble
// up to an enclosing one, so wrapping a whole recompute captures the union while each
// nested sketch/feature still gets its own footprint.
//
// Example: deps := params.Track(func() { sketch.Solve() }) // the dimensions that drive sketch
func (ps *Parameters) Track(fn func()) []ID {
	prev := ps.readSink
	sink := idSet{}
	ps.readSink = sink
	fn()
	ps.readSink = prev
	if prev != nil {
		for id := range sink {
			prev[id] = true
		}
	}
	return keysOf(sink)
}

// Key is the recompute-graph identity of a parameter (ADR-0044): a parameter read becomes
// a depend.Key the engine attributes to the consumers (sketches, features) that read it,
// so a later edit dirties only those. The conversion lives here, at the param↔depend
// boundary, so it is defined once.
func Key(id ID) depend.Key { return depend.Key{Kind: depend.ParameterKey, ID: uint64(id)} }

// keysOfIDs widens a parameter-id slice to dependency keys, preserving nil (an empty
// change-set must stay empty so the edit seam falls back to a full rebuild).
func keysOfIDs(ids []ID) []depend.Key {
	if len(ids) == 0 {
		return nil
	}
	keys := make([]depend.Key, len(ids))
	for i, id := range ids {
		keys[i] = Key(id)
	}
	return keys
}

// TrackKeys runs fn while capturing every parameter it read (through [Parameter.ModelValue])
// and returns them as dependency keys — the footprint the recompute engine stores for the
// piece of work fn performed (a sketch solve, a feature evaluation, a work-plane recompute).
func (ps *Parameters) TrackKeys(fn func()) []depend.Key { return keysOfIDs(ps.Track(fn)) }

// DrainChangedKeys returns the parameters an edit re-evaluated since the previous call as
// dependency keys, and clears the record. The edit seam intersects these against consumer
// footprints to dirty the minimal tail.
func (ps *Parameters) DrainChangedKeys() []depend.Key { return keysOfIDs(ps.DrainChanged()) }

// markChanged records that an edit re-evaluated parameter id (called from the graph
// reflow). The set accumulates until [DrainChanged] reads and clears it.
func (ps *Parameters) markChanged(id ID) {
	if ps.changed == nil {
		ps.changed = idSet{}
	}
	ps.changed[id] = true
}

// DrainChanged returns the parameters re-evaluated since the previous call and clears
// the record. The edit seam ([compdef] RecomputeAfterParameterEdit) uses it to learn
// exactly which parameters an edit changed, so it can dirty only their dependent
// features. An empty result means no parameter value changed (or the changes predate
// tracking) — the caller then rebuilds conservatively.
func (ps *Parameters) DrainChanged() []ID {
	out := keysOf(ps.changed)
	ps.changed = nil
	return out
}
