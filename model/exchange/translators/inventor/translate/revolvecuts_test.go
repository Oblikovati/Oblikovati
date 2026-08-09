// SPDX-License-Identifier: GPL-2.0-only

package translate

import (
	"math"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/model/analysis"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/contentset"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/exchange/translators/inventor/ipt"
	"oblikovati.org/model/sketch"
	"oblikovati.org/persistence"
)

// newPart is a test helper: a fresh empty part definition.
func newPart(t *testing.T) *compdef.PartComponentDefinition {
	t.Helper()
	ws := doc.NewWorkspace(persistence.NewPackageStore(), contentset.Default())
	d, err := compdef.AddPart(ws, "t.opd", true)
	if err != nil {
		t.Fatal(err)
	}
	return d.Content().(*compdef.PartComponentDefinition)
}

// rectAboutAxis is a closed rectangle profile at x∈[x0,x1], y∈[0,h] plus an isolated vertical
// centreline at x=0 — a synthetic revolve profile (its 360° sweep is an annular tube).
func rectAboutAxis(x0, x1, h float64) ipt.Sketch {
	return ipt.Sketch{
		Lines: []ipt.Line{
			ln(x0, 0, x1, 0), ln(x1, 0, x1, h), ln(x1, h, x0, h), ln(x0, h, x0, 0),
			ln(0, 0, 0, h+2), // isolated vertical axis
		},
		Resolved: true,
	}
}

// TestTryKernelRevolveBuildsSolidFromSyntheticProfile drives the whole kernel-revolve chain
// (revolveAxisIndex → closedProfileIndices → profileOneSideOfAxis → revolveClosedProfiles) on a
// synthetic profile, with no corpus part: the rectangle x∈[1,2], h=3 revolved about x=0 is an
// annular tube of volume π(2²−1²)·3 = 9π cm³.
func TestTryKernelRevolveBuildsSolidFromSyntheticProfile(t *testing.T) {
	profile := rectAboutAxis(1, 2, 3)
	if !graphRevolveCandidate([]ipt.Sketch{profile}) {
		t.Fatal("expected a revolve candidate")
	}
	if graphRevolveCandidate([]ipt.Sketch{{Lines: []ipt.Line{ln(0, 0, 1, 0)}, Resolved: true}}) {
		t.Error("a 1-line sketch is not a revolve candidate")
	}
	def := newPart(t)
	placed := placeGraphSketches([]ipt.Sketch{profile})
	emitted := emitSketches(def, placed)
	if ids := tryKernelRevolve(def, nil, placed, emitted, 0, revolveAxisRef{}); len(ids) == 0 {
		t.Fatal("tryKernelRevolve built nothing")
	}
	def.Recompute()
	if !firstBodyIsSolid(def) {
		t.Fatal("revolve did not close to a solid")
	}
	vol := analysis.MassPropertiesOf(def.SurfaceBodies().All(), 1, types.MassPropertiesHigh).VolumeMm3
	if want := 9 * math.Pi * 1000; math.Abs(vol-want) > 0.03*want {
		t.Errorf("tube volume = %.0f mm³, want ~%.0f", vol, want)
	}
}

// TestBuildGraphRevolveCutsBuildsBaseAndCuts drives the whole revolve-with-cuts orchestration
// (graphRevolveCandidate → revolve base → applyExtrudeCutsAndHole with the through-all retry) on
// synthetic graph sketches, with no corpus part: an annular base with a Ø0.6 bore cut, region named
// as a lone full circle.
func TestBuildGraphRevolveCutsBuildsBaseAndCuts(t *testing.T) {
	def := newPart(t)
	graph := []ipt.Sketch{
		rectAboutAxis(1, 2, 3), // profile sketch #0 → annular tube base
		{Circles: []ipt.Circle{{Center: ipt.Point2D{X: 1.5}, Radius: 0.3}}, Resolved: true}, // bore sketch #1
	}
	extrudes := []ipt.Extrude{{Operation: ipt.OpCut, ThroughAll: true}}
	profiles := []int{1}
	regions := [][]ipt.RegionLoop{{{Edges: []ipt.RegionEdge{{Kind: ipt.EdgeCircle, Circle: ipt.Circle{Center: ipt.Point2D{X: 1.5}, Radius: 0.3}}}}}}
	solid, notes := buildGraphRevolveCuts(def, nil, graph, 0, revolveAxisRef{}, extrudes, profiles, regions, ipt.Hole{}, false)
	if !solid {
		t.Fatalf("expected a solid turned+milled body; notes=%v", notes)
	}
	if !firstBodyIsSolid(def) {
		t.Error("firstBodyIsSolid disagrees with the returned solid flag")
	}
}

// TestRevolveCutBranches covers the remaining revolve/cut branches: the preferred=-1 SCAN path, the
// axis-reference fallback (a profile turning about an ordinary y=0 edge the heuristic can't spot), and
// applyExtrudeCutsAndHole's skip note for an unresolvable profile.
func TestRevolveCutBranches(t *testing.T) {
	// Scan path: preferred=-1 finds the revolve by scanning the emitted sketches.
	def := newPart(t)
	placed := placeGraphSketches([]ipt.Sketch{rectAboutAxis(1, 2, 3)})
	emitted := emitSketches(def, placed)
	if len(tryKernelRevolve(def, nil, placed, emitted, -1, revolveAxisRef{})) == 0 {
		t.Error("scan path built no revolve")
	}

	// Axis-reference fallback: a rectangle below y=0 with its top edge ON y=0 — no isolated/construction
	// centreline, so revolveAxisIndex declines and the decoded horizontal axis supplies the edge.
	def2 := newPart(t)
	edge := ipt.Sketch{Lines: []ipt.Line{
		ln(-2, 0, -1, 0), ln(-1, 0, -1, -3), ln(-1, -3, -2, -3), ln(-2, -3, -2, 0),
	}, Resolved: true}
	pl2 := placeGraphSketches([]ipt.Sketch{edge})
	em2 := emitSketches(def2, pl2)
	if len(tryKernelRevolve(def2, nil, pl2, em2, 0, revolveAxisRef{ox: 0, oy: 0, dx: -1, dy: 0, ok: true})) == 0 {
		t.Error("axis-reference revolve built nothing")
	}

	// applyExtrudeCutsAndHole skip note: an extrude whose profile index is out of range.
	notes := applyExtrudeCutsAndHole(newPart(t), []ipt.Extrude{{Operation: ipt.OpCut}}, []int{9}, nil, ipt.Hole{}, false, pl2, em2)
	if len(notes) == 0 {
		t.Error("expected a skip note for an unresolvable profile")
	}
}

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
