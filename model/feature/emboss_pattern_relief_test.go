// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// TestEmbossFromPlanePatternReplicatesTheReliefCut is the #2066 regression. A from-plane emboss
// applies TWO booleans — a raise in front of the sketch plane and a relief cut behind it — but the
// pattern machinery only ever recovered one tool per source, so a patterned copy kept the raise and
// dropped the relief cut.
//
// The profile sits fully INSIDE the block, so the raise (a prism already buried in solid material)
// is a no-op and ONLY the relief cut leaves a mark: a pocket behind the plane. A copy that missing
// the relief therefore has no pocket — solid where the seed is hollow — which is exactly what this
// test measures. Before the fix the copy's pocket point is solid; after it, both pockets are empty.
func TestEmbossFromPlanePatternReplicatesTheReliefCut(t *testing.T) {
	fs := NewPartFeatures(nil)
	NewExtrudeFeatures(fs).AddExtrude(squareSketch(10), []int{0}, ops.NewBody,
		Extent{Type: DistanceExtent, Direction: SymmetricDir, Distance: func() float64 { return 4 }}, 0)
	emb := NewEmbossFeatures(fs).Add(rectSketchOn(sketch.XYPlane(), 1, 1, 3, 3),
		[]int{0}, func() float64 { return 1 }, EmbossEngraveFromPlane, 0)
	// Two occurrences 4 cm apart in X: the seed's pocket at x∈[1,3], the copy's at x∈[5,7].
	NewPatternFeatures(fs).AddRectangular([]ID{emb.ID()},
		func() int { return 2 }, func() int { return 1 }, math.V3(4, 0, 0), math.Vector3{})
	fs.Recompute()

	body := fs.Result()[0]
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("patterned from-plane emboss is not a valid solid: %+v", r)
	}
	// The seed's relief pocket (behind the plane, inside the profile) is hollow.
	if p := math.P3(2, 2, -0.5); ops.PointInsideBody(body, p) {
		t.Errorf("seed relief pocket %v is solid, want the relief cut to have emptied it", p)
	}
	// The COPY's relief pocket must ALSO be hollow — the pattern must replicate the relief cut, not
	// only the raise (#2066). This is the assertion that fails before the fix.
	if p := math.P3(6, 2, -0.5); ops.PointInsideBody(body, p) {
		t.Errorf("patterned copy relief pocket %v is solid — the pattern replicated the raise but "+
			"dropped the relief cut (#2066)", p)
	}
	// A point behind the plane but outside both pockets stays solid — the cut is local, not global.
	if p := math.P3(8, 8, -0.5); !ops.PointInsideBody(body, p) {
		t.Errorf("%v lies outside every emboss profile and must stay solid", p)
	}
}

// TestEmbossFromPlaneMirrorReplicatesTheReliefCut is the mirror half of #2066 (the issue covers
// pattern AND mirror; both replicate through the same resolveGroup). A mirror of an interior
// from-plane emboss must reflect its relief cut, not only its raise.
func TestEmbossFromPlaneMirrorReplicatesTheReliefCut(t *testing.T) {
	fs := NewPartFeatures(nil)
	NewExtrudeFeatures(fs).AddExtrude(squareSketch(10), []int{0}, ops.NewBody,
		Extent{Type: DistanceExtent, Direction: SymmetricDir, Distance: func() float64 { return 4 }}, 0)
	emb := NewEmbossFeatures(fs).Add(rectSketchOn(sketch.XYPlane(), 1, 1, 3, 3),
		[]int{0}, func() float64 { return 1 }, EmbossEngraveFromPlane, 0)
	// Mirror across x=5: the seed pocket at x∈[1,3] reflects to x∈[7,9].
	NewPatternFeatures(fs).AddMirror([]ID{emb.ID()}, nil, math.P3(5, 0, 0), math.V3(1, 0, 0))
	fs.Recompute()

	body := fs.Result()[0]
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("mirrored from-plane emboss is not a valid solid: %+v", r)
	}
	if p := math.P3(2, 2, -0.5); ops.PointInsideBody(body, p) {
		t.Errorf("seed relief pocket %v is solid, want emptied", p)
	}
	if p := math.P3(8, 2, -0.5); ops.PointInsideBody(body, p) {
		t.Errorf("mirrored relief pocket %v is solid — the mirror dropped the relief cut (#2066)", p)
	}
}
