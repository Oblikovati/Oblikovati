// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/internal/tol"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Placing the cone-canal arm's stations, and measuring what the lofted surface costs (split out
// of fillet_conecanal.go for #2219).
//
// Stations are spaced by CURVATURE, not by arc length: a canal round a cone tightens toward the
// apex, so a uniform spacing would either over-sample the far end or under-sample the near one.
// The envelope error measured here is what decides whether the station count was enough.

// canalStations bundles a chosen station set: the node coordinates x_f and their EXACT columns (ball
// centre + plane/cone feet). It is what resolveStations produces, the loft consumes, and the envelope-error
// measure re-reads — kept together so the node x_f (needed to bracket the closest-spine search) travels
// with the columns.
type canalStations struct {
	xfs                   []float64
	centers, feetA, feetB []math.Point3
}

// resolveStations chooses the station density that BOUNDS the between-station envelope error to the
// model-relative res.Weld() — the same weld the arm's at-station exactness is measured to (ADR-0042). It
// starts at canalArmStationsMin and doubles, each round placing curvature-weighted stations, lofting them,
// and MEASURING the actual mid-interval envelope error (maxEnvelopeError), until the error is within bound —
// returning the winning stations and their surface. If the cap canalArmStationsMax is reached still over
// bound the spine is genuinely unresolvable (coneArmRulingUnresolved), honest-rejected rather than lofted
// with a known gap (CN2 review replaced the old bare 24 that left D1's snout 2.1e-2 off the true envelope).
func (s coneCanalSpine) resolveStations(lo, hi float64, res tol.Resolution) (canalStations, geom.BSplineSurface, coneArmReject) {
	for n := canalArmStationsMin; n <= canalArmStationsMax; n *= 2 {
		st, reason := s.stationsAt(lo, hi, n)
		if reason != coneArmBuilt {
			return canalStations{}, geom.BSplineSurface{}, reason
		}
		surf, err := geom.LoftCanalStations(st.centers, st.feetA, st.feetB, s.radius, res.Weld())
		if err != nil {
			return canalStations{}, geom.BSplineSurface{}, coneArmDegenerate
		}
		if s.maxEnvelopeError(surf, st) <= res.Weld() {
			return st, surf, coneArmBuilt
		}
	}
	return canalStations{}, geom.BSplineSurface{}, coneArmRulingUnresolved
}

// stationsAt builds the n+1 curvature-weighted stations across [lo, hi]: exact hyperbola centre and exact
// plane/cone feet per node, guarding each station's BAND ARC (plane foot → cone foot) against a canal
// self-intersection (fold). A cone foot on the axis or a folded band arc rejects (do-no-harm: never loft a
// malformed band). Nodes cluster near the hyperbola vertex (curvatureWeightedXf) rather than uniformly.
func (s coneCanalSpine) stationsAt(lo, hi float64, n int) (canalStations, coneArmReject) {
	xfs := s.curvatureWeightedXf(lo, hi, n)
	st := canalStations{xfs: xfs,
		centers: make([]math.Point3, n+1),
		feetA:   make([]math.Point3, n+1),
		feetB:   make([]math.Point3, n+1)}
	for i, xf := range xfs {
		m := s.center(xf)
		coneT, ok := s.coneFoot(m)
		if !ok {
			return canalStations{}, coneArmDegenerate
		}
		fP := s.planeFoot(m)
		if s.bandArcMinRegularity(xf, m, fP, coneT) <= 0 {
			return canalStations{}, coneArmRulingFold
		}
		st.centers[i], st.feetA[i], st.feetB[i] = m, fP, coneT
	}
	return st, coneArmBuilt
}

// curvatureWeightedXf places the n+1 station nodes over [lo, hi] with density ∝ κ(x_f)^¼, so the cubic
// loft's O(h⁴) position error κ·h⁴ is equalized across intervals — clustering nodes at the hyperbola vertex
// (κ = cotα/r maximal) instead of wasting them on the smooth far span, so a modest count resolves the D1
// snout. Reduces to uniform where κ is constant. Nodes are the inverse-cumulative-weight images of an even
// partition (an arc-length-style remap of x_f).
func (s coneCanalSpine) curvatureWeightedXf(lo, hi float64, n int) []float64 {
	cum := s.curvatureWeightCumulative(lo, hi)
	total := cum[canalCurvatureWeightSamples]
	span := float64(canalCurvatureWeightSamples)
	out := make([]float64, n+1)
	for j := 0; j <= n; j++ {
		target := total * float64(j) / float64(n)
		out[j] = lo + (hi-lo)*invertCumulative(cum, target)/span
	}
	return out
}

// curvatureWeightCumulative is the running trapezoidal integral of κ(x_f)^¼ over
// canalCurvatureWeightSamples+1 uniform samples of [lo, hi]. κ = |curvatureVector| > 0 for every finite x_f
// (ζ” > 0 on the hyperbola), so the cumulative is strictly increasing and invertible — no floor needed.
func (s coneCanalSpine) curvatureWeightCumulative(lo, hi float64) []float64 {
	cum := make([]float64, canalCurvatureWeightSamples+1)
	prev := s.curvatureWeight(lo)
	for i := 1; i <= canalCurvatureWeightSamples; i++ {
		xf := lo + (hi-lo)*float64(i)/float64(canalCurvatureWeightSamples)
		w := s.curvatureWeight(xf)
		cum[i] = cum[i-1] + 0.5*(w+prev)
		prev = w
	}
	return cum
}

// curvatureWeight is κ(x_f)^¼, the node-density weight (canalCurvatureWeightExp equalizes the O(h⁴) error).
func (s coneCanalSpine) curvatureWeight(xf float64) float64 {
	return stdmath.Pow(float64(s.curvatureVector(xf).Length()), canalCurvatureWeightExp)
}

// invertCumulative returns the fractional sample index t∈[0,S] where the strictly-increasing running
// integral `cum` reaches `target`, linearly interpolating within the bracketing pair.
func invertCumulative(cum []float64, target float64) float64 {
	for i := 1; i < len(cum); i++ {
		if cum[i] >= target {
			return float64(i-1) + (target-cum[i-1])/(cum[i]-cum[i-1])
		}
	}
	return float64(len(cum) - 1)
}

// maxEnvelopeError is the between-station envelope error the derivation must bound: the max over interval
// MIDPOINTS (in v) and a u-sweep of |dist(surface point, exact spine) − r|. On the true canal every surface
// point is at distance r from its characteristic ball centre; the cubic v-interp between exact station
// columns deviates, and this measures that deviation with the exact spine machinery (CN2 review). At-station
// columns are exact by construction, so the error peaks mid-interval.
func (s coneCanalSpine) maxEnvelopeError(surf geom.BSplineSurface, st canalStations) float64 {
	vp := spineChordParams(st.centers)
	worst := 0.0
	for j := 0; j+1 < len(st.centers); j++ {
		vmid := 0.5 * (vp[j] + vp[j+1])
		worst = stdmath.Max(worst, s.intervalEnvelopeError(surf, vmid, st.xfs[j], st.xfs[j+1]))
	}
	return worst
}

// intervalEnvelopeError samples the band across u at the mid-v of one station interval and returns the
// largest |dist(point, exact spine) − r|. A mid-interval point's characteristic x_f lies between the two
// bracketing stations, so the closest-spine search is confined to [x_f low, x_f high].
func (s coneCanalSpine) intervalEnvelopeError(surf geom.BSplineSurface, vmid, xfA, xfB float64) float64 {
	lo, hi := sortedSpan(xfA, xfB)
	worst := 0.0
	for _, u := range canalEnvelopeUSamples() {
		p := surf.PointAt(u, vmid)
		worst = stdmath.Max(worst, stdmath.Abs(s.distanceToSpine(p, lo, hi)-s.radius))
	}
	return worst
}

// canalEnvelopeUSamples are the across-band u parameters the envelope error is probed at (the two feet, the
// mid arc, and the quarter points — enough to catch the worst deviation of the interpolated cross-section).
func canalEnvelopeUSamples() []float64 { return []float64{0, 0.25, 0.5, 0.75, 1} }

// distanceToSpine is the distance from p to the exact hyperbola spine — min over x_f∈[lo, hi] of
// |p − center(x_f)| — by golden-section search (the distance is unimodal on a one-interval bracket around
// the characteristic station). Reuses the exact center machinery; no marched geometry.
func (s coneCanalSpine) distanceToSpine(p math.Point3, lo, hi float64) float64 {
	return goldenSectionMin(func(xf float64) float64 { return float64(p.DistanceTo(s.center(xf))) }, lo, hi, canalSpineSearchIters)
}

// goldenSectionMin returns the minimum of a unimodal f over [lo, hi] after `iters` golden-section
// contractions (each shrinks the bracket by the golden ratio ≈0.618; 64 reaches machine precision).
func goldenSectionMin(f func(float64) float64, lo, hi float64, iters int) float64 {
	const invPhi = 0.6180339887498949 // 1/φ
	a, b := lo, hi
	c, d := b-invPhi*(b-a), a+invPhi*(b-a)
	fc, fd := f(c), f(d)
	for range iters {
		if fc < fd {
			b, d, fd = d, c, fc
			c = b - invPhi*(b-a)
			fc = f(c)
			continue
		}
		a, c, fc = c, d, fd
		d = a + invPhi*(b-a)
		fd = f(d)
	}
	return stdmath.Min(fc, fd)
}

// spineChordParams is the loft's v-parametrization: the normalized cumulative chord length of the station
// centres (P&T §9.2.1, matching geom.alphaParams(·, 1)), so v_j is where station j's exact column lives on
// the built surface. Shared by the envelope-error measure and the at-station exactness test.
func spineChordParams(centers []math.Point3) []float64 {
	cum := make([]float64, len(centers))
	for k := 1; k < len(centers); k++ {
		cum[k] = cum[k-1] + float64(centers[k-1].DistanceTo(centers[k]))
	}
	out := make([]float64, len(centers))
	for k := range out {
		out[k] = cum[k] / cum[len(centers)-1]
	}
	out[len(out)-1] = 1
	return out
}

// bandArcMinRegularity is the minimum tube-regularity factor 1 − r·(K·d̂) over the BAND ARC (the
// characteristic sub-arc the fillet actually spans, from the plane foot d̂ to the cone foot d̂), NOT the
// full tube: the full tube of a canal over the D1 vertex (κ·r = cotα > 1) folds on its concave down-axis
// side, but the band never reaches there (a full-tube check would spuriously reject D1). > 0 everywhere
// ⇒ the band arc is a regular strip of the envelope.
func (s coneCanalSpine) bandArcMinRegularity(xf float64, m, fP, coneT math.Point3) float64 {
	k := s.curvatureVector(xf)
	dA := unitBetween(m, fP)
	dB := unitBetween(m, coneT)
	minF := stdmath.Inf(1)
	for i := 0; i <= canalFoldArcSamples; i++ {
		d := slerpUnit(dA, dB, float64(i)/float64(canalFoldArcSamples))
		minF = stdmath.Min(minF, 1-s.radius*float64(k.Dot(d)))
	}
	return minF
}

// unitBetween is the unit direction from a to b (assumes a != b — the feet sit at radius r > 0 from m).
func unitBetween(a, b math.Point3) math.Vector3 {
	d, _ := math.UnitVector3FromVector(a.VectorTo(b))
	return d.AsVector()
}

// slerpUnit spherically interpolates between unit directions a and b (great-arc, constant angular speed),
// so the sampled d̂ sweeps the actual characteristic arc from the plane foot to the cone foot. Coincident
// directions (angle below canalSlerpMinAngle) return a unchanged.
func slerpUnit(a, b math.Vector3, t float64) math.Vector3 {
	cosw := stdmath.Max(-1, stdmath.Min(1, float64(a.Dot(b))))
	w := stdmath.Acos(cosw)
	if w < canalSlerpMinAngle {
		return a
	}
	sw := stdmath.Sin(w)
	return a.Scale(stdmath.Sin((1-t)*w) / sw).Add(b.Scale(stdmath.Sin(t*w) / sw))
}

// coneCanalArmFillet builds the exact canal arm on a Cone∧Plane RULING edge, carried in the same
// edgeFillet the straight-edge/torus paths emit plus the hyperbola spine descriptor (armCanalSpine) the
// corner weld reads. Handles BOTH host material sides — convex-external (material inside the cone,
// coneHostMaterialSign>0, apexSign=+1) and the concave BORE (material outside, sign≤0, apexSign=−1) — by
// threading apexSign into the spine exactly as coneArmFillet's cap-plane sibling already does
// (coneArmFilletConvex/coneArmFilletConcave): a bore ruling is edge-CONVEX, the ball rolling in the
// material precisely as it does on a boss (the I1 insight, re-landed here for I4). Only an unreadable
// material sign (apex-adjacent edge) or a build decline (α bands, no fit, fold) still honest-rejects, so
// coneArmEdge reports the cause via coneArmError (do-no-harm).
func coneCanalArmFillet(e *topo.Edge, co geom.Cone, pl geom.Plane, coneFace, planeFace *topo.Face, r float64, res tol.Resolution) (edgeFillet, coneArmReject) {
	sgn, ok := coneHostMaterialSign(e, co, coneFace)
	if !ok {
		return edgeFillet{}, coneArmDegenerate // apex-adjacent edge: no readable radial direction
	}
	apexSign := 1.0
	if sgn <= 0 {
		apexSign = -1.0
	}
	nOut, err := math.UnitVector3FromVector(outwardPlaneNormal(planeFace, pl))
	if err != nil {
		return edgeFillet{}, coneArmDegenerate
	}
	surf, spine, reason := buildConeCanalArm(e, co, nOut, apexSign, r, res)
	if reason != coneArmBuilt {
		return edgeFillet{}, reason
	}
	faces := e.Faces()
	sp := spine
	return edgeFillet{a: faces[0], b: faces[1], edge: e, armSurface: surf, armCanalSpine: &sp}, coneArmBuilt
}

// coneCanalArmError names the ruling-edge (canal) build rejects — each carrying the offending radius and
// cone half-angle — mirroring coneArmSurfaceError for the torus arm.
func coneCanalArmError(reason coneArmReject, co geom.Cone, r float64) error {
	switch reason {
	case coneArmRulingNoFit:
		return fmt.Errorf("fillet: cannot round this Cone∧Plane RULING edge with radius %g — the rolling ball never fits "+
			"the picked span (cone half-angle %g): no hyperbola-spine station exists (tanα·(z_{A′}−z) < r everywhere)", r, co.HalfAngle)
	case coneArmRulingSpan:
		return fmt.Errorf("fillet: cannot round this Cone∧Plane RULING edge with radius %g — the fittable canal-spine span "+
			"collapses to a point (cone half-angle %g)", r, co.HalfAngle)
	case coneArmRulingUnresolved:
		return fmt.Errorf("fillet: cannot round this Cone∧Plane RULING edge with radius %g — the between-station envelope "+
			"error stays over the model-relative bound at the %d-station cap (cone half-angle %g): the hyperbola spine is "+
			"unresolvably high-curvature for this radius", r, canalArmStationsMax, co.HalfAngle)
	default: // coneArmRulingFold
		return fmt.Errorf("fillet: cannot round this Cone∧Plane RULING edge with radius %g — the constant-radius canal band "+
			"self-intersects at a station (irregular band arc, 1−κ·r·cosψ ≤ 0; cone half-angle %g)", r, co.HalfAngle)
	}
}
