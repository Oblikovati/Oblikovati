// SPDX-License-Identifier: GPL-2.0-only

package depend

import "testing"

func TestIntersectsReportsSharedKey(t *testing.T) {
	footprint := []Key{{ParameterKey, 1}, {ParameterKey, 2}}
	changed := NewSet([]Key{{ParameterKey, 2}, {ParameterKey, 9}})
	if !Intersects(footprint, changed) {
		t.Errorf("Intersects = false, want true (key {Parameter,2} is in both)")
	}
}

func TestIntersectsDisjointIsFalse(t *testing.T) {
	footprint := []Key{{ParameterKey, 1}}
	changed := NewSet([]Key{{ParameterKey, 2}})
	if Intersects(footprint, changed) {
		t.Errorf("Intersects = true, want false (disjoint)")
	}
}

// A ParameterKey and an ExternalGeometryKey that share the numeric ID must NOT be treated as
// the same dependency — the Kind discriminates. This is the property that lets the two kinds
// share the uint64 ID space without aliasing (ADR-0044).
func TestKindDiscriminatesEqualID(t *testing.T) {
	footprint := []Key{{ParameterKey, 7}}
	changed := NewSet([]Key{{ExternalGeometryKey, 7}})
	if Intersects(footprint, changed) {
		t.Errorf("Intersects = true, want false (same ID 7 but different Kind must not collide)")
	}
}

// The foundation's load-bearing promise (ADR-0044): the attribution is kind-agnostic, so a
// future external-geometry change flows through the SAME Intersects an adaptive reference will
// rely on, with no new machinery. This pins that an ExternalGeometryKey is a first-class match.
func TestExternalGeometryKeyMatchesThroughSameMachinery(t *testing.T) {
	footprint := []Key{{ParameterKey, 1}, {ExternalGeometryKey, 42}}
	changed := NewSet([]Key{{ExternalGeometryKey, 42}})
	if !Intersects(footprint, changed) {
		t.Errorf("Intersects = false, want true (external-geometry key must match like any other)")
	}
}

func TestNewSetHasAndEmpty(t *testing.T) {
	if !NewSet(nil).Empty() {
		t.Errorf("NewSet(nil).Empty() = false, want true")
	}
	s := NewSet([]Key{{ParameterKey, 3}})
	if s.Empty() {
		t.Errorf("non-empty set reports Empty()")
	}
	if !s.Has(Key{ParameterKey, 3}) || s.Has(Key{ParameterKey, 4}) {
		t.Errorf("Has wrong: have {Parameter,3}, want !{Parameter,4}")
	}
}
