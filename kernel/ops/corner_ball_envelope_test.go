// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// TestMaxBallDevRejectsAnInteriorBulgeTheOldCertificateAccepted is the falsification the certificate
// exists for. It mutates a CORRECT run-out canal with an ENDPOINT-PRESERVING INTERIOR BULGE — the
// shoulder control row is pushed out at every interior station along the direction it already points,
// which leaves all four boundary curves bit-unchanged and leaves the tangency-host isoparm's normal
// direction unchanged too — and asserts:
//
//	RED   the old five-field certificate (Closed / WeldsArms / NoFold / MaxDev / MaxAngleDev) ACCEPTS
//	      the bulged surface, exactly as coons4-audit.md §C.3 says it must: every one of those fields
//	      is a boundary or structural property and the boundary was not touched;
//	GREEN MaxBallDev REJECTS it, and reads ~1e-9 on the unmutated surface it was built from.
//
// A certificate that has not been falsified is a claim, not a guard; this is the guard.
func TestMaxBallDevRejectsAnInteriorBulgeTheOldCertificateAccepted(t *testing.T) {
	t.Parallel()
	loop, surf, res := s1FlankCanal(t)
	weld := res.Weld()

	trueDev := maxBallDev(surf, loop.Envelope)
	if trueDev > weld {
		t.Fatalf("the UNMUTATED canal must certify: MaxBallDev %g > weld %g", trueDev, weld)
	}

	bulged := bulgeShoulderRow(t, surf, 0.05*surf.Ctrl[0][0].DistanceTo(surf.Ctrl[2][0]))
	assertBoundaryUnmoved(t, surf, bulged)

	old := certifyRunoutCanalPatch(bulged, loop, maxLoopSurfaceDev(bulged, runoutPatchLoops(loop)), res)
	if !old.Closed || !old.WeldsArms || !old.NoFold || old.MaxDev > weld || old.MaxAngleDev > seamAngularTol {
		t.Fatalf("the bulge was meant to be invisible to the OLD fields; got Closed=%v WeldsArms=%v NoFold=%v "+
			"MaxDev=%g (weld %g) MaxAngleDev=%g (tol %g) — re-tune the mutation, do not weaken the claim",
			old.Closed, old.WeldsArms, old.NoFold, old.MaxDev, weld, old.MaxAngleDev, seamAngularTol)
	}

	bulgedDev := maxBallDev(bulged, loop.Envelope)
	if bulgedDev <= weld {
		t.Errorf("MaxBallDev accepted an interior bulge: %g <= weld %g — the certificate is not a guard", bulgedDev, weld)
	}
	t.Logf("RED-then-GREEN: true MaxBallDev=%.6e, bulged MaxBallDev=%.6e, weld=%.6e (rejection margin %.1fx)",
		trueDev, bulgedDev, weld, bulgedDev/weld)
}

// TestMaxBallDevIsZeroWithoutAnExtractorPayload pins the deliberate abstention: with no declared
// envelope the residual is 0, never a guess. coons4-audit.md §B.4 measured a certify-time guess at the
// roll hosts reading 5–19% even on OCCT's own CORRECT patches, so an inferred payload would reject good
// geometry — the claim has to come from the extractor that knows the topology.
func TestMaxBallDevIsZeroWithoutAnExtractorPayload(t *testing.T) {
	t.Parallel()
	_, surf, _ := s1FlankCanal(t)
	if got := maxBallDev(surf, nil); got != 0 {
		t.Errorf("maxBallDev(nil envelope) = %g, want 0", got)
	}
	if got := maxBallDev(surf, &BallEnvelope{Radius: 6}); got != 0 {
		t.Errorf("maxBallDev(empty envelope) = %g, want 0", got)
	}
}

// s1FlankCanal resolves the real S1 fixture's +x flank band and lofts it — the shared arrangement.
func s1FlankCanal(t *testing.T) (RailLoop, geom.BSplineSurface, Resolution) {
	t.Helper()
	ef, res := runoutFixtureCrossingBoss(t)
	b, ok := detectSetbackBands(ef, res)
	if !ok {
		t.Fatalf("detectSetbackBands declined the S1 fixture")
	}
	loops, ok := extractSetbackPatches(b, ef, res)
	if !ok || len(loops) != 3 {
		t.Fatalf("extractSetbackPatches: ok=%v loops=%d, want 3", ok, len(loops))
	}
	loop := loops[2] // the +x flank: one tangency host + one restriction curve
	rc := loop.Runout
	surf, err := geom.LoftCanalStations(rc.Centers, rc.FeetA, rc.FeetB, loop.Envelope.Radius, res.Weld())
	if err != nil {
		t.Fatalf("LoftCanalStations: %v", err)
	}
	return loop, surf, res
}

// bulgeShoulderRow pushes the canal's middle (shoulder) control row outward at every INTERIOR station,
// along the direction that row already points toward the u=1 row. Columns 0 and N−1 are untouched, so
// all four boundary curves are bit-identical; the direction choice keeps (P2−P1) parallel, so the
// tangency-host isoparm's normal — the one MaxAngleDev measures — is unchanged too.
func bulgeShoulderRow(t *testing.T, s geom.BSplineSurface, delta math.Scalar) geom.BSplineSurface {
	t.Helper()
	ctrl := make([][]math.Point3, len(s.Ctrl))
	for i := range s.Ctrl {
		ctrl[i] = append([]math.Point3(nil), s.Ctrl[i]...)
	}
	for j := 1; j < len(ctrl[1])-1; j++ {
		dir := ctrl[1][j].VectorTo(ctrl[2][j])
		l := dir.Length()
		if l == 0 {
			continue
		}
		ctrl[1][j] = ctrl[1][j].TranslateBy(dir.Scale(delta / l))
	}
	out, err := geom.NewBSplineSurface(s.UDegree, s.VDegree, ctrl, s.Weights, s.UKnots, s.VKnots)
	if err != nil {
		t.Fatalf("bulgeShoulderRow: %v", err)
	}
	return out
}

// assertBoundaryUnmoved proves the mutation really is endpoint-preserving: every one of the four
// boundary isoparms is bit-identical between the original and the bulged surface. Without this the
// RED-then-GREEN claim would be unfalsifiable in the other direction — a mutation that moved the
// boundary would be caught by MaxDev and would prove nothing about the interior.
func assertBoundaryUnmoved(t *testing.T, a, b geom.BSplineSurface) {
	t.Helper()
	u0, u1 := a.UDomain()
	v0, v1 := a.VDomain()
	for i := 0; i <= 32; i++ {
		f := float64(i) / 32
		for _, pair := range [][2]float64{{u0, v0 + (v1-v0)*f}, {u1, v0 + (v1-v0)*f},
			{u0 + (u1-u0)*f, v0}, {u0 + (u1-u0)*f, v1}} {
			if d := a.PointAt(pair[0], pair[1]).DistanceTo(b.PointAt(pair[0], pair[1])); d != 0 {
				t.Fatalf("boundary moved at (u=%g, v=%g) by %g; the mutation must preserve the endpoints",
					pair[0], pair[1], d)
			}
		}
	}
}

// TestSectionCurvePointFindsTheBandSideCrossing pins the residual's section solve: a plane through a
// point, normal to the spine, meets a footprint circle twice, and the measure must take the crossing on
// the sample's own side — the far one is a different part of the boss entirely.
func TestSectionCurvePointFindsTheBandSideCrossing(t *testing.T) {
	t.Parallel()
	circle, err := geom.NewCircle(math.P3(0, 0, 0), math.V3(0, 0, 1), 13)
	if err != nil {
		t.Fatalf("NewCircle: %v", err)
	}
	// A sample at x=5 on the −y side: the section plane x=5 meets the circle at y=±12.
	got, ok := sectionCurvePoint(circle, math.P3(5, -20, 0), math.V3(1, 0, 0))
	if !ok {
		t.Fatalf("sectionCurvePoint found no crossing")
	}
	want := math.P3(5, -12, 0)
	if d := float64(got.DistanceTo(want)); d > 1e-9 {
		t.Errorf("crossing %v, want %v (off by %g)", got, want, d)
	}
}

// TestSectionCurvePointDeclinesOffSpan pins the no-crossing reject: a section plane beyond the conic's
// spine span meets nothing, and the measure must say so rather than fall back on an unrelated point.
func TestSectionCurvePointDeclinesOffSpan(t *testing.T) {
	t.Parallel()
	circle, _ := geom.NewCircle(math.P3(0, 0, 0), math.V3(0, 0, 1), 13)
	if _, ok := sectionCurvePoint(circle, math.P3(40, 0, 0), math.V3(1, 0, 0)); ok {
		t.Errorf("sectionCurvePoint accepted a plane 40 out on a radius-13 circle; want a decline")
	}
}

// TestRunoutCanalDeclinesWithoutAnEnvelope pins the envelope as a REQUIREMENT of the run-out tier, not
// an optional extra. The certificate's interior condition abstains (reads 0) on a missing envelope —
// deliberately, see TestMaxBallDevIsZeroWithoutAnExtractorPayload — so a producer that supplied stations
// and forgot the envelope would once have shipped a patch whose Certificate.Valid passed on the five
// boundary/structural fields alone, i.e. the exact certificate this slice exists to replace. Build now
// refuses the loop outright, which makes the pairing structural rather than a convention.
func TestRunoutCanalDeclinesWithoutAnEnvelope(t *testing.T) {
	t.Parallel()
	loop, _, res := s1FlankCanal(t)
	if _, cert, ok := (runoutCanalProvider{}).Build(loop, res); !ok || !cert.Valid(res) {
		t.Fatalf("the intact S1 flank loop must build and certify: ok=%v cert=%+v", ok, cert)
	}
	orphan := loop
	orphan.Envelope = nil
	if _, cert, ok := (runoutCanalProvider{}).Build(orphan, res); ok {
		t.Errorf("Build accepted a run-out loop with no envelope (cert %+v, MaxBallDev=%g); the interior "+
			"condition would have abstained while Valid still passed", cert, cert.MaxBallDev)
	}
}
