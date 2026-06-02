// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"testing"

	"github.com/Oblikovati/oblikovati/kernel/geom"
	"github.com/Oblikovati/oblikovati/kernel/topo"
	"github.com/Oblikovati/oblikovati/math"
	"github.com/Oblikovati/oblikovati/model/doc"
)

// box builds a single-vertex solid stand-in body with a known range box corner, so
// the definition's bounding box is testable without a full solid builder.
func bodyWithCorner(x, y, z float64) *topo.Body {
	bld := topo.NewBuilder(true, topo.NewLineage(topo.Tok("f", "body", 0)))
	v0 := bld.AddVertex(math.P3(0, 0, 0), topo.NewLineage(topo.Tok("f", "vertex", 0)))
	v1 := bld.AddVertex(math.P3(x, y, z), topo.NewLineage(topo.Tok("f", "vertex", 1)))
	e := bld.AddEdge(geom.NewLineSegment(v0.Point(), v1.Point()), v0, v1, topo.NewLineage(topo.Tok("f", "edge", 0)))
	plane, _ := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	bld.AddFace(plane, topo.NewLineage(topo.Tok("f", "face", 0)), topo.OuterLoop(topo.Fwd(e)))
	return bld.Build()
}

func TestPartContentImplementsDocContent(t *testing.T) {
	var _ doc.Content = NewPartComponentDefinition()
	if NewPartComponentDefinition().DocumentType() != doc.Part {
		t.Error("part content should report DocumentType Part")
	}
}

func TestPartDocumentExposesBodiesAndBoundingBox(t *testing.T) {
	def := NewPartComponentDefinition()
	def.SurfaceBodies().Add(bodyWithCorner(4, 3, 2))

	// Wire the real content onto a part document (M03) and retrieve it back.
	ws := doc.NewWorkspace(nil)
	pd, _ := ws.Add(doc.Part, "bracket.obk", true)
	pd.SetContent(def)

	got, ok := pd.Content().(*PartComponentDefinition)
	if !ok || got != def {
		t.Fatal("part document did not expose its component definition")
	}
	if got.SurfaceBodies().Count() != 1 {
		t.Errorf("bodies = %d, want 1", got.SurfaceBodies().Count())
	}
	box := got.RangeBox()
	if !box.Contains(math.P3(4, 3, 2)) || !box.Contains(math.P3(0, 0, 0)) {
		t.Errorf("range box %v does not span the body", box)
	}
	obb := got.OrientedMinimumRangeBox()
	if !obb.Contains(math.P3(2, 1.5, 1)) {
		t.Error("oriented range box does not contain the body center")
	}
}

func TestGeometryVersionChangesOnEdit(t *testing.T) {
	def := NewPartComponentDefinition()
	v0 := def.ModelGeometryVersion()
	def.MarkChanged()
	v1 := def.ModelGeometryVersion()
	if v0 == v1 {
		t.Errorf("geometry version did not change on edit: %q == %q", v0, v1)
	}
	def.MarkChanged()
	if def.ModelGeometryVersion() == v1 {
		t.Error("geometry version did not advance on a second edit")
	}
}

func TestEmptyDefinitionAccessors(t *testing.T) {
	def := NewPartComponentDefinition()
	if def.SurfaceBodies().Count() != 0 || def.Parameters().Count() != 0 || def.Sketches().Count() != 0 {
		t.Error("new definition should be empty")
	}
	if !def.RangeBox().IsEmpty() || !def.PreciseRangeBox().IsEmpty() {
		t.Error("empty definition should have an empty range box")
	}
}

func TestEndOfPartRollback(t *testing.T) {
	def := NewPartComponentDefinition()
	if def.IsRolledBack() || def.EndOfPartPosition() != -1 {
		t.Fatal("new definition should be at end-of-part, not rolled back")
	}
	before := def.ModelGeometryVersion()
	def.SetEndOfPart(2) // roll back to feature index 2
	if !def.IsRolledBack() || def.EndOfPartPosition() != 2 {
		t.Errorf("after SetEndOfPart(2): rolledBack=%v pos=%d", def.IsRolledBack(), def.EndOfPartPosition())
	}
	if def.ModelGeometryVersion() == before {
		t.Error("moving the EOP marker should bump the geometry version (request re-eval)")
	}
	def.RollToEnd()
	if def.IsRolledBack() || def.EndOfPartPosition() != -1 {
		t.Error("RollToEnd should restore full evaluation")
	}
}
