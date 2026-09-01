// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	"errors"
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// errConcaveCoveSpills marks a concave boss-base rim whose R+r cove contact circle reaches past the cap
// face's outer boundary (or into another feature's hole) — i.e. the fillet would bite the adjacent side
// walls and the result needs the deep #2012 boss-root multi-face weld (R8: R+r=11 on a ±10 box; W9:
// r=15). resolveRim catches it via errors.Is and leaves those rims on the UNCHANGED solveRim ladder, so
// the deep cases stay byte-identical while the fit-within-cap concave rims (W6/W8) take the corrected
// R+r cove.
var errConcaveCoveSpills = errors.New("fillet: concave rim R+r cove spills past the cap boundary")

// rimCapFrame is the cap-orientation-resolved geometric frame both rimWithCapOrientation retry tiers
// share: the TRUE (Reversed-aware) outward cap normal, the r-into-the-solid torus centre it implies, and
// the radial reference direction toward the picked rim vertex. Factored out of rimWithCapOrientation to
// keep it under the funlen budget (CLAUDE.md: functions 4-20 lines).
type rimCapFrame struct {
	capCenter   math.Point3
	outward     math.UnitVector3
	torusCenter math.Point3
	ref         math.UnitVector3
}

// solveRimCapFrame resolves the cap's TRUE outward-from-material normal via outwardPlaneNormal (already
// used by the S2/S5 concave arm builder for the same reason) instead of solveRim's raw pl.Normal(), which
// coincidentally already IS the outward normal on every convex rim the corpus exercised before K1/Z1 but
// is backwards when capF.Reversed()==true (K1's plate cap AND Z1's bottom-of-tower cap both are) —
// negating the raw normal then pushes torusCenter to the WRONG side of the cap, failing the material-side
// probe for a reason that has nothing to do with which radius is tried.
func solveRimCapFrame(e *topo.Edge, capF *topo.Face, cyl geom.Cylinder, pl geom.Plane, r float64) (rimCapFrame, error) {
	rimV := e.StartVertex()
	capCenter := projectOntoAxis(rimV.Point(), cyl.Origin, cyl.AxisDir)
	outward, err := math.UnitVector3FromVector(outwardPlaneNormal(capF, pl))
	if err != nil {
		return rimCapFrame{}, fmt.Errorf("fillet: degenerate rim cap normal %v", pl.Normal())
	}
	torusCenter := capCenter.TranslateBy(outward.Negate().AsVector().Scale(r)) // r into the solid
	ref, err := math.UnitVector3FromVector(perpComponent(capCenter.VectorTo(rimV.Point()), cyl.AxisDir))
	if err != nil {
		return rimCapFrame{}, fmt.Errorf("fillet: degenerate rim frame")
	}
	return rimCapFrame{capCenter: capCenter, outward: outward, torusCenter: torusCenter, ref: ref}, nil
}

// concaveBossRim routes a CONCAVE rim (a boss/pin base, where the wall meets its plate in a reentrant
// corner) to solveConcaveBossRim, returning handled=true when the concave cove owns the result (a built
// fillet, or a hard error). It returns handled=false — leaving resolveRim's UNCHANGED solveRim ladder to
// run byte-identically — for a convex rim, or for a concave rim whose R+r cove SPILLS past the cap (the
// deep #2012 boss-root weld, R8/W9): solveRim's convex R−r round silently builds a corner-biting
// watertight solid with the WRONG footprint (W6 +3.8%, W8 +5.5%) that the cove corrects. Convex rims
// never build a cove here, so I9/J1/K1/Z1 stay byte-identical.
func concaveBossRim(e *topo.Edge, cylF, capF *topo.Face, cyl geom.Cylinder, r float64) (rf *rimFillet, handled bool, err error) {
	if ClassifyEdgeConvexity(e) != EdgeConcave {
		return nil, false, nil
	}
	rf, err = solveConcaveBossRim(e, cylF, capF, cyl, r)
	if errors.Is(err, errConcaveCoveSpills) {
		return nil, false, nil
	}
	return rf, true, err
}

// solveConcaveBossRim solves the fillet for a CONCAVE boss-base rim — the reentrant circle where a
// boss/pin wall (a cylinder standing on a plate) meets the plate. OCCT's blend fills that corner with a
// cove whose rolling ball is tangent to the wall from OUTSIDE (tube centre at radius R+r, not solveRim's
// convex R−r) and offset AXIALLY toward the boss (the direction the wall rises from the rim), so the
// plate hole OPENS to R+r and the wall recedes by r. The contacts land at tube v=π (wall) and v=π/2
// (cap), the same as rimWithCapOrientation's R+r mirror, so the concave winding (threeQuarterTube seam,
// reversed band) is reused verbatim; only the centre placement differs from that bore-lip mirror (whose
// centre sits r INTO the solid, correct for a hole but backwards for a boss).
//
// It is gated to non-spilling rims: if the R+r cove's cap-tangent circle reaches past the cap's outer
// boundary it returns errConcaveCoveSpills, so R8/W9 (whose cove bites the side walls and needs the deep
// #2012 boss-root weld) fall back to the unchanged solveRim ladder untouched.
func solveConcaveBossRim(e *topo.Edge, cylF, capF *topo.Face, cyl geom.Cylinder, r float64) (*rimFillet, error) {
	rimV := e.StartVertex()
	capCenter := projectOntoAxis(rimV.Point(), cyl.Origin, cyl.AxisDir)
	ref, err := math.UnitVector3FromVector(perpComponent(capCenter.VectorTo(rimV.Point()), cyl.AxisDir))
	if err != nil {
		return nil, fmt.Errorf("fillet: degenerate rim frame")
	}
	seamEdge, bottomV := wallSeam(cylF, e, rimV)
	if seamEdge == nil {
		return nil, fmt.Errorf("fillet: cylinder wall has no seam edge at the rim")
	}
	wallDir := wallRiseDir(rimV, bottomV, cyl.AxisDir) // toward the boss the wall rises into
	majorR := cyl.Radius + r
	torusCenter := capCenter.TranslateBy(wallDir.AsVector().Scale(r))
	if !capTangentCircleFits(capF, capCenter, cyl.AxisDir, ref, majorR) {
		return nil, errConcaveCoveSpills
	}
	// Torus axis = −wallDir so the tube's v runs cap-tangent (v=π/2, toward the plate) and wall-tangent
	// (v=π, at the boss); reused threeQuarterTube seam midpoint sits between them.
	tor, err := geom.NewTorusWithRef(torusCenter, wallDir.Negate().AsVector(), ref.AsVector(), majorR, r)
	if err != nil {
		return nil, err
	}
	frame := rimCapFrame{capCenter: capCenter, torusCenter: torusCenter, ref: ref}
	return orientedRimFillet(cylF, capF, e, seamEdge, rimV, bottomV, cyl, frame, majorR, r, tor, true), nil
}

// wallRiseDir is the unit axial direction the cylinder wall rises from the rim toward its far (bottomV)
// end — the side the boss occupies. It is +axis or −axis, whichever agrees with rimV→bottomV.
func wallRiseDir(rimV, bottomV *topo.Vertex, axis math.UnitVector3) math.UnitVector3 {
	if float64(rimV.Point().VectorTo(bottomV.Point()).Dot(axis.AsVector())) < 0 {
		return axis.Negate()
	}
	return axis
}

// capTangentCircleFits reports whether the R+r cap-tangent circle (radius majorR, on the cap plane about
// capCenter) lies entirely inside the cap face — outside every hole and inside the outer boundary.
// Sampled densely; a single point off the cap means the cove would bite the adjacent walls (spill).
func capTangentCircleFits(capF *topo.Face, capCenter math.Point3, axis, ref math.UnitVector3, majorR float64) bool {
	ev := topo.NewFaceEvaluator(capF)
	bi := axis.Cross(ref)
	const n = 64
	for i := range n {
		a := 2 * stdmath.Pi * float64(i) / float64(n)
		p := capCenter.
			TranslateBy(ref.AsVector().Scale(majorR * stdmath.Cos(a))).
			TranslateBy(bi.Scale(majorR * stdmath.Sin(a)))
		if !ev.Contains(p) {
			return false
		}
	}
	return true
}

// rimWithCapOrientation is solveRim's shared tail, resolved through the cap-orientation-corrected
// solveRimCapFrame. resolveRim calls this ONLY after solveRim has declined (errConvexRimProbeFailed), so
// it never runs on any rim solveRim already builds and the convex I9/J1 path stays byte-identical. Since
// solveRim's seat became SIGNED (fillet_rim_cap_side.go) that set includes the concave-but-spilling boss
// roots simple/R8 and simple/W9, whose ball sits in the material where their convexity says void: tier 1
// declines them and this tier rebuilds the very same convex R−r round they always shipped — the frame is
// identical because on an un-Reversed cap solveRimCapFrame's outward normal IS pl.Normal(). Two retry
// tiers share this ONE fix:
//   - majorR = R−r, concave = false: the SAME convex geometry as solveRim, just cap-orientation-corrected
//     (Z1: a plain convex rim, not actually a bore lip — its cap is simply stored bottom-up).
//   - majorR = R+r, concave = true: the CONCAVE bore-lip mirror (K1: the plate material genuinely sits
//     OUTSIDE the bore, so even the corrected R−r probe still lands in the hole void). The rolling ball is
//     tangent to the bore wall from the material side at radius R+r instead of R−r; the wall contact
//     accordingly sits at tube v=π (NewTorusWithRef's Major+Minor·cos v term hits R at v=π when
//     major=R+r), the mirror of the convex v=0 equator, so the seam midpoint moves from quarterTube (π/4)
//     to threeQuarterTube (3π/4) — the cap contact (v=π/2, a pure axial offset with no major/minor mixing)
//     is unaffected either way.
//
// If the probe still fails at this tier's majorR, the rim needs more than a cap-orientation or R+r fix
// (e.g. a non-perpendicular cap) and this returns an honest reject rather than forcing a bad build.
func rimWithCapOrientation(b *topo.Body, e *topo.Edge, cylF, capF *topo.Face, cyl geom.Cylinder, pl geom.Plane, r, majorR float64, concave bool) (*rimFillet, error) {
	frame, err := solveRimCapFrame(e, capF, cyl, pl, r)
	if err != nil {
		return nil, err
	}
	probe := frame.torusCenter.TranslateBy(frame.ref.AsVector().Scale(majorR))
	if !brep.PointInside(b, probe) {
		return nil, fmt.Errorf(
			"fillet: rim radius %g is unsupported at radius %g (cap-orientation-corrected material-side probe at %g landed outside the body)",
			r, cyl.Radius, majorR)
	}
	// The torus axis points OUTWARD along the true (Reversed-aware) cap normal, mirroring solveRim's own
	// "axis points outward" convention (fillet_rim.go).
	tor, err := geom.NewTorusWithRef(frame.torusCenter, frame.outward.AsVector(), frame.ref.AsVector(), majorR, r)
	if err != nil {
		return nil, err
	}
	rimV := e.StartVertex()
	seamEdge, bottomV := wallSeam(cylF, e, rimV)
	if seamEdge == nil {
		return nil, fmt.Errorf("fillet: cylinder wall has no seam edge at the rim")
	}
	return orientedRimFillet(cylF, capF, e, seamEdge, rimV, bottomV, cyl, frame, majorR, r, tor, concave), nil
}

// orientedRimFillet assembles the rimFillet for either rimWithCapOrientation tier: the seam midpoint is
// quarterTube (v=π/4, the convex R−r contacts) unless concave selects threeQuarterTube (v=3π/4, the
// bore-lip R+r contacts) — see rimWithCapOrientation's doc comment for the tube-angle derivation.
func orientedRimFillet(cylF, capF *topo.Face, rim, seamEdge *topo.Edge, rimV, bottomV *topo.Vertex, cyl geom.Cylinder, frame rimCapFrame, majorR, r float64, tor geom.Torus, concave bool) *rimFillet {
	seamV := quarterTube
	if concave {
		seamV = threeQuarterTube
	}
	return &rimFillet{
		cyl: cylF, cap: capF, rimEdge: rim, seamEdge: seamEdge, rimV: rimV, bottomV: bottomV,
		cylTan: geom.Circle{Center: frame.torusCenter, Normal: cyl.AxisDir, RefDir: frame.ref, Radius: cyl.Radius},
		capTan: geom.Circle{Center: frame.capCenter, Normal: cyl.AxisDir, RefDir: frame.ref, Radius: majorR},
		band:   tor, r: r, concave: concave,
		seamMid: tor.PointAt(0, seamV),
	}
}
