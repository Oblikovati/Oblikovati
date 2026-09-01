// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/model/sketch"
)

// TestReplaceFromKeepsPrefixRebuildsTail is the engine half of incremental undo (#1424): replacing
// the program from an index keeps the earlier features (same objects, same cached recompute
// counts) and rebuilds only the tail, so the next Recompute re-evaluates just the tail.
func TestReplaceFromKeepsPrefixRebuildsTail(t *testing.T) {
	t.Parallel()
	sk := sketch.NewSketches().Add(sketch.XYPlane())
	fs := NewPartFeatures(nil)
	for range 3 {
		NewExtrudeFeatures(fs).AddByDistanceExtent(sk, 0, ops.NewBody, func() float64 { return 5 })
	}
	fs.Recompute()
	data, err := fs.MarshalRecipe(oneSketch{sk})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	keep0, keep1 := fs.Item(0), fs.Item(1)
	count0, count1 := keep0.RecomputeCount(), keep1.RecomputeCount()
	oldTail := fs.Item(2)

	if err := fs.ReplaceFrom(2, data[2:], oneSketch{sk}, nil); err != nil {
		t.Fatalf("ReplaceFrom: %v", err)
	}

	if fs.Count() != 3 {
		t.Fatalf("after ReplaceFrom: %d features, want 3", fs.Count())
	}
	if fs.Item(0) != keep0 || fs.Item(1) != keep1 {
		t.Error("ReplaceFrom rebuilt the kept prefix; it must reuse those features")
	}
	if fs.Item(0).RecomputeCount() != count0 || fs.Item(1).RecomputeCount() != count1 {
		t.Error("kept prefix recompute counters changed; the prefix was not reused")
	}
	if fs.Item(2) == oldTail {
		t.Error("the tail feature was not rebuilt")
	}
	// The rebuilt tail is dirty, so the next recompute re-evaluates only it.
	if !fs.RequiresUpdate() {
		t.Error("rebuilt tail is not marked for recompute")
	}
}

func TestReplaceFromRejectsOutOfRangePrefix(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	sk := sketch.NewSketches().Add(sketch.XYPlane())
	NewExtrudeFeatures(fs).AddByDistanceExtent(sk, 0, ops.NewBody, func() float64 { return 5 })
	if err := fs.ReplaceFrom(2, nil, oneSketch{sk}, nil); err == nil {
		t.Error("ReplaceFrom past the program length returned no error")
	}
	if err := fs.ReplaceFrom(-1, nil, oneSketch{sk}, nil); err == nil {
		t.Error("ReplaceFrom with a negative prefix returned no error")
	}
}

// TestReplaceFromTruncatesTail: replacing from a prefix with no tail data drops the old tail and
// removes its id bindings (a restore that undid feature additions).
func TestReplaceFromTruncatesTail(t *testing.T) {
	t.Parallel()
	sk := sketch.NewSketches().Add(sketch.XYPlane())
	fs := NewPartFeatures(nil)
	for range 3 {
		NewExtrudeFeatures(fs).AddByDistanceExtent(sk, 0, ops.NewBody, func() float64 { return 5 })
	}
	removedID := fs.Item(2).ID()

	if err := fs.ReplaceFrom(2, nil, oneSketch{sk}, nil); err != nil {
		t.Fatalf("ReplaceFrom: %v", err)
	}
	if fs.Count() != 2 {
		t.Fatalf("after truncation: %d features, want 2", fs.Count())
	}
	if _, ok := fs.ByID(removedID); ok {
		t.Error("the dropped tail feature's id binding survived")
	}
}
