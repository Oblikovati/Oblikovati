// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
)

// TestSingleArmRunoutR1 is the whole-body + per-face gate for the curved-arm SINGLE-EDGE runout tracers
// B6 (exact cylinder arm) and C9 (exact torus arm) — the R1 slice that dispatches a one-pick, two-plane-
// capped curved arm to the corner-free both-ends weld (fillet_curved_single_runout.go) instead of the
// 3-arm trihedral floor. On the REAL imported STEP bodies it asserts the weld is a watertight solid with
// the oracle face count, the single FILLET face meshes to OCCT's area (B6 CylindricalSurface r=10 1823.48,
// C9 ToroidalSurface 1298.13), and EVERY face meshes FOLD-FREE (FoldEdgeCount 0) — the highest-priority
// tessellation gate. RED at base 4e7f0d6c ("trihedral corner needs 3 arms"); GREEN after the R1 dispatch.
func TestSingleArmRunoutR1(t *testing.T) {
	for _, tc := range []struct {
		name       string
		faces      int
		filletKind string  // "cylinder" | "torus" — the single added fillet face
		filletR    float64 // fillet tube radius (disambiguates B6's r=10 arm from the r=50 wall)
		filletArea float64 // OCCT per-face fillet area
	}{
		{"B6", 6, "cylinder", 10, 1823.48},
		{"C9", 5, "torus", 10, 1298.13},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := caseResultBody(t, tc.name)
			assertWatertight(t, tc.name, body, tc.faces)
			assertNoFaceFolds(t, tc.name, body)
			f := runoutFilletFace(t, tc.name, body, tc.filletKind, tc.filletR)
			if a := faceMeshArea2(f); stdmath.Abs(a-tc.filletArea) > 0.01*tc.filletArea {
				t.Fatalf("%s fillet-face area %.4f != OCCT %.4f (rel %.3f%% > 1%%)",
					tc.name, a, tc.filletArea, relErr(a, tc.filletArea)*100)
			}
		})
	}
}

// assertNoFaceFolds fails when ANY result face meshes with a fold (a self-overlapping boundary) — the
// tessellation-first invariant: a watertight B-rep still renders/measures broken if a face folds.
func assertNoFaceFolds(t *testing.T, name string, body *topo.Body) {
	t.Helper()
	for _, f := range body.Faces() {
		if n := ops.FoldEdgeCount(ops.TessellateFace(f, ops.PropertyQuality())); n != 0 {
			t.Fatalf("%s face %T meshed with %d fold edges, want 0", name, f.Geometry(), n)
		}
	}
}

// runoutFilletFace returns the single added fillet face: the torus arm (C9), or the r=10 cylinder arm (B6,
// disambiguated from the r=50 host wall by radius). Fails loud when it is absent or ambiguous.
func runoutFilletFace(t *testing.T, name string, body *topo.Body, kind string, r float64) *topo.Face {
	t.Helper()
	var match *topo.Face
	for _, f := range body.Faces() {
		if !runoutFilletKind(f, kind, r) {
			continue
		}
		if match != nil {
			t.Fatalf("%s: multiple %q faces (r=%g) match the fillet role", name, kind, r)
		}
		match = f
	}
	if match == nil {
		t.Fatalf("%s carries no %q fillet face (r=%g)", name, kind, r)
	}
	return match
}

// runoutFilletKind reports whether face f is the fillet's arm surface of the given kind and tube radius.
func runoutFilletKind(f *topo.Face, kind string, r float64) bool {
	switch kind {
	case "cylinder":
		c, ok := f.Geometry().(geom.Cylinder)
		return ok && stdmath.Abs(c.Radius-r) < 1e-6
	case "torus":
		tor, ok := f.Geometry().(geom.Torus)
		return ok && stdmath.Abs(tor.MinorRadius-r) < 1e-6
	default:
		return false
	}
}
