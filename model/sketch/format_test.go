// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/math"
)

// redFormat is a format with every field overridden, for tests that only care that a format
// travels rather than what it holds.
func redFormat() EntityFormat {
	return EntityFormat{LineType: "dashed", Color: types.NewColor(255, 0, 0), LineWeight: 0.5}
}

func TestEntityFormatSetGetClear(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	l := s.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(10, 0))

	if _, ok := s.EntityFormat(l.EntityID()); ok {
		t.Fatal("a new entity must have no format — absence is Default")
	}
	s.SetEntityFormat(l.EntityID(), redFormat())
	got, ok := s.EntityFormat(l.EntityID())
	if !ok || got != redFormat() {
		t.Fatalf("format = %+v ok=%v, want the stored override", got, ok)
	}
	s.ClearEntityFormat(l.EntityID())
	if _, ok := s.EntityFormat(l.EntityID()); ok {
		t.Error("clearing must return the entity to Default")
	}
}

// A format that overrides nothing must not be stored: absence is the single representation of
// Default, so an empty entry would create a second one.
func TestSettingADefaultFormatStoresNothing(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	l := s.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(10, 0))
	s.SetEntityFormat(l.EntityID(), EntityFormat{})
	if n := s.EntityFormatCount(); n != 0 {
		t.Errorf("format entries = %d, want 0", n)
	}
}

// A deleted entity's format must go with it: otherwise it leaks, and a later entity reusing the
// id would silently inherit it.
func TestEntityFormatPrunedOnDelete(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	l := s.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(10, 0))
	id := l.EntityID()
	s.SetEntityFormat(id, redFormat())

	s.DeleteEntities([]Entity{l})
	if _, ok := s.EntityFormat(id); ok {
		t.Error("deleting an entity must drop its format")
	}
	if n := s.EntityFormatCount(); n != 0 {
		t.Errorf("format entries = %d, want 0", n)
	}
}

func TestCopyEntityFormat(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	a := s.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(10, 0))
	b := s.Lines().AddByTwoPoints(math.P2(0, 5), math.P2(10, 5))
	s.SetEntityFormat(a.EntityID(), redFormat())

	s.CopyEntityFormat(a.EntityID(), b.EntityID())
	got, ok := s.EntityFormat(b.EntityID())
	if !ok || got != redFormat() {
		t.Errorf("copied format = %+v ok=%v, want the source's", got, ok)
	}
}

// Copying from an unstyled entity must not create an entry — that would turn Default into an
// explicit empty override.
func TestCopyEntityFormatFromDefaultCreatesNothing(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	a := s.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(10, 0))
	b := s.Lines().AddByTwoPoints(math.P2(0, 5), math.P2(10, 5))

	s.CopyEntityFormat(a.EntityID(), b.EntityID())
	if n := s.EntityFormatCount(); n != 0 {
		t.Errorf("format entries = %d, want 0", n)
	}
}

// Each field is independently optional, so an entity can override its colour while inheriting
// its line type.
func TestEntityFormatFieldsAreIndependent(t *testing.T) {
	f := EntityFormat{Color: types.NewColor(0, 0, 255)}
	if f.IsDefault() {
		t.Error("a colour-only override is not Default")
	}
	if (EntityFormat{LineWeight: 0.25}).IsDefault() {
		t.Error("a weight-only override is not Default")
	}
	if !(EntityFormat{}).IsDefault() {
		t.Error("an empty format is Default")
	}
	// The zero Color must NOT read as an override: its Source is 0, which is not a member of
	// the enum, and treating it as set would make every empty format look styled.
	if !(EntityFormat{Color: types.Color{}}).IsDefault() {
		t.Error("the zero Color must not count as a colour override")
	}
}
