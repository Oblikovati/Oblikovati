// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"sort"
	"testing"

	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// s1RebuiltShell assembles the runout-rebuilt S1 fillet shell WITH the rebuild enabled (no
// do-no-harm fallback), so a test can inspect the raw rebuilt topology — the open shell the
// closure work must weld shut. It mirrors assembleFilletBody's first line but skips the fallback.
func s1RebuiltShell(t *testing.T) *topo.Body {
	t.Helper()
	body := importCorpusSolid(t, "simple/S1")
	e := edgeAtMidpoint(body, math.P3(0, -10, 10))
	if e == nil {
		t.Fatal("s1RebuiltShell: front-top edge not found")
	}
	fil, err := computeEdgeFillet(body, filletPick{edge: e, r0: 6, r1: 6},
		map[uint64]*cornerBlend{}, map[uint64]*cornerMiter{}, FillConcaveOutward)
	if err != nil {
		t.Fatalf("s1RebuiltShell: computeEdgeFillet: %v", err)
	}
	faces, fired := filletResultFaces(body, []edgeFillet{fil}, map[uint64]*cornerBlend{}, true)
	if !fired {
		t.Fatal("s1RebuiltShell: runout rebuild did not fire")
	}
	return assembleBody(faces, "fillet")
}

// boundaryEdgeReport is the systematic-debugging evidence for closure: every edge used a number
// of times other than 2 (a watertight solid uses each exactly twice). endpoints + use-count let a
// test log WHICH open edges remain and drive the count to zero.
type boundaryEdgeReport struct {
	uses     int
	from, to math.Point3
}

// openEdges lists every non-manifold edge of body (co-edge use count != 2), sorted for stable
// logging. An empty result means the shell is watertight.
func openEdges(body *topo.Body) []boundaryEdgeReport {
	var out []boundaryEdgeReport
	for _, e := range body.Edges() {
		if n := len(e.Uses()); n != 2 {
			out = append(out, boundaryEdgeReport{uses: n, from: e.StartVertex().Point(), to: e.EndVertex().Point()})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].from.X != out[j].from.X {
			return out[i].from.X < out[j].from.X
		}
		return out[i].from.Y < out[j].from.Y
	})
	return out
}

// TestRunoutClosure_S1Watertight is the closure gate: the runout-rebuilt S1 body must be a
// watertight solid whose area is within OCCT's 1% (3662.79). It logs every remaining open edge as
// evidence while the closure is built up step by step (systematic-debugging: evidence before fixes).
func TestRunoutClosure_S1Watertight(t *testing.T) {
	body := s1RebuiltShell(t)
	open := openEdges(body)
	for _, o := range open {
		t.Logf("open edge (uses=%d): %v -> %v", o.uses, o.from, o.to)
	}
	t.Logf("open edge count: %d", len(open))
	if !body.IsSolid() {
		// Area of an open, non-manifold shell is meaningless (and its trim tessellation can thrash),
		// so it is only computed once the shell welds shut — the count above drives the closure work.
		t.Fatalf("S1 runout shell is not a solid: %d open edges", len(open))
	}
	// HolesContained is the same do-no-harm accessor the obstacle gate uses (obstacleImprovedSolid,
	// fillet.go:198-200): a watertight-but-open-holed shell would pass IsSolid alone, so the closure
	// gate must also confirm no hole survived the re-weld.
	if r := Validate(body); !r.HolesContained {
		t.Fatalf("S1 runout shell is not hole-contained: %+v", r)
	}
	area := BodyGeometryProperties(body, PropertyQuality()).Area
	t.Logf("rebuilt shell area: %.4f (OCCT 3662.79, gate [3626.16, 3699.42])", area)
	if area < 3626.16 || area > 3699.42 {
		t.Fatalf("S1 area %.4f outside OCCT 1%% gate [3626.16, 3699.42]", area)
	}
}

// t4RunoutFils resolves the SAME edgeFillet the production FilletEdges path builds for the T4
// corpus case (corpus.json simple/T4: radius 8 on the edge at midpoint (0,-30,0)) — routed through
// resolveFilletPicks + computeCorners + computeFillets exactly as filletResolvedEdges does
// (fillet.go:148-156), rather than a hand-rolled computeEdgeFillet call with fabricated blends/
// miters maps (those feed corner-tangent solving that an empty map skips silently).
func t4RunoutFils(t *testing.T) (*topo.Body, []edgeFillet) {
	t.Helper()
	body := importCorpusSolid(t, "simple/T4")
	e := edgeAtMidpoint(body, math.P3(0, -30, 0))
	if e == nil {
		t.Fatal("t4RunoutFils: T4 corpus-picked edge (midpoint 0,-30,0) not found")
	}
	edges, err := resolveFilletPicks(body, []EdgeFilletRadii{{Key: e.ReferenceKey(), R0: 8, R1: 8}})
	if err != nil {
		t.Fatalf("t4RunoutFils: resolveFilletPicks: %v", err)
	}
	blends, miters, err := computeCorners(edges)
	if err != nil {
		t.Fatalf("t4RunoutFils: computeCorners: %v", err)
	}
	fils, err := computeFillets(body, edges, blends, miters, FillConcaveOutward, nil)
	if err != nil {
		t.Fatalf("t4RunoutFils: computeFillets: %v", err)
	}
	return body, fils
}

// t4FragileRunoutFils adds the same filletRebuildMaps filletResultFaces builds (fillet_faces.go:
// 17-22) on top of t4RunoutFils's edgeFillet. buildRunoutHostsAndWalls consumes maps.abSubst to
// re-cut the host planes; with an empty map it honest-rejects for an unrelated reason (the host
// re-cut looks malformed), so a test built on a zero-value filletRebuildMaps would pass whether or
// not the fragile-band guard exists — silently proving nothing. Named so the test can assert
// directly against collectRunouts's real production inputs.
func t4FragileRunoutFils(t *testing.T) (*topo.Body, []edgeFillet, filletRebuildMaps) {
	t.Helper()
	body, fils := t4RunoutFils(t)
	abSubst, endCorner, edgeInserts := filletMaps(fils)
	fans, fanV := classifyEndCorners(fils)
	spreads, _ := buildSpreadMaps(fans, body)
	pruneEndCorners(endCorner, fanV)
	maps := filletRebuildMaps{abSubst: abSubst, endCorner: endCorner, edgeInserts: edgeInserts, spreads: spreads}
	return body, fils, maps
}

// TestCollectRunouts_DefersOnFragileBand pins the bodyHasFragileBand guard in collectRunouts
// (fillet_runout_faces.go:33) against the T4 regression: T4's imported STEP already carries a
// TOROIDAL_SURFACE survivor band, and firing the runout re-weld on its front-top edge (radius 8,
// the corpus pick) loses that torus's trim classification, inflating the rebuilt area from OCCT's
// 19514.7 to 32463.27 (full-domain torus fallback, confirmed locally via FilletEdges end-to-end
// with the guard temporarily disabled). The guard must defer the WHOLE edge to baseline (empty
// replace/extra/handled) whenever the body carries a fragile (torus/b-spline) band, same scope
// decision as the mid-span obstacle path (collectObstacles, ADR-4).
//
// RED check (2026-07-14): with `if bodyHasFragileBand(body) { ... }` in collectRunouts replaced by
// `if false && bodyHasFragileBand(body) { ... }`, this test FAILS — collectRunouts on this exact
// (body, fils) fires the runout rebuild (handled becomes non-empty) once the guard is bypassed.
// Verified locally (temporary edit, reverted, not committed) alongside the FilletEdges area check
// above (19640.09 with the guard vs 32463.27 without).
func TestCollectRunouts_DefersOnFragileBand(t *testing.T) {
	body, fils, maps := t4FragileRunoutFils(t)
	if !bodyHasFragileBand(body) {
		t.Fatal("TestCollectRunouts_DefersOnFragileBand: T4 fixture has no torus/b-spline survivor face — fixture regressed")
	}
	replace, extra, handled := collectRunouts(body, fils, ResolutionForBody(body),
		map[uint64]bool{}, maps)
	if len(replace) != 0 || len(extra) != 0 || len(handled) != 0 {
		t.Fatalf("collectRunouts fired on a fragile-band body (T4 torus survivor): replace=%d extra=%d handled=%d — expected the runout path to defer to baseline",
			len(replace), len(extra), len(handled))
	}
}
