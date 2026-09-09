// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"

	"oblikovati.org/math"
)

// The SECOND harmonic of the torus∩quadric reduction (ADR-0061 stage 5, third slice) — the half of the
// family torus_quadric_arc.go's one-harmonic closed form cannot reach.
//
// A point on the torus is
//
//	P(u, v) = C + ρ(v)·e(u) + r·sin v·â,   ρ(v) = R + r·cos v,   e(u) = cos u·ê₁ + sin u·ê₂
//
// and a quadric is Q(X) = W·(M W) + 2 G·W + K with W = X − Anchor. Writing W = W₀ + ρ·e(u), with
// W₀ = (C − Anchor) + r·sin v·â the station's own offset, and expanding,
//
//	Q = (W₀·MW₀ + 2 G·W₀ + K) + ρ·(e·T) + ρ²·(e·Me),   T = 2(M W₀ + G)
//
// so at a fixed tube angle v the whole azimuth dependence sits in two terms. The term LINEAR in e is
// e·T = x·cos u + y·sin u, with x = T·ê₁ and y = T·ê₂ — one harmonic. The term QUADRATIC in e is
//
//	e·Me = m₁₁·cos²u + 2·m₁₂·cos u·sin u + m₂₂·sin²u
//	     = (m₁₁+m₂₂)/2 + ((m₁₁−m₂₂)/2)·cos 2u + m₁₂·sin 2u
//
// with m₁₁ = ê₁·Mê₁, m₂₂ = ê₂·Mê₂ and m₁₂ = ê₁·Mê₂ — a SECOND harmonic plus a constant. So every torus
// against every quadric reduces, station by station, to
//
//	f(u) = Level + Cos1·cos u + Sin1·sin u + Cos2·cos 2u + Sin2·sin 2u = 0
//
//	Cos2 = ρ²(m₁₁−m₂₂)/2   Sin2 = ρ²·m₁₂   Cos1 = ρ·x   Sin1 = ρ·y
//	Level = W₀·MW₀ + 2 G·W₀ + K + ρ²(m₁₁+m₂₂)/2
//
// When M is INVARIANT about the torus axis, m₁₁ = m₂₂ and m₁₂ = 0: Cos2 and Sin2 vanish, Level falls
// back to the one-harmonic form's level, and (Cos1, Sin1) is its reach and phase in polar form. That
// closed form stays — an arccos is exact, cheap, and already certified where it applies. Everything
// else — a rod ACROSS a ring, a tilted drill, an off-axis cone — lands here, where f is a degree-two
// trigonometric polynomial with up to FOUR azimuths per station rather than two.
//
// Those azimuths are the same Weierstrass quartic the conic×conic bucket already solves
// (trigQuadraticRoots, conic2d_intersect.go): with t = tan(u/2), clearing (1+t²)² gives
//
//	(Cos2−Cos1+Level)·t⁴ + (2Sin1−4Sin2)·t³ + (2Level−6Cos2)·t² + (2Sin1+4Sin2)·t + (Cos2+Cos1+Level)
//
// whose leading coefficient is f(π) — so the half-turn the substitution cannot reach is a root exactly
// when the quartic drops to a cubic, which is how that solver recovers it. ONE quartic solver, at the
// layer both callers reach; this file adds no second one.

// torusStation is the quadric's constraint on ONE tube circle of the torus, before the azimuth is
// resolved. Both reductions read it: the axis-invariant arccos takes rho, constant and (x, y), and the
// general form takes the in-plane tensor entries with them.
type torusStation struct {
	rho           float64 // R + r·cos v, the tube circle's distance from the axis
	constant      float64 // W₀·MW₀ + 2 G·W₀ + K, the part of Q with no azimuth dependence
	x, y          float64 // T·ê₁ and T·ê₂ with T = 2(M W₀ + G): the term linear in e(u)
	m11, m22, m12 float64 // ê₁·Mê₁, ê₂·Mê₂, ê₁·Mê₂: the term quadratic in e(u)
	invariant     bool    // M acts the same way on every direction perpendicular to the torus axis
}

// torusStationAt reduces the quadric's constraint on the torus at tube angle v.
func torusStationAt(t Torus, q Quadric, v float64) torusStation {
	axis, e1, e2 := torusAxisFrame(t)
	cv, sv := cosSin(v)
	w0 := q.Anchor.VectorTo(t.Center).Add(axis.Scale(math.Scalar(t.MinorRadius * sv)))
	mw0 := q.M.Apply(w0)
	reachVec := mw0.Scale(2).Add(q.G.Scale(2))
	m11, m22, m12 := inPlaneTensorEntries(q, e1, e2)
	return torusStation{
		rho:       t.MajorRadius + t.MinorRadius*cv,
		constant:  float64(w0.Dot(mw0)) + 2*float64(q.G.Dot(w0)) + q.K,
		x:         float64(reachVec.Dot(e1)),
		y:         float64(reachVec.Dot(e2)),
		m11:       m11,
		m22:       m22,
		m12:       m12,
		invariant: axisInvariantEntries(q, m11, m22, m12),
	}
}

// inPlaneTensorEntries returns ê₁·Mê₁, ê₂·Mê₂ and ê₁·Mê₂ — how the quadric's quadratic form acts on
// the plane the torus's azimuth sweeps, which is the only part of M the reduction ever reads.
func inPlaneTensorEntries(q Quadric, e1, e2 math.Vector3) (m11, m22, m12 float64) {
	return float64(e1.Dot(q.M.Apply(e1))), float64(e2.Dot(q.M.Apply(e2))), float64(e1.Dot(q.M.Apply(e2)))
}

// axisInvariantEntries reports that the in-plane entries make M act the same way on every direction
// perpendicular to the torus axis — the condition that collapses the azimuth dependence to ONE
// harmonic. The departure is measured RELATIVE to the tensor's own entries, so it carries no model
// scale: the same pair written in metres and in millimetres classifies identically.
func axisInvariantEntries(q Quadric, m11, m22, m12 float64) bool {
	scale := stdmath.Max(q.M.Norm(), stdmath.Abs(m11))
	if scale <= 0 {
		return false // a degenerate (planar) quadric: torus∩plane is the spiric closed form
	}
	return stdmath.Abs(m11-m22) <= axisInvarianceTol*scale && stdmath.Abs(m12) <= axisInvarianceTol*scale
}

// harmonic reduces the station to the ONE-harmonic form level + reach·cos(u − phase), which describes
// it exactly while the station is axis-invariant.
//
// The level is READ from the general form rather than respelled as constant + m11·ρ². Two spellings of
// one quantity are two roundings of it: on a platform that fuses a multiply into the following add
// (arm64 does, amd64 never) the two parted by an ulp, so the reproduction proof this file rests on —
// that the arccos path is the general path written out, not an approximation of it — held on one
// platform and failed on the other (CI run 34280554924 macos-latest). One expression, one value.
func (st torusStation) harmonic() torusHarmonic {
	return torusHarmonic{
		level: st.secondHarmonic().Level,
		reach: st.rho * stdmath.Hypot(st.x, st.y),
		phase: stdmath.Atan2(st.y, st.x),
	}
}

// torusSecondHarmonic is the quadric's constraint on one tube circle written in full:
//
//	f(u) = Level + Cos1·cos u + Sin1·sin u + Cos2·cos 2u + Sin2·sin 2u
//
// It is zero exactly at the azimuths where the tube circle meets the quadric, and carries the sign of
// the quadric's own inside/outside sense between them.
type torusSecondHarmonic struct {
	Cos2, Sin2 float64 // the second harmonic — zero exactly when M is invariant about the torus axis
	Cos1, Sin1 float64 // the first harmonic, the one-harmonic form's reach and phase in cartesian form
	Level      float64 // the azimuth-independent term
}

// secondHarmonic rewrites the station's cos²/cos·sin/sin² terms as cos 2u and sin 2u (see the file
// comment's half-angle identities), leaving the five coefficients of f.
func (st torusStation) secondHarmonic() torusSecondHarmonic {
	rr := st.rho * st.rho
	return torusSecondHarmonic{
		Cos2: rr * (st.m11 - st.m22) / 2,
		Sin2: rr * st.m12,
		Cos1: st.rho * st.x,
		Sin1: st.rho * st.y,
		// The term added to the constant ends in a DIVISION, and a division result is not a product,
		// so no platform may fuse it into this add — which is what makes this spelling of the level the
		// stable one and the closed form's `constant + m11·ρ²` the one that parted by an ulp on arm64
		// (CI run 34280554924 macos-latest). harmonic() reads this Level rather than respelling it for
		// exactly that reason. Keep the division last if this line is ever rewritten; an explicit
		// float64() round of the term would pin it too, and was dropped only because it cannot fire
		// here and a conversion that documents a mechanism it does not use reads as a live guard.
		Level: st.constant + rr*(st.m11+st.m22)/2,
	}
}

// torusSecondHarmonicAt returns the five coefficients of the quadric's constraint on the torus at tube
// angle v. It is exact for EVERY quadric, axis-invariant or not.
//
//	h := torusSecondHarmonicAt(ring, rod.QuadricForm(), v) // h.valueAt(u) == rod.QuadricForm().ValueAt(ring.PointAt(u, v))
func torusSecondHarmonicAt(t Torus, q Quadric, v float64) torusSecondHarmonic {
	return torusStationAt(t, q, v).secondHarmonic()
}

// valueAt evaluates f at one azimuth.
func (h torusSecondHarmonic) valueAt(u float64) float64 {
	cu, su := cosSin(u)
	c2, s2 := cosSin(2 * u)
	return h.Level + h.Cos1*cu + h.Sin1*su + h.Cos2*c2 + h.Sin2*s2
}

// slopeAt evaluates df/du at one azimuth.
func (h torusSecondHarmonic) slopeAt(u float64) float64 {
	cu, su := cosSin(u)
	c2, s2 := cosSin(2 * u)
	return -h.Cos1*su + h.Sin1*cu + 2*(h.Sin2*c2-h.Cos2*s2)
}

// derivative returns df/du as a station polynomial of the SAME shape, so the extrema that name the
// lanes are found by the same solver the roots are — no second machinery for the critical points.
func (h torusSecondHarmonic) derivative() torusSecondHarmonic {
	return torusSecondHarmonic{Cos2: 2 * h.Sin2, Sin2: -2 * h.Cos2, Cos1: h.Sin1, Sin1: -h.Cos1}
}

// scale is the polynomial's own coefficient magnitude, what a residual is judged against. It carries
// the station's units, so a residual ratio against it is dimensionless.
func (h torusSecondHarmonic) scale() float64 {
	return polyScale(h.Cos2, h.Sin2, h.Cos1, h.Sin1, h.Level)
}

// azimuths returns the CERTIFIED azimuths where f vanishes, in ascending order. Each candidate the
// quartic produces is Newton-polished in the ANGLE — better conditioned than the tan(u/2) chart near
// the half-turn, where a root sits at a nearly infinite t — and then kept only if its residual on f
// itself is at the solve's own rounding. Nothing here decides which root is physical: every root that
// certifies is returned, and the caller's lane structure says which pair bounds which arc.
func (h torusSecondHarmonic) azimuths() []float64 {
	scale := h.scale()
	out := make([]float64, 0, 4)
	for _, u := range trigQuadraticRoots(2*h.Cos2, h.Sin2, h.Cos1, h.Sin1, h.Level-h.Cos2) {
		u = h.polish(u)
		if stdmath.Abs(h.valueAt(u)) <= torusRootResidualTol*scale {
			out = append(out, u)
		}
	}
	return sortedDedupedAngles(out)
}

// polish takes a candidate azimuth to the root itself by Newton on f. It stops at a stalled slope,
// which is a double root — a fold, where the caller reads the extremum rather than the root.
func (h torusSecondHarmonic) polish(u float64) float64 {
	for range torusRootPolishSteps {
		d := h.slopeAt(u)
		if d == 0 {
			break
		}
		u -= h.valueAt(u) / d
	}
	return wrapAngle(u)
}

// extrema returns the certified azimuths where df/du vanishes, in ascending order — the station's
// maxima and minima alternating round the circle. There are always at least two (f is continuous and
// periodic) and at most four (df/du is itself a degree-two trigonometric polynomial).
func (h torusSecondHarmonic) extrema() []float64 { return h.derivative().azimuths() }

// torusRootResidualTol is how large a candidate azimuth's residual on the station polynomial may be,
// relative to that polynomial's own largest coefficient, and still count as a root. It compares a
// value with the coefficients it was formed from, so it carries no model scale.
const torusRootResidualTol = 1e-9 // tol:numeric — relative residual of a certified station root

// torusRootPolishSteps is how many Newton steps in the angle a quartic root gets. The quartic solver
// already polishes in t; two more steps in u remove what the tan(u/2) chart's conditioning left.
const torusRootPolishSteps = 3
