// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The SPIRIC closed-rim canal SPINE (OCCT blend/simple J3, bfuseblend A4). Rounding the CLOSED rim
// where a MERIDIAN cap plane (a plane containing the torus axis — the |ẑ·n̂|≈0 dual of the latitude
// E7 arm) meets a host torus rolls a ball whose centre curve is a spiric section of Perseus: the
// intersection of the r-offset cap plane with the r-offset host torus. No exact-torus arm exists
// (fillet_torusarm.go's torusArmMeridian reject), but every STATION is closed-form exact:
//
//	frame:  m̂ = in-cap-plane direction from the torus axis to the rim-circle centre D (⊥ ẑ, ⊥ n̂),
//	        n̂ = the cap face's material-outward unit normal (⊥ ẑ within band — meridian gate).
//	centre: C(ψ) = O + w·n̂ + ξ(ψ)·m̂ + b·sinψ·ẑ, ξ(ψ) = √((R + b·cosψ)² − w²),
//	        w = σ·r (σ = +1 concave void-side / −1 convex material-side), b = a + σ·s·r
//	        (s = tubeMaterialSign — which side of the tube carries material).
//	feet:   cap foot = C − w·n̂ (exactly r off the cap plane); tube foot = c_u + (a/b)·(C − c_u)
//	        with c_u the tube-centre circle point at C's azimuth (exactly r off the host tube).
//
// DRAWEXE 8.0.0 receipts (wave-report-E): J3's blend band decodes station-for-station on these
// formulas (b = 40, w = −10), its trimmed cap disc area 5033.06 = the region inside the projected
// spiric rail; A4's cove band likewise at b = 60, w = +10.

// spiricRimSpine carries the resolved spiric frame: the host torus, the cap plane scalars, and the
// station formula inputs. All stations it emits are closed-form exact on the true envelope.
type spiricRimSpine struct {
	host geom.Torus
	nHat math.UnitVector3 // cap material-outward normal (⊥ axis within band)
	mHat math.UnitVector3 // in-cap-plane direction axis → rim-circle centre (⊥ ẑ, ⊥ n̂)
	capD float64          // signed n̂-offset of the cap plane from the torus centre (≈0, meridian gate)
	w    float64          // centre-plane n̂-offset: capD + σ·r
	b    float64          // offset-tube radius: a + σ·s·r
	side float64          // σ: +1 concave (void-side ball) / −1 convex (material-side ball)
	r    float64
}

// spiricParallelCoef scales the meridian band |ẑ·n̂| ≤ k·res.Weld()/ρ_rim — the mirror of
// torusLatitudeCoef's latitude band, model-relative to the rim radius (ADR-0042).
const spiricParallelCoef = 3

// spiricMeridianCut reports whether the cap plane is (near-)parallel to the torus axis — the spiric
// configuration. nDotZ is ẑ·n̂ (signed); the band is model-relative to the rim radius.
func spiricMeridianCut(nDotZ, rhoRim float64, res Resolution) bool {
	epsAng := spiricParallelCoef * res.Weld() / stdmath.Max(rhoRim, res.Weld())
	return stdmath.Abs(nDotZ) < epsAng
}

// newSpiricRimSpine resolves the spiric frame for a CLOSED meridian-cut Torus∧Plane rim. ok=false
// (fall through to the existing reject) when the plane is not a meridian cut, the plane misses the
// axis, the rim circle centre is unreadable, the tube material side is unreadable, the offset tube
// collapses (b ≤ 0), or the offset plane exits the offset torus (R − b ≤ |w|: the spine would not be
// a single closed loop around the tube).
func newSpiricRimSpine(body *topo.Body, e *topo.Edge, host geom.Torus, pl geom.Plane, hostFace, planeFace *topo.Face, r float64) (spiricRimSpine, bool) {
	res := ResolutionForBody(body)
	n, err := math.UnitVector3FromVector(outwardPlaneNormal(planeFace, pl))
	if err != nil {
		return spiricRimSpine{}, false
	}
	rhoRim := torusRimRadius(e, host)
	if !spiricMeridianCut(host.AxisDir.Dot(n), rhoRim, res) {
		return spiricRimSpine{}, false // not a meridian cut — not this engine's rim
	}
	capD := host.Center.VectorTo(pl.Origin).Dot(n.AsVector())
	if stdmath.Abs(float64(capD)) > spiricParallelCoef*res.Weld()*host.MajorRadius {
		return spiricRimSpine{}, false // cap plane off the axis — the rim is not a meridian circle
	}
	return assembleSpiricSpine(e, host, hostFace, n, float64(capD), r, res)
}

// assembleSpiricSpine finishes the frame: σ from the rim dihedral, s from the tube material side,
// m̂ from the rim-circle centre, then the closed-loop existence guards.
func assembleSpiricSpine(e *topo.Edge, host geom.Torus, hostFace *topo.Face, n math.UnitVector3, capD, r float64, res Resolution) (spiricRimSpine, bool) {
	side := -1.0 // convex: ball centre on the material side of the cap
	if ClassifyEdgeConvexity(e) == EdgeConcave {
		side = +1.0 // concave: ball in the void
	}
	s, ok := tubeMaterialSign(hostFace, host, edgeMidpoint(e))
	if !ok {
		return spiricRimSpine{}, false
	}
	d, ok := rimCircleCenter(e)
	if !ok {
		return spiricRimSpine{}, false
	}
	m, err := math.UnitVector3FromVector(perpComponent(host.Center.VectorTo(d), host.AxisDir))
	if err != nil {
		return spiricRimSpine{}, false // rim centre on the axis — degenerate frame
	}
	sp := spiricRimSpine{host: host, nHat: n, mHat: m, capD: capD, w: capD + side*r,
		b: host.MinorRadius + side*s*r, side: side, r: r}
	band := armSpindleBand * res.Weld()
	if sp.b < band || host.MajorRadius-sp.b <= stdmath.Abs(sp.w)+band {
		return spiricRimSpine{}, false // tube consumed, or the spine is not one closed loop
	}
	return sp, true
}

// rimCircleCenter is the closed rim edge's circle centre (a geom.Circle or a full-sweep geom.Arc3d —
// the forms a STEP rim import carries).
func rimCircleCenter(e *topo.Edge) (math.Point3, bool) {
	switch c := e.Geometry().(type) {
	case geom.Circle:
		return c.Center, true
	case geom.Arc3d:
		return c.Center, true
	default:
		return math.Point3{}, false
	}
}

// station evaluates one exact spiric station at tube angle ψ: the ball centre and its two exact feet
// (tube foot on the host torus, cap foot on the cap plane). ok=false when the offset plane exits the
// offset torus at this ψ (guarded against at spine build for the closed loop).
func (sp spiricRimSpine) station(psi float64) (c, tubeFoot, capFoot math.Point3, ok bool) {
	sinPsi, cosPsi := stdmath.Sincos(psi)
	rho := sp.host.MajorRadius + sp.b*cosPsi
	xiSq := rho*rho - sp.w*sp.w
	if xiSq <= 0 {
		return math.Point3{}, math.Point3{}, math.Point3{}, false
	}
	xi := stdmath.Sqrt(xiSq)
	nV, mV, zV := sp.nHat.AsVector(), sp.mHat.AsVector(), sp.host.AxisDir.AsVector()
	c = sp.host.Center.TranslateBy(nV.Scale(math.Scalar(sp.w))).
		TranslateBy(mV.Scale(math.Scalar(xi))).
		TranslateBy(zV.Scale(math.Scalar(sp.b * sinPsi)))
	capFoot = c.TranslateBy(nV.Scale(math.Scalar(sp.capD - sp.w))) // n̂-coord back to capD: exactly r off the plane
	cu := sp.host.Center.TranslateBy(mV.Scale(math.Scalar(xi)).Add(nV.Scale(math.Scalar(sp.w))).Scale(math.Scalar(sp.host.MajorRadius / rho)))
	tubeFoot = cu.TranslateBy(cu.VectorTo(c).Scale(math.Scalar(sp.host.MinorRadius / sp.b)))
	return c, tubeFoot, capFoot, true
}
