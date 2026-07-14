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
	ef, _ := runoutFixtureCrossingBoss(t)
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
	_ = ef
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
	area := BodyGeometryProperties(body, PropertyQuality()).Area
	t.Logf("rebuilt shell area: %.4f (OCCT 3662.79, gate [3626.16, 3699.42])", area)
	if area < 3626.16 || area > 3699.42 {
		t.Fatalf("S1 area %.4f outside OCCT 1%% gate [3626.16, 3699.42]", area)
	}
}
