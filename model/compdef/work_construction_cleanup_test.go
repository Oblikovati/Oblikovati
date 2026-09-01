// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"testing"

	"oblikovati.org/model/feature"
)

// TestConstructionConsumerSnapshotAndPrune drives the auto-delete lifecycle at the part level: a
// construction plane with no consumer is never in the snapshot; once a sketch hosts it, it is; and
// removing that sketch then pruning the snapshot tombstones the plane — its last consumer went (#1849).
func TestConstructionConsumerSnapshotAndPrune(t *testing.T) {
	t.Parallel()
	def := NewPartComponentDefinition()
	wp := def.WorkPlanes().AddByPlaneAndOffset(feature.OriginXYPlane, func() float64 { return 5 })
	wp.SetConstruction(true)
	def.Recompute()
	ref := string(wp.Key())

	if snap := def.ConstructionConsumerSnapshot(); len(snap) != 0 {
		t.Fatalf("a construction plane with no consumer must not be a prune candidate, got %v", snap)
	}

	sk := def.Sketches().Add(wp.Plane())
	sk.SetHostWorkRef(ref)
	snap := def.ConstructionConsumerSnapshot()
	if len(snap) != 1 || snap[0] != ref {
		t.Fatalf("snapshot = %v, want [%s] (the sketch consumes it)", snap, ref)
	}

	def.Sketches().Remove(sk.ID())
	if n := def.PruneConstructionOrphans(snap); n != 1 {
		t.Fatalf("prune removed %d, want 1 (the plane lost its last consumer)", n)
	}
	if !wp.Deleted() {
		t.Error("the construction plane should be tombstoned once its last consumer went")
	}
}

// TestPruneRetainsWithSketchConsumer: a construction plane still hosting a sketch is retained by
// PruneConstructionOrphans — datumHasConsumer sees the live sketch (#1849).
func TestPruneRetainsWithSketchConsumer(t *testing.T) {
	t.Parallel()
	def := NewPartComponentDefinition()
	wp := def.WorkPlanes().AddByPlaneAndOffset(feature.OriginXYPlane, func() float64 { return 5 })
	wp.SetConstruction(true)
	sk := def.Sketches().Add(wp.Plane())
	sk.SetHostWorkRef(string(wp.Key()))
	def.Recompute()

	if n := def.PruneConstructionOrphans([]string{string(wp.Key())}); n != 0 {
		t.Fatalf("prune removed %d, want 0 while a sketch still hosts the plane", n)
	}
	if wp.Deleted() {
		t.Error("a plane with a live sketch consumer must not be pruned")
	}
}

// TestRefTokenAppears pins the boundary rules of the recipe scan that detects a feature's retained
// datum reference: a ref matches only as a whole token, never as the prefix of a longer ref
// ("plane/5" must not match inside "plane/50") nor inside a larger identifier (#1849).
func TestRefTokenAppears(t *testing.T) {
	t.Parallel()
	cases := []struct {
		hay  string
		ref  string
		want bool
	}{
		{"plane: plane/5\n", "plane/5", true}, // a feature's plane field
		{"toPlane: plane/5\nfrom: axis/1\n", "axis/1", true},
		{"plane: plane/50\n", "plane/5", false},         // trailing digit extends the ref
		{"plane: plane/5\n", "plane/50", false},         // the longer ref is absent
		{"note: myplane/5x\n", "plane/5", false},        // leading letter + trailing letter: not a token
		{"", "plane/3", false},                          // empty recipe (no features)
		{"a: point/12\nb: plane/3\n", "point/12", true}, // multi-line, exact token
		{"a: point/120\n", "point/12", false},           // trailing digit again
	}
	for _, c := range cases {
		if got := refTokenAppears([]byte(c.hay), c.ref); got != c.want {
			t.Errorf("refTokenAppears(%q, %q) = %v, want %v", c.hay, c.ref, got, c.want)
		}
	}
}
