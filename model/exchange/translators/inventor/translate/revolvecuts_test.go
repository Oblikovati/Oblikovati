// SPDX-License-Identifier: GPL-2.0-only

package translate

import (
	"testing"

	"oblikovati.org/model/compdef"
	"oblikovati.org/model/contentset"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/exchange/translators/inventor/ipt"
	"oblikovati.org/model/sketch"
	"oblikovati.org/persistence"
)

// TestPlaceGraphSketchesPairsEachWithAPlane: every graph sketch comes back paired with a plane and
// its geometry preserved; a sketch stating no readable placement lands on XY (sketchPlaneOf's
// fallback), never dropped.
func TestPlaceGraphSketchesPairsEachWithAPlane(t *testing.T) {
	graph := []ipt.Sketch{
		{Lines: []ipt.Line{ln(0, 0, 0, 5)}}, // no plane → XY fallback
		{Circles: []ipt.Circle{{Center: ipt.Point2D{}, Radius: 1}}},
	}
	placed := placeGraphSketches(graph)
	if len(placed) != len(graph) {
		t.Fatalf("placed %d sketches, want %d", len(placed), len(graph))
	}
	if len(placed[0].geom.Lines) != 1 || len(placed[1].geom.Circles) != 1 {
		t.Error("placeGraphSketches must carry each sketch's geometry through unchanged")
	}
	if n := placed[0].plane.Normal().AsVector(); n.Z < 0.999 {
		t.Errorf("a placement-less sketch should fall back to XY (normal +Z), got normal.Z=%v", n.Z)
	}
}

// TestClearFeaturesAndSketchesEmptiesBoth: after emitting a sketch, clearFeaturesAndSketches returns
// the definition to zero sketches and zero features — what buildRevolveDispatch relies on to abandon
// one build attempt and start another from a different sketch source.
func TestClearFeaturesAndSketchesEmptiesBoth(t *testing.T) {
	ws := doc.NewWorkspace(persistence.NewPackageStore(), contentset.Default())
	document, err := compdef.AddPart(ws, "clear.opd", true)
	if err != nil {
		t.Fatal(err)
	}
	def := document.Content().(*compdef.PartComponentDefinition)
	emitSketchOn(def, ipt.Sketch{Lines: []ipt.Line{ln(0, 0, 2, 0), ln(2, 0, 2, 2)}}, sketch.XYPlane())
	if def.Sketches().Count() == 0 {
		t.Fatal("precondition: expected a sketch to have been emitted")
	}
	clearFeaturesAndSketches(def)
	if def.Sketches().Count() != 0 || def.Features().Count() != 0 {
		t.Errorf("after clear: sketches=%d features=%d, want 0/0", def.Sketches().Count(), def.Features().Count())
	}
}
