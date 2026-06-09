// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/model/sketch"
)

// Loft AREA-GRAPH matrix (Slice 5, kLoftWithAreaGraphSections): the cross-section area along the
// loft follows a graph of (t, scale) stops, with the end sections pinned to scale 1. A mid stop > 1
// bulges the middle (a barrel by area); < 1 waists it.

// areaGraphCircles lofts two equal circles (r=2, z 0 and 4) under the given area graph, asserting a
// valid solid.
func areaGraphCircles(t *testing.T, graph []LoftAreaStop) *topo.Body {
	t.Helper()
	fs := NewPartFeatures(nil, nil)
	secs := []LoftSection{sec(circleOn(sketch.XYPlane(), 2)), sec(circleOn(planeAtZ(4), 2))}
	pf := NewLoftFeatures(fs).AddGuided(secs, false, ops.NewBody, LoftEnd{}, LoftEnd{}, LoftGuideSet{AreaGraph: graph})
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("area-graph loft went sick: %+v", pf.Health())
	}
	b := fs.Result()[0]
	if r := ops.Validate(b); !r.Valid || !b.IsSolid() {
		t.Fatalf("area-graph loft not a valid solid: valid=%v solid=%v issues=%v", r.Valid, b.IsSolid(), capIssues3(r.Issues))
	}
	return b
}

// TestLoftAreaGraphBulges: a mid stop of scale 2 doubles the mid cross-section's area, so the mid
// radius grows by √2 (≈2.83) — the body bulges past the section radius, holding more volume.
func TestLoftAreaGraphBulges(t *testing.T) {
	plain := ops.BodyGeometryProperties(areaGraphCircles(t, nil), ops.DefaultQuality()).Volume
	b := areaGraphCircles(t, []LoftAreaStop{{T: 0.5, Scale: 2}})
	if maxX := float64(b.RangeBox().Max.X); maxX < 2.5 {
		t.Errorf("area graph did not bulge: max x = %.3f, want ≈2.83 (√2·2)", maxX)
	}
	if v := ops.BodyGeometryProperties(b, ops.DefaultQuality()).Volume; v <= plain*1.1 {
		t.Errorf("area-graph bulge did not add volume: %.3f vs plain %.3f", v, plain)
	}
}

// TestLoftAreaGraphWaists: a mid stop of scale 0.25 quarters the mid area (mid radius halves), so
// the body waists in and holds less than the plain cylinder.
func TestLoftAreaGraphWaists(t *testing.T) {
	plain := ops.BodyGeometryProperties(areaGraphCircles(t, nil), ops.DefaultQuality()).Volume
	waisted := ops.BodyGeometryProperties(areaGraphCircles(t, []LoftAreaStop{{T: 0.5, Scale: 0.25}}), ops.DefaultQuality()).Volume
	if waisted >= plain {
		t.Errorf("area-graph waist did not reduce volume: %.3f vs plain %.3f", waisted, plain)
	}
}

// TestLoftAreaGraphKeepsEnds: the end sections are pinned to scale 1, so the body still spans the
// section heights (z 0→4) and the end radius is unchanged (the graph only resizes the interior).
func TestLoftAreaGraphKeepsEnds(t *testing.T) {
	bb := areaGraphCircles(t, []LoftAreaStop{{T: 0.5, Scale: 2}}).RangeBox()
	if z0, z1 := float64(bb.Min.Z), float64(bb.Max.Z); z0 < -1e-6 || z0 > 1e-6 || z1 < 4-1e-6 || z1 > 4+1e-6 {
		t.Errorf("area graph moved the loft ends in z: span [%.4f,%.4f], want [0,4]", z0, z1)
	}
}

// TestLoftAreaGraphRoundTrip: the area-graph stops survive a recipe save/restore and the restored
// loft reports the area-graph type.
func TestLoftAreaGraphRoundTrip(t *testing.T) {
	bottom := circleOn(sketch.XYPlane(), 2)
	top := circleOn(planeAtZ(4), 2)
	idx := sketchList{sks: []*sketch.Sketch{bottom, top}}
	fs := NewPartFeatures(nil, nil)
	NewLoftFeatures(fs).AddGuided(
		[]LoftSection{{Sketch: bottom, ProfileIndex: 0}, {Sketch: top, ProfileIndex: 0}},
		false, ops.NewBody, LoftEnd{}, LoftEnd{}, LoftGuideSet{AreaGraph: []LoftAreaStop{{T: 0.5, Scale: 2}}},
	)
	data, err := fs.MarshalRecipe(idx)
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	fresh := NewPartFeatures(nil, nil)
	if err := fresh.ApplyRecipe(data, idx, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	got := fresh.Item(0).Definition().(*LoftFeature).Definition()
	if len(got.AreaGraph) != 1 || got.AreaGraph[0].T != 0.5 || got.AreaGraph[0].Scale != 2 {
		t.Errorf("area-graph round-trip lost the stop: %+v", got.AreaGraph)
	}
	if got.LoftType() != "area-graph" {
		t.Errorf("LoftType = %q, want \"area-graph\"", got.LoftType())
	}
}
