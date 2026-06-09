// SPDX-License-Identifier: GPL-2.0-only

package renderer

import (
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// TestPassSetForIsTotal asserts every style in the gallery resolves to a deliberate pass set
// (a row in styleTable), so adding a VisualStyle value without a resolver row is caught — the
// resolver must never fall through to the default for a known style (ADR-0023 §1).
func TestPassSetForIsTotal(t *testing.T) {
	for _, opt := range VisualStyleGallery() {
		found := false
		for _, sp := range styleTable {
			if sp.style == opt.Style {
				found = true
			}
		}
		if !found {
			t.Errorf("style %v (%q) has no resolver row", opt.Style, opt.Name)
		}
	}
	if len(VisualStyleGallery()) != 11 {
		t.Errorf("gallery has %d styles, want 11 (Inventor DisplayModeEnum parity)", len(VisualStyleGallery()))
	}
}

// TestPassSetComposition pins the pass set of each style so a resolver-table edit that changes
// what a mode draws is caught.
func TestPassSetComposition(t *testing.T) {
	cases := []struct {
		style VisualStyle
		want  PassSet
	}{
		{Shaded, PassSet{Faces: ShadeFlat, Edges: EdgesNone}},
		{ShadedWithEdges, PassSet{Faces: ShadeFlat, Edges: EdgesAll}},
		{Wireframe, PassSet{Faces: ShadeNone, Edges: EdgesAll}},
		{Realistic, PassSet{Faces: ShadePBR, Edges: EdgesNone}},
		{ShadedWithHiddenEdges, PassSet{Faces: ShadeFlat, Edges: EdgesVisiblePlusHidden}},
		{WireframeWithHiddenEdges, PassSet{Faces: ShadeNone, Edges: EdgesVisiblePlusHidden}},
		{WireframeVisibleOnly, PassSet{Faces: ShadeNone, Edges: EdgesVisible}},
		{Monochrome, PassSet{Faces: ShadeMonochrome, Edges: EdgesNone, Outline: true}},
		{Watercolor, PassSet{Faces: ShadeWatercolor, Edges: EdgesNone, Outline: true}},
		{Illustration, PassSet{Faces: ShadeCel, Edges: EdgesNone, Outline: true}},
		{TechnicalIllustration, PassSet{Faces: ShadeGooch, Edges: EdgesNone, Outline: true}},
	}
	for _, c := range cases {
		if got := PassSetFor(c.style); got != c.want {
			t.Errorf("PassSetFor(%v) = %+v, want %+v", c.style, got, c.want)
		}
	}
}

// TestHiddenEdgeStylesNeedHLR documents which styles require the per-frame hidden-line pass.
func TestHiddenEdgeStylesNeedHLR(t *testing.T) {
	hlr := map[VisualStyle]bool{
		ShadedWithHiddenEdges: true, WireframeWithHiddenEdges: true, WireframeVisibleOnly: true,
	}
	for _, opt := range VisualStyleGallery() {
		if PassSetFor(opt.Style).HidesEdges() != hlr[opt.Style] {
			t.Errorf("style %q HidesEdges=%v, want %v", opt.Name, PassSetFor(opt.Style).HidesEdges(), hlr[opt.Style])
		}
	}
}

// streamCounts classifies a draw list into the four HLR streams.
func streamCounts(l DrawList) (shadedTris, occluders, solidLines, hiddenLines int) {
	for _, it := range l.Items {
		switch {
		case it.Primitive == Triangles && it.Occluder:
			occluders++
		case it.Primitive == Triangles:
			shadedTris++
		case it.Primitive == Lines && it.Hidden:
			hiddenLines++
		case it.Primitive == Lines:
			solidLines++
		}
	}
	return
}

// TestHiddenEdgeStreamComposition pins which of the four streams (shaded faces, depth-only
// occluder, solid visible edges, dashed hidden edges) each edge mode emits — the draw-list
// contract the native HLR pipeline relies on.
func TestHiddenEdgeStreamComposition(t *testing.T) {
	b := box(2, math.V3(0, 0, 0))
	cam := frontCamera()
	cases := []struct {
		style                           VisualStyle
		shaded, occluder, solid, hidden bool
	}{
		{Wireframe, false, false, true, false},              // all edges, nothing hides them
		{WireframeVisibleOnly, false, true, true, false},    // occluder hides the back edges
		{WireframeWithHiddenEdges, false, true, true, true}, // + dashed occluded edges
		{ShadedWithHiddenEdges, true, false, true, true},    // shaded faces occlude + dashed hidden
		{ShadedWithEdges, true, false, true, false},         // faces + all edges, no hidden pass
	}
	for _, c := range cases {
		st, oc, sl, hl := streamCounts(BuildDrawListStyled([]*topo.Body{b}, cam, ops.DefaultQuality(), nil, c.style))
		if (st > 0) != c.shaded || (oc > 0) != c.occluder || (sl > 0) != c.solid || (hl > 0) != c.hidden {
			t.Errorf("%v streams: shaded=%d occluder=%d solid=%d hidden=%d; want presence %v/%v/%v/%v",
				c.style, st, oc, sl, hl, c.shaded, c.occluder, c.solid, c.hidden)
		}
	}
}

// TestNPRModesEmitOutline checks each NPR mode composites a dark "ink" outline (a solid line
// item in outlineColor) over its stylized faces.
func TestNPRModesEmitOutline(t *testing.T) {
	b := box(2, math.V3(0, 0, 0))
	cam := frontCamera()
	for _, style := range []VisualStyle{Monochrome, Watercolor, Illustration, TechnicalIllustration} {
		list := BuildDrawListStyled([]*topo.Body{b}, cam, ops.DefaultQuality(), nil, style)
		var faces, outline int
		for _, it := range list.Items {
			if it.Primitive == Triangles {
				faces++
			}
			if it.Primitive == Lines && it.Color == outlineColor {
				outline++
			}
		}
		if faces == 0 || outline == 0 {
			t.Errorf("%v: faces=%d outlineLines=%d, want both > 0", style, faces, outline)
		}
	}
}

// TestTriangleItemsCarryShading checks the draw list tags surface items with the style's
// shading mode (so the native pipeline can pick the face shader).
func TestTriangleItemsCarryShading(t *testing.T) {
	b := box(2, math.V3(0, 0, 0))
	cam := frontCamera()
	for _, c := range []struct {
		style VisualStyle
		want  Shading
	}{
		{Shaded, ShadeFlat},
		{Realistic, ShadePBR},
		{Monochrome, ShadeMonochrome},
		{Illustration, ShadeCel},
		{TechnicalIllustration, ShadeGooch},
		{Watercolor, ShadeWatercolor},
	} {
		list := BuildDrawListStyled([]*topo.Body{b}, cam, ops.DefaultQuality(), nil, c.style)
		var got Shading = 255
		for _, it := range list.Items {
			if it.Primitive == Triangles {
				got = it.Shading
			}
		}
		if got != c.want {
			t.Errorf("style %q surface shading = %v, want %v", c.style, got, c.want)
		}
	}
}
