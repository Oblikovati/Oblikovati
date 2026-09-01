// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// surfacePeriodic classifies each analytic surface's wrap: plane neither, cylinder/cone u only,
// sphere u only (latitude is bounded), torus both.
func TestSurfacePeriodic(t *testing.T) {
	t.Parallel()
	pl, _ := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	cyl, _ := geom.NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 1)
	sph, _ := geom.NewSphere(math.P3(0, 0, 0), 1)
	tor, _ := geom.NewTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 2)
	for _, c := range []struct {
		name       string
		s          geom.Surface
		uPer, vPer bool
	}{
		{"plane", pl, false, false},
		{"cylinder", cyl, true, false},
		{"sphere", sph, true, false},
		{"torus", tor, true, true},
	} {
		if u, v := surfacePeriodic(c.s); u != c.uPer || v != c.vPer {
			t.Errorf("surfacePeriodic(%s) = (%v,%v), want (%v,%v)", c.name, u, v, c.uPer, c.vPer)
		}
	}
}

// continueUV keeps a wrapped parameter sequence monotone across the seam in both directions.
func TestContinueUVUnwrap(t *testing.T) {
	t.Parallel()
	// u wraps from ~2π back to ~0: the seam jump is unwrapped forward past 2π.
	ring := []math.Point2{math.P2(2*stdmath.Pi-0.1, 3)}
	u, _ := continueUV(ring, 0.1, 3, true, false)
	if stdmath.Abs(u-(2*stdmath.Pi+0.1)) > 1e-9 {
		t.Errorf("u unwrap across seam = %g, want ≈2π+0.1", u)
	}
	// v periodic (torus tube) unwraps too; a non-periodic axis is left untouched.
	_, vv := continueUV([]math.Point2{math.P2(0, 2*stdmath.Pi-0.1)}, 0, 0.1, false, true)
	if stdmath.Abs(vv-(2*stdmath.Pi+0.1)) > 1e-9 {
		t.Errorf("v unwrap across seam = %g, want ≈2π+0.1", vv)
	}
	if _, keep := continueUV([]math.Point2{math.P2(0, 100)}, 0, 0.1, false, false); keep != 0.1 {
		t.Errorf("non-periodic v left = %g, want 0.1", keep)
	}
}

// castAxis routes the even-odd uv ray to a non-periodic UNBOUNDED axis (plane, cylinder, cone) and
// declines for a sphere/torus (no exterior endpoint), where the classifier uses the geodesic winding.
func TestCastAxisRouting(t *testing.T) {
	t.Parallel()
	pl, _ := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	cyl, _ := geom.NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 1)
	cone, _ := geom.NewCone(math.P3(0, 0, 0), math.V3(0, 0, 1), stdmath.Pi/4)
	sph, _ := geom.NewSphere(math.P3(0, 0, 0), 1)
	tor, _ := geom.NewTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 2)
	for _, c := range []struct {
		name   string
		s      geom.Surface
		wantOK bool
	}{
		{"plane", pl, true}, {"cylinder", cyl, true}, {"cone", cone, true},
		{"sphere", sph, false}, {"torus", tor, false},
	} {
		uPer, vPer := surfacePeriodic(c.s)
		if _, ok := castAxis(c.s, uPer, vPer); ok != c.wantOK {
			t.Errorf("castAxis(%s) ok=%v, want %v (sphere/torus must defer to the geodesic winding)", c.name, ok, c.wantOK)
		}
	}
}

// TestLoopToUVBridgesTheConeApex pins bridgePoleBranch and its helpers: a full cone's side loop runs
// down the seam, round the rim and back up the SAME ruling, so it closes in 3-D but not in u. The
// unwrap must re-branch across the apex and emit the pole on both branches, leaving a ring that spans
// a whole turn and closes (Oblikovati/Oblikovati#3447). A frustum's band loop already closes and must
// come back byte-identical.
func TestLoopToUVBridgesTheConeApex(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name      string
		topRadius float64
		wantSpan  float64
	}{
		{"full cone (apex pole)", 0, 2 * stdmath.Pi},
		{"frustum (no pole)", 6, 2 * stdmath.Pi},
	} {
		f := coneSideCurvedFace(t, c.topRadius)
		uPer, vPer := surfacePeriodic(f.surface)
		ring := loopToUV(f.surface, f.loops[0], uPer, vPer)
		if turns := ringMissingTurns(ring, false); turns != 0 {
			t.Errorf("%s: ring is %v whole turns short of closing, want 0", c.name, turns)
		}
		u0, u1, _, _, ok := polyBounds([][]math.Point2{ring})
		if !ok || stdmath.Abs((u1-u0)-c.wantSpan) > 1e-9 {
			t.Errorf("%s: ring u span = %g (ok=%v), want %g", c.name, u1-u0, ok, c.wantSpan)
		}
		if a := stdmath.Abs(signedArea2D(ring)); a < 1 {
			t.Errorf("%s: ring encloses %g in (u, v) — a degenerate sliver, not a region", c.name, a)
		}
	}
}

// coneSideCurvedFace returns the single curved face of a cone/frustum solid built about the z axis.
func coneSideCurvedFace(t *testing.T, topRadius float64) curvedFace {
	t.Helper()
	body, err := SolidCylinderCone(math.P3(0, 0, 0), math.P3(0, 0, 10), 3, topRadius, "cone")
	if err != nil {
		t.Fatalf("SolidCylinderCone: %v", err)
	}
	for _, f := range facesOfAny(body) {
		if _, isCone := f.surface.(geom.Cone); isCone {
			return f
		}
	}
	t.Fatalf("no cone face on a cone solid with top radius %g", topRadius)
	return curvedFace{}
}

// TestSampleOnPoleReadsTheTangentCollapse: the pole probe must fire only where the periodic tangent
// vanishes — the cone's apex station — and nowhere along its regular rulings.
func TestSampleOnPoleReadsTheTangentCollapse(t *testing.T) {
	t.Parallel()
	cone, err := geom.NewCone(math.P3(0, 0, 0), math.V3(0, 0, 1), stdmath.Pi/6)
	if err != nil {
		t.Fatalf("NewCone: %v", err)
	}
	for _, c := range []struct {
		name string
		q    math.Point2
		want bool
	}{
		{"the apex", math.P2(1.2, 0), true},
		{"the apex from another azimuth", math.P2(4, 0), true},
		{"a regular point up the ruling", math.P2(1.2, 5), false},
	} {
		if got := sampleOnPole(cone, c.q, false); got != c.want {
			t.Errorf("sampleOnPole(%s %v) = %v, want %v", c.name, c.q, got, c.want)
		}
	}
}

// TestRebranchRingTailKeepsThePoleOnBothBranches: the traverse along the pole isoline must become an
// explicit edge, so the shifted ring is one sample longer and holds the pole twice.
func TestRebranchRingTailKeepsThePoleOnBothBranches(t *testing.T) {
	t.Parallel()
	ring := []math.Point2{math.P2(0, 3), math.P2(-2, 0), math.P2(-2, 3)}
	out := rebranchRingTail(ring, 1, 2, false)
	want := []math.Point2{math.P2(0, 3), math.P2(-2, 0), math.P2(0, 0), math.P2(0, 3)}
	if len(out) != len(want) {
		t.Fatalf("rebranched ring has %d samples, want %d", len(out), len(want))
	}
	for i := range want {
		if out[i] != want[i] {
			t.Errorf("rebranched ring[%d] = %v, want %v", i, out[i], want[i])
		}
	}
}
