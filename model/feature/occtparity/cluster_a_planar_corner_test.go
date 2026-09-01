// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	stdmath "math"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/kernel/topo"
)

// Cluster A — planar K-edge/M-face corner topology + mixed radius (capability-breadth wave W-A).
// Three cases greened by three exact planar-corner mechanisms, each gated here BOTH directions:
//
//   - tolblend_simple/A1 — full-round K-arm corner: every edge at a K-valent planar vertex
//     filleted at one radius, closed by the EXACT common-tangent-sphere K-gon
//     (fillet_corner_fullround.go). DRAWEXE's own boundary arcs lie on that sphere at <1e-6; its
//     patch interior is a GeomPlate approximant (A1 12.9388 vs the exact spherical polygon
//     12.948 — ±0.6% on a ~100-area face, the ONLY divergent face; every other face rank-pairs
//     within mesh quantization).
//   - simple/A4 — mixed-radius trihedral TORUS corner [rB,rS,rS] (fillet_corner_radiustorus.go):
//     patch = exact torus (R=rB−rS, r=rS), per-face parity with DRAWEXE incl. the closed form
//     rS·(π/2)·(R·(π/2)+rS).
//   - tolblend_simple/D4 — vertex-only miter (fillet_miter_vertexonly.go): two arms sharing ONLY
//     a vertex mutually trim along the cylA∩cylB arc whose ends graze the sharp edges; DRAWEXE's
//     band end arc verified on BOTH cylinders.
//
// simple/X8 uses the SAME full-round mechanism as A1 (a 4-valent, 4-face pyramid apex) but is
// held in the decline table: TestEveryLoopSegmentLiesOnItsFace caught a real B-rep defect on it
// (a host plane's loop bounded by a corner arc 0.0135 off it) that A1 does not carry. Every
// hypothesis for a narrow, correct fix was checked by direct measurement and falsified (see
// fillet_corner_fullround.go's solveFullRoundCorner comment) — the sphere and every arm's own
// tangency are exactly right on X8 too, so the defect is downstream, in shared host-retrim code
// this cluster does not own. fullRoundDihedralsEqual gates on the one measured structural
// difference (A1's arms meet at an equal dihedral angle; X8's do not), honestly labelled as an
// empirical gate pending the real root cause, rather than shipping the defect or discarding the
// verified-clean symmetric population.
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

// clusterAPins are the captured fingerprints of the three cluster-A greens (two runs bit-identical
// on this HEAD; volume at 1e-9 rel, mesh at the 1e-6·diag quantum via the hash). Recaptured after
// wedgeAlignChainBranches (tessellate_wedge_band.go): the SAME oblique-band shape now also fires on
// bands whose two end chains land on different 2π branches of cyl.ParamAt (A1/D4 both have one),
// which is what fully closed the deg-4 free edge at the shared base corner (was 1 at DefaultQuality
// on A1/D4 even after the first wedge-band pass — see that file's history). Fewer, correctly-welded
// triangles; volumes move only at tessellation-chordal-error scale (≤1e-5 rel).
func clusterAPins() []fingerprintPin {
	return []fingerprintPin{
		{name: "A4", grid: "simple", vol: 98744.376504411470, tris: 8844, hash: 0x9c8b6ac93b3ab53e},
		{name: "A1", grid: "tolblend_simple", vol: 29055.024985750901, tris: 10438, hash: 0x3bdfbb69d9af9678},
		{name: "D4", grid: "tolblend_simple", vol: 29110.243662669993, tris: 276, hash: 0x6ee37d5c0f1984f1},
	}
}

// TestClusterACornerGreensPinnedAndWatertight is the green-direction gate: each cluster-A green
// rebuilds to its pinned fingerprint and its welded mesh carries 0 free edges and 0 folds at BOTH
// gate qualities.
func TestClusterACornerGreensPinnedAndWatertight(t *testing.T) {
	t.Parallel()
	for _, tc := range clusterAPins() {
		t.Run(tc.grid+"/"+tc.name, func(t *testing.T) {
			body := clusterAResultBody(t, tc.grid, tc.name)
			fp := bodyMeshFingerprint(body)
			// Triangle COUNT is architecture-stable and always checked; the HASH is not. It digests
			// quantized coordinates, and arm64 contracts x*y+z into FMA where amd64 does not, so the same
			// mesh hashes differently (tolblend_simple/A1 and D4 drifted on macOS with IDENTICAL counts —
			// 10438 and 276 — proving the mesh is the same and only the rounding differs). Pinned on amd64,
			// where both CI legs still run it. The volume and watertight/fold gates below run everywhere:
			// those are the structural claims, and they must never be architecture-gated.
			if fp.Triangles != tc.tris {
				t.Fatalf("triangle count drifted: tris=%d, want %d", fp.Triangles, tc.tris)
			}
			if runtime.GOARCH == "amd64" && fp.Hash != tc.hash {
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
		facets := tessellate.CalculateBodyFacets(body, gq.q)
		if free := tessellate.FreeEdgeCount(facets.Mesh); free != 0 {
			t.Fatalf("%s quality: %d free edges, want 0", gq.name, free)
		}
		if folds := meshFoldEdges(facets); folds != 0 {
			t.Fatalf("%s quality: %d fold edges, want 0", gq.name, folds)
		}
	}
}

// TestClusterACornerPatchClosedForms gates the corner patches on their DERIVED areas, so the gate
// is the geometry, not a snapshot: A1's sphere K-gon r²·(Σθ−2π)=12.9480 (DRAWEXE's plate approximant
// reads 12.9388 — the documented divergence), A4's torus quarter rS·(π/2)·(R·(π/2)+rS)=100.9556,
// and D4's two DRAWEXE-exact bands at 148.148.
func TestClusterACornerPatchClosedForms(t *testing.T) {
	t.Parallel()
	quarterPi := stdmath.Pi / 2
	assertUniqueKindFaceArea(t, clusterAResultBody(t, "tolblend_simple", "A1"), "Sphere", 6.25*2.07168, 1e-3)
	assertUniqueKindFaceArea(t, clusterAResultBody(t, "simple", "A4"), "Torus", 5*quarterPi*(5*quarterPi+5), 1e-3)
	d4 := clusterAResultBody(t, "tolblend_simple", "D4")
	for _, f := range d4.Faces() {
		if _, isCyl := f.Geometry().(geom.Cylinder); !isCyl {
			continue
		}
		if a := ops.MeshArea(tessellate.TessellateFace(f, ops.PropertyQuality())); relErr(a, 148.148) > 1e-3 {
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
		if a := ops.MeshArea(tessellate.TessellateFace(f, ops.PropertyQuality())); relErr(a, want) > tol {
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
// zeroing the common-sphere residual or loosening the torus orthogonality certificate) flips one
// of these reasons and fails loud.
func TestClusterAHonestDeclines(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ grid, name, want string }{
		// B1: glued drafted wedges — measured plane residual 0.263 (weld 8.3e-8): NOT a common sphere.
		{"tolblend_simple", "B1", "has no common tangent sphere"},
		// X5/B3: 6-valent alternating convex/concave — outside the all-convex full-round scope.
		{"simple", "X5", "requires all-convex arms"},
		{"tolblend_simple", "B3", "requires all-convex arms"},
		// D5: 2-of-4 edges filleted at a pyramid apex (2 ADJACENT apex edges, sharing one face). R2
		// wave: TRIED routing this into the partial-corner touched-face test too (the same mechanism
		// that now catches D3/E6/E7/E8 below), reasoning the ordinary miter's mirror-plane/cyl∩cyl
		// seam assumes exactly 3 faces at the vertex. FALSIFIED by the full -race suite:
		// TestClassifyEndCornersExcludesKGreaterThanOne (simple/V3, a genuine 2-edge shared-face
		// miter at its OWN 5-valent vertex) proved the ordinary miter is LOCAL to the two picked
		// edges and their 3 relevant faces — it does not need the vertex's total valence at all, and
		// works correctly at valence>3 in general. So D5's invalidity is a genuine, case-specific
		// seam defect on ITS geometry (not a missing valence-awareness), reached the SAME way it was
		// before this wave touched the file: solveMiter runs, the welded body fails IsSolid. Not
		// forced through the new mechanism — reverted to the pre-wave dispatch, unchanged.
		{"tolblend_simple", "D5", "result is not a valid solid"},
		// D3/E6/E7/E8: partial corners (a sharp edge survives at the vertex) — 3+ edges filleted,
		// unlike D5's 2, so len(ps)>=3 routes them to fillet_corner_partial.go's touched-face test
		// WITHOUT touching the ps==2 ordinary-miter path V3 depends on. Declines with a specific,
		// measured reason (the structural loop-closure check) instead of the generic default
		// "not a supported blend" fallthrough.
		{"tolblend_simple", "D3", "boundary-assembly capability is not yet built"},
		{"tolblend_simple", "E6", "boundary-assembly capability is not yet built"},
		{"tolblend_simple", "E7", "boundary-assembly capability is not yet built"},
		{"tolblend_simple", "E8", "boundary-assembly capability is not yet built"},
		// A2: mixed-radius 4-arm apex — no common tangent sphere exists at four distinct radii.
		{"tolblend_simple", "A2", "mixed-radius corner where 4 filleted edges"},
		// E3: drafted slab — its [19,7,7]/[14,7,7] corners are ~3° off orthogonal (elliptic-spine
		// canal corners, not tori); the certificate measures the skew and declines.
		{"complex", "E3", "needs a torus corner patch"},
		// C4: concave 4-valent runout whose cap crossing lands past the far edge's end (t=27.6 on a
		// 25-long edge) — the runout spill tier: the decline fires from validateRunoutFans (run-in/
		// run-out region, Cluster T's file, fillet.go:~826), NOT from solveCorner — this wave's
		// dispatch generalization does not reach it and must not (scope boundary), so it stays
		// declining with its own unrelated message rather than being forced into a corner reason.
		{"tolblend_simple", "C4", "no single crossing on far edge"},
		// A9/B2/C2: K-valent (K=faces=5/6/6) but MIXED radii with no [rB,rS,rS] bipartition (A9: 5
		// distinct radii; B2: 3 distinct with uneven multiplicity; C2: 3 distinct AND alternating
		// convex/concave, a saddle not a pyramid apex) — no common tangent sphere/torus closed form
		// exists. DRAWEXE-verified (mksurface/dump on the oracle build): OCCT itself closes A2's,
		// A9's, and C2's corner with exactly ONE BSplineSurface (A2 16×16 poles, C2 single Plate face
		// among 19) — the free-form GeomPlate mechanism, not an analytic patch, so this is the SAME
		// out-of-scope capability as D3/E7/D5, not a gap in the shipped torus/sphere machinery.
		{"tolblend_simple", "A9", "mixed-radius corner where 5 filleted edges"},
		{"tolblend_simple", "B2", "mixed-radius corner where 6 filleted edges"},
		{"complex", "C2", "mixed-radius corner where 6 filleted edges"},
		// H3/H5: PREMISE CORRECTION — the map filed these as planar (S2: "H3/H5→A"), but the fixture
		// is `revol` of an ellipse-arc + polyline 270°: the 2-face vertex is a Cone∧Plane (H3) or a
		// SurfaceOfRevolution∧Plane (H5) corner, and DRAWEXE's own oracle build closes it with FOUR
		// distinct BSplineSurface faces (mksurface/dump: result_3/4/8/9 all BSplineSurface urational
		// vrational, alongside a ConicalSurface and a SurfaceOfRevolution) — a multi-patch free-form
		// weld, not a single tangent-sphere/torus problem on two planes. This is a curved-host corner
		// (Cluster H/F territory: "corner face must be planar"), not this cluster's planar mechanism;
		// reported here rather than forced, per the map's own "verify the premise" rule.
		{"simple", "H3", "is not a supported blend"},
		{"simple", "H5", "is not a supported blend"},
		// X8: same full-round mechanism as A1 (4-valent pyramid apex) but an ASYMMETRIC one
		// (dihedral angles 90°/29°/29°/76.4°, vs A1's four equal 70.254° arms) — the one measured
		// discriminator between this case's genuine B-rep defect (TestEveryLoopSegmentLiesOnItsFace)
		// and A1's clean build. See the file doc comment and fillet_corner_fullround.go's
		// solveFullRoundCorner for the full falsification trail (sphere tangency, spine concurrence,
		// and the arc-midpoint formula were each independently verified EXACT on X8 too — none of
		// them is the defect).
		{"simple", "X8", "unequal dihedral angles"},
	} {
		t.Run(tc.grid+"/"+tc.name, func(t *testing.T) {
			reason := clusterADeclineReason(t, tc.grid, tc.name)
			if !strings.Contains(reason, tc.want) {
				t.Fatalf("decline reason %q does not carry %q", reason, tc.want)
			}
		})
	}
}
