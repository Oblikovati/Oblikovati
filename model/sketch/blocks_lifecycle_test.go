// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"strings"
	"testing"

	"oblikovati.org/math"
)

// TestCreateBlockFromSelection: the selected entities (and their points) move
// out of the sketch into the definition; an identity instance replaces them;
// constraints touching the moved geometry are dropped (M06-F07, #622).
func TestCreateBlockFromSelection(t *testing.T) {
	sc := NewSketches()
	s := sc.Add(XYPlane())
	l1 := s.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(1, 0))
	l2 := s.Lines().AddByTwoPoints(math.P2(1, 0), math.P2(1, 1))
	keep := s.Lines().AddByTwoPoints(math.P2(5, 5), math.P2(6, 5))
	s.GeometricConstraints().AddHorizontal(l1.A, l1.B)
	s.GeometricConstraints().AddHorizontal(keep.A, keep.B)

	def, inst, err := s.Blocks().CreateFromSelection(sc.BlockDefinitions(), "corner", []Entity{l1, l2})
	if err != nil {
		t.Fatalf("CreateFromSelection: %v", err)
	}
	if def.EntityCount() != 2 || inst.DefinitionName() != "corner" {
		t.Fatalf("definition/instance = %d entities / %q, want 2 / corner", def.EntityCount(), inst.DefinitionName())
	}
	if got := len(s.Entities()); got != 2 { // the kept line + the instance
		t.Errorf("sketch entities after create = %d, want 2", got)
	}
	if got := s.GeometricConstraints().Count(); got != 1 {
		t.Errorf("constraints after create = %d, want only the kept line's", got)
	}
	if !inst.Transform().IsEqualTo(math.Identity3(), tol) {
		t.Error("the replacing instance must place at identity (geometry stays put)")
	}
	if _, _, err := s.Blocks().CreateFromSelection(sc.BlockDefinitions(), "corner", []Entity{keep}); err == nil {
		t.Error("a duplicate definition name must be rejected")
	}
}

// TestBlockDeleteInUseNamesConsumers: deleting a definition with placed
// instances is refused, naming the block and the consuming sketches.
func TestBlockDeleteInUseNamesConsumers(t *testing.T) {
	sc := NewSketches()
	s := sc.Add(XYPlane())
	s.SetName("Plate")
	l := s.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(1, 0))
	if _, _, err := s.Blocks().CreateFromSelection(sc.BlockDefinitions(), "rib", []Entity{l}); err != nil {
		t.Fatalf("CreateFromSelection: %v", err)
	}
	err := sc.BlockDefinitions().Delete("rib")
	if err == nil {
		t.Fatal("deleting an in-use definition must fail")
	}
	if !strings.Contains(err.Error(), "rib") || !strings.Contains(err.Error(), "Plate") {
		t.Errorf("error %q must name the block and the consuming sketch", err)
	}

	// Removing the instance unblocks deletion.
	inst := s.Blocks().Item(0)
	s.deleteEntity(inst)
	if err := sc.BlockDefinitions().Delete("rib"); err != nil {
		t.Errorf("Delete after instance removal: %v", err)
	}
}

// TestBlockNestingAndCycleRejection: a definition can nest another block's
// instance; a definition cycle is rejected with both names.
func TestBlockNestingAndCycleRejection(t *testing.T) {
	sc := NewSketches()
	scratch := NewSketches().Add(XYPlane())
	inner, err := sc.BlockDefinitions().Define("inner")
	if err != nil {
		t.Fatalf("Define inner: %v", err)
	}
	if err := inner.Add(scratch.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(1, 0))); err != nil {
		t.Fatalf("Add: %v", err)
	}
	outer, err := sc.BlockDefinitions().Define("outer")
	if err != nil {
		t.Fatalf("Define outer: %v", err)
	}
	nested := &BlockInstance{entityBase: newEntity(), def: inner, transform: math.Translation3(math.V2(2, 0))}
	if err := outer.Add(nested); err != nil {
		t.Fatalf("nesting inner in outer: %v", err)
	}
	backRef := &BlockInstance{entityBase: newEntity(), def: outer, transform: math.Identity3()}
	if err := inner.Add(backRef); err == nil {
		t.Fatal("a definition cycle must be rejected")
	}

	// Expansion composes transforms through the nesting.
	s := sc.Add(XYPlane())
	inst := s.Blocks().Insert(outer, math.Translation3(math.V2(10, 0)))
	polys := inst.ExpandedPolylines()
	if len(polys) != 1 {
		t.Fatalf("expanded polylines = %d, want the nested line", len(polys))
	}
	if got := polys[0][0]; !got.IsEqualTo(math.P2(12, 0), tol) {
		t.Errorf("nested line start = %v, want (12, 0) (outer + nested translation)", got)
	}
}

// TestBlocksRoundTripThroughPartRecipe: definitions (incl. nesting) and
// placed instances survive the recipe round-trip with equal geometry.
func TestBlocksRoundTripThroughPartRecipe(t *testing.T) {
	sc := NewSketches()
	s := sc.Add(XYPlane())
	l := s.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(1, 0))
	c := s.Circles().AddByCenterRadius(math.P2(0.5, 0.5), 0.25)
	if _, _, err := s.Blocks().CreateFromSelection(sc.BlockDefinitions(), "fastener", []Entity{l, c}); err != nil {
		t.Fatalf("CreateFromSelection: %v", err)
	}
	s.Blocks().Insert(mustDef(t, sc, "fastener"), PlacementTransform(math.P2(4, 2), 0.5, 2))

	defs, err := sc.MarshalBlockDefinitions()
	if err != nil {
		t.Fatalf("MarshalBlockDefinitions: %v", err)
	}
	sketches, err := sc.MarshalRecipe()
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	fresh := NewSketches()
	if err := fresh.ApplyBlockDefinitions(defs); err != nil {
		t.Fatalf("ApplyBlockDefinitions: %v", err)
	}
	if err := fresh.ApplyRecipe(sketches); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}

	rdef := mustDef(t, fresh, "fastener")
	if rdef.EntityCount() != 2 {
		t.Errorf("restored definition entities = %d, want 2", rdef.EntityCount())
	}
	rs := fresh.Item(0)
	if rs.Blocks().InstanceCount() != 2 {
		t.Fatalf("restored instances = %d, want 2", rs.Blocks().InstanceCount())
	}
	want := s.Blocks().Item(1).Transform()
	if got := rs.Blocks().Item(1).Transform(); !got.IsEqualTo(want, tol) {
		t.Errorf("restored transform = %v, want %v", got, want)
	}
	// Expanded geometry equality: the realized polylines match pre-save.
	before := s.Blocks().Item(1).ExpandedPolylines()
	after := rs.Blocks().Item(1).ExpandedPolylines()
	if len(before) != len(after) {
		t.Fatalf("expanded polyline count %d != %d", len(after), len(before))
	}
	for i := range before {
		for j := range before[i] {
			if !before[i][j].IsEqualTo(after[i][j], tol) {
				t.Fatalf("expanded geometry differs at polyline %d point %d", i, j)
			}
		}
	}
}

func mustDef(t *testing.T, sc *Sketches, name string) *BlockDefinition {
	t.Helper()
	def, ok := sc.BlockDefinitions().ByName(name)
	if !ok {
		t.Fatalf("definition %q missing", name)
	}
	return def
}
