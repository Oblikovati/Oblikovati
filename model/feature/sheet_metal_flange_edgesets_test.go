// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
)

// topEdgesAlongX returns every top-face boundary edge that runs in X (the two opposite sides of a
// square plate), so a test can flange more than one edge in one feature.
func topEdgesAlongX(t *testing.T, body *topo.Body) []*topo.Edge {
	t.Helper()
	maxZ := math.Inf(-1)
	for _, e := range body.Edges() {
		if z := float64(e.StartVertex().Point().Z); z > maxZ {
			maxZ = z
		}
	}
	var out []*topo.Edge
	for _, e := range body.Edges() {
		a, b := e.StartVertex().Point(), e.EndVertex().Point()
		if math.Abs(float64(a.X-b.X)) > 1e-6 && math.Abs(float64(a.Y-b.Y)) < 1e-6 &&
			math.Abs(float64(a.Z)-maxZ) < 1e-6 && math.Abs(float64(b.Z)-maxZ) < 1e-6 {
			out = append(out, e)
		}
	}
	return out
}

// TestFlangeEdgeSetsFlangesEveryEdge is the #2071 acceptance: one flange feature with several edge
// sets folds a wall on every edge and records one bend per edge, so on two opposite edges it adds
// twice a single flange's band and reports two placements.
func TestFlangeEdgeSetsFlangesEveryEdge(t *testing.T) {
	const side, r, h, th = 4.0, 0.2, 1.0, 0.2
	fs, _ := seedSheetMetalSheet(t, side, nil)
	edges := topEdgesAlongX(t, fs.Result()[0])
	if len(edges) < 2 {
		t.Fatalf("the seed plate has %d top X-edges, want ≥2 to flange", len(edges))
	}
	pf := NewSheetMetalFlangeFeatures(fs).Add(&SheetMetalFlangeDefinition{
		Height: func() float64 { return h }, Radius: func() float64 { return r },
		Angle: func() float64 { return math.Pi / 2 },
		EdgeSets: []FlangeEdgeSet{
			{EdgeKeys: [][]byte{edges[0].ReferenceKey()}},
			{EdgeKeys: [][]byte{edges[1].ReferenceKey()}},
		},
	})
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("multi-edge flange went sick: %+v", pf.Health())
	}
	body := fs.Result()[0]
	if !body.IsSolid() || !ops.Validate(body).Valid {
		t.Fatalf("multi-edge flange is not a valid solid: %+v", ops.Validate(body))
	}
	// One bend per flanged edge.
	fl := pf.Definition().(*SheetMetalFlangeFeature)
	if got := len(fl.Placements()); got != 2 {
		t.Errorf("Placements() = %d, want 2 (one per edge)", got)
	}
	if got := len(fl.BendSpecs(th)); got != 2 {
		t.Errorf("BendSpecs len = %d, want 2 (one bend per edge for the flat pattern)", got)
	}
	// Two opposite flanges add ~twice a single flange's band.
	single := sheetWithFlange(t, side, r, h)
	singleAdded := smSolidVolume(single[0]) - side*side*th
	twoAdded := smSolidVolume(body) - side*side*th
	if math.Abs(twoAdded-2*singleAdded) > 0.05*singleAdded {
		t.Errorf("two-edge flange added %g cm³, want ~2× the single-edge %g", twoAdded, singleAdded)
	}
}

// TestFlangeEdgeSetsRoundTrip persists and restores a multi-edge flange's edge-set collection,
// including a per-set width; a legacy single-edge recipe (no edgeSets) reads back unchanged.
func TestFlangeEdgeSetsRoundTrip(t *testing.T) {
	def := &SheetMetalFlangeDefinition{
		Height: func() float64 { return 1 },
		EdgeSets: []FlangeEdgeSet{
			{EdgeKeys: [][]byte{[]byte("e1"), []byte("e2")}},
			{EdgeKeys: [][]byte{[]byte("e3")}, Width: FlangeWidth{Type: WidthCentered, Width: constFloat(2)}},
		},
	}
	data := serializeSheetMetalFlange(def)
	if len(data.EdgeSets) != 2 || len(data.EdgeSets[0].Edges) != 2 || data.EdgeSets[1].Width == nil {
		t.Fatalf("persisted edge sets = %+v, want 2 sets (2 edges / 1 edge with a width)", data.EdgeSets)
	}
	restored, err := restoreSheetMetalFlange(NewPartFeatures(nil), data)
	if err != nil {
		t.Fatalf("restore: %v", err)
	}
	rdef := restored.Definition().(*SheetMetalFlangeFeature).Definition()
	if len(rdef.EdgeSets) != 2 || len(rdef.EdgeSets[0].EdgeKeys) != 2 ||
		string(rdef.EdgeSets[1].EdgeKeys[0]) != "e3" || rdef.EdgeSets[1].Width.Type != WidthCentered {
		t.Errorf("restored edge sets = %+v, want the two sets back with the per-set width", rdef.EdgeSets)
	}

	legacy := serializeSheetMetalFlange(&SheetMetalFlangeDefinition{EdgeKey: []byte("only"), Height: func() float64 { return 1 }})
	if legacy.EdgeSets != nil {
		t.Errorf("a single-edge flange persisted edgeSets %+v, want none", legacy.EdgeSets)
	}
}
