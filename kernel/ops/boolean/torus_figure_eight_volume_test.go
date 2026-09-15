// SPDX-License-Identifier: GPL-2.0-only

package boolean_test

import (
	"fmt"
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/kernel/topo"
)

// The figure-eight pieces' ANALYTIC VOLUME oracle (Oblikovati/Oblikovati#3519).
//
// The two pieces had an analytic AREA per torus face and, for volume, nothing but the partition
// identity — "the two add up to the whole torus". That identity is satisfied by any pair of numbers
// summing to 2π²Rr², including a pair that is wrong by the same amount in opposite directions, which
// is exactly the failure the pinch produces (the retired spiric loft covered it twice). So each piece
// now carries its OWN analytic volume, and the identity is the third assertion rather than the only one.
//
// The derivation. In cylindrical coordinates the solid torus is (ρ−R)² + z² ≤ r², so at tube angle t
// the ring radius is ρ = R + r·sin t, the z-extent is 2r·cos t and dρ = r·cos t dt. The half-space
// y > c is ρ·sin u > c, whose azimuth measure at that ρ is π − 2·arcsin(c/ρ) = 2·arccos(c/ρ) (it is
// never empty here, because c = R − r is the smallest ρ the torus reaches). Multiplying the three and
// the Jacobian ρ:
//
//	V_above = ∫ ρ · 2r·cos t · 2·arccos(c/ρ) · r·cos t dt = 4r² ∫ (R + r·sin t)·cos²t·arccos(c/ρ) dt
//
// over t ∈ [−π/2, π/2]. The integrand is ANALYTIC on that interval, which is why a fixed-order rule
// reaches machine precision on it: at t = −π/2 the plane is tangent, ρ → c and arccos(c/ρ) → 0, and
// since ρ − c = 2r·sin²((t+π/2)/2) the arccos series √(2δ)·(1 + δ/12 + …) is a power series in
// (t + π/2) with no fractional powers left. Measured below: the rule's own N-against-4N spread is
// 1.4e-13 relative (TestTheFigureEightVolumeOracleIsConverged).
//
// It is more exact than the numbers it gates, which is the point (kernel ground rules: "an oracle that
// gates a result must be more exact than the result it gates"). It gates the MESH volumes, whose own
// chord deficit is 2.6e-4 at PropertyQuality and 1.2e-2 at DefaultQuality — nine decades coarser. It
// does NOT gate the kernel's analytic B-rep integrator, which reaches the same 114.886320193 to twelve
// digits; that agreement is asserted as a two-way cross-check, not as a gate.
const (
	figureEightRingRadius = 5.0 // R
	figureEightTubeRadius = 2.0 // r
	figureEightCutY       = 3.0 // = R − r, so the plane y = c touches the inner equator
)

// figureEight*Volume are that integral's value and its complement, in mm³. The literals are the
// receipt; figureEightAboveVolumeQuadrature recomputes them every run and the two must agree, so
// neither a mistyped digit nor a broken rule can pass unseen.
const (
	// 2π²Rr², the whole solid torus. It equals the whole torus's AREA here only because 2π²Rr² and
	// 4π²Rr are both 40π² at R=5, r=2 — a coincidence of this fixture, not an identity.
	figureEightTorusVolume = 2 * stdmath.Pi * stdmath.Pi * figureEightRingRadius *
		figureEightTubeRadius * figureEightTubeRadius
	figureEightAboveVolume = 114.886320193 // the y > 3 piece, from the integral above
	figureEightBelowVolume = 279.897855851 // the whole torus less it
)

// torusTangentPlanePanels is the composite-Simpson panel count the oracle ships at. The rule
// is exact for cubics and the integrand is analytic, so the error falls as h⁴: 4096 panels over an
// interval of π put it at 1e-13 relative, which the convergence row measures rather than assumes.
const torusTangentPlanePanels = 4096

// torusTangentPlaneAboveVolume is the oracle, for ANY torus of the family: the volume of the solid
// torus (R, r) on the far side of the plane tangent to its inner equator, by composite Simpson over the
// tube angle. It is parameterised because the corpus carries more than one aspect ratio — the second
// self-touching row is R=5 r=1.5 (torus_tangent_selftouch_test.go) — and two copies of one integral
// would be two chances to mistype it.
//
// Example: got := torusTangentPlaneAboveVolume(5, 2, 4096) // 114.886320193
func torusTangentPlaneAboveVolume(ringR, tubeR float64, panels int) float64 {
	return simpsonQuadrature(-stdmath.Pi/2, stdmath.Pi/2, panels, func(t float64) float64 {
		rho := ringR + tubeR*stdmath.Sin(t)
		cosT := stdmath.Cos(t)
		return 4 * tubeR * tubeR * rho * cosT * cosT * torusTangentPlaneArccos(ringR, tubeR, rho)
	})
}

// torusTangentPlaneArccos is arccos(c/ρ) with c = R − r, the azimuth half-measure of the half-space at
// ring radius ρ. The argument is CLAMPED at 1: ρ reaches exactly c at the tangency and rounding there
// can put c/ρ a last place above 1, where math.Acos answers NaN — the clamp restores the value the
// limit has, it does not widen the domain.
func torusTangentPlaneArccos(ringR, tubeR, rho float64) float64 {
	return stdmath.Acos(stdmath.Min(1, (ringR-tubeR)/rho))
}

// torusTangentPlaneAboveArea is the same oracle for the torus SURFACE, and it is what re-measured the
// two area literals in torus_figure_eight_band_test.go (#3519). The area element is r(R + r·cos v) du dv
// and the u-measure at tube angle v is again 2·arccos(c/ρ), so
//
//	A_above = 2r ∫ ρ·arccos(c/ρ) dv over v ∈ [0, 2π]
//
// integrated over HALF the period and doubled: ρ is even in v, and the tangency sits at v = π, where
// ρ − c = 2r·cos²(v/2) makes the arccos series carry |cos(v/2)| — a KINK in the middle of the full
// period, and a rule whose stencil straddles it loses its order. On [0, π] that point is an endpoint
// and the integrand is one-sided analytic again.
//
// Example: got := torusTangentPlaneAboveArea(5, 2, 4096) // 111.683566487
func torusTangentPlaneAboveArea(ringR, tubeR float64, panels int) float64 {
	return 2 * simpsonQuadrature(0, stdmath.Pi, panels, func(v float64) float64 {
		rho := ringR + tubeR*stdmath.Cos(v)
		return 2 * tubeR * rho * torusTangentPlaneArccos(ringR, tubeR, rho)
	})
}

// simpsonQuadrature integrates f over [a, b] with composite Simpson on an even number of panels. It
// panics on an odd count rather than silently rounding it, because a rule applied on the wrong stencil
// answers a plausible number for a different integral.
func simpsonQuadrature(a, b float64, panels int, f func(float64) float64) float64 {
	if panels < 2 || panels%2 != 0 {
		panic(fmt.Sprintf("simpsonQuadrature needs an even panel count of at least 2, got %d", panels))
	}
	h := (b - a) / float64(panels)
	sum := f(a) + f(b)
	for i := 1; i < panels; i++ {
		sum += simpsonWeight(i) * f(a+float64(i)*h)
	}
	return sum * h / 3
}

// simpsonWeight is composite Simpson's alternating interior weight: 4 at an odd node, 2 at an even one.
func simpsonWeight(i int) float64 {
	if i%2 == 1 {
		return 4
	}
	return 2
}

// TestTheFigureEightVolumeOracleIsConverged is the oracle's own receipt, in both directions: the rule
// has converged (its N-against-4N spread is machine noise) AND it lands on the literal written down
// above. Either alone is weak — a converged rule can converge on the wrong integral, and a literal
// nothing recomputes is a transcription.
func TestTheFigureEightVolumeOracleIsConverged(t *testing.T) {
	t.Parallel()
	coarse := torusTangentPlaneAboveVolume(figureEightRingRadius, figureEightTubeRadius, torusTangentPlanePanels)
	fine := torusTangentPlaneAboveVolume(figureEightRingRadius, figureEightTubeRadius, 4*torusTangentPlanePanels)
	if spread := stdmath.Abs(fine-coarse) / fine; spread > 1e-12 {
		t.Errorf("the volume quadrature moves %.3e relative between %d and %d panels; it has not converged",
			spread, torusTangentPlanePanels, 4*torusTangentPlanePanels)
	}
	if off := stdmath.Abs(coarse-figureEightAboveVolume) / figureEightAboveVolume; off > 1e-11 {
		t.Errorf("the volume quadrature gives %.9f where the literal says %.9f (rel %.3e); one of the two "+
			"is wrong and the oracle cannot be used until they agree", coarse, figureEightAboveVolume, off)
	}
	if sum := figureEightAboveVolume + figureEightBelowVolume; stdmath.Abs(sum-figureEightTorusVolume) > 1e-8 {
		t.Errorf("the two literal piece volumes sum to %.9f where the whole solid torus is %.9f",
			sum, figureEightTorusVolume)
	}
}

// TestTheFigureEightAreaOracleIsConverged is the same receipt for the area literals the band gate
// reads, which this slice re-measured. It is a separate row from the volume's so a failure names which
// oracle moved.
func TestTheFigureEightAreaOracleIsConverged(t *testing.T) {
	t.Parallel()
	coarse := torusTangentPlaneAboveArea(figureEightRingRadius, figureEightTubeRadius, torusTangentPlanePanels)
	fine := torusTangentPlaneAboveArea(figureEightRingRadius, figureEightTubeRadius, 4*torusTangentPlanePanels)
	if spread := stdmath.Abs(fine-coarse) / fine; spread > 1e-12 {
		t.Errorf("the area quadrature moves %.3e relative between %d and %d panels; it has not converged",
			spread, torusTangentPlanePanels, 4*torusTangentPlanePanels)
	}
	if off := stdmath.Abs(coarse-figureEightAboveArea) / figureEightAboveArea; off > 1e-11 {
		t.Errorf("the area quadrature gives %.9f where the literal says %.9f (rel %.3e)",
			coarse, figureEightAboveArea, off)
	}
	if sum := figureEightAboveArea + figureEightBelowArea; stdmath.Abs(sum-figureEightTorusArea) > 1e-8 {
		t.Errorf("the two literal piece areas sum to %.9f where the whole torus is %.9f", sum, figureEightTorusArea)
	}
}

// TestTheFigureEightPiecesIntegrateToTheirOwnAnalyticVolume is the volume half of the per-piece gate.
//
// Each piece's MESH volume must be its analytic volume less a chord deficit — under it, never over —
// at both facetings, and the two must then partition the solid torus. A mesher that covers the pinch
// twice overshoots one piece; one that loses the material corner there undershoots by more than a
// chord deficit. The partition identity alone sees neither, which is why it is the third assertion.
func TestTheFigureEightPiecesIntegrateToTheirOwnAnalyticVolume(t *testing.T) {
	t.Parallel()
	cut, intersect := figureEightPiece(t, ops.Cut), figureEightPiece(t, ops.Intersect)
	for _, gq := range figureEightQualities() {
		band := chordDeficitVolumeCoarse
		if gq.name == "property" {
			band = chordDeficitVolumeFine
		}
		below, above := figureEightMeshVolume(t, cut, gq.q), figureEightMeshVolume(t, intersect, gq.q)
		assertChordDeficitWithin(t, gq.name+" below y=3 volume", below, figureEightBelowVolume, band)
		assertChordDeficitWithin(t, gq.name+" above y=3 volume", above, figureEightAboveVolume, band)
		assertChordDeficitWithin(t, gq.name+" volume sum", below+above, figureEightTorusVolume, band)
	}
}

// figureEightMeshVolume is one piece's whole-body mesh volume at one faceting.
func figureEightMeshVolume(t *testing.T, b *topo.Body, q ops.Quality) float64 {
	t.Helper()
	mesh, _ := tessellate.TessellateBody(b, q)
	return tessellate.MeshGeometryProperties(mesh).Volume
}

// TestTheAnalyticIntegratorAgreesWithTheFigureEightOracle is the two-way cross-check, and the pin on
// how far it reaches.
//
// The kernel's own analytic B-rep integrator answers for the INTERSECT piece and declines the CUT
// piece today. Both halves are asserted: where it answers it must agree with the quadrature to nine
// digits (two independent derivations of one number), and the COUNT of pieces it answers for is pinned
// two-sided — a fall means it lost a piece it had, and a rise means the cut piece became integrable and
// this row should gate it too, rather than sitting green on a skip.
func TestTheAnalyticIntegratorAgreesWithTheFigureEightOracle(t *testing.T) {
	t.Parallel()
	answered := 0
	for _, row := range []struct {
		op   ops.PartFeatureOperation
		want float64
	}{{ops.Cut, figureEightBelowVolume}, {ops.Intersect, figureEightAboveVolume}} {
		an, ok := query.AnalyticGeometryProperties(figureEightPiece(t, row.op))
		if !ok {
			continue
		}
		answered++
		if rel := stdmath.Abs(an.Volume-row.want) / row.want; rel > 1e-9 {
			t.Errorf("%v piece: the analytic integrator reads %.9f mm³ against the quadrature's %.9f (rel %.3e)",
				row.op, an.Volume, row.want, rel)
		}
	}
	if answered != 1 {
		t.Errorf("the analytic integrator answers for %d of the two figure-eight pieces; the measurement is 1 "+
			"(intersect only — it declines the cut piece, whose lid is bounded by the figure eight itself). "+
			"More means the cut piece became integrable and this row must gate it; fewer means the "+
			"cross-check now covers nothing", answered)
	}
}
