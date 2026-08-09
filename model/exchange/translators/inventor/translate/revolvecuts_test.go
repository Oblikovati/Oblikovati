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

// TestAxisLineFromReferenceFindsTheEdge: given a decoded horizontal axis on y=0, the collinear
// profile edge (the y=0 line) is selected, and an off-axis or oblique line is not. This is the
// centreline the geometric heuristic misses when the axis is an ordinary profile edge.
func TestAxisLineFromReferenceFindsTheEdge(t *testing.T) {
	s := ipt.Sketch{Lines: []ipt.Line{
		ln(0, 0, -1.29, 0), // on y=0 — the axis edge
		ln(-1.29, 0, -1.29, -4.3),
		ln(-1.29, -4.3, 0.3, -4.3),
	}}
	axis := revolveAxisRef{ox: 0, oy: 0, dx: -1.29, dy: 0, ok: true}
	i, ok := axisLineFromReference(s, axis)
	if !ok || i != 0 {
		t.Fatalf("want the y=0 edge (index 0), got index %d ok=%v", i, ok)
	}
	// no line on a y=2 axis
	if _, ok := axisLineFromReference(s, revolveAxisRef{ox: 0, oy: 2, dx: 1, dy: 0, ok: true}); ok {
		t.Error("no profile edge lies on y=2, expected no match")
	}
}

// TestReviseAxisCrossingCirclesRecoversArc: a non-construction full circle that crosses the axis but
// whose endpoints are one-sided is rebuilt as an arc; a genuine bore (no endpoints) and an
// axis-straddling pair are left as circles.
func TestReviseAxisCrossingCirclesRecoversArc(t *testing.T) {
	s := ipt.Sketch{
		Circles: []ipt.Circle{
			{Center: ipt.Point2D{X: 17.779, Y: -0.001}, Radius: 20.329, // crosses x=0, ends one-sided → arc
				ArcStart: ipt.Point2D{X: -2.466, Y: 1.85}, ArcEnd: ipt.Point2D{X: -2.55, Y: 0}, ArcEndsOK: true},
			{Center: ipt.Point2D{X: 0, Y: 2.1}, Radius: 0.19}, // genuine bore, no endpoints → circle
			{Center: ipt.Point2D{}, Radius: 1, // ends straddle x=0 → circle
				ArcStart: ipt.Point2D{X: 1}, ArcEnd: ipt.Point2D{X: -1}, ArcEndsOK: true},
		},
		CircleConstruction: []bool{false, false, false},
	}
	axis := revolveAxisRef{ox: 0, oy: 0, dx: 0, dy: 1.7, ok: true}
	got := reviseAxisCrossingCircles(s, axis)
	if len(got.Arcs) != 1 {
		t.Fatalf("want 1 recovered arc, got %d", len(got.Arcs))
	}
	if got.Arcs[0].Radius != 20.329 {
		t.Errorf("recovered arc has radius %.3f, want 20.329", got.Arcs[0].Radius)
	}
	if len(got.Circles) != 2 {
		t.Errorf("want 2 circles left (bore + straddling), got %d", len(got.Circles))
	}
}
