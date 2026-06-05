// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"github.com/Oblikovati/oblikovati/kernel/ops"
	"github.com/Oblikovati/oblikovati/kernel/topo"
	"github.com/Oblikovati/oblikovati/math"
)

// negXFaceKey returns the reference key of the body's first face whose outward normal is −X.
func negXFaceKey(t *testing.T, b *topo.Body) []byte {
	t.Helper()
	for _, f := range b.Faces() {
		if f.Geometry().NormalAt(0, 0).Dot(math.V3(-1, 0, 0)) > 0.9 {
			return f.ReferenceKey()
		}
	}
	t.Fatal("no −X face")
	return nil
}

// TestCombineFaceKeySurvivesIntoResultAndRecompute is the feature-level proof of K1a: a
// face picked on box A keeps its reference key after A is Combined with an overlapping box
// B (the planar boolean carries the lineage), and still resolves after a parameter-driven
// recompute — so a downstream reference (fillet/dimension/sketch) on the combined solid
// stays bound across edits.
func TestCombineFaceKeySurvivesIntoResultAndRecompute(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	exA := NewExtrudeFeatures(fs).AddByDistanceExtent(squareSketch(4), 0, ops.NewBody, func() float64 { return 5 })
	fs.Recompute()

	// A's −X face (at x=0) is far from where B will overlap, so it survives the boolean whole.
	aKey := negXFaceKey(t, fs.Result()[0])

	// Add an overlapping box B and Join them (intersecting → the planar B-rep boolean).
	NewExtrudeFeatures(fs).AddByDistanceExtent(squareSketchAt(4, 2), 0, ops.NewBody, func() float64 { return 5 })
	NewModifyFeatures(fs).AddCombine(0, 1, ops.Join)
	fs.Recompute()

	combined := fs.Result()
	if len(combined) != 1 {
		t.Fatalf("after join = %d bodies, want 1", len(combined))
	}
	if _, ok := combined[0].FindFaceByKey(aKey); !ok {
		t.Fatal("A's face key did not survive INTO the combine result (K1a)")
	}

	// Edit: grow A's height and recompute; the same key still binds.
	exA.Definition().(*ExtrudeFeature).SetDistance(8)
	fs.MarkDirty(exA)
	fs.Recompute()
	if _, ok := fs.Result()[0].FindFaceByKey(aKey); !ok {
		t.Fatal("A's face key did not survive the recompute after the combine (K1a)")
	}
}
