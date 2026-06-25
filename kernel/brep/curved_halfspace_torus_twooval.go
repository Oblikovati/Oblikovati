// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
)

// Two-oval torus half-space cut (M2 Phase-1 follow-up, Oblikovati/Oblikovati#1375). A plane PARALLEL to a
// torus axis whose offset is LESS than the inner tube radius (|K|/M < R−r) passes through the central hole
// and cuts BOTH tube walls — the spiric section is TWO ovals, each wrapping the whole tube (the v-period).
// The two ovals split the torus into two v-wrapping BANDS; the kept half-space keeps one of them. The kept
// band (one torus face wrapping the tube between the two ovals) is closed by TWO planar oval-disk lids (the
// two tube cross-sections the plane carves), unlike the single-oval cut's one lid.

// torusTwoOvalBand reports whether plane cuts two ovals from torus: PARALLEL to the axis (m·axis ≈ 0) with
// the offset up to the inner tube radius (0 < |K|/M ≤ R−r). Below R−r the plane passes through the hole and
// cuts both walls as two separate ovals; AT R−r exactly it is tangent to the inner equator and the two ovals
// touch in a FIGURE-EIGHT pinch (Oblikovati/Oblikovati#1375). The band loft handles the pinch as the band's
// zero-width limit there, so the same path serves both; above R−r the section is a single oval, handled
// elsewhere. At |K|/M = 0 the plane passes through the axis (the ovals degenerate to meridian circles, still
// deferred).
func torusTwoOvalBand(t geom.Torus, plane geom.Plane) bool {
	_, m, k, c := geom.TorusSectionCoeffs(t, plane)
	if stdmath.Abs(c) > cylinderAxisTol || m <= cylinderAxisTol {
		return false
	}
	ratio := stdmath.Abs(k) / m
	return ratio > cylinderAxisTol && ratio < t.MajorRadius-t.MinorRadius+cylinderAxisTol
}

// torusTwoOvalHalfSpace keeps the v-wrapping band a plane parallel to the torus axis leaves on its negative
// side: the band {g ≤ 0} swept around the tube between the two section ovals (through u = Phi+π, the side the
// half-space keeps), closed by the two planar oval-disk lids. The caller must have checked [torusTwoOvalBand].
func torusTwoOvalHalfSpace(t geom.Torus, plane geom.Plane) (*topo.Body, error) {
	phi, m, k, c := geom.TorusSectionCoeffs(t, plane)
	n := unit(plane.Normal())
	lidPlane, err := geom.NewPlane(plane.Origin, n)
	if err != nil {
		return nil, err
	}
	plus := geom.SpiricArc{Torus: t, Phi: phi, M: m, K: k, C: c, Branch: +1, V0: 0, V1: 2 * stdmath.Pi}
	minus := geom.SpiricArc{Torus: t, Phi: phi, M: m, K: k, C: c, Branch: -1, V0: 0, V1: 2 * stdmath.Pi}
	// The seam bridges the two ovals at v=0 the long way, through u=Phi+π — the side the kept band wraps.
	seam, err := geom.Arc3dByThreePoints(plus.PointAt(0), t.PointAt(phi+stdmath.Pi, 0), minus.PointAt(0))
	if err != nil {
		return nil, err
	}
	return buildTorusTwoOvalSolid(t, lidPlane, plus, minus, seam), nil
}

// buildTorusTwoOvalSolid assembles the kept band: one torus BAND face (the two closed-oval edges bridged by
// a v=0 seam, mirroring the perpendicular band) and TWO planar oval-disk lids, one bounded by each oval. The
// lids share each oval edge with the band in the opposite sense, so every edge has two uses (watertight).
func buildTorusTwoOvalSolid(t geom.Torus, lidPlane geom.Plane, plus, minus geom.SpiricArc, seam geom.Arc3d) *topo.Body {
	bld := topo.NewBuilder(true, topo.NewLineage(topo.Tok("halfspace", "torustwooval", 0)))
	vPlus := bld.AddVertex(plus.PointAt(0), torusLin("twoovalvp"))
	vMinus := bld.AddVertex(minus.PointAt(0), torusLin("twoovalvm"))
	ePlus := bld.AddEdge(plus, vPlus, vPlus, torusLin("twoovalplus"))    // closed oval (one seam vertex)
	eMinus := bld.AddEdge(minus, vMinus, vMinus, torusLin("twoovalmin")) // closed oval
	seamE := bld.AddEdge(seam, vPlus, vMinus, torusLin("twoovalseam"))
	bld.AddFace(t, torusLin("twoovalband"),
		topo.OuterLoop(topo.Fwd(seamE), topo.Rev(eMinus), topo.Rev(seamE), topo.Fwd(ePlus)))
	bld.AddFace(lidPlane, torusLin("twoovallidp"), topo.OuterLoop(topo.Rev(ePlus)))
	bld.AddFace(lidPlane, torusLin("twoovallidm"), topo.OuterLoop(topo.Fwd(eMinus)))
	return bld.Build()
}
