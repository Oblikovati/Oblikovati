// SPDX-License-Identifier: GPL-2.0-only

package param

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
