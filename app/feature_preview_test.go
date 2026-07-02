// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/sketch"
	"oblikovati.org/renderer"
)

// addSquareSketch adds a side×side square sketch on the XY plane with its lower-left corner
// at (ox,oy) and returns the profile handle for its single region.
func addSquareSketch(def *compdef.PartComponentDefinition, ox, oy, side float64) ProfileHandle {
	sk := def.Sketches().Add(sketch.XYPlane())
	c0 := sk.Points().Add(math.P2(ox, oy))
	c1 := sk.Points().Add(math.P2(ox+side, oy))
	c2 := sk.Points().Add(math.P2(ox+side, oy+side))
	c3 := sk.Points().Add(math.P2(ox, oy+side))
	sk.Lines().Add(c0, c1)
	sk.Lines().Add(c1, c2)
	sk.Lines().Add(c2, c3)
	sk.Lines().Add(c3, c0)
	return ProfileHandle{Sketch: sk, ProfileIndex: 0}
}

// TestFeaturePreviewCutIsRed builds a base solid, then previews an extrude-cut that bores a
// hole through it. Removing material drops the volume, so the live preview is RED.
func TestFeaturePreviewCutIsRed(t *testing.T) {
	s, base := newPartWithSquare(t, 4)
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)

	// Commit a 4×4×4 base solid.
	s.SetPicker(stubPicker{sel: base})
	e0 := NewExtrudeTool()
	s.StartTool(e0)
	s.Click(0, 0)
	e0.SetOperation(ops.NewBody)
	e0.SetDistance(4)
	if err := s.OK(); err != nil {
		t.Fatalf("base extrude OK: %v", err)
	}
	baseVol := totalVolume(s.VisibleBodies())

	// Preview a 2×2 through-cut centered in the base.
	cut := addSquareSketch(def, 1, 1, 2)
	s.SetPicker(stubPicker{sel: cut})
	e1 := NewExtrudeTool()
	s.StartTool(e1)
	s.Click(0, 0)
	e1.SetOperation(ops.Cut)
	e1.SetDistance(4)

	null := &renderer.NullBackend{}
	s.RenderFrame(null)
	prev := previewItems(null.LastFrame())
	if len(prev) == 0 {
		t.Fatal("no cut preview items")
	}
	if !sameHue(prev[0].Color, previewRemoveColor) {
		t.Errorf("cut preview color = %v, want red %v", prev[0].Color, previewRemoveColor)
	}
	// The preview is non-destructive: the committed base is untouched.
	if got := totalVolume(s.VisibleBodies()); got != baseVol {
		t.Errorf("preview mutated the model volume: %v → %v", baseVol, got)
	}
}

// TestPreviewDeltaHelpers covers the dress-up delta path (boolean difference) used when a
// feature has no separable tool body.
func TestPreviewDeltaHelpers(t *testing.T) {
	s := extrudedBox(t, 4, 4)
	base := s.VisibleBodies()
	// Identical body sets differ by nothing → no delta solid to show.
	if items := deltaPreviewItems(base, base); items != nil {
		t.Errorf("identical bodies should yield no delta items, got %d", len(items))
	}
	// The box volume is ~64 cm³ (4×4×4).
	if v := totalVolume(base); v < 63 || v > 65 {
		t.Errorf("box volume = %v, want ~64", v)
	}
}

// TestDressUpPreviewIsTheDeltaSolid is the regression guard for the "flooding" bug: a chamfer
// on one edge of a 4×4×4 box must preview only the small wedge of material it removes (a red
// solid ghost), NOT the whole part. We assert the preview's footprint is thin in X and Y (the
// wedge spans one edge) — a flooded preview would span the full 4-unit box.
func TestDressUpPreviewIsTheDeltaSolid(t *testing.T) {
	s := extrudedBox(t, 4, 4)
	edge := firstVerticalEdge(t, s.VisibleBodies()[0])

	ch := NewChamferTool()
	s.StartTool(ch)
	ch.Pick(s, EdgeHandle{Edge: edge})
	ch.SetDistance(1)

	null := &renderer.NullBackend{}
	s.RenderFrame(null)
	prev := previewItems(null.LastFrame())
	if len(prev) == 0 {
		t.Fatal("chamfer preview produced no delta solid")
	}
	if !sameHue(prev[0].Color, previewRemoveColor) {
		t.Errorf("chamfer (removes material) preview color = %v, want red %v", prev[0].Color, previewRemoveColor)
	}
	dx, dy, _ := previewExtent(prev)
	if dx > 2 || dy > 2 { // the wedge is ~1×1 in cross-section; flooding would be the full 4×4
		t.Errorf("chamfer delta footprint = %.2f×%.2f; expected a thin edge wedge (flooding regression)", dx, dy)
	}
}

// sameHue reports whether two colors share RGB (ignoring alpha — the fill bakes opacity into
// alpha, so only the hue is asserted).
func sameHue(a, b [4]float32) bool {
	return a[0] == b[0] && a[1] == b[1] && a[2] == b[2]
}

// previewExtent returns the X/Y/Z span of all preview vertex positions.
func previewExtent(items []renderer.DrawItem) (dx, dy, dz float64) {
	first := true
	var minX, minY, minZ, maxX, maxY, maxZ float64
	for _, it := range items {
		for _, p := range it.Positions {
			if first {
				minX, minY, minZ = p.X, p.Y, p.Z
				maxX, maxY, maxZ = p.X, p.Y, p.Z
				first = false
				continue
			}
			minX, maxX = min(minX, p.X), max(maxX, p.X)
			minY, maxY = min(minY, p.Y), max(maxY, p.Y)
			minZ, maxZ = min(minZ, p.Z), max(maxZ, p.Z)
		}
	}
	return maxX - minX, maxY - minY, maxZ - minZ
}

// firstVerticalEdge returns a body edge running mostly along Z.
func firstVerticalEdge(t *testing.T, b *topo.Body) *topo.Edge {
	t.Helper()
	for _, e := range b.Edges() {
		pts := ops.TessellateEdge(e, ops.DefaultQuality())
		if len(pts) >= 2 {
			dz := pts[0].Z - pts[len(pts)-1].Z
			if dz < 0 {
				dz = -dz
			}
			if dz > 1 {
				return e
			}
		}
	}
	t.Fatal("no vertical edge found")
	return nil
}

// TestFeaturePreviewEmptyUntilReady asserts no preview is drawn before the tool has enough
// input to commit (no region picked yet).
func TestFeaturePreviewEmptyUntilReady(t *testing.T) {
	s, _ := newPartWithSquare(t, 2)
	ext := NewExtrudeTool()
	s.StartTool(ext)
	if items := s.ToolPreview(); items != nil {
		t.Errorf("preview before any profile picked = %d items, want none", len(items))
	}
}

// TestFeaturePreviewIsAGhostSolid checks the tool-body preview is one translucent mesh of the
// whole feature solid (the prism), accompanied by its opaque feature edges.
func TestFeaturePreviewIsAGhostSolid(t *testing.T) {
	s, profile := newPartWithSquare(t, 2)
	s.SetPicker(stubPicker{sel: profile})
	ext := NewExtrudeTool()
	s.StartTool(ext)
	s.Click(0, 0)
	ext.SetOperation(ops.NewBody)
	ext.SetDistance(5)

	null := &renderer.NullBackend{}
	s.RenderFrame(null)
	prev := previewItems(null.LastFrame())
	if len(prev) != 1 {
		t.Errorf("preview fill items = %d, want 1 merged ghost mesh", len(prev))
	}
	if previewTriangles(prev) < 12 { // a box prism has ≥12 triangles
		t.Errorf("preview has %d triangles, want the whole prism (≥12)", previewTriangles(prev))
	}
	// The ghost is accompanied by opaque feature-edge lines.
	if edgeLines(null.LastFrame()) == 0 {
		t.Error("preview has no feature edge lines")
	}
}

// edgeLines counts the opaque (alpha==1) on-top line items — the preview's feature edges.
func edgeLines(frame renderer.DrawList) int {
	n := 0
	for _, it := range frame.Items {
		if it.Primitive == renderer.Lines && it.OnTop && it.Color[3] == 1 {
			n++
		}
	}
	return n
}

// TestFaceSigHelpers pins the surface-signature comparison helpers used by changedFacePreview
// to tell a feature's NEW faces from the base body's faces. A surface and its reverse share a
// signature (signNorm), the freeform branch compares area+centroid (exercising maxVal), and a
// different kind/anchor/radius never matches.
func TestFaceSigHelpers(t *testing.T) {
	// signNorm flips a vector so its dominant component is positive, so opposite normals match.
	up, down := math.V3(0, 0, 1), math.V3(0, 0, -1)
	if !signNorm(up).IsEqualTo(signNorm(down), 1e-9) {
		t.Errorf("signNorm(%v) != signNorm(%v)", up, down)
	}
	// A cylinder axis foot is invariant to where along the axis the origin sits.
	a := axisFoot(math.P3(0, 0, 0), math.V3(0, 0, 1))
	b := axisFoot(math.P3(0, 0, 7), math.V3(0, 0, 1))
	if !a.IsEqualTo(b, 1e-9) {
		t.Errorf("axisFoot not invariant along axis: %v vs %v", a, b)
	}
	if got := max(2.0, 5.0); got != 5 {
		t.Errorf("max(2,5) = %g, want 5", got)
	}
	if got := stdmath.Abs(-3); got != 3 {
		t.Errorf("stdmath.Abs(-3) = %g, want 3", got)
	}

	// Two freeform sigs (kind 0) with equal area+centroid match; differing area does not.
	c := math.P3(1, 2, 3)
	free := faceSig{kind: 0, centroid: c, area: 10}
	if !sigEqual(free, faceSig{kind: 0, centroid: c, area: 10}) {
		t.Error("identical freeform sigs should match")
	}
	if sigEqual(free, faceSig{kind: 0, centroid: c, area: 12}) {
		t.Error("freeform sigs with different area should not match")
	}
	// A plane and a cylinder (different kind) never match.
	if sigEqual(faceSig{kind: 1}, faceSig{kind: 2}) {
		t.Error("different surface kinds should not match")
	}
	// Same-kind cylinders with different radii do not match.
	cyl := faceSig{kind: 2, dir: math.V3(0, 0, 1), anchor: c, r1: 1}
	if sigEqual(cyl, faceSig{kind: 2, dir: math.V3(0, 0, 1), anchor: c, r1: 2}) {
		t.Error("cylinders with different radii should not match")
	}
}
