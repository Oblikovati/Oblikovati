// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	"math"
	"strings"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/blend"
	"oblikovati.org/kernel/topo"
	gmath "oblikovati.org/math"
)

// countPlanes counts a body's planar faces.
func countPlanes(b *topo.Body) int {
	n := 0
	for _, f := range b.Faces() {
		if _, ok := f.Geometry().(geom.Plane); ok {
			n++
		}
	}
	return n
}

// TestFilletOpenCurvedTangentStripe is the ADR-0050 P6 open-run acceptance: filleting a CONTIGUOUS OPEN
// sub-run of a curved tangent chain (a straight–arc–straight stretch of the top rim of a box whose
// vertical edges are already rounded) rounds the run as one stripe and closes each end with a flat
// setback cap (OCCT's free-end ChFi3d_CoupeParPlan). The result is a valid closed manifold solid with
// two new planar cap faces, and the removed volume matches the exact rolling-ball notch integral (the
// ground truth OCCT approximates) — verifying the caps orient the tube outward, not inside-out.
func TestFilletOpenCurvedTangentStripe(t *testing.T) {
	t.Parallel()
	filleted := boxWithRoundedVerticals(t, 4, 0.5)
	before := ops.BodyGeometryProperties(filleted, ops.DefaultQuality()).Volume
	planesBefore := countPlanes(filleted)

	seed := firstStraightTopEdge(t, filleted)
	chain, _, err := blend.TangentEdgeChain(filleted, seed, blend.DefaultTangentChainAngle)
	if err != nil {
		t.Fatal(err)
	}
	open := chain[:3] // straight, arc, straight — a contiguous open sub-run of the closed loop
	const r = 0.25

	res, err := blend.FilletEdges(filleted, open, r)
	if err != nil {
		t.Fatalf("open curved tangent-stripe fillet failed: %v", err)
	}
	rep := ops.Validate(res)
	if !rep.Valid || !res.IsSolid() || !rep.Manifold || !rep.Closed || !rep.OrientationOK {
		t.Fatalf("open-stripe result invalid: valid=%v solid=%v manifold=%v closed=%v orient=%v issues=%v",
			rep.Valid, res.IsSolid(), rep.Manifold, rep.Closed, rep.OrientationOK, rep.Issues)
	}
	if rep.EulerCharacteristic != 2 {
		t.Errorf("Euler characteristic = %d, want 2 (genus-0 solid)", rep.EulerCharacteristic)
	}
	// Two flat run-out caps: the open run adds exactly two new planar faces over the closed body.
	if got := countPlanes(res) - planesBefore; got != 2 {
		t.Errorf("new planar faces = %d, want 2 (one flat setback cap per open end)", got)
	}
	// One torus over the single arc segment, two cylinders over the two straight segments (plus the four
	// pre-existing vertical-edge cylinders).
	if tori := countTorus(res); tori != 1 {
		t.Errorf("torus faces = %d, want 1 (the single arc segment)", tori)
	}
	after := ops.BodyGeometryProperties(res, ops.DefaultQuality()).Volume
	removed := before - after
	want := openStripeRemovedVolume(r)
	// Cross-check against OCCT: BRepFilletAPI_MakeFillet removes 0.198382 over the WHOLE 15.14-long
	// tangent loop (it propagates); the same per-unit cross-section over just this 6.79-long run scales
	// to ≈0.089, matching both `want` (the exact analytic) and our tessellated 0.0919 to a few percent.
	if math.Abs(removed-want) > 0.1*want {
		t.Errorf("removed volume = %g, want ≈%g (rolling-ball notch over straight–arc–straight)", removed, want)
	}
}

// openStripeRemovedVolume is the EXACT material a radius-r rolling-ball fillet removes over the
// straight–arc–straight run of the top rim (box side 4, vertical fillets 0.5). Each 90° convex edge
// removes the notch area r²(1−π/4) per unit length; the straight tops are length (4−2·0.5)=3 each, and
// the arc is a quarter turn of radius R=0.5 whose swept volume follows Pappus about the corner axis.
// This is the ground truth OCCT's BRepFilletAPI_MakeFillet converges to.
func openStripeRemovedVolume(r float64) float64 {
	const R, side, vfil = 0.5, 4.0, 0.5
	notch := r * r * (1 - math.Pi/4)
	straightLen := side - 2*vfil // 3
	straight := 2 * notch * straightLen
	// Pappus: the notch centroid rides at radius R − cx from the corner axis, where cx is the notch's
	// own centroid offset from the rim; swept through a quarter turn (π/2).
	cx := notchCentroidOffset(r)
	arc := notch * (math.Pi / 2) * (R - cx)
	return straight + arc
}

// notchCentroidOffset is the inward distance from the convex edge to the centroid of the corner notch
// (the r×r square minus its quarter disc), = r·(2/3)/(4−π) along each leg by symmetry.
func notchCentroidOffset(r float64) float64 {
	// square centroid at r/2, area r²; quarter-disc centroid at 4r/3π from the corner, area πr²/4.
	num := r*r*(r/2) - (math.Pi/4)*r*r*(4*r/(3*math.Pi))
	return num / (r * r * (1 - math.Pi/4))
}

// TestFilletSingleVerticalOpenChain fillets a genuinely OPEN maximal tangent chain: a box with ONE
// vertical edge rounded (r=0.5) has a top rim whose tangent run is straight–arc–straight, terminating at
// two sharp 90° corners (not a closed loop). Filleting it (r=0.25) builds the open stripe with a flat
// setback cap at each sharp-corner run-out. The removed volume is checked against a locally-built OCCT
// BRepFilletAPI_MakeFillet oracle (0.103246) — confirming our flat cap matches OCCT's intersection-at-end
// for a right-angle terminal, to tessellation tolerance.
func TestFilletSingleVerticalOpenChain(t *testing.T) {
	t.Parallel()
	box := csgBox(gmath.P3(0, 0, 0), 4, 4, 4)
	var one [][]byte
	for _, e := range box.Edges() {
		if a, c := e.StartVertex().Point(), e.EndVertex().Point(); a.X == c.X && a.Y == c.Y {
			one = append(one, e.ReferenceKey())
			break
		}
	}
	f, err := blend.FilletEdges(box, one, 0.5)
	if err != nil {
		t.Fatalf("single vertical fillet: %v", err)
	}
	before := ops.BodyGeometryProperties(f, ops.DefaultQuality()).Volume

	chain, closed, err := blend.TangentEdgeChain(f, firstStraightTopEdge(t, f), blend.DefaultTangentChainAngle)
	if err != nil {
		t.Fatal(err)
	}
	if closed || len(chain) != 3 {
		t.Fatalf("expected an OPEN 3-edge (straight–arc–straight) chain, got closed=%v len=%d", closed, len(chain))
	}
	res, err := blend.FilletEdges(f, chain, 0.25)
	if err != nil {
		t.Fatalf("open sas-chain fillet failed: %v", err)
	}
	rep := ops.Validate(res)
	if !rep.Valid || !res.IsSolid() || rep.EulerCharacteristic != 2 {
		t.Fatalf("open sas result invalid: valid=%v solid=%v chi=%d issues=%v",
			rep.Valid, res.IsSolid(), rep.EulerCharacteristic, rep.Issues)
	}
	after := ops.BodyGeometryProperties(res, ops.DefaultQuality()).Volume
	const occt = 0.103246
	if removed := before - after; math.Abs(removed-occt) > 0.05*occt {
		t.Errorf("removed volume = %g, want ≈%g (OCCT oracle)", removed, occt)
	}
}

// TestFilletStripeUnbuildableIsLocalized is the ADR-0050 P6 partial-result contract: a radius that
// cannot seat on the tangent chain (here r=0.5 equals the vertical-fillet radius, so the rolling-ball
// centre curve collapses on the arc segments) fails with a localized, actionable error naming the faulty
// segment and guide point — not a panic, not a silently wrong body. Mirrors OCCT ChFiDS_ErrorStatus.
func TestFilletStripeUnbuildableIsLocalized(t *testing.T) {
	t.Parallel()
	filleted := boxWithRoundedVerticals(t, 4, 0.5)
	top := topPerimeterKeys(t, filleted)

	res, err := blend.FilletEdges(filleted, top, 0.5) // r == arc radius ⇒ no rolling-ball fit on the arcs
	if err == nil {
		t.Fatal("expected a localized partial-result error for the unbuildable radius, got a body")
	}
	if res != nil {
		t.Errorf("unbuildable stripe must return no body, got %d faces", len(res.Faces()))
	}
	for _, want := range []string{"chain segment", "near ("} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q should localize the failure (contain %q)", err.Error(), want)
		}
	}
}

// boxWithRoundedVerticals builds a box of the given side with its four vertical edges filleted at vr.
func boxWithRoundedVerticals(t *testing.T, side, vr float64) *topo.Body {
	t.Helper()
	box := csgBox(gmath.P3(0, 0, 0), side, side, side)
	var verts [][]byte
	for _, e := range box.Edges() {
		if a, c := e.StartVertex().Point(), e.EndVertex().Point(); a.X == c.X && a.Y == c.Y {
			verts = append(verts, e.ReferenceKey())
		}
	}
	filleted, err := blend.FilletEdges(box, verts, vr)
	if err != nil {
		t.Fatalf("vertical fillet setup: %v", err)
	}
	return filleted
}

// firstStraightTopEdge returns a top-rim straight edge (a line, not an arc) to seed the tangent chain.
func firstStraightTopEdge(t *testing.T, b *topo.Body) []byte {
	t.Helper()
	maxZ := 0.0
	for _, v := range b.Vertices() {
		if v.Point().Z > maxZ {
			maxZ = v.Point().Z
		}
	}
	for _, e := range b.Edges() {
		if e.StartVertex().Point().Z >= maxZ-1e-9 && e.EndVertex().Point().Z >= maxZ-1e-9 {
			if _, isArc := e.Geometry().(geom.Arc3d); !isArc {
				return e.ReferenceKey()
			}
		}
	}
	t.Fatal("no straight top-rim edge found")
	return nil
}
