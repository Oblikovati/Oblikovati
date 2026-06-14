// SPDX-License-Identifier: GPL-2.0-only

package occurrence

import (
	"testing"

	"oblikovati.org/math"
)

// TestMirrorComponentsAreCorrectlyHanded is the PBI-122 mirror acceptance: a mirror is
// orientation-reversed (the reflection flips handedness), so a chiral component comes
// out opposite-handed.
func TestMirrorComponentsAreCorrectlyHanded(t *testing.T) {
	asm := NewOccurrences()
	src := asm.AddByComponentDefinition("part:1", unitComponent(), math.Translation4(math.V3(3, 0, 0)))
	mirrors := MirrorComponents(asm, []*Occurrence{src}, math.P3(0, 0, 0), unitX(t)) // across the x=0 plane
	if len(mirrors) != 1 {
		t.Fatalf("mirrored %d, want 1", len(mirrors))
	}
	m := mirrors[0]
	if det := m.Transform().Determinant(); det >= 0 {
		t.Errorf("mirror placement determinant = %g, want < 0 (opposite hand)", det)
	}
	if got := m.Transform().TransformPoint(math.P3(0, 0, 0)); got != (math.P3(-3, 0, 0)) {
		t.Errorf("mirrored placement = %v, want {-3 0 0}", got)
	}
	if m.Definition() != src.Definition() {
		t.Error("mirror should share the source definition (handed by its transform)")
	}
	if asm.Count() != 2 {
		t.Errorf("assembly count = %d, want 2 (source + mirror)", asm.Count())
	}
}

func TestCopyComponentsAreIndependentInstances(t *testing.T) {
	asm := NewOccurrences()
	src := asm.AddByComponentDefinition("part:1", unitComponent(), math.Translation4(math.V3(2, 0, 0)))
	copies := CopyComponents(asm, []*Occurrence{src})
	if len(copies) != 1 || copies[0] == src || copies[0].ID() == src.ID() {
		t.Fatalf("copy = %v, want one new instance with its own id", copies)
	}
	if c := copies[0]; c.Definition() != src.Definition() || c.Transform() != src.Transform() {
		t.Error("copy should share the source's definition and placement")
	}
}

// TestReplicasInheritComponentName is the #765 regression: a copy/mirror/pattern element must
// carry the source's componentName, so it persists and re-binds on reopen exactly like the source
// (the assembly recipe drops a nameless, in-memory-only placement, which also made replicas
// un-undoable). Source placed with a component name; every replica must keep it.
func TestReplicasInheritComponentName(t *testing.T) {
	asm := NewOccurrences()
	src := asm.AddByComponentName("widget:1", unitComponent(), "widget.obk", math.Identity4())

	copies := CopyComponents(asm, []*Occurrence{src})
	if got := copies[0].ComponentName(); got != "widget.obk" {
		t.Errorf("copy componentName = %q, want widget.obk", got)
	}
	mirrors := MirrorComponents(asm, []*Occurrence{src}, math.P3(0, 0, 0), unitX(t))
	if got := mirrors[0].ComponentName(); got != "widget.obk" {
		t.Errorf("mirror componentName = %q, want widget.obk", got)
	}
	pat := NewOccurrencePattern(src.Definition(), src.Transform(), RectangularArrangement{
		Dir1: unitX(t), Spacing1: 2, Count1: 2, Dir2: unitX(t), Spacing2: 0, Count2: 1,
	})
	elems := PatternComponents(asm, src, pat)
	if len(elems) != 1 {
		t.Fatalf("2×1 pattern placed %d occurrences beyond the seed, want 1", len(elems))
	}
	if got := elems[0].ComponentName(); got != "widget.obk" {
		t.Errorf("pattern element componentName = %q, want widget.obk", got)
	}
}

// TestSubstituteSwapsInSimplifiedRep is the PBI-122 substitution acceptance: the
// detailed sources are suppressed and a simplified representation stands in for them.
func TestSubstituteSwapsInSimplifiedRep(t *testing.T) {
	asm := NewOccurrences()
	a := asm.AddByComponentDefinition("a:1", unitComponent(), math.Identity4())
	b := asm.AddByComponentDefinition("b:1", unitComponent(), math.Translation4(math.V3(5, 0, 0)))
	simplified := &mutableComponent{box: math.NewBox(math.P3(0, 0, 0), math.P3(6, 1, 1))}

	sub := Substitute(asm, []*Occurrence{a, b}, "shrinkwrap:1", simplified, math.Identity4())
	if !sub.IsSubstitute() {
		t.Error("substitute occurrence should be flagged IsSubstitute")
	}
	if sub.Definition() != simplified {
		t.Error("substitute should reference the simplified definition")
	}
	if !a.Suppressed() || !b.Suppressed() {
		t.Error("substitution should suppress the detailed sources")
	}
	if got := asm.RangeBox().Max; got != (math.P3(6, 1, 1)) {
		t.Errorf("assembly box = %v, want the simplified rep {6 1 1}", got)
	}
}
