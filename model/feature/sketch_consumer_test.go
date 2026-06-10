// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// consumerSketch returns a minimal empty sketch. ConsumedSketches reports sketch IDENTITY
// only (it never resolves profiles), so the features need no real geometry — and no healthy
// recompute — to report what they consume.
func consumerSketch() *sketch.Sketch { return sketch.NewSketches().Add(sketch.XYPlane()) }

// consumerPatchLoops builds boundary-patch loops over the given sketches (nil entries
// included, to exercise the nil-loop handling).
func consumerPatchLoops(sks ...*sketch.Sketch) *BoundaryPatchLoops {
	ls := &BoundaryPatchLoops{}
	for _, sk := range sks {
		ls.Add(sk, 0, PatchFree)
	}
	return ls
}

// TestConsumedSketchesPerFeatureKind pins ConsumedSketches for every implementation in
// sketch_consumer.go: each kind reports exactly the distinct non-nil sketches it depends
// on, in definition order, with no nil entries and no duplicates (issue #132 — the browser
// nests each consumed sketch under its feature exactly once).
func TestConsumedSketchesPerFeatureKind(t *testing.T) {
	skA, skB := consumerSketch(), consumerSketch()
	apex := math.P3(0, 0, 5)
	cases := []struct {
		name     string
		consumer SketchConsumer
		want     []*sketch.Sketch
	}{
		{"extrude reports its profile sketch",
			&ExtrudeFeature{def: &ExtrudeDefinition{Sketch: skA}},
			[]*sketch.Sketch{skA}},
		{"emboss reports its profile sketch",
			&EmbossFeature{def: &EmbossDefinition{Sketch: skA}},
			[]*sketch.Sketch{skA}},
		{"coil reports its profile sketch",
			&CoilFeature{def: &CoilDefinition{Sketch: skA}},
			[]*sketch.Sketch{skA}},
		{"rib reports its profile sketch",
			&RibFeature{def: &RibDefinition{Sketch: skA}},
			[]*sketch.Sketch{skA}},
		{"sweep reports its profile sketch",
			&SweepFeature{def: &SweepDefinition{Sketch: skA}},
			[]*sketch.Sketch{skA}},
		{"revolve about an explicit work axis reports only the profile sketch",
			&RevolveFeature{def: &RevolveDefinition{Sketch: skA}},
			[]*sketch.Sketch{skA}},
		{"revolve whose centerline lives in the profile sketch dedupes it",
			&RevolveFeature{def: &RevolveDefinition{Sketch: skA, AxisCenterlineSketch: skA}},
			[]*sketch.Sketch{skA}},
		{"revolve with a separate centerline sketch reports both",
			&RevolveFeature{def: &RevolveDefinition{Sketch: skA, AxisCenterlineSketch: skB}},
			[]*sketch.Sketch{skA, skB}},
		{"loft skips a point (apex) section without a nil entry",
			&LoftFeature{def: &LoftDefinition{Sections: []LoftSection{{Sketch: skA}, {Point: &apex}, {Sketch: skB}}}},
			[]*sketch.Sketch{skA, skB}},
		{"loft of only point sections consumes nothing",
			&LoftFeature{def: &LoftDefinition{Sections: []LoftSection{{Point: &apex}, {Point: &apex}}}},
			nil},
		{"boundary patch with nil loops consumes nothing",
			&BoundaryPatchFeature{def: &BoundaryPatchDefinition{}},
			nil},
		{"boundary patch skips a nil-sketch loop",
			&BoundaryPatchFeature{def: &BoundaryPatchDefinition{Loops: consumerPatchLoops(skA, nil, skB)}},
			[]*sketch.Sketch{skA, skB}},
		{"boundary patch dedupes loops on one sketch",
			&BoundaryPatchFeature{def: &BoundaryPatchDefinition{Loops: consumerPatchLoops(skA, skA)}},
			[]*sketch.Sketch{skA}},
		{"ruled surface with a nil sketch consumes nothing",
			&RuledSurfaceFeature{def: &RuledSurfaceDefinition{}},
			nil},
		{"ruled surface reports its profile sketch",
			&RuledSurfaceFeature{def: &RuledSurfaceDefinition{Sketch: skA}},
			[]*sketch.Sketch{skA}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.consumer.ConsumedSketches()
			if len(got) != len(tc.want) {
				t.Fatalf("ConsumedSketches returned %d sketches, want %d", len(got), len(tc.want))
			}
			for i, sk := range got {
				if sk == nil {
					t.Fatalf("ConsumedSketches[%d] is nil — the browser would nest a nil sketch", i)
				}
				if sk != tc.want[i] {
					t.Errorf("ConsumedSketches[%d] = %p, want %p", i, sk, tc.want[i])
				}
			}
		})
	}
}

// TestPartFeatureConsumedSketchesForwards: the PartFeature wrapper forwards to the wrapped
// feature when it is a SketchConsumer and reports nil for a kind that consumes no sketch
// (a dress-up), so the browser can call it uniformly on every tree node.
func TestPartFeatureConsumedSketchesForwards(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	sk := consumerSketch()
	pf := fs.Add(&ExtrudeFeature{def: &ExtrudeDefinition{Sketch: sk}})
	if got := pf.ConsumedSketches(); len(got) != 1 || got[0] != sk {
		t.Errorf("PartFeature.ConsumedSketches = %v, want the wrapped extrude's sketch", got)
	}
	fillet := fs.Add(&FilletFeature{def: &FilletDefinition{}})
	if got := fillet.ConsumedSketches(); got != nil {
		t.Errorf("a non-consumer (fillet) PartFeature.ConsumedSketches = %v, want nil", got)
	}
}
