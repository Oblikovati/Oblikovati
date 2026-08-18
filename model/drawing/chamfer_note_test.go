// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	stdmath "math"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/subd"
	"oblikovati.org/kernel/topo"
	gmath "oblikovati.org/math"
)

// chamferedSquarePrism builds a prism whose cross-section is a square (half-side 1 cm) with its four
// corners cut by a 45° chamfer of leg c — so each diagonal side face is a real chamfer between two
// axis-aligned faces, with setback distance c and angle 45°. The prism runs along +Z from 0 to h.
func chamferedSquarePrism(c, h float64) *topo.Body {
	prof := [][2]float64{
		{1, -1 + c}, {1, 1 - c}, // +X face
		{1 - c, 1}, {-1 + c, 1}, // chamfer at (1,1); +Y face
		{-1, 1 - c}, {-1, -1 + c}, // chamfer at (-1,1); -X face
		{-1 + c, -1}, {1 - c, -1}, // chamfer at (-1,-1); -Y face
	}
	n := len(prof)
	verts := make([]gmath.Point3, 0, 2*n)
	for _, p := range prof { // bottom cap z=0
		verts = append(verts, gmath.P3(p[0], p[1], 0))
	}
	for _, p := range prof { // top cap z=h
		verts = append(verts, gmath.P3(p[0], p[1], h))
	}
	faces := [][]int{octRingReversed(n), octRingForward(n, n)}
	for i := 0; i < n; i++ {
		j := (i + 1) % n
		faces = append(faces, []int{i, j, n + j, n + i}) // outward side quad
	}
	return subd.ToBody(subd.Mesh{Verts: verts, Faces: faces}, "chamfered")
}

func octRingForward(n, offset int) []int {
	r := make([]int, n)
	for i := range r {
		r[i] = offset + i
	}
	return r
}

func octRingReversed(n int) []int {
	r := make([]int, n)
	for i := range r {
		r[i] = n - 1 - i
	}
	return r
}

// chamferFaceEdges finds a 45°-corner chamfer face (outward normal ≈ (±1,±1,0)/√2) and returns its
// two axis-parallel (vertical) edge keys — the chamfer's two edges.
func chamferFaceEdges(t *testing.T, body *topo.Body) ([]byte, []byte) {
	t.Helper()
	for _, f := range body.Faces() {
		n, ok := planeOutwardNormal(f)
		if !ok || stdmath.Abs(float64(n.Z)) > 1e-6 || stdmath.Abs(float64(n.X)) < 0.5 || stdmath.Abs(float64(n.Y)) < 0.5 {
			continue
		}
		var vertical [][]byte
		for _, e := range f.Edges() {
			line, ok := e.Geometry().(geom.LineSegment)
			if !ok {
				continue
			}
			d := line.StartPoint.VectorTo(line.EndPoint)
			if stdmath.Abs(float64(d.X)) < 1e-6 && stdmath.Abs(float64(d.Y)) < 1e-6 {
				vertical = append(vertical, e.ReferenceKey())
			}
		}
		if len(vertical) == 2 {
			return vertical[0], vertical[1]
		}
	}
	t.Fatal("no 45° chamfer face with two vertical edges found")
	return nil, nil
}

// TestChamferNoteSurvivesRecipeRoundTrip: a chamfer note's two edge keys persist through the recipe,
// so the reopened drawing re-derives the same callout (the round-trip gap that bit earlier notes).
func TestChamferNoteSurvivesRecipeRoundTrip(t *testing.T) {
	body := chamferedSquarePrism(0.4, 3)
	c := NewContent()
	c.SetBodyResolver(fakeBodyResolver{body: body})
	c.SetModelReference("prism.opd")
	topBase(t, c.Sheets().Active().Views())
	ka, kb := chamferFaceEdges(t, body)
	if _, err := c.Sheets().Active().Annotations().AddChamferNote("CH", "TOP", ka, kb); err != nil {
		t.Fatalf("AddChamferNote: %v", err)
	}

	model, err := c.MarshalRecipe()
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	rc := NewContent()
	rc.SetBodyResolver(fakeBodyResolver{body: body})
	if err := rc.ApplyRecipe(model); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	rc.RecomputeViews()
	cn, ok := rc.Sheets().Active().Annotations().ByName("CH")
	if !ok {
		t.Fatal("reopened drawing lost the chamfer note")
	}
	if len(cn.Labels()) != 1 || cn.Labels()[0].Text != "4.00 × 45°" {
		t.Errorf("reopened chamfer note = %v, want 4.00 × 45° (edge keys survived)", cn.Labels())
	}
}

// TestChamferMetricsFromBody: the chamfer of a chamfered-square prism reads 45° and its setback
// distance equals the chamfer leg — derived from the model geometry.
func TestChamferMetricsFromBody(t *testing.T) {
	body := chamferedSquarePrism(0.4, 3) // leg 0.4 cm
	ka, kb := chamferFaceEdges(t, body)
	distCm, angleDeg, _, ok := chamferMetricsFromBody(body, ka, kb)
	if !ok {
		t.Fatal("chamferMetricsFromBody failed to resolve the chamfer")
	}
	if stdmath.Abs(angleDeg-45) > 1e-6 {
		t.Errorf("chamfer angle = %.4f°, want 45°", angleDeg)
	}
	if stdmath.Abs(distCm-0.4) > 1e-6 {
		t.Errorf("chamfer setback = %.4f cm, want 0.4", distCm)
	}
}

// TestAddChamferNoteRendersDistanceAngle: a chamfer note reads "<d> × <angle>°" (4.00 × 45° for a
// 0.4 cm leg = 4 mm) and re-resolves when the chamfer grows.
func TestAddChamferNoteRendersDistanceAngle(t *testing.T) {
	body := chamferedSquarePrism(0.4, 3)
	c := NewContent()
	c.SetBodyResolver(fakeBodyResolver{body: body})
	c.SetModelReference("prism.opd")
	topBase(t, c.Sheets().Active().Views())
	ka, kb := chamferFaceEdges(t, body)

	cn, err := c.Sheets().Active().Annotations().AddChamferNote("CH", "TOP", ka, kb)
	if err != nil {
		t.Fatalf("AddChamferNote: %v", err)
	}
	if cn.Kind() != types.ChamferNoteAnnotation || len(cn.Labels()) != 1 || cn.Labels()[0].Text != "4.00 × 45°" {
		t.Fatalf("chamfer note = %v, want one label 4.00 × 45°", cn.Labels())
	}

	// Grow the chamfer to 0.6 cm (6 mm) and recompute: the note re-resolves (the same-structure
	// prism keeps the same edge reference keys).
	c.SetBodyResolver(fakeBodyResolver{body: chamferedSquarePrism(0.6, 3)})
	c.RecomputeViews()
	if cn.Labels()[0].Text != "6.00 × 45°" {
		t.Errorf("after the chamfer grew, note = %q, want 6.00 × 45°", cn.Labels()[0].Text)
	}
}
