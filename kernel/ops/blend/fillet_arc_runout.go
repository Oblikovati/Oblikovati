// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// THE RUN-OUT TERMINATION — why an arc band cannot always be closed by a flat cross-section.
//
// rebuildWithArcFillet closes each end of the band with the tube cross-section in that end's RADIAL
// plane (vc → vt), which is right exactly when the side face the arc runs into IS that radial plane —
// a wall through the axis (simple/B2, simple/H6, simple/N6's first end). When the side plane is instead
// PARALLEL to the axis but stands off it, the cap-tangent point vt = capCentre + ref·majorR can land on
// the far side of that plane: the cross-section is then drawn through solid boundary into the void, and
// the band spills out of the solid. Measured on simple/W2 at the seat OCCT itself uses (cylR+r = 1.2):
// vt at the second end is (2.98303, 0, −0.19998) — a FULL r = 0.2 below the bottom plane z = 0 — and the
// shipped mesh reports it exactly, 3 → 29 free edges all strung along that spilled arc.
//
// OCCT does not run the band to the arc's ends. It terminates the band ON the side plane, and the
// boundary it terminates against is the SPIRIC section that plane cuts from the torus. Re-derived from
// DRAWEXE 8.0.0 on simple/W2 (`blend result s 0.2 s_2`, `dump result_3`): the band face carries FOUR
// edges, and one of them has a pcurve on the torus running (u,v) = (0.98506, π/2) → (1.55675, π) and a
// second pcurve on the plane z = 0 running (x,y) = (2.33652, 0) → (2.98586, 0.2). Both endpoints are the
// closed form of this section — 3 − √(1.2²−0.9999²) = 2.336524 where the cap-tangent circle meets z = 0,
// and 3 − √(1²−0.9999²) = 2.985858 where the cyl-tangent circle does — so OCCT's "run-out lobe" is not a
// separate face at all: it is the part of the SAME torus patch between the clipped rectangle and this
// section curve, worth 0.418101 − r·U·((R+r)·π/2 − r) = 0.0861.
//
// ★ READING DRAWEXE: identify a face by area + closed form, never by `bounding` (on a trimmed face that
// returns the underlying surface's POLE box, not the trimmed region), and read a per-face `sprops` area
// at `1.e-12` or tighter — its tolerance argument is a QUADRATURE target, and on a trimmed oblique
// quadric it moves the answer by up to 1.6 % (simple/T6's obstacle wall reads 2355.61 / 2393.32 / 2384.17
// at 1e-6 / 1e-9 / 1e-12 and emits NOTHING at 1e-13). Both caveats, with the receipts, are in
// model/feature/occtparity/perface_oracle_test.go.
//
// ★ The construction GENERALISES the flat cross-section rather than special-casing it. For a wall that
// contains the axis the section coefficients collapse to K = C = 0, w(v) ≡ 0, and u(v) ≡ Phi ± π/2 — a
// constant azimuth, i.e. the radial-plane tube arc itself. So a diametral end and a run-out end are the
// same construction evaluated at different stand-offs, which is why simple/W2's near-diametral first end
// (its top plane misses the axis by 1e-4) comes out of it as OCCT's own 1.99142 top face instead of the
// separately-shipped 0.008587 setback triangle.

// capTube is v = π/2, the tube angle of the CAP contact on every arc band: Torus.PointAt's axial term
// r·sin v reaches the cap plane there, whichever seat (cylR−r or cylR+r) the ball took.
const capTube = 1.5707963267948966

// arcSpillFloor is the stand-off, RELATIVE TO r, past a side plane at which a cap-tangent point counts as
// spilled rather than as construction noise. It is deliberately at noise level, not a modelling budget:
// a wall that truly contains the axis puts vt on the plane to the last bit (simple/B2's two walls measure
// 1e-16·r), while a genuine stand-off is a first-order fraction of r (simple/W2's second end measures
// 0.99990·r, its imported-cylinder-tilted first end 1.0e-4·r). Seven decades of headroom either way.
const arcSpillFloor = 1e-9

// resolveTerminal solves how end i's band terminates: on the end's own radial cross-section when that
// stays inside the side face, or ON the side plane — the spiric section — when it would spill through it.
// It also fixes the end's two contact azimuths, which a run-out end no longer shares.
func (af *arcFillet) resolveTerminal(i int) {
	end := &af.ends[i]
	u := unwrapNear(af.torusAzimuth(end.refDir), end.uAnchor)
	end.uCyl, end.uCap = u, u
	end.vc, end.vt = af.torus.PointAt(u, af.vCyl), af.torus.PointAt(u, capTube)
	sp, isPlane := end.sideF.Geometry().(geom.Plane)
	if !isPlane || !af.capTangentSpills(i, sp) {
		return
	}
	sec, ok := arcSideSection(af.torus, sp, af.vCyl, u)
	if !ok {
		return // the plane cuts no admissible section over the whole tube range: keep the setback
	}
	end.runout = &sec
	end.uCyl = unwrapNear(sec.UAt(af.vCyl), end.uAnchor)
	end.uCap = unwrapNear(sec.UAt(capTube), end.uAnchor)
	end.vc, end.vt = sec.PointAt(0), sec.PointAt(1)
}

// capTangentSpills reports whether end i's cap-tangent point lies OUTSIDE the solid past its own side
// face — the defect the run-out exists to prevent, measured with that face's outward normal so a
// Reversed side plane cannot read backwards.
func (af *arcFillet) capTangentSpills(i int, sp geom.Plane) bool {
	out := outwardPlaneNormal(af.ends[i].sideF, sp)
	return float64(sp.Origin.VectorTo(af.ends[i].vt).Dot(out)) > arcSpillFloor*af.r
}

// torusAzimuth returns the band's u at a radial direction, in the torus's own Ref/binormal frame.
func (af *arcFillet) torusAzimuth(ref math.UnitVector3) float64 {
	v, refDir := ref.AsVector(), af.torus.Ref.AsVector()
	binormal := af.torus.AxisDir.AsVector().Cross(refDir)
	return stdmath.Atan2(float64(v.Dot(binormal)), float64(v.Dot(refDir)))
}

// arcSideSection returns the spiric section the side plane cuts from the band, parameterised from the
// cyl-tangent contact (t=0, v = vCyl) to the cap-tangent contact (t=1, v = π/2). ok=false when the plane
// is perpendicular to the axis (it cuts circles, not a run-out) or when the section leaves the torus
// anywhere in that tube range — a band cannot be terminated on a curve that does not exist along it.
func arcSideSection(tor geom.Torus, sp geom.Plane, vCyl, uEnd float64) (geom.SpiricArc, bool) {
	phi, m, k, c := geom.TorusSectionCoeffs(tor, sp)
	if m <= arcSpillFloor {
		return geom.SpiricArc{}, false
	}
	plus := geom.SpiricArc{Torus: tor, Phi: phi, M: m, K: k, C: c, Branch: 1, V0: vCyl, V1: capTube}
	minus := plus
	minus.Branch = -1
	sec := plus
	if angleGap(plus.UAt(vCyl), uEnd) > angleGap(minus.UAt(vCyl), uEnd) {
		sec = minus // the arccos root that terminates on the arc's OWN end, not its mirror
	}
	if !arcSectionOnTorus(sec) {
		return geom.SpiricArc{}, false
	}
	return sec, true
}

// angleGap is the absolute difference between two azimuths, folded into [0, π].
func angleGap(a, b float64) float64 {
	d := stdmath.Mod(stdmath.Abs(a-b), 2*stdmath.Pi)
	if d > stdmath.Pi {
		d = 2*stdmath.Pi - d
	}
	return d
}

// arcSectionOnTorus reports whether the section's arccos argument stays inside [−1, 1] across the band's
// tube range — i.e. whether the plane really does cut the torus at every v the band spans. SpiricArc
// clamps rather than returning NaN, so an unchecked section would silently collapse onto u = Phi.
func arcSectionOnTorus(sec geom.SpiricArc) bool {
	const samples = 32
	for i := 0; i <= samples; i++ {
		v := sec.V0 + (sec.V1-sec.V0)*float64(i)/samples
		cv, sv := stdmath.Cos(v), stdmath.Sin(v)
		w := (sec.K - sec.C*sec.Torus.MinorRadius*sv) / (sec.M * (sec.Torus.MajorRadius + sec.Torus.MinorRadius*cv))
		if stdmath.Abs(w) > 1 {
			return false
		}
	}
	return true
}
