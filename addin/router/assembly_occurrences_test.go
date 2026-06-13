// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"
	"testing"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
)

// openPartDoc adds a part document to the workspace (keeping the active document
// unchanged) and returns it, so a test can place it into the active assembly by id.
func openPartDoc(t *testing.T, s *app.Session, name string) *doc.Document {
	t.Helper()
	active := s.ActiveDocument()
	d, err := compdef.AddPart(s.Workspace(), name, true)
	if err != nil {
		t.Fatalf("AddPart: %v", err)
	}
	if err := s.Workspace().SetActiveDocument(active); err != nil {
		t.Fatalf("restore active: %v", err)
	}
	return d
}

// transformJSON renders a row-major translation matrix (tx,ty,tz in cells 3/7/11) as the
// flat 16-cell array the wire matrix expects.
func transformJSON(tx, ty, tz float64) string {
	return fmt.Sprintf("[1,0,0,%g, 0,1,0,%g, 0,0,1,%g, 0,0,0,1]", tx, ty, tz)
}

// TestAssemblyOccurrencesTreeOverWire reads the occurrence tree of an assembly with two
// placed components and checks ids, names, and placements round-trip.
func TestAssemblyOccurrencesTreeOverWire(t *testing.T) {
	r, s, _, occs := assemblySessionWithBoxes(t, 0, 5)

	var tree wire.OccurrencesResult
	call(t, r, s, "assembly.occurrences", `{}`, &tree)
	if len(tree.Occurrences) != 2 {
		t.Fatalf("tree = %d occurrences, want 2", len(tree.Occurrences))
	}
	if tree.Occurrences[0].ID != occs[0].ID() || tree.Occurrences[0].Name != occs[0].Name() {
		t.Errorf("node[0] = {%d,%q}, want {%d,%q}", tree.Occurrences[0].ID, tree.Occurrences[0].Name, occs[0].ID(), occs[0].Name())
	}
	// occs[1] was placed at x=5 (cells[3]).
	if got := tree.Occurrences[1].Transform.Cells[3]; got != 5 {
		t.Errorf("node[1] x-translation = %g, want 5", got)
	}
}

// TestAssemblyPlaceOverWire places an open part document into the active assembly and
// checks the new occurrence carries the requested name/transform and joins the tree.
func TestAssemblyPlaceOverWire(t *testing.T) {
	r, s, _, _ := assemblySessionWithBoxes(t)
	pin := openPartDoc(t, s, "pin.obk")

	var placed wire.OccurrenceResult
	args := fmt.Sprintf(`{"document":%d,"name":"pin:1","transform":%s}`, uint64(pin.ID()), transformJSON(2, 0, 0))
	call(t, r, s, "assembly.place", args, &placed)
	if placed.Occurrence.Name != "pin:1" || placed.Occurrence.Transform.Cells[3] != 2 {
		t.Fatalf("placed = %+v, want pin:1 at x=2", placed.Occurrence)
	}

	var tree wire.OccurrencesResult
	call(t, r, s, "assembly.occurrences", `{}`, &tree)
	if len(tree.Occurrences) != 1 || tree.Occurrences[0].ID != placed.Occurrence.ID {
		t.Fatalf("tree = %+v, want the one placed occurrence", tree.Occurrences)
	}

	if _, err := r.Handle(s, "assembly.place", []byte(`{"document":99999,"name":"x","transform":[1,0,0,0,0,1,0,0,0,0,1,0,0,0,0,1]}`)); err == nil {
		t.Error("place with an unknown document id should fail")
	}
}

// TestAssemblyPlaceByDefinitionOverWire places a second instance reusing an existing
// occurrence's definition.
func TestAssemblyPlaceByDefinitionOverWire(t *testing.T) {
	r, s, _, occs := assemblySessionWithBoxes(t, 0)

	var placed wire.OccurrenceResult
	args := fmt.Sprintf(`{"source":%d,"name":"box:2","transform":%s}`, occs[0].ID(), transformJSON(3, 0, 0))
	call(t, r, s, "assembly.placeByDefinition", args, &placed)
	if placed.Occurrence.Name != "box:2" || placed.Occurrence.ID == occs[0].ID() {
		t.Fatalf("placed = %+v, want a new occurrence named box:2", placed.Occurrence)
	}

	if _, err := r.Handle(s, "assembly.placeByDefinition", []byte(`{"source":99999,"name":"x","transform":[1,0,0,0,0,1,0,0,0,0,1,0,0,0,0,1]}`)); err == nil {
		t.Error("placeByDefinition with an unknown source id should fail")
	}
}

// TestAssemblyTransformGroundSuppressOverWire drives the per-occurrence state mutators and
// checks each is reflected on the occurrence.
func TestAssemblyTransformGroundSuppressOverWire(t *testing.T) {
	r, s, _, occs := assemblySessionWithBoxes(t, 0)
	id := occs[0].ID()

	var moved wire.OccurrenceResult
	call(t, r, s, "assembly.transform", fmt.Sprintf(`{"id":%d,"transform":%s}`, id, transformJSON(7, 0, 0)), &moved)
	if moved.Occurrence.Transform.Cells[3] != 7 || occs[0].Transform().Cells()[3] != 7 {
		t.Errorf("after transform: wire=%g model=%g, want 7", moved.Occurrence.Transform.Cells[3], occs[0].Transform().Cells()[3])
	}

	var grounded wire.OccurrenceResult
	call(t, r, s, "assembly.ground", fmt.Sprintf(`{"id":%d,"grounded":true}`, id), &grounded)
	if !grounded.Occurrence.Grounded || !occs[0].Grounded() {
		t.Error("after ground: occurrence should be grounded")
	}

	var suppressed wire.OccurrenceResult
	call(t, r, s, "assembly.suppress", fmt.Sprintf(`{"id":%d,"suppressed":true}`, id), &suppressed)
	if !suppressed.Occurrence.Suppressed || !occs[0].Suppressed() {
		t.Error("after suppress: occurrence should be suppressed")
	}

	if _, err := r.Handle(s, "assembly.transform", []byte(`{"id":99999,"transform":[1,0,0,0,0,1,0,0,0,0,1,0,0,0,0,1]}`)); err == nil {
		t.Error("transform of an unknown occurrence id should fail")
	}
}

// TestAssemblyReplaceOverWire swaps an occurrence's component for another document's,
// keeping the occurrence's id, name, and placement.
func TestAssemblyReplaceOverWire(t *testing.T) {
	r, s, _, occs := assemblySessionWithBoxes(t, 4)
	id, name := occs[0].ID(), occs[0].Name()
	pin := openPartDoc(t, s, "pin.obk")

	var replaced wire.OccurrenceResult
	call(t, r, s, "assembly.replace", fmt.Sprintf(`{"id":%d,"document":%d}`, id, uint64(pin.ID())), &replaced)
	if replaced.Occurrence.ID != id || replaced.Occurrence.Name != name || replaced.Occurrence.Transform.Cells[3] != 4 {
		t.Errorf("replaced = %+v, want id/name/placement kept (id=%d name=%q x=4)", replaced.Occurrence, id, name)
	}
}

// TestAssemblyRemoveOverWire deletes an occurrence and checks the tree shrinks.
func TestAssemblyRemoveOverWire(t *testing.T) {
	r, s, _, occs := assemblySessionWithBoxes(t, 0, 5)

	var tree wire.OccurrencesResult
	call(t, r, s, "assembly.remove", fmt.Sprintf(`{"id":%d}`, occs[0].ID()), &tree)
	if len(tree.Occurrences) != 1 || tree.Occurrences[0].ID != occs[1].ID() {
		t.Fatalf("after remove: tree = %+v, want only occurrence %d", tree.Occurrences, occs[1].ID())
	}

	if _, err := r.Handle(s, "assembly.remove", []byte(`{"id":99999}`)); err == nil {
		t.Error("remove of an unknown occurrence id should fail")
	}
}
