// SPDX-License-Identifier: GPL-2.0-only

package bom

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/occurrence"
)

// fakePart is a named leaf component: it satisfies occurrence.Definition (RangeBox)
// and bom.Component (part metadata). Always used by pointer, so BOM grouping keys on
// its identity (the flyweight).
type fakePart struct {
	num, desc string
	structure Structure
	props     map[string]string
}

func (p *fakePart) RangeBox() math.Box            { return math.NewBox(math.P3(0, 0, 0), math.P3(1, 1, 1)) }
func (p *fakePart) PartNumber() string            { return p.num }
func (p *fakePart) Description() string           { return p.desc }
func (p *fakePart) BOMStructure() Structure       { return p.structure }
func (p *fakePart) Properties() map[string]string { return p.props }

// fakeAssembly is a named sub-assembly component: a Composite (owns occurrences) that
// is also a bom.Component.
type fakeAssembly struct {
	fakePart
	children *occurrence.Occurrences
}

func (a *fakeAssembly) Occurrences() *occurrence.Occurrences { return a.children }

func newSub(num string, structure Structure) *fakeAssembly {
	a := &fakeAssembly{children: occurrence.NewOccurrences()}
	a.num, a.structure = num, structure
	return a
}

func rowNames(rows []*Row) []string {
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = r.PartNumber
	}
	return out
}

func TestPartsOnlyTotalsAcrossNestedAssemblies(t *testing.T) {
	bolt := &fakePart{num: "BOLT-1", structure: Normal}
	sub := newSub("SUB-1", Normal)
	sub.children.AddByComponentDefinition("bolt:1", bolt, math.Identity4())
	sub.children.AddByComponentDefinition("bolt:2", bolt, math.Translation4(math.V3(1, 0, 0)))

	top := occurrence.NewOccurrences()
	top.AddByComponentDefinition("sub:1", sub, math.Identity4())
	top.AddByComponentDefinition("sub:2", sub, math.Translation4(math.V3(10, 0, 0)))

	pv := New(top).PartsOnly()
	// The sub-assembly is traversed; 2 placements × 2 bolts = 4 bolts, the only part.
	if len(pv.Rows) != 1 {
		t.Fatalf("parts-only rows = %v, want just the bolt", rowNames(pv.Rows))
	}
	if pv.Rows[0].PartNumber != "BOLT-1" || pv.Rows[0].Quantity != 4 {
		t.Errorf("bolt row = %s qty %d, want BOLT-1 qty 4", pv.Rows[0].PartNumber, pv.Rows[0].Quantity)
	}
}

func TestStructuredViewNestsAndCountsPerParent(t *testing.T) {
	bolt := &fakePart{num: "BOLT-1", structure: Normal}
	sub := newSub("SUB-1", Normal)
	sub.children.AddByComponentDefinition("bolt:1", bolt, math.Identity4())
	sub.children.AddByComponentDefinition("bolt:2", bolt, math.Identity4())

	top := occurrence.NewOccurrences()
	top.AddByComponentDefinition("sub:1", sub, math.Identity4())
	top.AddByComponentDefinition("sub:2", sub, math.Identity4())

	sv := New(top).Structured()
	if len(sv.Rows) != 1 || sv.Rows[0].PartNumber != "SUB-1" || sv.Rows[0].Quantity != 2 {
		t.Fatalf("structured top = %v, want one SUB-1 qty 2", rowNames(sv.Rows))
	}
	children := sv.Rows[0].Children
	if len(children) != 1 || children[0].Quantity != 2 || children[0].ItemNumber != 1 {
		t.Errorf("sub children = %+v, want one BOLT row qty 2, item 1", children)
	}
}

// TestPhantomSubAssemblyCollapses is part of the PBI-123 acceptance: a phantom
// sub-assembly is not a row; its children are promoted to the parent.
func TestPhantomSubAssemblyCollapses(t *testing.T) {
	plate := &fakePart{num: "PLATE", structure: Normal}
	rig := newSub("RIG", Phantom)
	rig.children.AddByComponentDefinition("plate:1", plate, math.Identity4())

	top := occurrence.NewOccurrences()
	top.AddByComponentDefinition("rig:1", rig, math.Identity4())

	sv := New(top).Structured()
	if len(sv.Rows) != 1 || sv.Rows[0].PartNumber != "PLATE" {
		t.Fatalf("structured = %v, want the promoted PLATE (phantom collapsed out)", rowNames(sv.Rows))
	}
}

func TestPurchasedSubAssemblyCountsAsOne(t *testing.T) {
	screw := &fakePart{num: "SCREW", structure: Normal}
	motor := newSub("MOTOR", Purchased)
	motor.children.AddByComponentDefinition("screw:1", screw, math.Identity4())

	top := occurrence.NewOccurrences()
	top.AddByComponentDefinition("motor:1", motor, math.Identity4())

	pv := New(top).PartsOnly()
	if len(pv.Rows) != 1 || pv.Rows[0].PartNumber != "MOTOR" || pv.Rows[0].Quantity != 1 {
		t.Errorf("parts-only = %v, want one MOTOR qty 1 (purchased, not broken out)", rowNames(pv.Rows))
	}
}

func TestReferenceComponentExcludedFromPartsOnly(t *testing.T) {
	top := occurrence.NewOccurrences()
	top.AddByComponentDefinition("skel:1", &fakePart{num: "SKEL", structure: Reference}, math.Identity4())
	top.AddByComponentDefinition("real:1", &fakePart{num: "REAL", structure: Normal}, math.Identity4())

	pv := New(top).PartsOnly()
	if len(pv.Rows) != 1 || pv.Rows[0].PartNumber != "REAL" {
		t.Errorf("parts-only = %v, want only REAL (reference excluded)", rowNames(pv.Rows))
	}
}

// TestItemNumbersStableAndQuantityTracksSuppression is the rest of the PBI-123
// acceptance: quantities reflect the structure and item numbers are stable across an
// unrelated edit.
func TestItemNumbersStableAndQuantityTracksSuppression(t *testing.T) {
	a := &fakePart{num: "A", structure: Normal}
	b := &fakePart{num: "B", structure: Normal}
	top := occurrence.NewOccurrences()
	top.AddByComponentDefinition("a:1", a, math.Identity4())
	top.AddByComponentDefinition("a:2", a, math.Identity4())
	bOcc := top.AddByComponentDefinition("b:1", b, math.Identity4())

	first := New(top).PartsOnly()
	if first.Rows[0].PartNumber != "A" || first.Rows[0].ItemNumber != 1 || first.Rows[0].Quantity != 2 {
		t.Errorf("A row = %+v, want item 1 qty 2", first.Rows[0])
	}
	if first.Rows[1].PartNumber != "B" || first.Rows[1].ItemNumber != 2 {
		t.Errorf("B item number = %d, want 2", first.Rows[1].ItemNumber)
	}
	// Suppressing B drops it; A keeps its item number (stable).
	bOcc.SetSuppressed(true)
	second := New(top).PartsOnly()
	if len(second.Rows) != 1 || second.Rows[0].PartNumber != "A" || second.Rows[0].ItemNumber != 1 {
		t.Errorf("after suppressing B: %v, want A still item 1", rowNames(second.Rows))
	}
}
