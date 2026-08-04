// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"sort"
	"testing"

	"oblikovati.org/kernel/geom"
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
	faces, fired := filletResultFaces(body, []edgeFillet{fil}, map[uint64]*cornerBlend{}, true, true)
	if !fired {
		t.Fatal("s1RebuiltShell: runout rebuild did not fire")
	}
	return assembleBody(faces)
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
	blends, miters, err := computeCorners(body, edges)
	if err != nil {
		t.Fatalf("t4RunoutFils: computeCorners: %v", err)
	}
	fils, err := computeFillets(body, edges, blends, miters, FillConcaveOutward, nil)
	if err != nil {
		t.Fatalf("t4RunoutFils: computeFillets: %v", err)
	}
	return body, fils
}

// t4FragileRunoutFils adds the same filletRebuildMaps filletResultFaces builds (filletBuildMaps,
// fillet_faces.go) on top of t4RunoutFils's edgeFillet. This is real production input, not a zero-value
// filletRebuildMaps, so TestCollectRunouts_TorusFiresOnRunoutPath actually exercises collectRunouts
// end-to-end: T4's torus survivor no longer trips runoutDefersBody, so the intact-boss runout builds.
// Named so the test can assert directly against collectRunouts's real production inputs.
func t4FragileRunoutFils(t *testing.T) (*topo.Body, []edgeFillet, filletRebuildMaps) {
	t.Helper()
	body, fils := t4RunoutFils(t)
	abSubst, endCorner, edgeInserts := filletMaps(fils)
	fans, fanV := classifyEndCorners(fils)
	spreads, _ := buildSpreadMaps(fans, body)
	pruneEndCorners(endCorner, fanV)
	maps := filletRebuildMaps{abSubst: abSubst, endCorner: endCorner, edgeInserts: edgeInserts,
		insertCurves: map[*topo.Face]map[uint64][]geom.Curve3{}, spreads: spreads}
	return body, fils, maps
}

// TestCollectRunouts_TorusFiresOnRunoutPath pins the M4-Task-3 narrowing of the runout deferral: a TORUS
// survivor body (T4's imported STEP carries a TOROIDAL_SURFACE band + a crossing torus boss) is NO LONGER
// deferred on the runout path — runoutDefersBody(T4)==false — so collectRunouts now FIRES the intact-boss
// runout (handled non-empty), keeping the torus boss whole (band_ring_chain.go meshes it correctly). The
// old boss-splitting path had to defer (torus split → full-domain donut, 32463.27 vs OCCT 19514.7); the
// intact path never splits, so the per-boss-type setbackBossesFaithful whitelist is the real gate now.
//
// The SHARED obstacle gate stays UNTOUCHED: bodyHasFragileBand(T4) is STILL true (a torus IS fragile on
// the obstacle re-weld), so collectObstacles keeps deferring it — this asserts both that the narrowing is
// scoped to the runout path and that the obstacle path is byte-identical.
func TestCollectRunouts_TorusFiresOnRunoutPath(t *testing.T) {
	body, fils, maps := t4FragileRunoutFils(t)
	if runoutDefersBody(body) {
		t.Fatal("runoutDefersBody(T4 torus) = true — the runout narrowing (M4 Task 3) regressed; a torus survivor must NOT defer here")
	}
	if !bodyHasFragileBand(body) {
		t.Fatal("bodyHasFragileBand(T4 torus) = false — the SHARED obstacle gate was changed; it must stay true (obstacle path untouched)")
	}
	_, _, handled := collectRunouts(body, fils, ResolutionForBody(body), map[uint64]bool{}, maps)
	if len(handled) == 0 {
		t.Fatal("collectRunouts did not fire the intact-boss runout on the T4 torus body — expected it to build, not defer")
	}
}

// TestRunoutDefersBody_BSplineOnly pins that the runout deferral is narrowed to FREE-FORM (b-spline)
// survivor bands ONLY: a b-spline body (T9: {BSplineSurface:1, Plane:7}, m4-spike §T9) still defers (no
// band mesher recovers a doubly-periodic b-spline on re-weld), while a torus body (T4) does NOT — the
// exact split from the shared bodyHasFragileBand (which defers BOTH). Guards against a future re-widening.
func TestRunoutDefersBody_BSplineOnly(t *testing.T) {
	bspline := importCorpusSolid(t, "simple/T9")
	if !runoutDefersBody(bspline) {
		t.Fatal("runoutDefersBody(T9 b-spline) = false — a free-form survivor band must still defer on the runout path")
	}
	torus, _ := t4RunoutFils(t)
	if runoutDefersBody(torus) {
		t.Fatal("runoutDefersBody(T4 torus) = true — a torus survivor must no longer defer (M4 Task 3)")
	}
}
