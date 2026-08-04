// SPDX-License-Identifier: GPL-2.0-only

package viewport

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/renderer"
)

// Stroked lines (#2015 line weight): a line item wider than a hairline is expanded into quads the
// shader offsets in screen space. These pin the routing and the per-corner encoding the shader
// reads — get the side sign wrong at the far end and every segment renders as a bow tie.

// wideLineItem is one segment from (0,0,0) to (4,0,0) stroked at w pixels.
func wideLineItem(w float32) renderer.DrawItem {
	return renderer.DrawItem{
		Primitive: renderer.Lines,
		Positions: []math.Point3{math.P3(0, 0, 0), math.P3(4, 0, 0)},
		Indices:   []int{0, 1},
		Color:     [4]float32{1, 0, 0, 1},
		Width:     w,
	}
}

// corner reads the i-th expanded vertex out of a wide-line stream.
type corner struct {
	at, other [3]float32
	side      float32
	width     float32
}

func cornerAt(verts []float32, i int) corner {
	v := verts[i*VertexFloats : (i+1)*VertexFloats]
	return corner{
		at:    [3]float32{v[0], v[1], v[2]},
		other: [3]float32{v[3], v[4], v[5]},
		side:  v[10],
		width: v[11],
	}
}

// TestWideLineRoutesAwayFromTheHairlineStream: a stroked line must not fall through to the line
// pipeline, which has no width and would draw it as a hairline.
func TestWideLineRoutesAwayFromTheHairlineStream(t *testing.T) {
	m := Flatten(renderer.DrawList{Items: []renderer.DrawItem{wideLineItem(4)}})
	if m.LineVCount != 0 {
		t.Errorf("wide line leaked into the hairline stream (%d verts)", m.LineVCount)
	}
	if m.WideLineVCount != 4 {
		t.Errorf("WideLineVCount = %d, want 4 (one quad)", m.WideLineVCount)
	}
	if len(m.WideLineIndices) != 6 {
		t.Errorf("WideLineIndices = %d, want 6 (two triangles)", len(m.WideLineIndices))
	}
}

// TestOnTopWideLineUsesTheOnTopStream keeps the depth-tested and depth-ignoring lanes distinct,
// as they are for hairlines.
func TestOnTopWideLineUsesTheOnTopStream(t *testing.T) {
	item := wideLineItem(4)
	item.OnTop = true
	m := Flatten(renderer.DrawList{Items: []renderer.DrawItem{item}})
	if m.TopWideLineVCount != 4 || m.WideLineVCount != 0 {
		t.Errorf("on-top wide line went to the wrong stream (top=%d, depth-tested=%d)",
			m.TopWideLineVCount, m.WideLineVCount)
	}
}

// TestHairlineStaysOnTheLinePipeline: width 0 and width 1 are hairlines and must keep the cheap
// two-vertices-per-segment path — expanding every line would triple the vertex cost of a dense
// DWG import for no visible gain.
func TestHairlineStaysOnTheLinePipeline(t *testing.T) {
	for _, w := range []float32{0, 1} {
		m := Flatten(renderer.DrawList{Items: []renderer.DrawItem{wideLineItem(w)}})
		if m.LineVCount != 2 || m.WideLineVCount != 0 {
			t.Errorf("width %v: line=%d wide=%d, want the hairline stream", w, m.LineVCount, m.WideLineVCount)
		}
	}
}

// TestWideTrianglesAreNotExpanded: Width is a stroke attribute, so a triangle item carrying one
// still renders as a filled triangle.
func TestWideTrianglesAreNotExpanded(t *testing.T) {
	item := triItem(0)
	item.Width = 8
	m := Flatten(renderer.DrawList{Items: []renderer.DrawItem{item}})
	if m.TriVCount != 3 || m.WideLineVCount != 0 {
		t.Errorf("triangle with a Width was expanded (tri=%d wide=%d)", m.TriVCount, m.WideLineVCount)
	}
}

// TestWideLineCornersEncodeBothEndpoints: every corner must carry its own endpoint AND the
// opposite one, because the shader derives the stroke direction from the pair.
func TestWideLineCornersEncodeBothEndpoints(t *testing.T) {
	m := Flatten(renderer.DrawList{Items: []renderer.DrawItem{wideLineItem(6)}})
	a, b := [3]float32{0, 0, 0}, [3]float32{4, 0, 0}
	for i, want := range []corner{
		{at: a, other: b}, {at: a, other: b}, {at: b, other: a}, {at: b, other: a},
	} {
		got := cornerAt(m.WideLineVerts, i)
		if got.at != want.at || got.other != want.other {
			t.Errorf("corner %d: at=%v other=%v, want at=%v other=%v", i, got.at, got.other, want.at, want.other)
		}
		if got.width != 6 {
			t.Errorf("corner %d: width=%v, want 6", i, got.width)
		}
	}
}

// TestFarEndCornersMirrorTheSide is the bow-tie guard. The shader takes the stroke direction as
// (other - at), which points the opposite way at the far end, flipping the perpendicular it
// offsets along. The far corners must therefore carry the opposite sign to land on the same side
// of the stroke; if they do not, each segment renders as an hourglass pinched at its middle.
func TestFarEndCornersMirrorTheSide(t *testing.T) {
	m := Flatten(renderer.DrawList{Items: []renderer.DrawItem{wideLineItem(4)}})
	near0, near1 := cornerAt(m.WideLineVerts, 0).side, cornerAt(m.WideLineVerts, 1).side
	far0, far1 := cornerAt(m.WideLineVerts, 2).side, cornerAt(m.WideLineVerts, 3).side
	if near0 == near1 || far0 == far1 {
		t.Fatalf("both corners of one end share a side (near %v/%v, far %v/%v)", near0, near1, far0, far1)
	}
	// Corner 0 and corner 3 are the two ends of the SAME side of the stroke, so with the direction
	// reversed at the far end their encoded signs must differ.
	if near0 == far1 {
		t.Errorf("far corner did not mirror its side: near=%v far=%v; the quad is a bow tie", near0, far1)
	}
}

// TestWideLineExpandsEverySegment: a polyline's segments each become their own quad, with indices
// rebased so the second quad does not reference the first's vertices.
func TestWideLineExpandsEverySegment(t *testing.T) {
	item := renderer.DrawItem{
		Primitive: renderer.Lines,
		Positions: []math.Point3{math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(2, 0, 0)},
		Indices:   []int{0, 1, 1, 2},
		Width:     3,
	}
	m := Flatten(renderer.DrawList{Items: []renderer.DrawItem{item}})
	if m.WideLineVCount != 8 || len(m.WideLineIndices) != 12 {
		t.Fatalf("two segments gave %d verts / %d indices, want 8 / 12", m.WideLineVCount, len(m.WideLineIndices))
	}
	if m.WideLineIndices[6] < 4 {
		t.Errorf("second quad's indices were not rebased: %v", m.WideLineIndices[6:])
	}
}

// TestWideLineIgnoresAMalformedItem: an odd or out-of-range index list must not panic the render
// loop — a dropped segment is recoverable, a crashed frame is not.
func TestWideLineIgnoresAMalformedItem(t *testing.T) {
	item := renderer.DrawItem{
		Primitive: renderer.Lines,
		Positions: []math.Point3{math.P3(0, 0, 0)},
		Indices:   []int{0, 9, 0}, // out of range, then a dangling index
		Width:     3,
	}
	m := Flatten(renderer.DrawList{Items: []renderer.DrawItem{item}})
	if m.WideLineVCount != 0 {
		t.Errorf("malformed item produced %d vertices, want none", m.WideLineVCount)
	}
}

// TestHiddenEdgeKeepsItsLaneOverWidth: the hidden lane exists for its reversed depth test, which
// the stroked lanes have no equivalent of. A hidden edge that also carries a width must stay
// hidden and draw thin, rather than becoming a plainly visible stroked edge.
func TestHiddenEdgeKeepsItsLaneOverWidth(t *testing.T) {
	item := wideLineItem(6)
	item.Hidden = true
	m := Flatten(renderer.DrawList{Items: []renderer.DrawItem{item}})
	if m.HidVCount != 2 || m.WideLineVCount != 0 {
		t.Errorf("hidden wide edge went to hid=%d wide=%d, want the hidden lane", m.HidVCount, m.WideLineVCount)
	}
}

// TestOnTopStillBeatsHidden pins the precedence the single routing switch had before the stroked
// lanes were added: an item flagged both draws on top.
func TestOnTopStillBeatsHidden(t *testing.T) {
	item := wideLineItem(0)
	item.OnTop, item.Hidden = true, true
	m := Flatten(renderer.DrawList{Items: []renderer.DrawItem{item}})
	if m.TopLineVCount != 2 || m.HidVCount != 0 {
		t.Errorf("on-top hidden line went to top=%d hid=%d, want the on-top lane", m.TopLineVCount, m.HidVCount)
	}
}
