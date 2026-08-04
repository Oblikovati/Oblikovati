// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"testing"

	"oblikovati.org/math"
)

// A pattern or mirror copy must carry the source's format, or a styled sketch loses its
// formatting the moment it is patterned.
func TestFormatCarriesAcrossInSketchClone(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	src := s.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(10, 0))
	s.SetEntityFormat(src.EntityID(), redFormat())

	clones := s.cloneEntities([]Entity{src}, translation(math.V2(0, 5)))
	if len(clones) != 1 {
		t.Fatalf("clones = %d, want 1", len(clones))
	}
	got, ok := s.EntityFormat(clones[0].EntityID())
	if !ok || got != redFormat() {
		t.Errorf("cloned format = %+v ok=%v, want the source's", got, ok)
	}
}

// A cross-sketch copy reads the originals' formats from the SOURCE sketch — the target has never
// seen them, and format is the one part of an entity's state that is not reachable through the
// entity pointer.
func TestFormatCarriesAcrossSketches(t *testing.T) {
	sketches := NewSketches()
	source := sketches.Add(XYPlane())
	target := sketches.Add(XYPlane())
	src := source.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(10, 0))
	source.SetEntityFormat(src.EntityID(), redFormat())

	clones, _ := target.CopyEntitiesWithConstraints(source, []Entity{src}, math.V2(50, 0))
	if len(clones) != 1 {
		t.Fatalf("clones = %d, want 1", len(clones))
	}
	got, ok := target.EntityFormat(clones[0].EntityID())
	if !ok || got != redFormat() {
		t.Errorf("copied format = %+v ok=%v, want the source's", got, ok)
	}
}

// Cloning unstyled geometry must not manufacture format entries.
func TestCloningDefaultGeometryStoresNoFormat(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	src := s.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(10, 0))
	s.cloneEntities([]Entity{src}, translation(math.V2(0, 5)))
	if n := s.EntityFormatCount(); n != 0 {
		t.Errorf("format entries = %d, want 0", n)
	}
}
