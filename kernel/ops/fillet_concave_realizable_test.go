// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"testing"

	"oblikovati.org/kernel/geom"
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

// recessUnderExposedFace builds the ACCEPT geometry: a slab plus a standalone edge whose stations lie
// on the slab's exposed top face, with the frame (nA=nB=+Z, offDir=−Z) that seats the rolling ball in
// the material directly UNDER that exposed face. Its two tangent points then land on the top face —
// material behind (−Z, inside the slab), void in front (+Z) — so tangentBackedByMaterial holds at
// every station and the gate returns realizable.
//
// This construction is the endorsed predicate-level accept case, because the NATURAL reentrant frame
// can never reach here: at any concave/reentrant edge the surrounding material spans >180°, so the
// inward ball's tangents always fall into the bulk (empirically every supported prism, pocket, T-rib
// and V-groove concave edge rejects — buried tangent). The recess is realizable only where a tangent
// lands on an exposed face with void in front, which the slab-under-face frame supplies directly.
func recessUnderExposedFace(t *testing.T) (*topo.Body, *topo.Edge, m.Vector3, m.Vector3, m.Vector3) {
	t.Helper()
	slab := zPrism([]m.Point2{{X: 0, Y: 0}, {X: 10, Y: 0}, {X: 10, Y: 10}, {X: 0, Y: 10}}, 0, 4, "slab")
	lin := topo.NewLineage(topo.Tok("test", "recess", 0))
	bld := topo.NewBuilder(true, lin)
	v0 := bld.AddVertex(m.P3(3, 5, 4), lin) // both endpoints interior to the z=4 top face
	v1 := bld.AddVertex(m.P3(7, 5, 4), lin)
	e := bld.AddEdge(geom.NewLineSegment(m.P3(3, 5, 4), m.P3(7, 5, 4)), v0, v1, lin)
	up := m.V3(0, 0, 1)
	return slab, e, up, up, up.Scale(-1) // nA=nB=+Z, offDir=−Z (natural frame for coplanar up-facing walls)
}

// TestConcaveInwardRealizableAcceptsRecessUnderFace exercises the gate's ACCEPT path end-to-end (its
// return-true branch, which the reject-only regressions never reached). A rolling ball seated in the
// material beneath an exposed planar face is realizable, so the gate must return true and the zero
// sentinel point. A reject-always mutation (return false) fails this test.
func TestConcaveInwardRealizableAcceptsRecessUnderFace(t *testing.T) {
	body, e, nA, nB, offDir := recessUnderExposedFace(t)
	p, ok := concaveInwardRealizable(body, e, nA, nB, offDir, 1.5)
	if !ok {
		t.Fatalf("recess ball under an exposed top face must be realizable, got reject at %v", p)
	}
	if p != (m.Point3{}) {
		t.Errorf("accept must return the zero sentinel point, got %v", p)
	}
}

// TestConcaveInwardRealizableRejectsFloatingStation proves the gate evaluates EACH station's tangent
// placement, not merely that the body has material: nudging one endpoint off the slab (x=15, over
// void) makes that station's tangent float outside the solid, so the same frame that accepts in
// TestConcaveInwardRealizableAcceptsRecessUnderFace now rejects — the tangent geometry is load-bearing.
func TestConcaveInwardRealizableRejectsFloatingStation(t *testing.T) {
	slab := zPrism([]m.Point2{{X: 0, Y: 0}, {X: 10, Y: 0}, {X: 10, Y: 10}, {X: 0, Y: 10}}, 0, 4, "slab")
	lin := topo.NewLineage(topo.Tok("test", "float", 0))
	bld := topo.NewBuilder(true, lin)
	v0 := bld.AddVertex(m.P3(3, 5, 4), lin)
	v1 := bld.AddVertex(m.P3(15, 5, 4), lin) // off the slab (x>10): its tangent floats over void
	e := bld.AddEdge(geom.NewLineSegment(m.P3(3, 5, 4), m.P3(15, 5, 4)), v0, v1, lin)
	up := m.V3(0, 0, 1)
	if p, ok := concaveInwardRealizable(slab, e, up, up, up.Scale(-1), 1.5); ok {
		t.Errorf("a station whose tangent floats over void must be rejected, got realizable (last point %v)", p)
	}
}
