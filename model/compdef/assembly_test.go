// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/occurrence"
)

// fakeComponentSource is a named stand-in for a placed component's definition,
// reporting a fixed local box so the assembly's range box can be exercised without
// building real child geometry (real placement arrives in M11-F02).
type fakeComponentSource struct{ box math.Box }

func (f fakeComponentSource) RangeBox() math.Box { return f.box }

var _ occurrence.Definition = fakeComponentSource{}

// unitSource is a 1×1×1 component at its own origin.
func unitSource() fakeComponentSource {
	return fakeComponentSource{box: math.NewBox(math.P3(0, 0, 0), math.P3(1, 1, 1))}
}

func TestNewAssemblyIsEmptyAssemblyContent(t *testing.T) {
	t.Parallel()
	a := NewAssemblyComponentDefinition()
	if a.DocumentType() != doc.Assembly {
		t.Errorf("DocumentType = %v, want assembly", a.DocumentType())
	}
	if a.Occurrences().Count() != 0 || !a.RangeBox().IsEmpty() {
		t.Errorf("new assembly count=%d boxEmpty=%v, want 0 occurrences and an empty box",
			a.Occurrences().Count(), a.RangeBox().IsEmpty())
	}
}

func TestAssemblyRangeBoxUnionsOccurrencesAndVersionTracks(t *testing.T) {
	t.Parallel()
	a := NewAssemblyComponentDefinition()
	v0 := a.ModelGeometryVersion()
	a.Place("base", unitSource(), math.Identity4())
	a.Place("offset", unitSource(), math.Translation4(math.V3(4, 0, 0)))
	// [0,1]³ ∪ [4,5]×[0,1]×[0,1] = [0,5]×[0,1]×[0,1].
	box := a.RangeBox()
	if box.Min != (math.P3(0, 0, 0)) || box.Max != (math.P3(5, 1, 1)) {
		t.Errorf("assembly box = %v..%v, want {0 0 0}..{5 1 1}", box.Min, box.Max)
	}
	if a.ModelGeometryVersion() == v0 {
		t.Errorf("ModelGeometryVersion did not change after placing occurrences (still %s)", v0)
	}
}

func TestAddAssemblyInstallsAssemblyContentAndActivates(t *testing.T) {
	t.Parallel()
	ws := doc.NewWorkspace(nil, testContentFactories())
	d, err := AddAssembly(ws, "frame.oad", true)
	if err != nil {
		t.Fatalf("AddAssembly: %v", err)
	}
	if _, ok := d.Content().(*AssemblyComponentDefinition); !ok {
		t.Fatalf("content is %T, want *AssemblyComponentDefinition", d.Content())
	}
	if ws.ActiveDocument() != d || d.DocumentType() != doc.Assembly {
		t.Fatalf("AddAssembly active=%v type=%v, want active assembly", ws.ActiveDocument() == d, d.DocumentType())
	}
}

// TestAssemblyFactoryRegisteredForWorkspaceAdd proves the init() registration: a
// plain ws.Add(Assembly) (the path documents.create and the CLI use) yields the real
// content, not the identity-only doc stub.
func TestAssemblyFactoryRegisteredForWorkspaceAdd(t *testing.T) {
	t.Parallel()
	ws := doc.NewWorkspace(nil, testContentFactories())
	d, err := ws.Add(doc.Assembly, "sub.oad", true)
	if err != nil {
		t.Fatalf("ws.Add(assembly): %v", err)
	}
	if _, ok := d.Content().(*AssemblyComponentDefinition); !ok {
		t.Errorf("factory content = %T, want *AssemblyComponentDefinition", d.Content())
	}
}

// TestPlaceComponentSharesDefinitionAcrossPlacements is the PBI-118 acceptance at the
// document level: two placements of one part document share its definition (the
// flyweight), so editing the part would update both.
func TestPlaceComponentSharesDefinitionAcrossPlacements(t *testing.T) {
	t.Parallel()
	ws := doc.NewWorkspace(nil, testContentFactories())
	partDoc, err := AddPart(ws, "pin.opd", true)
	if err != nil {
		t.Fatalf("AddPart: %v", err)
	}
	asm := NewAssemblyComponentDefinition()
	o1, err := asm.PlaceComponent("pin:1", partDoc, math.Identity4())
	if err != nil {
		t.Fatalf("PlaceComponent pin:1: %v", err)
	}
	o2, err := asm.PlaceComponent("pin:2", partDoc, math.Translation4(math.V3(5, 0, 0)))
	if err != nil {
		t.Fatalf("PlaceComponent pin:2: %v", err)
	}
	if asm.Occurrences().Count() != 2 {
		t.Fatalf("occurrence count = %d, want 2", asm.Occurrences().Count())
	}
	if o1.Definition() != o2.Definition() {
		t.Error("two placements of one part document should share its definition (flyweight)")
	}
}

func TestPlaceComponentRejectsNonComponentDocument(t *testing.T) {
	t.Parallel()
	ws := doc.NewWorkspace(nil, testContentFactories())
	drawingDoc, err := ws.Add(doc.Drawing, "sheet.odd", true) // drawing content has no range box
	if err != nil {
		t.Fatalf("ws.Add(drawing): %v", err)
	}
	if _, err := NewAssemblyComponentDefinition().PlaceComponent("x", drawingDoc, math.Identity4()); err == nil {
		t.Error("PlaceComponent of a drawing document should error (not a component definition)")
	}
}

// TestNestedAssemblyResolvableByPath is the PBI-119 acceptance with real definitions:
// a sub-assembly placed in a parent assembly has its nested part addressable by path.
func TestNestedAssemblyResolvableByPath(t *testing.T) {
	t.Parallel()
	ws := doc.NewWorkspace(nil, testContentFactories())
	pinDoc, err := AddPart(ws, "pin.opd", true)
	if err != nil {
		t.Fatalf("AddPart: %v", err)
	}
	sub := NewAssemblyComponentDefinition()
	if _, err := sub.PlaceComponent("pin:1", pinDoc, math.Identity4()); err != nil {
		t.Fatalf("place pin in sub-assembly: %v", err)
	}
	top := NewAssemblyComponentDefinition()
	top.Place("gearbox:1", sub, math.Identity4())

	got, ok := top.Occurrences().Resolve(occurrence.OccurrencePath{"gearbox:1", "pin:1"})
	if !ok || got.Name() != "pin:1" {
		t.Errorf("Resolve([gearbox:1 pin:1]) ok=%v got=%v, want the nested pin:1", ok, got)
	}
}
