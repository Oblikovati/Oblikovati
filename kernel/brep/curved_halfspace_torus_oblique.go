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
// the outer tube. The plane splits the torus into the small CAP (the surface inside the oval) and its
// genus-1 COMPLEMENT (the full torus minus that cap); BOTH are kept exactly here, depending on which
// side the half-space keeps. Either kept solid is one torus face plus one planar oval lid, stitched
// along the two spiric branches. Other spiric topologies (two ovals, the figure-eight pinch, general
// oblique) still demote to CSG.

// torusAxisParallelOval reports whether plane cuts a single outer-tube oval from torus, returning the
// section's signed offset K. ok requires the plane be PARALLEL to the axis (m·axis ≈ 0) and its offset
// sit between the inner and outer tube radii (R−r < |K|/M < R+r) so the section is a single contractible
// oval — not two ovals (offset < R−r), a clearing plane, nor a perpendicular cut. The kept side (cap vs
// complement) is decided by the caller from the sign of K (K<0 keeps the small cap on the −m side).
func torusAxisParallelOval(t geom.Torus, plane geom.Plane) (k float64, ok bool) {
	_, m, k, c := geom.TorusSectionCoeffs(t, plane)
	if stdmath.Abs(c) > cylinderAxisCosTol || m <= cylinderAxisCosTol {
		return k, false
	}
	tol := torusSectionTol(t)
	ratio := stdmath.Abs(k) / m // the plane's distance from the axis, in tube radii
	ok = ratio > t.MajorRadius-t.MinorRadius+tol && ratio < t.MajorRadius+t.MinorRadius-tol
	return k, ok
}

// torusSingleOvalCap reports whether keeping the plane's negative side isolates the small outer-tube
// oval CAP (axis-parallel single oval with the kept side the small one, K<0).
func torusSingleOvalCap(t geom.Torus, plane geom.Plane) bool {
	k, ok := torusAxisParallelOval(t, plane)
	return ok && k < 0
}

// torusSingleOvalComplement reports whether keeping the plane's negative side keeps the genus-1
// COMPLEMENT (the full torus minus the oval cap, K>0 — the kept side is the big one).
func torusSingleOvalComplement(t geom.Torus, plane geom.Plane) bool {
	k, ok := torusAxisParallelOval(t, plane)
	return ok && k > 0
}

// The single-oval CAP (formerly torusObliqueHalfSpace) AND its genus-1 COMPLEMENT (formerly
// torusComplementHalfSpace) now route through the unified (u,v)-arrangement trimmer (torusSideSplit,
// curved_halfspace_torus_uv.go) — the same spiric section, the kept side chosen by the section's sign, the
// complement emitted as an outerless face with the oval as a hole (#1406). buildTorusOvalSolid below still
// serves the oblique single-oval cut until it too is migrated.

// buildTorusOvalSolid assembles a torus cap or its complement: the two spiric branches (+1 and −1) over
// the same tube-angle range form a bigon (they meet at the oval's v-extremes), closed by one torus face
// and one planar oval lid on the cut plane with outward normal +n. exterior=false builds the small CAP
// (the oval is the torus face's OUTER loop, bounding the small interior). exterior=true builds the genus-1
// COMPLEMENT (the oval is a HOLE on a torus face with no outer loop — it wraps the whole closed surface
// minus the oval). The winding cannot tell the two apart on a closed surface, so the representation does:
// an outer-loop oval is the small cap; a hole-loop oval is its complement.
func buildTorusOvalSolid(t geom.Torus, phi, m, k, c, v0, v1 float64, n math.Vector3, plane geom.Plane, exterior bool) (*topo.Body, error) {
	lidPlane, err := geom.NewPlane(plane.Origin, n)
	if err != nil {
		return nil, err
	}
	plus := geom.SpiricArc{Torus: t, Phi: phi, M: m, K: k, C: c, Branch: +1, V0: v0, V1: v1}
	minus := geom.SpiricArc{Torus: t, Phi: phi, M: m, K: k, C: c, Branch: -1, V0: v0, V1: v1}
	bld := topo.NewBuilder(true, topo.NewLineage(topo.Tok("halfspace", "torusoval", 0)))
	vBot := bld.AddVertex(plus.PointAt(0), torusLin("ovalvbot"))
	vTop := bld.AddVertex(plus.PointAt(1), torusLin("ovalvtop"))
	ePlus := bld.AddEdge(plus, vBot, vTop, torusLin("ovalplus"))
	eMinus := bld.AddEdge(minus, vBot, vTop, torusLin("ovalminus"))
	if exterior {
		// Oval is a hole; the torus face wraps the whole surface, the lid caps the oval the other way.
		bld.AddFace(t, torusLin("ovalface"), topo.InnerLoop(topo.Fwd(eMinus), topo.Rev(ePlus)))
		bld.AddFace(lidPlane, torusLin("ovallid"), topo.OuterLoop(topo.Fwd(ePlus), topo.Rev(eMinus)))
		return bld.Build(), nil
	}
	bld.AddFace(t, torusLin("ovalface"), topo.OuterLoop(topo.Fwd(ePlus), topo.Rev(eMinus)))
	bld.AddFace(lidPlane, torusLin("ovallid"), topo.OuterLoop(topo.Fwd(eMinus), topo.Rev(ePlus)))
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
