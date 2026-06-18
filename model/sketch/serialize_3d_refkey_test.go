// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/identity"
)

// roundTrip3D serializes a 3D sketch collection and rebuilds it in a fresh collection,
// returning the reopened first sketch — the real save→restore path the part recipe uses.
func roundTrip3D(t *testing.T, sc *Sketches3D) *Sketch3D {
	t.Helper()
	data, err := sc.MarshalRecipe3D()
	if err != nil {
		t.Fatalf("MarshalRecipe3D: %v", err)
	}
	fresh := NewSketches3D()
	if err := fresh.ApplyRecipe3D(data); err != nil {
		t.Fatalf("ApplyRecipe3D: %v", err)
	}
	if fresh.Count() != sc.Count() {
		t.Fatalf("3D sketch count after round trip = %d, want %d", fresh.Count(), sc.Count())
	}
	return fresh.Item(0)
}

// buildKeyed3DSketch makes a 3D sketch with a line, a circle, and a standalone point.
func buildKeyed3DSketch(t *testing.T) (*Sketches3D, *Line3D, *Circle3D, *Point3D) {
	t.Helper()
	sc := NewSketches3D()
	s := sc.Add()
	line := s.AddLine3D(math.P3(0, 0, 0), math.P3(4, 0, 0))
	axis, err := math.NewUnitVector3(0, 0, 1)
	if err != nil {
		t.Fatalf("axis: %v", err)
	}
	circle := s.AddCircle3D(math.P3(2, 3, 0), axis, 1.5)
	pt := s.AddPoint3D(math.P3(5, 5, 5))
	return sc, line, circle, pt
}

// entity3DKeys derives the persistent reference key of each 3D entity (#153).
func entity3DKeys(t *testing.T, ents ...Entity) []string {
	t.Helper()
	out := make([]string, len(ents))
	for i, e := range ents {
		k, err := identity.SketchEntityKey(docGUID, uint64(e.EntityID()))
		if err != nil {
			t.Fatalf("derive 3D key %d: %v", i, err)
		}
		out[i] = k
	}
	return out
}

// firstOfEach3D returns the first line, circle, and standalone point of a restored 3D sketch.
func firstOfEach3D(t *testing.T, s *Sketch3D) (*Line3D, *Circle3D, *Point3D) {
	t.Helper()
	var line *Line3D
	var circle *Circle3D
	var pt *Point3D
	for _, e := range s.Entities() {
		switch v := e.(type) {
		case *Line3D:
			line = v
		case *Circle3D:
			circle = v
		case *Point3D:
			pt = v
		}
	}
	if line == nil || circle == nil || pt == nil {
		t.Fatalf("restored 3D sketch missing entities: line=%v circle=%v point=%v", line, circle, pt)
	}
	return line, circle, pt
}

// TestEntity3DIDsSurviveRoundTrip is the #153 fast-follow for 3D sketches: a 3D entity's
// local id — and its derived persistent reference key — must be identical after a save/load
// cycle (previously the id was re-minted on load, like the 2D case fixed in the foundation).
func TestEntity3DIDsSurviveRoundTrip(t *testing.T) {
	sc, line, circle, pt := buildKeyed3DSketch(t)
	wantIDs := []ID{line.EntityID(), circle.EntityID(), pt.EntityID()}
	wantKeys := entity3DKeys(t, line, circle, pt)

	out := roundTrip3D(t, sc)
	gotLine, gotCircle, gotPoint := firstOfEach3D(t, out)
	gotIDs := []ID{gotLine.EntityID(), gotCircle.EntityID(), gotPoint.EntityID()}
	for i := range wantIDs {
		if gotIDs[i] != wantIDs[i] {
			t.Errorf("3D entity %d id = %d after round trip, want %d (re-minted)", i, gotIDs[i], wantIDs[i])
		}
	}
	gotKeys := entity3DKeys(t, gotLine, gotCircle, gotPoint)
	for i := range wantKeys {
		if gotKeys[i] != wantKeys[i] {
			t.Errorf("3D entity %d persistent key changed: %q -> %q", i, wantKeys[i], gotKeys[i])
		}
	}
}

// TestSketch3DIDSurvivesRoundTrip: the 3D sketch's own local id survives the round trip.
func TestSketch3DIDSurvivesRoundTrip(t *testing.T) {
	sc, _, _, _ := buildKeyed3DSketch(t)
	want := sc.Item(0).ID()
	if got := roundTrip3D(t, sc).ID(); got != want {
		t.Errorf("3D sketch id = %d after round trip, want %d", got, want)
	}
}

// TestSketch3DByIDReindexedAfterRestore: restoring pins ids verbatim AND re-keys the byID
// index, so EntityByID resolves the restored entity by its persisted id (a regression guard
// for the 3D-specific id index the 2D path does not have).
func TestSketch3DByIDReindexedAfterRestore(t *testing.T) {
	sc, line, _, pt := buildKeyed3DSketch(t)
	out := roundTrip3D(t, sc)

	if e, ok := out.EntityByID(line.EntityID()); !ok || e.EntityID() != line.EntityID() {
		t.Errorf("EntityByID(line) after restore failed: ok=%v", ok)
	}
	if e, ok := out.EntityByID(pt.EntityID()); !ok || e.EntityID() != pt.EntityID() {
		t.Errorf("EntityByID(standalone point) after restore failed: ok=%v", ok)
	}
}

// TestNew3DEntityAfterRestoreDoesNotCollide: an entity created after a 3D restore gets an id
// past every restored id (guards the raiseIDSeq clock-raise).
func TestNew3DEntityAfterRestoreDoesNotCollide(t *testing.T) {
	sc, _, _, _ := buildKeyed3DSketch(t)
	out := roundTrip3D(t, sc)
	var maxRestored ID
	for _, e := range out.Entities() {
		if e.EntityID() > maxRestored {
			maxRestored = e.EntityID()
		}
	}
	added := out.AddLine3D(math.P3(9, 9, 9), math.P3(9, 8, 9))
	if added.EntityID() <= maxRestored {
		t.Errorf("new 3D entity id %d not past max restored id %d", added.EntityID(), maxRestored)
	}
}
