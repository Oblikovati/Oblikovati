// SPDX-License-Identifier: GPL-2.0-only

package viewport

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/renderer"
)

func triItem(z float64) renderer.DrawItem {
	return renderer.DrawItem{
		Primitive: renderer.Triangles,
		Positions: []math.Point3{math.P3(0, 0, z), math.P3(1, 0, z), math.P3(0, 1, z)},
		Normals:   []math.Vector3{math.V3(0, 0, 1), math.V3(0, 0, 1), math.V3(0, 0, 1)},
		Indices:   []int{0, 1, 2},
		Color:     [4]float32{1, 0, 0, 1},
	}
}

func lineItem() renderer.DrawItem {
	return renderer.DrawItem{
		Primitive: renderer.Lines,
		Positions: []math.Point3{math.P3(0, 0, 0), math.P3(1, 1, 1)},
		Indices:   []int{0, 1},
		Color:     [4]float32{0, 1, 0, 1},
	}
}

func TestFlattenSplitsByPrimitive(t *testing.T) {
	m := Flatten(renderer.DrawList{Items: []renderer.DrawItem{triItem(0), lineItem()}})
	if m.TriVCount != 3 || len(m.TriIndices) != 3 {
		t.Errorf("triangles: vcount=%d idx=%d, want 3/3", m.TriVCount, len(m.TriIndices))
	}
	if m.LineVCount != 2 || len(m.LineIndices) != 2 {
		t.Errorf("lines: vcount=%d idx=%d, want 2/2", m.LineVCount, len(m.LineIndices))
	}
	if len(m.TriVerts) != 3*VertexFloats || len(m.LineVerts) != 2*VertexFloats {
		t.Errorf("vertex float counts wrong: tri=%d line=%d", len(m.TriVerts), len(m.LineVerts))
	}
}

// TestFlattenRebasesIndicesPerItem verifies a second triangle item's indices are
// offset by the running vertex count (so two items share one buffer correctly).
func TestFlattenRebasesIndicesPerItem(t *testing.T) {
	m := Flatten(renderer.DrawList{Items: []renderer.DrawItem{triItem(0), triItem(5)}})
	if m.TriVCount != 6 {
		t.Fatalf("vcount = %d, want 6", m.TriVCount)
	}
	// The second item's indices {0,1,2} must have been rebased to {3,4,5}.
	want := []uint32{0, 1, 2, 3, 4, 5}
	for i, v := range want {
		if m.TriIndices[i] != v {
			t.Errorf("TriIndices = %v, want %v", m.TriIndices, want)
			break
		}
	}
}

func TestFlattenInterleavesPositionNormalColor(t *testing.T) {
	m := Flatten(renderer.DrawList{Items: []renderer.DrawItem{triItem(2)}})
	// First vertex: pos (0,0,2), normal (0,0,1), color (1,0,0,1), then PBR fields + mode
	// (all zero for this item: metallic, roughness, emissive.rgb, mode=ShadeNone).
	got := m.TriVerts[:VertexFloats]
	want := []float32{0, 0, 2, 0, 0, 1, 1, 0, 0, 1, 0, 0, 0, 0, 0, 0}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("interleaved vertex = %v, want %v", got, want)
		}
	}
}

// TestFlattenCarriesShadingAndMaterial checks the PBR/shading-mode slots are packed from the
// draw item (so the native pipeline can pick the per-body shader).
func TestFlattenCarriesShadingAndMaterial(t *testing.T) {
	item := renderer.DrawItem{
		Primitive: renderer.Triangles,
		Positions: []math.Point3{math.P3(0, 0, 0)},
		Color:     [4]float32{0.3, 0.4, 0.5, 1},
		Metallic:  0.8, Roughness: 0.25, Emissive: [3]float32{0.1, 0, 0},
		Shading: renderer.ShadePBR,
	}
	m := Flatten(renderer.DrawList{Items: []renderer.DrawItem{item}})
	v := m.TriVerts[:VertexFloats]
	if v[10] != 0.8 || v[11] != 0.25 || v[12] != 0.1 || v[15] != float32(renderer.ShadePBR) {
		t.Errorf("material/mode slots = %v, want metallic .8, roughness .25, emissive.r .1, mode %d",
			v[10:], renderer.ShadePBR)
	}
}

// TestFlattenEmitsPerVertexColors checks that DrawItem.Colors overrides the single Color
// per vertex — the client-graphics heatmap path. Vertex 0 stays red, vertex 1 turns blue.
func TestFlattenEmitsPerVertexColors(t *testing.T) {
	item := triItem(0)
	item.Colors = [][4]float32{{1, 0, 0, 1}, {0, 0, 1, 1}, {0, 1, 0, 1}}
	m := Flatten(renderer.DrawList{Items: []renderer.DrawItem{item}})
	// color is floats 6..10 of each vertex.
	v0 := m.TriVerts[6:10]
	v1 := m.TriVerts[VertexFloats+6 : VertexFloats+10]
	if v0[2] != 0 || v1[2] != 1 {
		t.Errorf("per-vertex colors not applied: v0=%v v1=%v", v0, v1)
	}
}

// TestFlattenFallsBackToBroadcastColor checks that without Colors the single Color still
// broadcasts to every vertex (legacy behavior preserved).
func TestFlattenFallsBackToBroadcastColor(t *testing.T) {
	m := Flatten(renderer.DrawList{Items: []renderer.DrawItem{triItem(0)}})
	for v := 0; v < 3; v++ {
		c := m.TriVerts[v*VertexFloats+6 : v*VertexFloats+10]
		if c[0] != 1 || c[1] != 0 || c[2] != 0 || c[3] != 1 {
			t.Fatalf("vertex %d color = %v, want broadcast red", v, c)
		}
	}
}

// TestFlattenRoutesOnTopToTopStreams checks that DrawItem.OnTop triangles and lines route
// to the depth-disabled on-top streams rather than the regular ones (PBI-067).
func TestFlattenRoutesOnTopToTopStreams(t *testing.T) {
	tri := triItem(0)
	tri.OnTop = true
	ln := lineItem()
	ln.OnTop = true
	m := Flatten(renderer.DrawList{Items: []renderer.DrawItem{tri, ln}})
	if m.TopTriVCount != 3 || len(m.TopTriIndices) != 3 {
		t.Errorf("on-top triangles: vcount=%d idx=%d, want 3/3", m.TopTriVCount, len(m.TopTriIndices))
	}
	if m.TopLineVCount != 2 || len(m.TopLineIndices) != 2 {
		t.Errorf("on-top lines: vcount=%d idx=%d, want 2/2", m.TopLineVCount, len(m.TopLineIndices))
	}
	// They must NOT have leaked into the regular streams.
	if m.TriVCount != 0 || m.LineVCount != 0 {
		t.Errorf("on-top items leaked into regular streams: tri=%d line=%d", m.TriVCount, m.LineVCount)
	}
}

// TestFlattenSplitsSolidOnTopToTail reproduces issue #1489: an OPAQUE, normal-bearing on-top
// triangle (a client-graphics solid glyph — support cube / load arrow) must be held to the TAIL
// of the on-top triangle stream (at/after TopTriSolidFirst) so the native pass can clear depth and
// draw it depth-tested + lit, making it self-occlude. The flat on-top overlays — a translucent
// ghost/highlight and a normal-less flood plot — must stay in the HEAD (before TopTriSolidFirst),
// drawn flat with the depth test disabled, exactly as before.
func TestFlattenSplitsSolidOnTopToTail(t *testing.T) {
	ghost := triItem(0) // translucent on-top overlay (a feature-preview ghost / face highlight)
	ghost.OnTop, ghost.Opacity = true, 0.5
	flood := triItem(1) // normal-less heatmap flood plot (opaque, but flat data, not a solid)
	flood.OnTop, flood.Normals = true, nil
	glyph := triItem(2) // opaque, normal-bearing solid glyph — the #1489 case
	glyph.OnTop = true

	m := Flatten(renderer.DrawList{Items: []renderer.DrawItem{ghost, flood, glyph}})

	// All three are on-top triangles, so all nine indices land in the on-top stream.
	if got := len(m.TopTriIndices); got != 9 {
		t.Fatalf("TopTriIndices = %d, want 9 (3 items × 3)", got)
	}
	// The two flat overlays (ghost, flood) form the head; the solid glyph is the tail.
	if m.TopTriSolidFirst != 6 {
		t.Errorf("TopTriSolidFirst = %d, want 6 (the glyph begins after the two flat overlays)", m.TopTriSolidFirst)
	}
	// Nothing leaked into the regular (depth-tested-against-model) triangle stream.
	if m.TriVCount != 0 {
		t.Errorf("on-top items leaked into the regular tri stream: vcount=%d", m.TriVCount)
	}
}

// TestIsSolidOnTopTriangle pins the discriminator that routes a glyph to the self-occluding tail:
// only an OPAQUE, normal-bearing, on-top TRIANGLE qualifies (#1489).
func TestIsSolidOnTopTriangle(t *testing.T) {
	base := triItem(0)
	base.OnTop = true
	cases := []struct {
		name string
		mut  func(*renderer.DrawItem)
		want bool
	}{
		{"opaque glyph with normals", func(i *renderer.DrawItem) {}, true},
		{"not on top", func(i *renderer.DrawItem) { i.OnTop = false }, false},
		{"translucent (Opacity)", func(i *renderer.DrawItem) { i.Opacity = 0.5 }, false},
		{"translucent (color alpha)", func(i *renderer.DrawItem) { i.Color = [4]float32{1, 0, 0, 0.4} }, false},
		{"no normals (flood plot)", func(i *renderer.DrawItem) { i.Normals = nil }, false},
		{"a line, not a triangle", func(i *renderer.DrawItem) { i.Primitive = renderer.Lines }, false},
	}
	for _, c := range cases {
		item := base
		c.mut(&item)
		if got := isSolidOnTopTriangle(item); got != c.want {
			t.Errorf("%s: isSolidOnTopTriangle = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestFlattenLinesTolerateMissingNormals(t *testing.T) {
	m := Flatten(renderer.DrawList{Items: []renderer.DrawItem{lineItem()}})
	// Normal slot (floats 3..6) should be zero, not a panic / garbage.
	if m.LineVerts[3] != 0 || m.LineVerts[4] != 0 || m.LineVerts[5] != 0 {
		t.Errorf("missing normals should default to zero, got %v", m.LineVerts[3:6])
	}
}
