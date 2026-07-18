// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"testing"

	"oblikovati.org/kernel/topo"
	m "oblikovati.org/math"
)

// unitBoxMesh tessellates the [0,1]³ prism and returns its mesh plus the model-relative nudge —
// a named fake solid for probing tangentBackedByMaterial without wiring a whole fillet.
func unitBoxMesh(t *testing.T) (*Mesh, float64) {
	t.Helper()
	box := zPrism([]m.Point2{{X: 0, Y: 0}, {X: 1, Y: 0}, {X: 1, Y: 1}, {X: 0, Y: 1}}, 0, 1, "box")
	mesh, _ := TessellateBody(box, DefaultQuality())
	return mesh, ResolutionForBody(box).Plane()
}

// TestTangentBackedByMaterialPassesRealFace proves the gate is NOT a blanket inward ban: a tangent
// point ON a bounded face (material behind, void in front) is material-backed, so a genuinely
// realizable inward recess still builds.
func TestTangentBackedByMaterialPassesRealFace(t *testing.T) {
	mesh, eps := unitBoxMesh(t)
	nZ := m.V3(0, 0, 1) // the box's +Z face outward normal
	if !tangentBackedByMaterial(mesh, m.P3(0.5, 0.5, 1), nZ, eps) {
		t.Error("tangent on the box's +Z face must be material-backed (material behind, void in front)")
	}
}

// TestTangentBackedByMaterialRejectsBuriedAndFloating covers the two unrealizable configurations:
// a BURIED tangent (material on both sides — the reflex-corner L case) and a FLOATING one (void on
// both sides). Both must be rejected so the recess wall is never trimmed to a phantom line.
func TestTangentBackedByMaterialRejectsBuriedAndFloating(t *testing.T) {
	mesh, eps := unitBoxMesh(t)
	nZ := m.V3(0, 0, 1)
	if tangentBackedByMaterial(mesh, m.P3(0.5, 0.5, 0.5), nZ, eps) {
		t.Error("a buried tangent (interior plane, material on both sides) must NOT be material-backed")
	}
	if tangentBackedByMaterial(mesh, m.P3(0.5, 0.5, 2), nZ, eps) {
		t.Error("a floating tangent (void on both sides) must NOT be material-backed")
	}
}

// TestConcaveInwardRealizableRejectsReflexL is the kernel-level twin of
// TestFilletConcaveInwardDegenerate: on the reflex-corner L both tangent points fall into the bulk,
// so the recess is unrealizable and the gate returns a concrete offending point.
func TestConcaveInwardRealizableRejectsReflexL(t *testing.T) {
	b := zPrism([]m.Point2{{X: 0, Y: 0}, {X: 4, Y: 0}, {X: 4, Y: 2}, {X: 2, Y: 2}, {X: 2, Y: 4}, {X: 0, Y: 4}}, 0, 4, "L")
	var e *topo.Edge
	for _, ed := range b.Edges() {
		a, c := ed.StartVertex().Point(), ed.EndVertex().Point()
		if a.X == c.X && a.Y == c.Y && ClassifyEdgeConvexity(ed) == EdgeConcave {
			e = ed
			break
		}
	}
	if e == nil {
		t.Fatal("no concave vertical edge on the L-prism")
	}
	_, _, nA, nB, err := edgePlanarFaces(e)
	if err != nil {
		t.Fatalf("edgePlanarFaces: %v", err)
	}
	offDir := nA.Add(nB).Scale(-1 / (1 + nA.Dot(nB)))
	if p, ok := concaveInwardRealizable(b, e, nA, nB, offDir, 1.0); ok {
		t.Errorf("inward recess on the reflex L must be rejected, got realizable (last point %v)", p)
	}
}
