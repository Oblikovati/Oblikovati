// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/identity"
)

// docGUID is a stable document namespace for deriving persistent reference keys in tests.
const docGUID = "11111111-2222-3333-4444-555555555555"

// buildKeyedSketch makes a sketch with a line, a circle, and a standalone point, returning
// the collection and the three entities for id comparison.
func buildKeyedSketch(t *testing.T) (*Sketches, *Line, *Circle, *Point) {
	t.Helper()
	sc := NewSketches()
	s := sc.Add(XYPlane())
	a := s.NewPoint(math.P2(0, 0))
	b := s.NewPoint(math.P2(4, 0))
	line := s.Lines().Add(a, b)
	center := s.NewPoint(math.P2(2, 3))
	circle := s.Circles().Add(center, 1.5)
	pt := s.Points().Add(math.P2(5, 5))
	return sc, line, circle, pt
}

// entityKeys derives the persistent reference key of each entity (#153). It fails the test
// on a derivation error so callers can compare keys directly.
func entityKeys(t *testing.T, ents ...Entity) []string {
	t.Helper()
	out := make([]string, len(ents))
	for i, e := range ents {
		k, err := identity.SketchEntityKey(docGUID, uint64(e.EntityID()))
		if err != nil {
			t.Fatalf("derive key for entity %d: %v", i, err)
		}
		out[i] = k
	}
	return out
}

// TestEntityIDsSurviveRoundTrip is the #153 foundation: a sketch entity's local id — and
// therefore its document-derived persistent reference key — must be identical after a
// save/load cycle. Before this change the id was re-minted on load, so any durable
// reference an add-in held went stale.
func TestEntityIDsSurviveRoundTrip(t *testing.T) {
	sc, line, circle, pt := buildKeyedSketch(t)
	wantIDs := []ID{line.EntityID(), circle.EntityID(), pt.EntityID()}
	wantKeys := entityKeys(t, line, circle, pt)

	out := roundTrip(t, sc)
	gotLine, gotCircle, gotPoint := firstOfEach(t, out)

	gotIDs := []ID{gotLine.EntityID(), gotCircle.EntityID(), gotPoint.EntityID()}
	for i := range wantIDs {
		if gotIDs[i] != wantIDs[i] {
			t.Errorf("entity %d id = %d after round trip, want %d (re-minted)", i, gotIDs[i], wantIDs[i])
		}
	}
	gotKeys := entityKeys(t, gotLine, gotCircle, gotPoint)
	for i := range wantKeys {
		if gotKeys[i] != wantKeys[i] {
			t.Errorf("entity %d persistent key changed: %q -> %q", i, wantKeys[i], gotKeys[i])
		}
	}
}

// TestSketchIDSurvivesRoundTrip: the sketch's own local id (and its derived key) survive
// the round trip, so a reference to "this sketch" stays valid (#153).
func TestSketchIDSurvivesRoundTrip(t *testing.T) {
	sc, _, _, _ := buildKeyedSketch(t)
	want := sc.Item(0).ID()
	if got := roundTrip(t, sc).ID(); got != want {
		t.Errorf("sketch id = %d after round trip, want %d", got, want)
	}
}

// TestNewEntityAfterRestoreDoesNotCollide: an entity created after a restore gets an id
// past every restored id, so it never collides with a pinned (verbatim) id. This guards
// the raiseIDSeq clock-raise.
func TestNewEntityAfterRestoreDoesNotCollide(t *testing.T) {
	sc, _, _, _ := buildKeyedSketch(t)
	data, err := sc.MarshalRecipe()
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	fresh := NewSketches()
	if err := fresh.ApplyRecipe(data); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	restored := fresh.Item(0)
	maxRestored := maxEntityID(restored)

	added := restored.Lines().Add(restored.NewPoint(math.P2(9, 9)), restored.NewPoint(math.P2(9, 8)))
	if added.EntityID() <= maxRestored {
		t.Errorf("new entity id %d not past max restored id %d (collision risk)", added.EntityID(), maxRestored)
	}
}

// TestIDsStableAcrossUnrelatedEdit: editing a sketch (adding geometry) must not perturb
// the ids/keys of existing entities, so references to them survive an edit (#153).
func TestIDsStableAcrossUnrelatedEdit(t *testing.T) {
	sc, line, circle, pt := buildKeyedSketch(t)
	wantKeys := entityKeys(t, line, circle, pt)

	// An unrelated edit: add another line, then round-trip again.
	s := sc.Item(0)
	s.Lines().Add(s.NewPoint(math.P2(1, 1)), s.NewPoint(math.P2(2, 2)))

	out := roundTrip(t, sc)
	gotLine, gotCircle, gotPoint := firstOfEach(t, out)
	gotKeys := entityKeys(t, gotLine, gotCircle, gotPoint)
	for i := range wantKeys {
		if gotKeys[i] != wantKeys[i] {
			t.Errorf("entity %d key changed after an unrelated edit: %q -> %q", i, wantKeys[i], gotKeys[i])
		}
	}
}

// firstOfEach returns the first line, circle, and standalone point of a restored sketch.
func firstOfEach(t *testing.T, s *Sketch) (*Line, *Circle, *Point) {
	t.Helper()
	var line *Line
	var circle *Circle
	for _, e := range s.Entities() {
		switch v := e.(type) {
		case *Line:
			if line == nil {
				line = v
			}
		case *Circle:
			circle = v
		}
	}
	pts := s.Points().items
	if line == nil || circle == nil || len(pts) == 0 {
		t.Fatalf("restored sketch missing entities: line=%v circle=%v points=%d", line, circle, len(pts))
	}
	return line, circle, pts[0]
}

// maxEntityID returns the largest local id among a sketch's points and entities.
func maxEntityID(s *Sketch) ID {
	var max ID
	for _, p := range s.AllPoints() {
		if p.EntityID() > max {
			max = p.EntityID()
		}
	}
	for _, e := range s.Entities() {
		if e.EntityID() > max {
			max = e.EntityID()
		}
	}
	return max
}
