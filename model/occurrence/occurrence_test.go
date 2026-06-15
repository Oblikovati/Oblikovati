// SPDX-License-Identifier: GPL-2.0-only

package occurrence

import (
	"testing"

	"oblikovati.org/math"
)

// fakeComponent is a named test double for a leaf component definition (a part): it
// reports a fixed local range box and is not a Composite, so its occurrences have no
// sub-occurrences.
type fakeComponent struct{ box math.Box }

func (f fakeComponent) RangeBox() math.Box { return f.box }

// unitComponent is a 1×1×1 leaf component at the origin.
func unitComponent() fakeComponent {
	return fakeComponent{box: math.NewBox(math.P3(0, 0, 0), math.P3(1, 1, 1))}
}

// mutableComponent is a named Definition with a pointer identity and a changeable box,
// to prove the flyweight: editing one shared definition updates every occurrence of it.
type mutableComponent struct{ box math.Box }

func (m *mutableComponent) RangeBox() math.Box { return m.box }

// fakeAssembly is a named Composite test double: a definition that owns a nested
// occurrence collection, standing in for an assembly definition so nesting can be
// exercised without model/compdef.
type fakeAssembly struct {
	box      math.Box
	children *Occurrences
}

func (f *fakeAssembly) RangeBox() math.Box        { return f.box }
func (f *fakeAssembly) Occurrences() *Occurrences { return f.children }

var _ Composite = (*fakeAssembly)(nil)

func TestOccurrencesAddAssignsIdsAndCounts(t *testing.T) {
	occ := NewOccurrences()
	a := occ.AddByComponentDefinition("a:1", unitComponent(), math.Identity4())
	b := occ.AddByComponentDefinition("b:1", unitComponent(), math.Identity4())
	if occ.Count() != 2 || a.ID() == b.ID() {
		t.Fatalf("Count=%d ids=(%d,%d), want 2 distinct", occ.Count(), a.ID(), b.ID())
	}
	got, ok := occ.ByID(a.ID())
	if !ok || got != a || occ.Item(0) != a {
		t.Errorf("lookup mismatch: ByID ok=%v got=%p item0=%p a=%p", ok, got, occ.Item(0), a)
	}
}

func TestOccurrenceRangeBoxPlacesByTransform(t *testing.T) {
	occ := NewOccurrences()
	occ.AddByComponentDefinition("origin", unitComponent(), math.Identity4())
	occ.AddByComponentDefinition("moved", unitComponent(), math.Translation4(math.V3(10, 0, 0)))
	// One unit box at [0,1]³ and another at [10,11]×[0,1]×[0,1] → union [0,11]×[0,1]×[0,1].
	box := occ.RangeBox()
	if box.Min != (math.P3(0, 0, 0)) || box.Max != (math.P3(11, 1, 1)) {
		t.Errorf("assembly box = %v..%v, want {0 0 0}..{11 1 1}", box.Min, box.Max)
	}
}

func TestSuppressedOccurrenceLeavesNoTraceInBox(t *testing.T) {
	occ := NewOccurrences()
	occ.AddByComponentDefinition("kept", unitComponent(), math.Identity4())
	far := occ.AddByComponentDefinition("dropped", unitComponent(), math.Translation4(math.V3(100, 0, 0)))
	far.SetSuppressed(true)
	box := occ.RangeBox()
	if box.Min != (math.P3(0, 0, 0)) || box.Max != (math.P3(1, 1, 1)) {
		t.Errorf("box with suppressed occurrence = %v..%v, want just the kept unit box", box.Min, box.Max)
	}
}

func TestMutationsAdvanceRevision(t *testing.T) {
	occ := NewOccurrences()
	r0 := occ.Revision()
	o := occ.AddByComponentDefinition("x", unitComponent(), math.Identity4())
	rAdd := occ.Revision()
	o.SetTransform(math.Translation4(math.V3(1, 0, 0)))
	rMove := occ.Revision()
	o.SetSuppressed(true)
	rSuppress := occ.Revision()
	o.SetSuppressed(true) // no-op: state unchanged, must not advance
	rNoop := occ.Revision()
	if r0 >= rAdd || rAdd >= rMove || rMove >= rSuppress {
		t.Errorf("revisions not strictly increasing: %d,%d,%d,%d", r0, rAdd, rMove, rSuppress)
	}
	if rNoop != rSuppress {
		t.Errorf("no-op SetSuppressed advanced revision %d→%d", rSuppress, rNoop)
	}
	if !occ.Remove(o) || occ.Revision() <= rNoop {
		t.Errorf("Remove should advance revision (was %d, now %d)", rNoop, occ.Revision())
	}
}

func TestRemoveUnknownOccurrenceIsNoop(t *testing.T) {
	occ := NewOccurrences()
	other := NewOccurrences()
	stray := other.AddByComponentDefinition("stray", unitComponent(), math.Identity4())
	if occ.Remove(stray) {
		t.Error("Remove of an occurrence from another collection returned true")
	}
}

func TestEmptyAssemblyHasEmptyBox(t *testing.T) {
	if box := NewOccurrences().RangeBox(); !box.IsEmpty() {
		t.Errorf("empty assembly box = %v..%v, want empty", box.Min, box.Max)
	}
}

// TestEditingSharedDefinitionUpdatesAllOccurrences is the PBI-118 flyweight: two
// placements of one definition share it, and editing the definition updates both
// without re-placing.
func TestEditingSharedDefinitionUpdatesAllOccurrences(t *testing.T) {
	occ := NewOccurrences()
	def := &mutableComponent{box: math.NewBox(math.P3(0, 0, 0), math.P3(1, 1, 1))}
	a := occ.AddByComponentDefinition("u:1", def, math.Identity4())
	b := occ.AddByComponentDefinition("u:2", def, math.Translation4(math.V3(10, 0, 0)))
	if a.Definition() != b.Definition() {
		t.Fatal("placements should share the same definition pointer (flyweight)")
	}
	def.box = math.NewBox(math.P3(0, 0, 0), math.P3(2, 2, 2)) // edit the shared definition
	if a.RangeBox().Max != (math.P3(2, 2, 2)) {
		t.Errorf("occurrence a box max = %v, want grown to {2 2 2}", a.RangeBox().Max)
	}
	if b.RangeBox().Min != (math.P3(10, 0, 0)) || b.RangeBox().Max != (math.P3(12, 2, 2)) {
		t.Errorf("occurrence b box = %v..%v, want {10 0 0}..{12 2 2}", b.RangeBox().Min, b.RangeBox().Max)
	}
}

func TestOccurrenceStateFlags(t *testing.T) {
	occ := NewOccurrences()
	o := occ.AddByComponentDefinition("u:1", unitComponent(), math.Identity4())
	if o.Grounded() || o.Adaptive() || o.Suppressed() {
		t.Error("new occurrence should be ungrounded, non-adaptive, unsuppressed")
	}
	rev := occ.Revision()
	o.SetGrounded(true)
	o.SetAdaptive(true)
	if !o.Grounded() || !o.Adaptive() {
		t.Error("grounded/adaptive flags did not stick")
	}
	if occ.Revision() != rev {
		t.Errorf("grounding/adaptivity bumped the geometry revision %d→%d (non-geometric, should not)", rev, occ.Revision())
	}
}

func TestReplaceSwapsDefinitionKeepingPlacement(t *testing.T) {
	occ := NewOccurrences()
	small := &mutableComponent{box: math.NewBox(math.P3(0, 0, 0), math.P3(1, 1, 1))}
	big := &mutableComponent{box: math.NewBox(math.P3(0, 0, 0), math.P3(3, 3, 3))}
	o := occ.AddByComponentDefinition("u:1", small, math.Translation4(math.V3(5, 0, 0)))
	id, name, tf := o.ID(), o.Name(), o.Transform()
	if !occ.Replace(o, big) {
		t.Fatal("Replace returned false for an owned occurrence")
	}
	if o.Definition() != Definition(big) || o.ID() != id || o.Name() != name || o.Transform() != tf {
		t.Error("Replace should swap the definition but keep id/name/transform")
	}
	if o.RangeBox().Max != (math.P3(8, 3, 3)) { // big [0,3]³ placed at +5x
		t.Errorf("replaced box max = %v, want {8 3 3}", o.RangeBox().Max)
	}
	if occ.Replace(NewOccurrences().AddByComponentDefinition("x", small, math.Identity4()), big) {
		t.Error("Replace of a foreign occurrence returned true")
	}
}

// TestNestedOccurrencesResolveByPath is the PBI-119 nesting acceptance: a
// sub-assembly's instances are addressable by occurrence path, and the parent is a
// property of the path (the shared child has no single parent).
func TestNestedOccurrencesResolveByPath(t *testing.T) {
	// Sub-assembly B holding one pin.
	bChildren := NewOccurrences()
	pin := bChildren.AddByComponentDefinition("pin:1", unitComponent(), math.Identity4())
	b := &fakeAssembly{box: math.NewBox(math.P3(0, 0, 0), math.P3(2, 2, 2)), children: bChildren}

	// Top assembly A places B twice.
	a := NewOccurrences()
	g1 := a.AddByComponentDefinition("gearbox:1", b, math.Identity4())
	a.AddByComponentDefinition("gearbox:2", b, math.Translation4(math.V3(10, 0, 0)))

	if g1.SubOccurrences() != bChildren {
		t.Error("SubOccurrences should be the shared sub-assembly children (flyweight)")
	}
	got, ok := a.Resolve(OccurrencePath{"gearbox:1", "pin:1"})
	if !ok || got != pin {
		t.Errorf("Resolve([gearbox:1 pin:1]) ok=%v got=%p, want the pin %p", ok, got, pin)
	}
	// The same shared pin is reachable under gearbox:2, with gearbox:2 as its parent.
	parent, ok := a.ParentInPath(OccurrencePath{"gearbox:2", "pin:1"})
	if !ok || parent == nil || parent.Name() != "gearbox:2" {
		t.Errorf("ParentInPath([gearbox:2 pin:1]) = %v ok=%v, want gearbox:2", parent, ok)
	}
	// A top-level target resolves but has no parent.
	if p, ok := a.ParentInPath(OccurrencePath{"gearbox:1"}); !ok || p != nil {
		t.Errorf("ParentInPath([gearbox:1]) = %v ok=%v, want nil parent (top-level)", p, ok)
	}
}

func TestResolveRejectsBadPaths(t *testing.T) {
	a := NewOccurrences()
	a.AddByComponentDefinition("bolt:1", unitComponent(), math.Identity4()) // a leaf part
	cases := []OccurrencePath{
		{},                 // empty
		{"missing:1"},      // unknown name
		{"bolt:1", "deep"}, // descending into a leaf part
	}
	for _, p := range cases {
		if _, ok := a.Resolve(p); ok {
			t.Errorf("Resolve(%v) = ok, want failure", p)
		}
	}
}

// TestVisibilityFlagDefaultsVisible checks the M12-F04 display-visibility flag: a new
// occurrence is visible, and hiding/showing round-trips without touching suppression.
func TestVisibilityFlagDefaultsVisible(t *testing.T) {
	occs := NewOccurrences()
	o := occs.AddByComponentDefinition("widget:1", unitComponent(), math.Identity4())
	if !o.Visible() {
		t.Error("a new occurrence should be visible by default")
	}
	o.SetVisible(false)
	if o.Visible() || o.Suppressed() {
		t.Errorf("after SetVisible(false): Visible=%v Suppressed=%v, want false/false", o.Visible(), o.Suppressed())
	}
	o.SetVisible(true)
	if !o.Visible() {
		t.Error("after SetVisible(true) the occurrence should be visible again")
	}
}

// TestFlexibleFlag checks the M12-F06 flexible flag: only a sub-assembly occurrence can be
// flexible, and flexible is mutually exclusive with adaptive.
func TestFlexibleFlag(t *testing.T) {
	occs := NewOccurrences()

	// A leaf part cannot be flexible.
	leaf := occs.AddByComponentDefinition("part:1", unitComponent(), math.Identity4())
	leaf.SetFlexible(true)
	if leaf.Flexible() {
		t.Error("a leaf part occurrence should not become flexible")
	}

	// A sub-assembly occurrence can.
	sub := occs.AddByComponentDefinition("sub:1", &fakeAssembly{children: NewOccurrences()}, math.Identity4())
	sub.SetFlexible(true)
	if !sub.Flexible() {
		t.Error("a sub-assembly occurrence should become flexible")
	}

	// Flexible and adaptive are mutually exclusive.
	sub.SetAdaptive(true)
	if sub.Flexible() || !sub.Adaptive() {
		t.Errorf("after SetAdaptive: flexible=%v adaptive=%v, want false/true", sub.Flexible(), sub.Adaptive())
	}
	sub.SetFlexible(true)
	if !sub.Flexible() || sub.Adaptive() {
		t.Errorf("after SetFlexible: flexible=%v adaptive=%v, want true/false", sub.Flexible(), sub.Adaptive())
	}
}
