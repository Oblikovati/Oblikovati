// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"
	stdmath "math"
	"testing"

	"oblikovati.org/api/wire"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
)

// TestAssemblyPatternCircularOverWire replicates a seed component four-up around the Z
// axis: the pattern adds three new occurrences (elements 1–3 beyond the seed), so the
// tree grows from one to four.
func TestAssemblyPatternCircularOverWire(t *testing.T) {
	t.Parallel()
	r, s, _, occs := assemblySessionWithBoxes(t, 0)

	var created wire.NewOccurrencesResult
	args := fmt.Sprintf(`{"seed":%d,"kind":"circular","origin":[0,0,0],"axis":[0,0,1],"angle":%g,"count":4}`, occs[0].ID(), stdmath.Pi/2)
	call(t, r, s, "assembly.patternCreate", args, &created)
	if len(created.Created) != 3 {
		t.Fatalf("circular pattern created %d occurrences, want 3 (4 total minus the seed)", len(created.Created))
	}

	var tree wire.OccurrencesResult
	call(t, r, s, "assembly.occurrences", `{}`, &tree)
	if len(tree.Occurrences) != 4 {
		t.Errorf("tree = %d occurrences, want 4 after the 4-up pattern", len(tree.Occurrences))
	}

	if _, err := r.Handle(s, "assembly.patternCreate", []byte(fmt.Sprintf(`{"seed":%d,"kind":"helical","count":2}`, occs[0].ID()))); err == nil {
		t.Error("an unknown pattern kind should fail")
	}
}

// TestAssemblyMirrorOverWire mirrors a component (placed at x=2) across the YZ plane
// through the origin: the new occurrence sits at x=-2.
func TestAssemblyMirrorOverWire(t *testing.T) {
	t.Parallel()
	r, s, _, occs := assemblySessionWithBoxes(t, 2)

	var mirrored wire.NewOccurrencesResult
	args := fmt.Sprintf(`{"sources":[%d],"origin":[0,0,0],"normal":[1,0,0]}`, occs[0].ID())
	call(t, r, s, "assembly.mirror", args, &mirrored)
	if len(mirrored.Created) != 1 {
		t.Fatalf("mirror created %d occurrences, want 1", len(mirrored.Created))
	}
	if got := mirrored.Created[0].Transform.Cells[3]; got != -2 {
		t.Errorf("mirror x-translation = %g, want -2 (reflected across x=0)", got)
	}

	if _, err := r.Handle(s, "assembly.mirror", []byte(`{"sources":[99999],"origin":[0,0,0],"normal":[1,0,0]}`)); err == nil {
		t.Error("mirror of an unknown occurrence id should fail")
	}
}

// TestAssemblyCopyOverWire copies a component, producing an independent occurrence at the
// same placement.
func TestAssemblyCopyOverWire(t *testing.T) {
	t.Parallel()
	r, s, _, occs := assemblySessionWithBoxes(t, 3)

	var copied wire.NewOccurrencesResult
	call(t, r, s, "assembly.copy", fmt.Sprintf(`{"sources":[%d]}`, occs[0].ID()), &copied)
	if len(copied.Created) != 1 {
		t.Fatalf("copy created %d occurrences, want 1", len(copied.Created))
	}
	if copied.Created[0].ID == occs[0].ID() || copied.Created[0].Transform.Cells[3] != 3 {
		t.Errorf("copy = %+v, want a new occurrence at the source's placement (x=3)", copied.Created[0])
	}
}

// TestAssemblySubstituteOverWire substitutes two components with one simplified part: the
// substitute occurrence is flagged, and the sources are suppressed.
func TestAssemblySubstituteOverWire(t *testing.T) {
	t.Parallel()
	r, s, _, occs := assemblySessionWithBoxes(t, 0, 5)
	lod := openPartDoc(t, s, "lod.obk")

	var sub wire.OccurrenceResult
	args := fmt.Sprintf(`{"sources":[%d,%d],"document":%d,"name":"lod:1","transform":%s}`,
		occs[0].ID(), occs[1].ID(), uint64(lod.ID()), transformJSON(0, 0, 0))
	call(t, r, s, "assembly.substitute", args, &sub)
	if sub.Occurrence.Name != "lod:1" || !sub.Occurrence.Substitute {
		t.Fatalf("substitute = %+v, want a substitute occurrence named lod:1", sub.Occurrence)
	}
	if !occs[0].Suppressed() || !occs[1].Suppressed() {
		t.Error("substituted sources should be suppressed")
	}

	if _, err := r.Handle(s, "assembly.substitute", []byte(`{"sources":[99999],"document":1,"name":"x","transform":[1,0,0,0,0,1,0,0,0,0,1,0,0,0,0,1]}`)); err == nil {
		t.Error("substitute of an unknown source id should fail")
	}
}

// TestAssemblyMirrorIntoPartOverWire mirrors a file-backed component into a new opposite-
// hand part document (#717): the created occurrence is placed by a proper (det>0)
// transform — handedness lives in the part — and a new separately-openable "*-mirror.obk"
// part holds the source geometry reflected across the local plane (volume preserved,
// centroid crossed to -x).
func TestAssemblyMirrorIntoPartOverWire(t *testing.T) {
	t.Parallel()
	r, s, _, _ := assemblySessionWithBoxes(t)
	widget := partDocWithBox(t, s, "widget.obk") // a unit box [0,1]³ at the part origin

	// Place the widget (file-backed, so the occurrence records a component document).
	var placed wire.OccurrenceResult
	call(t, r, s, "assembly.place", fmt.Sprintf(`{"document":%d,"name":"widget:1","transform":%s}`, uint64(widget.ID()), transformJSON(2, 0, 0)), &placed)

	var mirrored wire.NewOccurrencesResult
	call(t, r, s, "assembly.mirrorIntoPart", fmt.Sprintf(`{"sources":[%d],"origin":[0,0,0],"normal":[1,0,0]}`, placed.Occurrence.ID), &mirrored)
	if len(mirrored.Created) != 1 {
		t.Fatalf("mirror-into-part created %d occurrences, want 1", len(mirrored.Created))
	}
	if placement := math.Matrix4FromCells(mirrored.Created[0].Transform.Cells); placement.Determinant() <= 0 {
		t.Errorf("placement determinant = %g, want > 0 (a real opposite-hand part, not a reflected instance)", placement.Determinant())
	}

	mirrorDoc, ok := s.Workspace().ByName("widget-mirror.obk")
	if !ok {
		t.Fatal("mirror-into-part should create a new widget-mirror.obk part document")
	}
	bodies := mirrorDoc.Content().(*compdef.PartComponentDefinition).SurfaceBodies().All()
	if len(bodies) != 1 {
		t.Fatalf("mirror part has %d bodies, want 1 (the reflected source)", len(bodies))
	}
	props := query.BodyGeometryProperties(bodies[0], ops.DefaultQuality())
	if stdmath.Abs(props.Volume-1) > 1e-6 {
		t.Errorf("mirror part volume = %g, want 1 (reflection preserves the unit box)", props.Volume)
	}
	if props.Centroid.X > 0 {
		t.Errorf("mirror part centroid x = %g, want < 0 (geometry reflected about the local YZ plane)", props.Centroid.X)
	}

	// The source occurrence remains; the assembly now has the original plus the mirror.
	if _, err := r.Handle(s, "assembly.mirrorIntoPart", []byte(`{"sources":[999],"origin":[0,0,0],"normal":[1,0,0]}`)); err == nil {
		t.Error("mirror-into-part of an unknown occurrence id should fail")
	}
}
