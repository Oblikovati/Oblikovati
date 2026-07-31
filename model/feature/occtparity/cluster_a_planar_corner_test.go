// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	stdmath "math"
	"path/filepath"
	"strings"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
)

// Cluster A — planar K-edge/M-face corner topology + mixed radius (capability-breadth wave W-A).
// Four cases greened by three exact planar-corner mechanisms, each gated here BOTH directions:
//
//   - tolblend_simple/A1 + simple/X8 — full-round K-arm corner: every edge at a K-valent planar
//     vertex filleted at one radius, closed by the EXACT common-tangent-sphere K-gon
//     (fillet_corner_fullround.go). DRAWEXE's own boundary arcs lie on that sphere at <1e-6; its
//     patch interior is a GeomPlate approximant (A1 12.9388 / X8 90.832 vs the exact spherical
//     polygon 12.948 / 90.31 — ±0.6% on a ~100-area face, the ONLY divergent face; every other
//     face rank-pairs within mesh quantization).
//   - simple/A4 — mixed-radius trihedral TORUS corner [rB,rS,rS] (fillet_corner_radiustorus.go):
//     patch = exact torus (R=rB−rS, r=rS), per-face parity with DRAWEXE incl. the closed form
//     rS·(π/2)·(R·(π/2)+rS).
//   - tolblend_simple/D4 — vertex-only miter (fillet_miter_vertexonly.go): two arms sharing ONLY
//     a vertex mutually trim along the cylA∩cylB arc whose ends graze the sharp edges; DRAWEXE's
//     band end arc verified on BOTH cylinders.
//
// The decline direction pins the honest reject for every neighbour configuration this wave does
// NOT build (free-form plate corners), so a laundering mutation cannot green them silently.

// clusterAResultBody runs a corpus case's fillet in any grid and returns its single healthy body.
func clusterAResultBody(t *testing.T, grid, name string) *topo.Body {
	t.Helper()
	res, okFillet, reason := clusterARun(t, grid, name)
	if !okFillet || len(res) != 1 || res[0] == nil {
		t.Fatalf("%s/%s fillet unhealthy: ok=%v reason=%q results=%d", grid, name, okFillet, reason, len(res))
	}
	return res[0]
}

// clusterADeclineReason runs a case expected to decline and returns the fillet's reason string.
func clusterADeclineReason(t *testing.T, grid, name string) string {
	t.Helper()
	res, okFillet, reason := clusterARun(t, grid, name)
	if okFillet && len(res) == 1 && res[0] != nil {
		t.Fatalf("%s/%s built a body but this gate expects the honest decline", grid, name)
	}
	return reason
}

// clusterARun locates and runs one corpus case, returning the raw fillet outcome.
func clusterARun(t *testing.T, grid, name string) ([]*topo.Body, bool, string) {
	t.Helper()
	var rec Record
	for _, r := range Corpus() {
		if r.Grid == grid && r.Case == name {
			rec = r
		}
	}
	body, err := importInput(filepath.Join(CorpusFixtureDir(), rec.InputStep))
	if err != nil {
		t.Fatalf("%s/%s import: %v", grid, name, err)
	}
	sets, ok := scoreLocate(rec, body)
	if !ok {
		t.Fatalf("%s/%s picks not located", grid, name)
	}
	return runFillet(body, sets)
}

// clusterAPins are the captured fingerprints of the four cluster-A greens (two runs bit-identical
// on this HEAD; volume at 1e-9 rel, mesh at the 1e-6·diag quantum via the hash).
func clusterAPins() []fingerprintPin {
	return []fingerprintPin{
		{name: "X8", grid: "simple", vol: 1297075.612691443646, tris: 18890, hash: 0xe1a4c02598744574},
		{name: "A4", grid: "simple", vol: 98744.376504411470, tris: 8844, hash: 0x9c8b6ac93b3ab53e},
		{name: "A1", grid: "tolblend_simple", vol: 29054.780695677222, tris: 13966, hash: 0xe76b671876c836f0},
		{name: "D4", grid: "tolblend_simple", vol: 29111.202165111859, tris: 2040, hash: 0xbe57217790ba23ea},
	}
}

// TestClusterACornerGreensPinnedAndWatertight is the green-direction gate: each cluster-A green
// rebuilds to its pinned fingerprint and its welded mesh carries 0 free edges and 0 folds at BOTH
// gate qualities.
func TestClusterACornerGreensPinnedAndWatertight(t *testing.T) {
	for _, tc := range clusterAPins() {
		t.Run(tc.grid+"/"+tc.name, func(t *testing.T) {
			body := clusterAResultBody(t, tc.grid, tc.name)
			fp := bodyMeshFingerprint(body)
			if fp.Hash != tc.hash || fp.Triangles != tc.tris {
				t.Fatalf("fingerprint drifted: hash=%#x tris=%d, want hash=%#x tris=%d", fp.Hash, fp.Triangles, tc.hash, tc.tris)
			}
			if rel := relErr(fp.Volume, tc.vol); rel > 1e-9 {
				t.Fatalf("volume %.12f != pinned %.12f (rel %.3g)", fp.Volume, tc.vol, rel)
			}
			assertClusterAWatertight(t, body)
		})
	}
}

// assertClusterAWatertight asserts 0 free edges / 0 folds at every gate quality.
func assertClusterAWatertight(t *testing.T, body *topo.Body) {
	t.Helper()
	for _, gq := range gateQualities() {
		facets := ops.CalculateBodyFacets(body, gq.q)
		if free := ops.FreeEdgeCount(facets.Mesh); free != 0 {
			t.Fatalf("%s quality: %d free edges, want 0", gq.name, free)
		}
		if folds := meshFoldEdges(facets); folds != 0 {
			t.Fatalf("%s quality: %d fold edges, want 0", gq.name, folds)
		}
	}
}

// TestClusterACornerPatchClosedForms gates the corner patches on their DERIVED areas, so the gate
// is the geometry, not a snapshot: A1's sphere K-gon r²·(Σθ−2π)=12.9480, X8's Girard sum 90.31
// (DRAWEXE's plate approximants read 12.9388/90.832 — the documented sole divergence), A4's torus
// quarter rS·(π/2)·(R·(π/2)+rS)=100.9556, and D4's two DRAWEXE-exact bands at 148.148.
func TestClusterACornerPatchClosedForms(t *testing.T) {
	quarterPi := stdmath.Pi / 2
	assertUniqueKindFaceArea(t, clusterAResultBody(t, "tolblend_simple", "A1"), "Sphere", 6.25*2.07168, 1e-3)
	assertUniqueKindFaceArea(t, clusterAResultBody(t, "simple", "X8"), "Sphere", 90.31, 2e-3)
	assertUniqueKindFaceArea(t, clusterAResultBody(t, "simple", "A4"), "Torus", 5*quarterPi*(5*quarterPi+5), 1e-3)
	d4 := clusterAResultBody(t, "tolblend_simple", "D4")
	for _, f := range d4.Faces() {
		if _, isCyl := f.Geometry().(geom.Cylinder); !isCyl {
			continue
		}
		if a := ops.MeshArea(ops.TessellateFace(f, ops.PropertyQuality())); relErr(a, 148.148) > 1e-3 {
			t.Fatalf("D4 band area %.4f, want 148.148 (DRAWEXE, ±0.1%%)", a)
		}
	}
}

// assertUniqueKindFaceArea finds the body's single face of the named surface kind and compares its
// PropertyQuality mesh area against the closed form at rel tolerance.
func assertUniqueKindFaceArea(t *testing.T, body *topo.Body, kind string, want, tol float64) {
	t.Helper()
	found := 0
	for _, f := range body.Faces() {
		if !strings.HasSuffix(strings.TrimPrefix(surfaceKindName(f), "*"), "."+kind) {
			continue
		}
		found++
		if a := ops.MeshArea(ops.TessellateFace(f, ops.PropertyQuality())); relErr(a, want) > tol {
			t.Fatalf("%s patch area %.6f, want %.6f (rel %.3g > %.1g)", kind, a, want, relErr(a, want), tol)
		}
	}
	if found != 1 {
		t.Fatalf("found %d %s faces, want exactly 1 (the corner patch)", found, kind)
	}
}

// surfaceKindName is the face surface's concrete type name (e.g. "geom.Sphere").
func surfaceKindName(f *topo.Face) string {
	switch f.Geometry().(type) {
	case geom.Sphere:
		return "geom.Sphere"
	case geom.Torus:
		return "geom.Torus"
	default:
		return "geom.Other"
	}
}

// TestClusterAHonestDeclines is the decline-direction gate: every neighbour configuration this
// wave measured as needing the free-form (GeomPlate/batten) corner filling — or a capability of
// another cluster — must keep its cause-specific honest reject. A gate-laundering mutation (e.g.
// zeroing the common-sphere residual, dropping the miter valence guard, or loosening the torus
// orthogonality certificate) flips one of these reasons and fails loud.
func TestClusterAHonestDeclines(t *testing.T) {
	for _, tc := range []struct{ grid, name, want string }{
		// B1: glued drafted wedges — measured plane residual 0.263 (weld 8.3e-8): NOT a common sphere.
		{"tolblend_simple", "B1", "has no common tangent sphere"},
		// X5/B3: 6-valent alternating convex/concave — outside the all-convex full-round scope.
		{"simple", "X5", "requires all-convex arms"},
		{"tolblend_simple", "B3", "requires all-convex arms"},
		// D5: 2-of-4 edges miter at the pyramid apex — OCCT closes it with a plate + battens.
		{"tolblend_simple", "D5", "needs a free-form corner filling"},
		// D3/E7: partial corners (a sharp edge survives at the vertex) — plate class.
		{"tolblend_simple", "D3", "is not a supported blend"},
		{"tolblend_simple", "E7", "is not a supported blend"},
		// A2: mixed-radius 4-arm apex — no common tangent sphere exists at four distinct radii.
		{"tolblend_simple", "A2", "mixed-radius corner where 4 filleted edges"},
		// E3: drafted slab — its [19,7,7]/[14,7,7] corners are ~3° off orthogonal (elliptic-spine
		// canal corners, not tori); the certificate measures the skew and declines.
		{"complex", "E3", "needs a torus corner patch"},
		// C4: concave 4-valent runout whose cap crossing lands past the far edge's end (t=27.6 on a
		// 25-long edge) — the runout spill tier, outside this wave.
		{"tolblend_simple", "C4", "no single crossing on far edge"},
	} {
		t.Run(tc.grid+"/"+tc.name, func(t *testing.T) {
			reason := clusterADeclineReason(t, tc.grid, tc.name)
			if !strings.Contains(reason, tc.want) {
				t.Fatalf("decline reason %q does not carry %q", reason, tc.want)
			}
		})
	}
}
