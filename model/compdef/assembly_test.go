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

var _ occurrence.RangeBoxSource = fakeComponentSource{}

// unitSource is a 1×1×1 component at its own origin.
func unitSource() fakeComponentSource {
	return fakeComponentSource{box: math.NewBox(math.P3(0, 0, 0), math.P3(1, 1, 1))}
}

func TestNewAssemblyIsEmptyAssemblyContent(t *testing.T) {
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
	a := NewAssemblyComponentDefinition()
	v0 := a.ModelGeometryVersion()
	a.Occurrences().Add("base", unitSource(), math.Identity4())
	a.Occurrences().Add("offset", unitSource(), math.Translation4(math.V3(4, 0, 0)))
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
	ws := doc.NewWorkspace(nil)
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
	ws := doc.NewWorkspace(nil)
	d, err := ws.Add(doc.Assembly, "sub.oad", true)
	if err != nil {
		t.Fatalf("ws.Add(assembly): %v", err)
	}
	if _, ok := d.Content().(*AssemblyComponentDefinition); !ok {
		t.Errorf("factory content = %T, want *AssemblyComponentDefinition", d.Content())
	}
}
