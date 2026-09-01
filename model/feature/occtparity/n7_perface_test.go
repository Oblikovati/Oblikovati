// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/kernel/topo"
)

// perFaceGate is one canal-corner face's oracle area gate on the real N7 body: the surface kind that
// identifies it (with an area bucket to disambiguate the three cylinders) and OCCT's per-face area.
type perFaceGate struct {
	name     string
	kind     string  // "bspline" | "torus" | "cylinder"
	loArea   float64 // area bucket lower bound (cylinders only; 0 = ignore)
	hiArea   float64
	expected float64 // OCCT per-face oracle area (checkprops role-map)
}

// n7PerFaceGates is the OCCT role-map for the three faces that MIS-tessellated before the canal
// sub-edge trimmed-sub-span fix plus the torus arm that must stay put (n7-tessellation-diagnosis.md):
// the BSpline canal patch folded+diverged to 176, the s_4 arm tiled exactly half (273), the s_10 arm
// overshot 13% (221); the torus was already correct. Now all four mesh to OCCT's areas.
var n7PerFaceGates = []perFaceGate{
	{"canal patch", "bspline", 0, 0, 90.194},    // was 176.27, 13 folds, 245s → convergent, 0 folds
	{"s_4 arm", "cylinder", 400, 800, 546.695},  // was 273.358 (exactly half)
	{"s_10 arm", "cylinder", 100, 300, 195.464}, // was 220.93 (+13%)
	{"torus arm", "torus", 0, 0, 212.306},       // unchanged control
}

// TestN7PerFaceAreas is the per-face tessellation gate the F3 whole-body area test deferred: on the
// REAL N7 body each canal-corner face now meshes to its OCCT oracle area within OCCT's 1% (deps), the
// canal patch has ZERO folds, and it CONVERGES (Default and Property quality agree, not diverge) — the
// signature that the self-overlapping boundary is gone. RED at HEAD a5bfe089 (patch 176/diverging,
// s_4 273); GREEN after the canal sub-edges present their trimmed sub-span geometry.
func TestN7PerFaceAreas(t *testing.T) {
	t.Parallel()
	faces := n7ResultBody(t).Faces()
	for _, g := range n7PerFaceGates {
		f := findN7Face(t, faces, g)
		mesh := tessellate.TessellateFace(f, ops.PropertyQuality())
		area := ops.MeshArea(mesh)
		if rel := relErr(area, g.expected); rel > 0.01 {
			t.Errorf("%s: Property area %.4f != OCCT %.4f (rel %.3f%% > 1%%)", g.name, area, g.expected, rel*100)
		}
		if g.kind == "bspline" {
			assertPatchConverges(t, f, mesh)
		}
	}
}

// assertPatchConverges gates the canal patch's two failure modes that survived a watertight B-rep: a
// FOLD (self-overlapping boundary) and DIVERGENCE under refinement (Default 152 → Property 176). Zero
// folds and Default≈Property (within 0.5%) together certify the boundary is a simple polygon.
func assertPatchConverges(t *testing.T, f *topo.Face, prop *ops.Mesh) {
	t.Helper()
	if folds := ops.FoldEdgeCount(prop); folds != 0 {
		t.Errorf("canal patch: %d fold edges at Property quality, want 0", folds)
	}
	def := ops.MeshArea(tessellate.TessellateFace(f, ops.DefaultQuality()))
	if rel := relErr(ops.MeshArea(prop), def); rel > 0.005 {
		t.Errorf("canal patch diverges: Default %.4f vs Property %.4f (rel %.3f%% > 0.5%%)", def, ops.MeshArea(prop), rel*100)
	}
}

// findN7Face selects the one face matching gate g by surface kind (and area bucket for the three
// cylinders), failing loud if it is absent or ambiguous — a face-count/role regression must not pass.
func findN7Face(t *testing.T, faces []*topo.Face, g perFaceGate) *topo.Face {
	t.Helper()
	var match *topo.Face
	for _, f := range faces {
		if !faceKindMatches(f, g) {
			continue
		}
		if match != nil {
			t.Fatalf("%s: multiple faces match kind %q + bucket [%g,%g]", g.name, g.kind, g.loArea, g.hiArea)
		}
		match = f
	}
	if match == nil {
		t.Fatalf("%s: no %q face in bucket [%g,%g] on the N7 body", g.name, g.kind, g.loArea, g.hiArea)
	}
	return match
}

// faceKindMatches reports whether f is g's surface kind and (for cylinders) falls in g's area bucket.
func faceKindMatches(f *topo.Face, g perFaceGate) bool {
	switch g.kind {
	case "bspline":
		_, ok := f.Geometry().(geom.BSplineSurface)
		return ok
	case "torus":
		_, ok := f.Geometry().(geom.Torus)
		return ok
	case "cylinder":
		if _, ok := f.Geometry().(geom.Cylinder); !ok {
			return false
		}
		a := ops.MeshArea(tessellate.TessellateFace(f, ops.PropertyQuality()))
		return a >= g.loArea && a <= g.hiArea
	default:
		return false
	}
}

// relErr is the relative error between got and want.
func relErr(got, want float64) float64 {
	return stdmath.Abs(got-want) / stdmath.Abs(want)
}
