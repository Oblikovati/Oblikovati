// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

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

// rimWithCapOrientation is solveRim's shared tail, resolved through the cap-orientation-corrected
// solveRimCapFrame instead of solveRim's raw pl.Normal(). resolveRim calls this ONLY after solveRim's own
// R−r probe has already failed (errConvexRimProbeFailed), so it never runs on any rim solveRim already
// builds and the convex I9/J1 path stays byte-identical. Two retry tiers share this ONE fix:
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
	if !PointInsideBody(b, probe) {
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
		torus:  tor, r: r, concave: concave,
		seamMid: tor.PointAt(0, seamV),
	}
}
