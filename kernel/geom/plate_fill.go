// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/math"
)

// P4a of M6's degenerate-corner plate fill: compose the plate pipeline (P1 average-plane, P2
// Duchon solve, P3 rail discretisation) into a single fillable BSpline surface.
//
// PlateFill is the finish stage: it fits the fair plate over the corner region and returns a
// tensor-product B-spline the caller (a corner provider) can hand to the boolean/tessellator.
// Any stage error is returned unchanged so the caller falls through to coons4 (do-no-harm floor).
//
// The pipeline (see .superpowers/sdd/plate-math-kit.md and n7-fill-rails-rederivation.md):
//  1. average plane Ω from the rail sample anchors (AveragePlane, P1);
//  2. discretise the 4 rail sides into G0/G1 PlateConstraints + X/Y/Z target columns
//     (DiscretizeSides, P3);
//  3. decimate rows that collapse onto the SAME Ω point — the 4 shared corners, where two
//     abutting rails emit coincident (and, for G1, conflicting) rows that would make the shared
//     saddle matrix singular (kit §4 guard #1, sample decimation);
//  4. solve the three coordinate fields X(u,v), Y(u,v), Z(u,v) over the shared matrix
//     (PlateSolveMulti, P2);
//  5. grid-eval the lifted surface over a transfinite (ξ,η) parameterisation of the corner
//     region and least-squares fit it to a B-spline (ApproximateSurfaceLS).
//
// Grid-eval coordinate reconciliation (step 5, load-bearing): the three PlateCoeffs are the
// coordinate FIELDS X/Y/Z as direct functions of the Ω parameter (u,v) — P3's G0 rows carry the
// foot's WORLD coordinates (foot.X, foot.Y, foot.Z), so coeffs[c].Eval(u,v) reconstructs world
// coordinate c directly. The surface point is therefore math.P3(coeffX.Eval, coeffY.Eval,
// coeffZ.Eval), NOT d.Lift(...): the average plane is only the parameter domain, never a height
// reference (kit §Problem framing, "Not a Monge patch").

// plateRailSamples is the number of position/derivative sample stations taken along each of the
// 4 rails (both for the average-plane anchors and the solver rows). 8 gives ~40–96 constraints —
// enough to pin the fair surface without over-conditioning the dense RBF block on the small corner.
const plateRailSamples = 8

// plateGridSpan is the (ξ,η) grid resolution the solved plate is evaluated on before the B-spline
// fit. It must exceed the fitted control count each way so the least-squares system is
// over-determined; 21×21 = 441 samples over a 9×9 net is comfortably so.
const plateGridSpan = 21

// plateFitControls / plateFitDegree pin the finish B-spline to OCCT GeomPlate's nbcarreau≈9
// control points per direction at cubic degree (degmax ≤ 8; a bicubic net is fair yet flexible
// enough for the corner's curvature). ApproximateSurfaceLS needs nu ≥ du+1, satisfied by 9 ≥ 4.
const (
	plateFitControls = 9
	plateFitDegree   = 3
)

// PlateFill fills the 4-rail degenerate corner bounded by sides with a single fair B-spline: it
// builds the average-plane domain, discretises the rails, solves the Duchon plate per coordinate,
// and least-squares fits the evaluated surface. tol is the model-relative acceptance tolerance
// (ADR-0042) for the post-fit corner certificate. Any stage error is returned so the caller can
// fall through to coons4.
//
// The G1 tangency to each Order==1 rail's Adjacent surface is carried by the plate's derivative
// rows (P3), not a post-fit MatchSurface: MatchSurface needs a B-spline target, and a degenerate
// corner's arms are analytic (cylinder/torus/wall) — so we rely on the G1-constrained solve and
// verify the result with the runtime G1 witness in the tests (kit §3).
//
// Example:
//
//	surf, err := PlateFill(sides, res.Weld()*modelSize)
func PlateFill(sides [4]PlateSide, tol float64) (BSplineSurface, error) {
	d, err := plateFillDomain(sides)
	if err != nil {
		return BSplineSurface{}, err
	}
	coeffs, err := plateFillSolve(sides, d)
	if err != nil {
		return BSplineSurface{}, err
	}
	pts, us, vs := plateGrid(sides, d, coeffs)
	surf, err := ApproximateSurfaceLS(pts, us, vs, plateFitDegree, plateFitDegree, plateFitControls, plateFitControls)
	if err != nil {
		return BSplineSurface{}, err
	}
	if err := plateCornerCertificate(surf, sides, tol); err != nil {
		return BSplineSurface{}, err
	}
	return surf, nil
}

// plateFillDomain fits the average-plane Ω through every rail sample anchor (reusing P3's
// allSampleWorldPoints, which also validates the 4 rails are present and G1 sides carry Adjacent).
func plateFillDomain(sides [4]PlateSide) (PlateDomain, error) {
	anchors, err := allSampleWorldPoints(sides, plateRailSamples)
	if err != nil {
		return PlateDomain{}, err
	}
	return AveragePlane(anchors)
}

// plateFillSolve discretises the rails and solves the three coordinate fields, decimating the
// coincident corner rows first so the shared saddle matrix stays non-singular.
func plateFillSolve(sides [4]PlateSide, d PlateDomain) ([]PlateCoeffs, error) {
	cs, vals, err := DiscretizeSides(sides, d, plateRailSamples)
	if err != nil {
		return nil, err
	}
	cs, vals = decimateCoincidentRows(cs, vals)
	return PlateSolveMulti(cs, vals[:])
}

// decimateCoincidentRows drops any constraint that lands on an already-kept constraint of the SAME
// Order within the Ω weld tolerance — i.e. the 4 shared corners, where two abutting rails emit
// duplicate G0 rows and conflicting G1 derivative rows. Left in, the identical rows make the
// bordered saddle matrix exactly singular (kit §4). Keeping the first-seen arm's corner row loses
// only that station's specific along-rail vector; the neighbouring interior G1 rows still pin the
// tangent plane (which is shared at a corner — the two arms are tangent to their common host there).
func decimateCoincidentRows(cs []PlateConstraint, vals [3][]float64) ([]PlateConstraint, [3][]float64) {
	hmin := ResolutionForSize(plateDomainDiameter(cs)).Weld()
	var keptCs []PlateConstraint
	var keptVals [3][]float64
	for i := range cs {
		if isCoincidentRow(keptCs, cs[i], hmin) {
			continue
		}
		keptCs = append(keptCs, cs[i])
		for c := range 3 {
			keptVals[c] = append(keptVals[c], vals[c][i])
		}
	}
	return keptCs, keptVals
}

// isCoincidentRow reports whether c duplicates an already-kept constraint of the same Order within
// hmin in the Ω chart (a shared-corner collision, not a distinct interior station).
func isCoincidentRow(kept []PlateConstraint, c PlateConstraint, hmin float64) bool {
	for _, k := range kept {
		if k.Order == c.Order && stdmath.Hypot(k.U-c.U, k.V-c.V) <= hmin {
			return true
		}
	}
	return false
}

// plateGrid evaluates the solved plate over a transfinite (ξ,η) parameterisation of the corner
// region: the 4 rail domain-curves map to the 4 edges of the unit square (bilinear Coons blend),
// so the fitted surface's parameter square covers exactly the trimmed corner (its integrated area
// is the corner area, not the Ω bounding box). Returns the 3D grid plus the ξ,η fit parameters.
func plateGrid(sides [4]PlateSide, d PlateDomain, coeffs []PlateCoeffs) (pts []math.Point3, us, vs []float64) {
	n := plateGridSpan
	pts = make([]math.Point3, 0, n*n)
	us = make([]float64, 0, n*n)
	vs = make([]float64, 0, n*n)
	for i := range n {
		xi := float64(i) / float64(n-1)
		for j := range n {
			eta := float64(j) / float64(n-1)
			u, v := coonsDomainPoint(sides, d, xi, eta)
			pts = append(pts, plateSurfacePoint(coeffs, u, v))
			us = append(us, xi)
			vs = append(vs, eta)
		}
	}
	return pts, us, vs
}

// coonsDomainPoint returns the Ω point (u,v) at unit-square parameter (ξ,η) via the bilinear-
// blended Coons interpolation of the 4 rail domain-curves. The loop is V0→V1→V2→V3→V0, so the
// bottom edge (η=0) is side 0, right (ξ=1) side 1, top (η=1) side 2 reversed, left (ξ=0) side 3
// reversed; corners are V0=(0,0), V1=(1,0), V2=(1,1), V3=(0,1).
func coonsDomainPoint(sides [4]PlateSide, d PlateDomain, xi, eta float64) (u, v float64) {
	bottom := railDomainAt(sides[0], d, xi)
	right := railDomainAt(sides[1], d, eta)
	top := railDomainAt(sides[2], d, 1-xi)
	left := railDomainAt(sides[3], d, 1-eta)
	v0, v1 := railDomainAt(sides[0], d, 0), railDomainAt(sides[0], d, 1)
	v2, v3 := railDomainAt(sides[1], d, 1), railDomainAt(sides[3], d, 0)
	u = coonsBlend(bottom.X, top.X, left.X, right.X, v0.X, v1.X, v2.X, v3.X, xi, eta)
	v = coonsBlend(bottom.Y, top.Y, left.Y, right.Y, v0.Y, v1.Y, v2.Y, v3.Y, xi, eta)
	return u, v
}

// coonsBlend is the scalar bilinear-blended Coons value: the sum of the two ruled surfaces minus
// the bilinear corner term (Farin, Curves and Surfaces for CAGD).
func coonsBlend(bottom, top, left, right, c00, c10, c11, c01, xi, eta float64) float64 {
	ruled := (1-eta)*bottom + eta*top + (1-xi)*left + xi*right
	corner := (1-xi)*(1-eta)*c00 + xi*(1-eta)*c10 + xi*eta*c11 + (1-xi)*eta*c01
	return ruled - corner
}

// railDomainAt maps the unit parameter s∈[0,1] to side's curve domain and projects the point into
// Ω, returning the domain (u,v) as a Point2.
func railDomainAt(side PlateSide, d PlateDomain, s float64) math.Point2 {
	lo, hi := side.Curve.Domain()
	u, v := d.Project(side.Curve.PointAt(lo + (hi-lo)*s))
	return math.P2(u, v)
}

// plateSurfacePoint reconstructs the world surface point at Ω parameter (u,v) directly from the
// three coordinate fields (see the package doc's coordinate-reconciliation note — NOT d.Lift).
func plateSurfacePoint(coeffs []PlateCoeffs, u, v float64) math.Point3 {
	return math.P3(
		math.Scalar(coeffs[0].Eval(u, v)),
		math.Scalar(coeffs[1].Eval(u, v)),
		math.Scalar(coeffs[2].Eval(u, v)),
	)
}

// plateCornerCertificate rejects the fit when any of its four parameter-square corners drifts from
// the corresponding rail endpoint by more than tol — the cheap do-no-harm gate that a clamped
// B-spline interpolates exactly for a healthy solve, so a large drift signals a degenerate fit
// that must fall through to coons4 rather than ship a torn corner.
func plateCornerCertificate(surf BSplineSurface, sides [4]PlateSide, tol float64) error {
	lo0, hi0 := sides[0].Curve.Domain()
	_, hi1 := sides[1].Curve.Domain()
	lo3, _ := sides[3].Curve.Domain()
	corners := [4]struct {
		xi, eta float64
		want    math.Point3
	}{
		{0, 0, sides[0].Curve.PointAt(lo0)},
		{1, 0, sides[0].Curve.PointAt(hi0)},
		{1, 1, sides[1].Curve.PointAt(hi1)},
		{0, 1, sides[3].Curve.PointAt(lo3)},
	}
	for k, c := range corners {
		if dist := float64(surf.PointAt(c.xi, c.eta).DistanceTo(c.want)); dist > tol {
			return fmt.Errorf(
				"geom: PlateFill corner %d drifted %.6g from rail endpoint %v (> tol %.6g); degenerate fit",
				k, dist, c.want, tol)
		}
	}
	return nil
}
