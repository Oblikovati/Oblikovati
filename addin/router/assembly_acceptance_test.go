// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"
	stdmath "math"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/kernel/brep"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/feature"
)

// partDocWithBox adds a part document carrying a unit-box body and returns it (active
// document unchanged), so a test can place it into an assembly over the wire and have
// real geometry to derive — the doc-backed counterpart of blockPart.
func partDocWithBox(t *testing.T, s *app.Session, name string) *doc.Document {
	t.Helper()
	active := s.ActiveDocument()
	d, err := compdef.AddPart(s.Workspace(), name, true)
	if err != nil {
		t.Fatalf("AddPart %q: %v", name, err)
	}
	part := d.Content().(*compdef.PartComponentDefinition)
	block, err := brep.SolidBlock(math.P3(0, 0, 0), math.P3(1, 1, 1), "widget")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	feature.NewBaseFeatures(part.Features()).AddBase(block)
	part.Recompute()
	if err := s.Workspace().SetActiveDocument(active); err != nil {
		t.Fatalf("restore active: %v", err)
	}
	return d
}

// TestAssemblyModelDrivenOverWire is the #716 acceptance round-trip: an add-in drives the
// whole assembly model over the wire — places components, reads the occurrence tree,
// replicates it (pattern then mirror), pulls a BOM view, and derives the assembly into a
// part — proving the public surface composes end to end, not just one method at a time.
//
// The placements are chosen disjoint so the numbers are exact: a unit box at the origin
// and at x=3; a 3-up circular pattern of the first about Z (0°/90°/180°, each an
// axis-aligned box in a different quadrant); and a mirror of the second across x=0 to
// x=-3. Five disjoint unit boxes, all sharing one part definition ⇒ a parts-only BOM of
// quantity 5 and a derived part of volume 5.
func TestAssemblyModelDrivenOverWire(t *testing.T) {
	t.Parallel()
	r, s, _, _ := assemblySessionWithBoxes(t) // empty assembly, active
	asmDoc := s.ActiveDocument()
	widget := partDocWithBox(t, s, "widget.obk")

	// 1. Place two instances of the shared widget over the wire.
	var first, second wire.OccurrenceResult
	call(t, r, s, "assembly.place", fmt.Sprintf(`{"document":%d,"name":"widget:1","transform":%s}`, uint64(widget.ID()), transformJSON(0, 0, 0)), &first)
	call(t, r, s, "assembly.place", fmt.Sprintf(`{"document":%d,"name":"widget:2","transform":%s}`, uint64(widget.ID()), transformJSON(3, 0, 0)), &second)

	// 2. Read the occurrence tree.
	var tree wire.OccurrencesResult
	call(t, r, s, "assembly.occurrences", `{}`, &tree)
	if len(tree.Occurrences) != 2 {
		t.Fatalf("after placing two components the tree has %d occurrences, want 2", len(tree.Occurrences))
	}

	// 3. Replicate: a 3-up circular pattern of the first, then a mirror of the second.
	var pat wire.NewOccurrencesResult
	call(t, r, s, "assembly.patternCreate", fmt.Sprintf(`{"seed":%d,"kind":"circular","origin":[0,0,0],"axis":[0,0,1],"angle":%g,"count":3}`, first.Occurrence.ID, stdmath.Pi/2), &pat)
	if len(pat.Created) != 2 {
		t.Fatalf("circular pattern created %d occurrences, want 2", len(pat.Created))
	}
	var mir wire.NewOccurrencesResult
	call(t, r, s, "assembly.mirror", fmt.Sprintf(`{"sources":[%d],"origin":[0,0,0],"normal":[1,0,0]}`, second.Occurrence.ID), &mir)
	if len(mir.Created) != 1 {
		t.Fatalf("mirror created %d occurrences, want 1", len(mir.Created))
	}
	call(t, r, s, "assembly.occurrences", `{}`, &tree)
	if len(tree.Occurrences) != 5 {
		t.Fatalf("after pattern (+2) and mirror (+1) the tree has %d occurrences, want 5", len(tree.Occurrences))
	}

	// 4. Pull a parts-only BOM: every instance shares the one widget definition.
	var bom wire.BOMViewResult
	call(t, r, s, "assembly.bomView", `{"view":"partsOnly"}`, &bom)
	if bom.View != types.BOMPartsOnly || len(bom.Rows) != 1 || bom.Rows[0].Quantity != 5 {
		t.Fatalf("parts-only BOM = %+v, want one row of quantity 5", bom.Rows)
	}

	// 5. Derive the whole assembly into a fresh part and confirm its geometry flowed through.
	if _, err := compdef.AddPart(s.Workspace(), "derived.obk", true); err != nil {
		t.Fatalf("AddPart derived: %v", err)
	}
	var detail wire.FeatureDetailResult
	call(t, r, s, "assembly.deriveCreate", fmt.Sprintf(`{"source":%d}`, uint64(asmDoc.ID())), &detail)
	if detail.Feature.Kind != "derivedAssembly" {
		t.Errorf("derived feature kind = %q, want derivedAssembly", detail.Feature.Kind)
	}
	if got := activePartVolume(t, s); stdmath.Abs(got-5.0) > 1e-6 {
		t.Errorf("derived part volume = %g, want 5.0 (five disjoint unit boxes)", got)
	}
}
