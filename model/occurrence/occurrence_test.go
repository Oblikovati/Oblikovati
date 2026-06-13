// SPDX-License-Identifier: GPL-2.0-only

package occurrence

import (
	"testing"

	"oblikovati.org/math"
)

// fakeComponent is a named test double for a component definition: it reports a fixed
// local range box, standing in for a real part/assembly definition (which the
// reference graph supplies from M11-F02).
type fakeComponent struct{ box math.Box }

func (f fakeComponent) RangeBox() math.Box { return f.box }

// unitComponent is a 1×1×1 box at the origin.
func unitComponent() fakeComponent {
	return fakeComponent{box: math.NewBox(math.P3(0, 0, 0), math.P3(1, 1, 1))}
}

func TestOccurrencesAddAssignsIdsAndCounts(t *testing.T) {
	occ := NewOccurrences()
	a := occ.Add("a:1", unitComponent(), math.Identity4())
	b := occ.Add("b:1", unitComponent(), math.Identity4())
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
	occ.Add("origin", unitComponent(), math.Identity4())
	occ.Add("moved", unitComponent(), math.Translation4(math.V3(10, 0, 0)))
	// One unit box at [0,1]³ and another at [10,11]×[0,1]×[0,1] → union [0,11]×[0,1]×[0,1].
	box := occ.RangeBox()
	if box.Min != (math.P3(0, 0, 0)) || box.Max != (math.P3(11, 1, 1)) {
		t.Errorf("assembly box = %v..%v, want {0 0 0}..{11 1 1}", box.Min, box.Max)
	}
}

func TestSuppressedOccurrenceLeavesNoTraceInBox(t *testing.T) {
	occ := NewOccurrences()
	occ.Add("kept", unitComponent(), math.Identity4())
	far := occ.Add("dropped", unitComponent(), math.Translation4(math.V3(100, 0, 0)))
	far.SetSuppressed(true)
	box := occ.RangeBox()
	if box.Min != (math.P3(0, 0, 0)) || box.Max != (math.P3(1, 1, 1)) {
		t.Errorf("box with suppressed occurrence = %v..%v, want just the kept unit box", box.Min, box.Max)
	}
}

func TestMutationsAdvanceRevision(t *testing.T) {
	occ := NewOccurrences()
	r0 := occ.Revision()
	o := occ.Add("x", unitComponent(), math.Identity4())
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
	stray := other.Add("stray", unitComponent(), math.Identity4())
	if occ.Remove(stray) {
		t.Error("Remove of an occurrence from another collection returned true")
	}
}

func TestEmptyAssemblyHasEmptyBox(t *testing.T) {
	if box := NewOccurrences().RangeBox(); !box.IsEmpty() {
		t.Errorf("empty assembly box = %v..%v, want empty", box.Min, box.Max)
	}
}
