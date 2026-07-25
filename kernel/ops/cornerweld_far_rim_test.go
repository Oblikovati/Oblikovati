// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"os"
	"strings"
	"testing"

	"oblikovati.org/kernel/exchange"
	"oblikovati.org/kernel/exchange/step"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Gates for the rim-continuation far termination (design Axis B3) driven on the REAL N4 fixture — the same
// STEP file the corpus scores. They pin BOTH directions of the admission gate: a capped far vertex must NOT
// enter the walk (this is the count==0 unreachability that keeps every existing green out of the new branch),
// and a seam vertex whose continuation is unavailable must DECLINE rather than guess.

const cornerWeldFixtureDir = "../../model/feature/occtparity/fixtures/simple/"

// importCornerWeldFixture loads one corpus fixture solid.
func importCornerWeldFixture(t *testing.T, name string) *topo.Body {
	t.Helper()
	data, err := os.ReadFile(cornerWeldFixtureDir + name + ".step")
	if err != nil {
		t.Skipf("fixture %s unavailable: %v", name, err)
	}
	bodies, _, err := step.Reader{}.ImportSolids(data, exchange.TranslationOptions{TargetUnitMM: 1})
	if err != nil || len(bodies) != 1 {
		t.Skipf("fixture %s import-divergence: %v (%d bodies)", name, err, len(bodies))
	}
	return bodies[0]
}

// edgeNearMidpoint finds the body edge whose curve midpoint is nearest want (the corpus locator's own way of
// resolving a pick).
func edgeNearMidpoint(t *testing.T, b *topo.Body, want math.Point3) *topo.Edge {
	t.Helper()
	var best *topo.Edge
	bestD := 1e18
	for _, e := range b.Edges() {
		if d := float64(e.Geometry().PointAt(0.5).DistanceTo(want)); d < bestD {
			best, bestD = e, d
		}
	}
	if best == nil || bestD > 1e-3 {
		t.Fatalf("no edge with midpoint near %v (best %.4g away)", want, bestD)
	}
	return best
}

// n4RimLink builds the cornerArmLink for N4's picked CONVEX cap-rim arc (hostA = the cap plane, hostB = the
// boss wall) plus the arm's torus surface and a plan carrying the three picked edge ids.
func n4RimLink(t *testing.T) (cornerArmLink, geom.Torus, cornerWeldPlan) {
	t.Helper()
	b := importCornerWeldFixture(t, "N4")
	corner := math.P3(112.37262951457501, 61.4198024742836, 50) // the shared trihedral vertex
	rim := edgeNearMidpoint(t, b, math.P3(127.31712179477491, 64.732916648636035, 50))
	ccyl := edgeNearMidpoint(t, b, math.P3(112.3726295145, 61.4198024742, 25))
	band := edgeNearMidpoint(t, b, math.P3(114.1091112912442, 71.267880004405725, 50))
	faces := rim.Faces()
	if len(faces) != 2 {
		t.Fatalf("the picked rim edge has %d faces, want 2", len(faces))
	}
	hostA, hostB := faces[0], faces[1]
	if _, isPlane := hostA.Geometry().(geom.Plane); !isPlane {
		hostA, hostB = hostB, hostA
	}
	torus := mustTorus(t, math.P3(115.84559306791, 81.115957534528, 45), math.V3(0, 0, 1), 15, 5)
	plan := cornerWeldPlan{
		ledger: newCornerWeldLedger(), vertex: corner, radius: 5,
		filleted: map[uint64]bool{rim.ID(): true, ccyl.ID(): true, band.ID(): true},
	}
	return cornerArmLink{edge: rim, hostA: hostA, hostB: hostB, farVtx: farEndVertex(rim, corner)}, torus, plan
}

// TestRimVertexIsSeamOnlyWhenNoCapExists is the count==0 gate, both ways. N4's picked cap-rim arc ends at a
// G1 seam (the boss wall's two faces meet there tangentially, so there is NO transverse capping face) → seam.
// Its OTHER end is the corner vertex, which has three faces around it including transverse ones → not a seam.
// This is the property that makes the rim-continuation branch unreachable for every existing green: they all
// terminate at a vertex with exactly ONE transverse capping face.
func TestRimVertexIsSeamOnlyWhenNoCapExists(t *testing.T) {
	link, _, plan := n4RimLink(t)
	seam, reason := rimVertexIsSeam(link)
	if reason != "" || !seam {
		t.Fatalf("N4's rim far vertex %d: seam=%v reason=%q, want a seam (0 transverse capping faces)", link.farVtx.ID(), seam, reason)
	}
	atCorner := link
	atCorner.farVtx = otherEdgeVertex(link.edge, link.farVtx)
	seam, reason = rimVertexIsSeam(atCorner)
	if reason != "" || seam {
		t.Fatalf("the corner end of N4's rim: seam=%v reason=%q, want NOT a seam (a capped/trihedral vertex must never enter the walk)", seam, reason)
	}
	_ = plan
}

// TestRimContinuationWalksToTheCappedEnd pins the walk's result on N4: exactly two links (the picked 90° arc
// plus the 180° continuation across the boss wall's second face), ending at a vertex the far-runout engine
// can cap. A regression that stops at the seam would emit a different solid.
func TestRimContinuationWalksToTheCappedEnd(t *testing.T) {
	link, torus, plan := n4RimLink(t)
	res := ResolutionForPoints([]math.Point3{math.P3(0, 0, 0), math.P3(200, 200, 100)})
	links, reason := rimContinuationLinks(link, torus, plan, res)
	if reason != "" {
		t.Fatalf("rim continuation declined on N4: %s", reason)
	}
	if len(links) != 2 {
		t.Fatalf("N4's rim chain has %d links, want 2 (the picked 90° arc + the 180° continuation)", len(links))
	}
	if links[1].hostA != links[0].hostA {
		t.Fatal("the continuation must keep the cap plane as hostA (the rim runs across ONE top plane)")
	}
	if links[1].hostB == links[0].hostB {
		t.Fatal("the continuation must cross to the boss wall's OTHER face — that seam is what splits the band")
	}
	if seam, _ := rimVertexIsSeam(links[1]); seam {
		t.Fatal("the chain's last link must end at a CAPPED vertex, not another seam")
	}
}

// TestRimContinuationDeclinesWhenTheRimIsAlreadyFilleted is the do-no-harm guard: a continuation edge that is
// itself a picked edge is the fillet-fillet regime, not a plain rim, so the walk must DECLINE with the
// offending vertex rather than run the arm through it.
func TestRimContinuationDeclinesWhenTheRimIsAlreadyFilleted(t *testing.T) {
	link, torus, plan := n4RimLink(t)
	for _, e := range link.farVtx.Edges() {
		plan.filleted[e.ID()] = true // every candidate is now a picked edge
	}
	res := ResolutionForPoints([]math.Point3{math.P3(0, 0, 0), math.P3(200, 200, 100)})
	_, reason := rimContinuationLinks(link, torus, plan, res)
	if !strings.Contains(reason, "rim continuation") || !strings.Contains(reason, "want exactly 1") {
		t.Fatalf("declined with %q, want a named rim-continuation decline carrying the candidate count", reason)
	}
}

// TestRimContinuationDeclinesWhenTheBallLeavesAHost is the geometric guard: the continuation must carry the
// SAME rolling ball. Walking with a ball radius the hosts cannot host at all must decline, never accept a
// continuation that is a different fillet.
func TestRimContinuationDeclinesWhenTheBallLeavesAHost(t *testing.T) {
	link, torus, plan := n4RimLink(t)
	plan.radius = 7 // no longer the r=5 tube the two hosts are tangent to at this vertex
	res := ResolutionForPoints([]math.Point3{math.P3(0, 0, 0), math.P3(200, 200, 100)})
	if _, reason := rimContinuationLinks(link, torus, plan, res); reason == "" {
		t.Fatal("a continuation whose hosts do not host the arm ball at radius r must decline")
	}
}
