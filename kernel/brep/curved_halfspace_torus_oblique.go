// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Oblique torus half-space cut (M2 Phase-1 follow-up, Oblikovati/Oblikovati#1375). A plane NOT
// perpendicular to a torus axis cuts a quartic SPIRIC section (no analytic conic). This file handles
// the cleanest spiric topology: a plane PARALLEL to the axis whose offset isolates a single OVAL on
// the outer tube — the small CAP of the torus poking past the plane. The kept cap is one torus disk
// face (the surface inside the oval) closed by one planar oval lid, the two stitched along the two
// spiric branches. Other spiric topologies (two ovals, the figure-eight pinch, the genus-1 complement,
// general oblique) still demote to CSG; this is the first exact oblique torus cut.

// torusSingleOvalCap reports whether keeping the plane's negative side isolates a single outer-tube
// oval cap: the plane must be PARALLEL to the axis (m·axis ≈ 0), the kept side must be the small cap
// (K<0, so the negative half-space pokes toward the cap rather than holding the big complement), and
// the offset must sit between the inner and outer tube radii (R−r < |K|/M < R+r) so the section is a
// single contractible oval, not two ovals or a clearing plane.
func torusSingleOvalCap(t geom.Torus, plane geom.Plane) bool {
	_, m, k, c := geom.TorusSectionCoeffs(t, plane)
	if stdmath.Abs(c) > cylinderAxisTol || m <= cylinderAxisTol || k >= 0 {
		return false
	}
	ratio := -k / m // |K|/M: the plane's distance from the axis, in tube radii
	return ratio > t.MajorRadius-t.MinorRadius+cylinderAxisTol &&
		ratio < t.MajorRadius+t.MinorRadius-cylinderAxisTol
}

// torusObliqueHalfSpace keeps the single outer-tube oval cap a plane parallel to the torus axis slices
// off (the negative side). The caller must have checked [torusSingleOvalCap]. The cap's v-extent runs
// to the tube angle where the section oval pinches (cos v = (|K|/M − R)/r, where the two branches meet).
func torusObliqueHalfSpace(t geom.Torus, plane geom.Plane) (*topo.Body, error) {
	phi, m, k, c := geom.TorusSectionCoeffs(t, plane)
	cosVc := (-k/m - t.MajorRadius) / t.MinorRadius
	vc := stdmath.Acos(clampUnitF(cosVc))
	return buildTorusCapSolid(t, phi, m, k, c, -vc, vc, unit(plane.Normal()), plane)
}

// buildTorusCapSolid assembles the kept cap: one torus disk face (the surface patch inside the spiric
// oval) and one planar oval lid on the cut plane with outward normal +n, stitched along the two spiric
// branches (+1 and −1) over the same tube-angle range. The branches meet at the oval's v-extremes
// (u = Phi+π, where both arccos roots collapse), giving the two shared vertices.
func buildTorusCapSolid(t geom.Torus, phi, m, k, c, v0, v1 float64, n math.Vector3, plane geom.Plane) (*topo.Body, error) {
	lidPlane, err := geom.NewPlane(plane.Origin, n)
	if err != nil {
		return nil, err
	}
	plus := geom.SpiricArc{Torus: t, Phi: phi, M: m, K: k, C: c, Branch: +1, V0: v0, V1: v1}
	minus := geom.SpiricArc{Torus: t, Phi: phi, M: m, K: k, C: c, Branch: -1, V0: v0, V1: v1}
	bld := topo.NewBuilder(true, topo.NewLineage(topo.Tok("halfspace", "toruscap", 0)))
	vBot := bld.AddVertex(plus.PointAt(0), torusLin("capvbot")) // v=v0 pinch (u=Phi+π)
	vTop := bld.AddVertex(plus.PointAt(1), torusLin("capvtop")) // v=v1 pinch
	ePlus := bld.AddEdge(plus, vBot, vTop, torusLin("capplus"))
	eMinus := bld.AddEdge(minus, vBot, vTop, torusLin("capminus"))
	bld.AddFace(t, torusLin("capface"),
		topo.OuterLoop(topo.Fwd(ePlus), topo.Rev(eMinus)))
	bld.AddFace(lidPlane, torusLin("caplid"),
		topo.OuterLoop(topo.Fwd(eMinus), topo.Rev(ePlus)))
	return bld.Build(), nil
}

// clampUnitF folds a float64 into [−1, 1] for arccos arguments that floating-point error nudges past
// the unit interval at the oval's v-extremes.
func clampUnitF(x float64) float64 {
	if x > 1 {
		return 1
	}
	if x < -1 {
		return -1
	}
	return x
}
