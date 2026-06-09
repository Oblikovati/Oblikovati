// SPDX-License-Identifier: GPL-2.0-only

// Package seq provides one process-wide monotonic creation clock so modeling nodes that
// live in separate collections — sketches, work features, and part features — share a
// single creation order. The model browser interleaves them by this stamp (Inventor's
// chronological model tree, issue #132), and edit-scope visibility uses it to tell what
// was created "after" the node being edited.
//
// Stamps are only ever compared within one document, so a single global clock is
// sufficient (cross-document ordering is never meaningful). A loaded document restores
// its nodes' saved stamps and calls [Bump] so freshly created nodes still sort after them.
package seq

import "sync/atomic"

// clock is the shared monotonic source. It mirrors the package-level id counter the
// feature engine already uses (model/feature.idSeq), kept here so both the sketch and
// feature packages can stamp from the same sequence without an import cycle.
var clock atomic.Uint64

// Next returns the next creation stamp: strictly increasing and never zero. Zero is
// reserved for the static origin coordinate frame, which must always sort first.
//
// Example: sk.seq = seq.Next() — stamped once, when the sketch is added to its collection.
func Next() uint64 { return clock.Add(1) }

// Bump raises the clock floor to at least v, so a node created after a document is loaded
// (its peers' stamps restored from the recipe) still receives a strictly larger stamp than
// every restored one. Safe for concurrent use; a no-op when v is already below the floor.
func Bump(v uint64) {
	for {
		cur := clock.Load()
		if v <= cur || clock.CompareAndSwap(cur, v) {
			return
		}
	}
}
