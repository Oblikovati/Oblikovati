// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	stdmath "math"
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// These tests cover the geometric-selector path (placementFaceGeom / edgesGeom / facesGeom): an
// external author selects a hole/dress-up's faces and edges by GEOMETRY (centroid+normal,
// midpoint+direction) instead of a reference key it cannot mint, and the host binds them to the
// body. Each geometric selection must build the SAME solid the equivalent reference-key selection
// does, since both name the same topology.

func xyz(p math.Point3) []float64   { return []float64{float64(p.X), float64(p.Y), float64(p.Z)} }
func dxyz(v math.Vector3) []float64 { return []float64{float64(v.X), float64(v.Y), float64(v.Z)} }

// topFaceOf returns the extruded box's uppermost face object (the natural hole placement face).
func topFaceOf(t *testing.T, s *app.Session) *topo.Face {
	t.Helper()
	b := activePartBody(t, s)
	var top *topo.Face
	for _, f := range b.Faces() {
		if top == nil || faceVertexMean(f).Z > faceVertexMean(top).Z {
			top = f
		}
	}
	if top == nil {
		t.Fatal("box has no faces")
	}
	return top
}

// TestHoleGeomFaceMatchesFaceRef: drilling by placementFaceGeom removes the same material as
// drilling by faceRef on the same top face — proof the geometric placement binds the right face
// (the failure this whole path fixes was a lost key leaving the hole cutting nothing).
func TestHoleGeomFaceMatchesFaceRef(t *testing.T) {
	byRef, _, _ := extrudedSolid(t)
	refKey, _ := boxTopFace(t, byRef)
	if _, err := applyMap(t, byRef, "hole", map[string]any{"faceRef": refKey, "diameter": "3 mm", "depth": "5 mm"}); err != nil {
		t.Fatalf("drill by faceRef: %v", err)
	}

	byGeom, _, _ := extrudedSolid(t)
	g := topo.DescribeFace(topFaceOf(t, byGeom))
	args := map[string]any{"diameter": "3 mm", "depth": "5 mm",
		"placementFaceGeom": map[string]any{"centroid": xyz(g.Centroid), "normal": dxyz(g.Normal)}}
	if _, err := applyMap(t, byGeom, "hole", args); err != nil {
		t.Fatalf("drill by placementFaceGeom: %v", err)
	}

	if vr, vg := bodyVolume(t, byRef), bodyVolume(t, byGeom); stdmath.Abs(vr-vg) > 1e-6 {
		t.Errorf("geom-face hole volume = %v, faceRef hole = %v, want equal", vg, vr)
	}
}

// TestHoleNeedsFaceRefOrGeom: a hole with neither faceRef nor placementFaceGeom is a clean error.
func TestHoleNeedsFaceRefOrGeom(t *testing.T) {
	s, _, _ := extrudedSolid(t)
	if _, err := applyMap(t, s, "hole", map[string]any{"diameter": "3 mm", "depth": "5 mm"}); err == nil {
		t.Error("hole with neither faceRef nor placementFaceGeom should error")
	}
}

// TestFilletGeomEdgesMatchesEdgeRefs: rounding an edge by edgesGeom removes the same material as
// rounding it by edgeRefs (same edge, same radius).
func TestFilletGeomEdgesMatchesEdgeRefs(t *testing.T) {
	byRef, edgeKey, _ := extrudedSolid(t)
	if _, err := applyMap(t, byRef, "fillet", map[string]any{"edgeRefs": []string{edgeKey}, "radius": "1 mm"}); err != nil {
		t.Fatalf("fillet by edgeRefs: %v", err)
	}

	byGeom, _, _ := extrudedSolid(t)
	g := topo.DescribeEdge(activePartBody(t, byGeom).Edges()[0])
	args := map[string]any{"radius": "1 mm",
		"edgesGeom": []any{map[string]any{"midpoint": xyz(g.Midpoint), "direction": dxyz(g.Direction)}}}
	if _, err := applyMap(t, byGeom, "fillet", args); err != nil {
		t.Fatalf("fillet by edgesGeom: %v", err)
	}

	if vr, vg := bodyVolume(t, byRef), bodyVolume(t, byGeom); stdmath.Abs(vr-vg) > 1e-6 {
		t.Errorf("geom-edge fillet volume = %v, edgeRefs fillet = %v, want equal", vg, vr)
	}
}

// TestChamferGeomEdgesMatchesEdgeRefs: bevelling an edge by edgesGeom matches edgeRefs.
func TestChamferGeomEdgesMatchesEdgeRefs(t *testing.T) {
	byRef, edgeKey, _ := extrudedSolid(t)
	if _, err := applyMap(t, byRef, "chamfer", map[string]any{"edgeRefs": []string{edgeKey}, "distance": "1 mm"}); err != nil {
		t.Fatalf("chamfer by edgeRefs: %v", err)
	}

	byGeom, _, _ := extrudedSolid(t)
	g := topo.DescribeEdge(activePartBody(t, byGeom).Edges()[0])
	args := map[string]any{"distance": "1 mm",
		"edgesGeom": []any{map[string]any{"midpoint": xyz(g.Midpoint), "direction": dxyz(g.Direction)}}}
	if _, err := applyMap(t, byGeom, "chamfer", args); err != nil {
		t.Fatalf("chamfer by edgesGeom: %v", err)
	}

	if vr, vg := bodyVolume(t, byRef), bodyVolume(t, byGeom); stdmath.Abs(vr-vg) > 1e-6 {
		t.Errorf("geom-edge chamfer volume = %v, edgeRefs chamfer = %v, want equal", vg, vr)
	}
}

// TestShellGeomFacesMatchesFaceRefs: hollowing while removing a face by facesGeom matches faceRefs.
func TestShellGeomFacesMatchesFaceRefs(t *testing.T) {
	byRef, _, faceKey := extrudedSolid(t)
	if _, err := applyMap(t, byRef, "shell", map[string]any{"faceRefs": []string{faceKey}, "thickness": "1 mm"}); err != nil {
		t.Fatalf("shell by faceRefs: %v", err)
	}

	byGeom, _, _ := extrudedSolid(t)
	g := topo.DescribeFace(activePartBody(t, byGeom).Faces()[0])
	args := map[string]any{"thickness": "1 mm",
		"facesGeom": []any{map[string]any{"centroid": xyz(g.Centroid), "normal": dxyz(g.Normal)}}}
	if _, err := applyMap(t, byGeom, "shell", args); err != nil {
		t.Fatalf("shell by facesGeom: %v", err)
	}

	if vr, vg := bodyVolume(t, byRef), bodyVolume(t, byGeom); stdmath.Abs(vr-vg) > 1e-6 {
		t.Errorf("geom-face shell volume = %v, faceRefs shell = %v, want equal", vg, vr)
	}
}

// TestDraftGeomFacesMatchesFaceRefs: drafting a face by facesGeom matches faceRefs.
func TestDraftGeomFacesMatchesFaceRefs(t *testing.T) {
	byRef, _, faceKey := extrudedSolid(t)
	if _, err := applyMap(t, byRef, "draft", map[string]any{"faceRefs": []string{faceKey}, "angle": "3 deg"}); err != nil {
		t.Fatalf("draft by faceRefs: %v", err)
	}

	byGeom, _, _ := extrudedSolid(t)
	g := topo.DescribeFace(activePartBody(t, byGeom).Faces()[0])
	args := map[string]any{"angle": "3 deg",
		"facesGeom": []any{map[string]any{"centroid": xyz(g.Centroid), "normal": dxyz(g.Normal)}}}
	if _, err := applyMap(t, byGeom, "draft", args); err != nil {
		t.Fatalf("draft by facesGeom: %v", err)
	}

	if vr, vg := bodyVolume(t, byRef), bodyVolume(t, byGeom); stdmath.Abs(vr-vg) > 1e-6 {
		t.Errorf("geom-face draft volume = %v, faceRefs draft = %v, want equal", vg, vr)
	}
}

// TestGeomSelectorBadVectorErrors: a geometric selector whose centroid/normal/midpoint is not a
// 3-vector is a clean error, not a panic (covers the conversion error paths).
func TestGeomSelectorBadVectorErrors(t *testing.T) {
	s, _, _ := extrudedSolid(t)
	badFace := map[string]any{"diameter": "3 mm", "depth": "5 mm",
		"placementFaceGeom": map[string]any{"centroid": []float64{1, 2}, "normal": []float64{0, 0, 1}}}
	if _, err := applyMap(t, s, "hole", badFace); err == nil {
		t.Error("placementFaceGeom with a 2-vector centroid should error")
	}

	s2, _, _ := extrudedSolid(t)
	badEdge := map[string]any{"radius": "1 mm",
		"edgesGeom": []any{map[string]any{"midpoint": []float64{1, 2, 3}, "direction": []float64{0, 1}}}}
	if _, err := applyMap(t, s2, "fillet", badEdge); err == nil {
		t.Error("edgesGeom with a 2-vector direction should error")
	}
}
