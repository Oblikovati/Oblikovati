// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/math"
)

// TestBossCapturesFaceAnchorOnAuthor pins the authoring-seam half of the face geometric tier
// (#1579): authoring a boss against an already-recomputed tip records the placement face's
// centroid, so the geometric recovery tier has a witness without depending on the creation path.
func TestBossCapturesFaceAnchorOnAuthor(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	NewBaseFeatures(fs).AddBase(prismBody())
	fs.Recompute() // a tip body must exist for capture to resolve the face
	top := fs.Result()[0].Faces()[0].ReferenceKey()

	boss := NewBossFeatures(fs).Add(top, func() float64 { return 1 }, func() float64 { return 1 })
	anchors := boss.Definition().(*BossFeature).Definition().FaceAnchors
	if len(anchors) != 1 {
		t.Fatalf("boss should capture one face anchor on author, got %v", anchors)
	}
	if _, ok := anchors[string(top)]; !ok {
		t.Errorf("captured anchor is not keyed by the placement face key")
	}
}

// TestNoAnchorCaptureBeforeRecompute pins that a batch build (author before the first recompute,
// when there is no tip body) skips capture and degrades to ancestral-only recovery — never panics.
func TestNoAnchorCaptureBeforeRecompute(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	NewBaseFeatures(fs).AddBase(prismBody()) // no Recompute: tip is empty
	boss := NewBossFeatures(fs).Add([]byte("some-face"), func() float64 { return 1 }, func() float64 { return 1 })
	if a := boss.Definition().(*BossFeature).Definition().FaceAnchors; a != nil {
		t.Errorf("capture before the first recompute should store nil anchors, got %v", a)
	}
}

// TestFaceAnchorsCodecRoundTrip pins the serialized form: an authored face-anchor map survives
// encode→decode unchanged (the persistence half of the geometric tier).
func TestFaceAnchorsCodecRoundTrip(t *testing.T) {
	t.Parallel()
	key := faceKeyFor(7)
	anchors := map[string]math.Point3{string(key): math.P3(1.25, -2.5, 3)}

	back, err := decodeFaceAnchors(encodeFaceAnchors(anchors))
	if err != nil {
		t.Fatalf("decodeFaceAnchors: %v", err)
	}
	if len(back) != 1 {
		t.Fatalf("round trip lost the anchor: got %v", back)
	}
	if p, ok := back[string(key)]; !ok || !p.IsEqualTo(math.P3(1.25, -2.5, 3), 1e-9) {
		t.Errorf("anchor after round trip = %v (present=%v), want (1.25,-2.5,3)", p, ok)
	}
	if encodeFaceAnchors(nil) != nil {
		t.Error("an empty anchor map must encode to nil (omitempty), not an empty slice")
	}
}

// TestBossFaceAnchorsSurviveRecipeRoundTrip is the persistence guarantee: a boss whose definition
// carries a face anchor round-trips through MarshalRecipe/ApplyRecipe with the centroid intact, and
// the restore path (addBoss) does NOT recapture — it preserves the persisted anchor.
func TestBossFaceAnchorsSurviveRecipeRoundTrip(t *testing.T) {
	t.Parallel()
	key := []byte("face-2")
	want := math.P3(2, -1, 0.5)
	fs := NewPartFeatures(nil)
	NewBossFeatures(fs).addBoss(&BossDefinition{
		PlacementFaceKey: key, Diameter: func() float64 { return 8 }, Height: func() float64 { return 4 },
		FaceAnchors: map[string]math.Point3{string(key): want},
	})

	data, err := fs.MarshalRecipe(oneSketch{})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	fresh := NewPartFeatures(nil)
	if err := fresh.ApplyRecipe(data, oneSketch{}, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	restored := fresh.Item(0).Definition().(*BossFeature).Definition().FaceAnchors
	if len(restored) != 1 {
		t.Fatalf("face anchors after recipe round trip = %v, want one entry", restored)
	}
	if got, ok := restored[string(key)]; !ok || !got.IsEqualTo(want, 1e-9) {
		t.Errorf("restored anchor = %v (present=%v), want %v", got, ok, want)
	}
}
