// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// The obstacle path's two boundary NODES — where the obstacle rim crosses the fillet's receded tangent
// line on its host — used to be solved on the 64-chord SAMPLED rim polyline (rimCrossings/lerpAtZero),
// so every one of them sat a chord SAGITTA inside the true curve: measured 3.05e-03 … 3.74e-02 across
// the eleven obstacle-detecting corpus cases. The node is the END of both cylinder wings and the corner
// of the notch, so that error scaled real faces — U4's boss-B node was 3.815e-03 short of its exact
// ±√44 and shortened each sliver's z-span by 0.93 %. analyticNode (kernel/ops/fillet_obstacle_detect.go)
// re-solves the node on the rim CURVE inside the bracket the polyline found.
//
// These gates assert the RESULT on the shipped bodies, at the CLOSED FORM derived from each solid — not
// from any oracle's printout and not from our own solver. They fail loud if a later slice puts the node
// back on the polyline.
//
// WHAT IT MEASURES. The nearest vertex of the shipped result body (ops-driven feature recompute via
// shippedCaseBody) to the closed-form node point, in model units. The nodes survive into the body as
// real vertices: the notch loop, the split obstacle wall and both wings all meet there.

// obstacleNodeTol is how far a shipped node vertex may sit from its closed form. 1e-5 is ~380x tighter
// than the sampled-rim error this gate protects against (3.8e-03 on U4, 3.7e-03 on U3, 4.6e-03 on R9)
// and ~23x looser than the worst residual an EXACT rim can carry here — U4's boss-B rim is an imported
// b-spline whose own deviation from the true circle is 4.4e-07 (DRAWEXE's own patch end sits at the
// same 4.4e-07 from √44, so this is the oracle's floor, not ours).
const obstacleNodeTol = 1e-5

// TestObstacleNodesLandOnTheirClosedForm pins the exact node on the three obstacle cases whose closed
// form is derivable from the solid itself.
//
//   - simple/R9 — a 20-cube with an r=8 boss on its z=10 top, filleted r=3 along the y=−10 ∧ z=10 edge.
//     The receded boundary on the top plane is y = −10+3 = −7, the rim is the circle of radius 8 about
//     (0,0,10), so the nodes are x² = 64−49 = 15: (±√15, −7, 10).
//   - simple/U3 — the r=12 pipe boss on the z=10 host; its rim circle's centre is 10 from the receded
//     band line, so the nodes are y² = 144−100 = 44: (0, ±√44, 10).
//   - simple/U4 — boss B's mouth on the x=10 host reduces exactly to the circle of radius 12 about
//     (10,−5,0) and the fillet's B-tangent there is y=−15, so its OUTER nodes are z² = 144−100 = 44:
//     (10, −15, ±√44). DRAWEXE 8.0.0's own sliver patch ends at −6.63324914227121, 4.4e-07 from that
//     closed form — its tolerance — and we now land 1.2e-11 from OCCT's value.
func TestObstacleNodesLandOnTheirClosedForm(t *testing.T) {
	sqrt15, sqrt44 := stdmath.Sqrt(15), stdmath.Sqrt(44)
	for _, tc := range []struct {
		name  string
		nodes []math.Point3
	}{
		{"R9", []math.Point3{math.P3(sqrt15, -7, 10), math.P3(-sqrt15, -7, 10)}},
		{"U3", []math.Point3{math.P3(0, sqrt44, 10), math.P3(0, -sqrt44, 10)}},
		{"U4", []math.Point3{math.P3(10, -15, sqrt44), math.P3(10, -15, -sqrt44)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body, ok := shippedCaseBody(caseRecord(t, "simple", tc.name), CorpusFixtureDir())
			if !ok {
				t.Fatalf("simple/%s: no shipped body", tc.name)
			}
			for _, want := range tc.nodes {
				if d := nearestVertexDistance(body, want); d > obstacleNodeTol {
					t.Errorf("simple/%s: no body vertex within %g of the closed-form node (%.12f, %.12f, %.12f); nearest is %.9e away — the node is back on the sampled rim polyline",
						tc.name, obstacleNodeTol, want.X, want.Y, want.Z, d)
				}
			}
		})
	}
}

// TestU4HostANodeKeepsItsCoupledStation is the OTHER half of the node solver: U4's host-A node must NOT
// be refined to the fixed-tangent closed form, because on a DUAL-host edge that closed form is not the
// truth. At host A's node station boss B is ALREADY setting the rolling ball back, so the ball centre has
// migrated off the plain fillet axis and the true node is where boss A's r=8 rim meets the MIGRATED
// tangency foot, not the fixed A-tangent line x=5. DRAWEXE 8.0.0's own sliver pole
// (5.00625411, −20, −6.23998556) lies on boss A's r=8 rim to 1.4e-05 and so fixes that true node's axis
// STATION at |z| = 6.23998556. Ours reads 6.23981590, 1.697e-04 off it; the fixed-tangent closed form
// √39 = 6.24499800 is 5.012e-03 off it, 30x worse. coupledNodeStation
// (kernel/ops/fillet_obstacle_detect_face.go) therefore leaves this node exactly where the sampled
// polyline had it, and the coupled solve is a tracked follow-up. This gate fails if that guard is removed.
//
// The node's x (5, vs the true foot's 5.00625) is a DIFFERENT, already-recorded defect — sectionEndA
// holds host A's rail on the fixed tangent line across the whole sliver span (u4-canal-report.md §4.4) —
// and is deliberately not gated here: the STATION is what partitions the sliver/core panels and scales
// their areas, and it is the only thing the node solver decides.
func TestU4HostANodeKeepsItsCoupledStation(t *testing.T) {
	body, ok := shippedCaseBody(caseRecord(t, "simple", "U4"), CorpusFixtureDir())
	if !ok {
		t.Fatal("simple/U4: no shipped body")
	}
	const occtStation = 6.23998556 // DRAWEXE 8.0.0's own sliver-patch v=1 station on boss A's rim
	got, found := stdmath.Inf(1), false
	for _, v := range body.Vertices() {
		p := v.Point()
		if stdmath.Abs(p.X-5) > 1e-9 || stdmath.Abs(p.Y+20) > 1e-9 {
			continue // not on host A's plane at the fillet's A-tangent line
		}
		if z := stdmath.Abs(p.Z); z > 5 && z < 6.5 && stdmath.Abs(z-occtStation) < stdmath.Abs(got-occtStation) {
			got, found = z, true
		}
	}
	if !found {
		t.Fatal("simple/U4: no host-A node vertex on the A-tangent line — the dual-host partition changed shape")
	}
	if d := stdmath.Abs(got - occtStation); d > 5e-4 {
		t.Errorf("simple/U4: host A's node station |z|=%.12f is %.9e from OCCT's own coupled station %.8f, want within 5e-4 — a node whose station the OTHER boss governs was solved against the fixed tangent line (closed form √39=%.8f, 5.0e-3 the wrong way)",
			got, d, occtStation, stdmath.Sqrt(39))
	}
}
