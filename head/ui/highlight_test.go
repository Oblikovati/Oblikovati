// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati/app"
	"oblikovati/kernel/geom"
	"oblikovati/kernel/topo"
	"oblikovati/math"
	"oblikovati/renderer"
)

// tinyBody builds the smallest valid body so its ObjectID can be matched in a draw list.
func tinyBody() *topo.Body {
	bld := topo.NewBuilder(true, topo.NewLineage(topo.Tok("t", "body", 0)))
	v := bld.AddVertex(math.P3(0, 0, 0), topo.NewLineage(topo.Tok("t", "vertex", 0)))
	e := bld.AddEdge(geom.NewLineSegment(math.P3(0, 0, 0), math.P3(1, 0, 0)), v, v, topo.NewLineage(topo.Tok("t", "edge", 0)))
	plane, _ := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	bld.AddFace(plane, topo.NewLineage(topo.Tok("t", "face", 0)), topo.OuterLoop(topo.Fwd(e)))
	return bld.Build()
}

func TestHighlightSelectionRecolorsOnlyTheSelectedBody(t *testing.T) {
	b := tinyBody()
	other := b.ID() + 1
	base := [4]float32{0.5, 0.5, 0.5, 1}
	list := renderer.DrawList{Items: []renderer.DrawItem{
		{ObjectID: b.ID(), Color: base},
		{ObjectID: other, Color: base},
	}}

	got := highlightSelection(list, app.BodyHandle{Body: b}, nil)

	if got.Items[0].Color != selectionHighlight {
		t.Errorf("selected body item not highlighted: %v", got.Items[0].Color)
	}
	if got.Items[1].Color != base {
		t.Errorf("unselected body item was recolored: %v", got.Items[1].Color)
	}
}

func TestHighlightSelectionNoSelectionLeavesColors(t *testing.T) {
	base := [4]float32{0.5, 0.5, 0.5, 1}
	list := renderer.DrawList{Items: []renderer.DrawItem{{ObjectID: 7, Color: base}}}
	if got := highlightSelection(list, nil, nil); got.Items[0].Color != base {
		t.Errorf("nil selection recolored an item: %v", got.Items[0].Color)
	}
}

func TestBodyHighlightIDsForFeatureCoversWholePart(t *testing.T) {
	b := tinyBody()
	f := app.FeatureHandle{} // a feature selection highlights every part body
	ids := bodyHighlightIDs(f, []*topo.Body{b})
	if !ids[b.ID()] || len(ids) != 1 {
		t.Errorf("feature selection should highlight all part bodies, got %v", ids)
	}
}
