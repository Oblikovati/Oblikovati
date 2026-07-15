// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// imprintGrazeEps is the dimensionless grazing-tangency threshold for the circle∩band-line
// solve below: a discriminant smaller than (scale·imprintGrazeEps)² is treated as a tangent
// (or numerically indistinguishable-from-tangent) chord, not a genuine two-point crossing —
// the "Numerical pitfalls" tangency guard, mirrored from rimCrossings' weld-tolerance guard
// but expressed relative to the host's own size (ADR-0042) since this solve works in exact
// (unsampled) conic space where there is no polyline weld to reuse.
const imprintGrazeEps = 1e-6

// imprintCut is the exact circle∩band-line imprint solve result: the two crossing points in 3D
// (on the host plane) and the footprint sub-arc between them on the OUTBOARD side — the piece a
// later task merges into the host loop and trims the fillet against (plan
// docs/superpowers/plans/2026-07-14-curved-runout-imprint-fillet.md).
type imprintCut struct {
	pMinus, pPlus math.Point3
	arc           geom.Curve3
}

// solveImprint computes the exact crossing of im's footprint against the receded fillet band
// (reconstructed from im.nodes) and the outboard sub-arc between the crossings. It dispatches on the
// footprint curve type: a circular conic (geom.Circle / geom.Arc3d — imported STEP feature footprints
// arrive as Arc3d) via footprintConic + lineCircleRoots, and an ELLIPSE (geom.EllipseFull — the oblique
// elliptical-cylinder boss of T7, setback-patch-derivation.md D4) via solveImprintEllipse. A b-spline or
// other non-conic footprint honest-rejects (ok=false), same as a tangential/grazing crossing.
//
// Example: a boss footprint circle centered at the origin (r=8) crossing the band at y=-4
// crosses at (±√48,−4); solveImprint returns those points and the ~300° arc that stays above
// the band (geom.Arc3d, PointAt/TangentAt/Domain).
func solveImprint(im runoutImprint, res Resolution) (imprintCut, bool) {
	if e, ok := im.footprintEdge.Geometry().(geom.EllipseFull); ok {
		return solveImprintEllipse(im, e, res) // oblique elliptical-cylinder boss (T7): line∩ellipse
	}
	center, radius, ok := footprintConic(im.footprintEdge)
	if !ok {
		return imprintCut{}, false // e.g. a b-spline footprint: not an analytic setback boss
	}
	if im.nodes[0].P.DistanceTo(im.nodes[1].P) <= res.Weld() {
		return imprintCut{}, false // nodes too close to fix a band direction from
	}
	band := bandLineFromNodes(im.nodes)
	center2 := im.flat(center)
	t0, t1, ok := lineCircleRoots(band, center2, radius, hostBoundingDiag(im.host))
	if !ok {
		return imprintCut{}, false
	}
	p0, p1 := im.back(band.origin.TranslateBy(band.dir.Scale(t0))), im.back(band.origin.TranslateBy(band.dir.Scale(t1)))
	// outboardArc needs a full geom.Circle (Normal/RefDir) to pick the outbound sub-arc; im.plane's
	// normal gives it one (NewCircle's arbitrary RefDir is fine — outboardArc's candidate/fallback
	// pair is exhaustive and symmetric under any RefDir or Normal-sign choice, see its doc comment).
	circle, _ := geom.NewCircle(center, im.plane.Normal(), radius)
	return imprintCut{pMinus: p0, pPlus: p1, arc: outboardArc(im, circle, p0, p1)}, true
}

// footprintConic returns the exact center and radius of edge's footprint conic when it is a full
// geom.Circle or a geom.Arc3d (both already carry Center/Radius directly — no sample-and-fit,
// which would lose exactness). It reports ok=false for any other concrete geometry (e.g.
// geom.EllipseFull or a B-spline curve): those footprint kinds are Tasks 9/12's scope, not this
// one's, same honest-reject solveImprint used before Arc3d support.
func footprintConic(edge *topo.Edge) (center math.Point3, radius float64, ok bool) {
	switch g := edge.Geometry().(type) {
	case geom.Circle:
		return g.Center, g.Radius, true
	case geom.Arc3d:
		return g.Center, g.Radius, true
	default:
		return math.Point3{}, 0, false
	}
}

// bandLineFromNodes rebuilds the receded fillet band as a 2D line through both crossing nodes:
// each is a signedDist==0 point on that band by construction (Task 2's bandCrossings found
// them there), so the pair determines the exact same line without re-deriving it from
// ef/boundaryFromTangents — solveImprint only ever sees the packaged runoutImprint.
func bandLineFromNodes(nodes [2]crossing) boundaryLine2 {
	d := nodes[0].P.VectorTo(nodes[1].P)
	return boundaryLine2{origin: nodes[0].P, dir: d.Scale(1 / d.Length())}
}

// lineCircleRoots solves for the two line-parameter roots where the band line P(t)=origin+t·dir
// crosses the circle (center c, radius r): substituting P(t) into |P−c|²=r² gives
// t² + 2t·(dir·(origin−c)) + (|origin−c|²−r²) = 0. scale is the host's model-relative size
// (its vertex bounding-box diagonal, ADR-0042); a discriminant below (scale·imprintGrazeEps)²
// is a tangential/grazing chord, not a genuine crossing.
func lineCircleRoots(b boundaryLine2, c math.Point2, r, scale float64) (t0, t1 float64, ok bool) {
	w := c.VectorTo(b.origin) // origin − c
	bb := b.dir.Dot(w)
	cc := w.Dot(w) - r*r
	disc := bb*bb - cc
	eps := scale * imprintGrazeEps
	if disc < eps*eps {
		return 0, 0, false
	}
	s := stdmath.Sqrt(disc)
	return -bb - s, -bb + s, true
}

// hostBoundingDiag is host's characteristic model size (its vertex bounding-box diagonal, via
// math.BoxFromPoints/Diagonal/Length — math/box.go — rather than a hand-rolled min/max fold,
// audit finding: this used to reimplement math.Box), mirroring occtparity.boundingDiag / the
// body-scale pattern used elsewhere in kernel/ops — the grazing guard above must scale with the
// model, not a hard-coded constant.
func hostBoundingDiag(host *topo.Face) float64 {
	verts := host.Vertices()
	if len(verts) == 0 {
		return 1
	}
	pts := make([]math.Point3, len(verts))
	for i, v := range verts {
		pts[i] = v.Point()
	}
	return math.BoxFromPoints(pts...).Diagonal().Length()
}

// outboardArc builds the footprint circle's OUTBOARD sub-arc between p0 and p1 as an exact
// geom.Arc3d (no re-sampling). It picks between the chord's two candidate sub-arcs by an exact
// signed test — the candidate whose angular MIDPOINT lies on the HOST side of im's band line
// (im.side*im.boundary.signedDist(...) > 0, mirroring dipsPast's host/fillet sign convention,
// fillet_obstacle_detect.go) is outboard. This replaced a "pick the larger (major) arc" size
// heuristic that is WRONG for a DEEP dip: when the footprint circle's center sits on the FILLET
// side of the band, the host-side cap is the MINOR arc, not the major one (Finding 1; see
// TestSolveImprint_DeepDipSelectsMinorOutboardArc). The two candidates' midpoints are always
// antipodal on the circle (they split the full turn in two), so testing the first candidate and
// falling back to its complement is exhaustive.
func outboardArc(im runoutImprint, c geom.Circle, p0, p1 math.Point3) geom.Arc3d {
	binormal := c.Normal.Cross(c.RefDir)
	a0 := circleAngleOf(c, binormal, p0)
	ccw := ccwSweep(circleAngleOf(c, binormal, p1) - a0)
	first, _ := geom.NewArc3d(c.Center, c.Normal.AsVector(), c.RefDir.AsVector(), c.Radius, a0, ccw)
	if onHostSide(im, first.PointAt(0.5)) {
		return first
	}
	second, _ := geom.NewArc3d(c.Center, c.Normal.AsVector(), c.RefDir.AsVector(), c.Radius, a0, ccw-2*stdmath.Pi)
	return second
}

// onHostSide reports whether p lies on the HOST (outboard) side of im's band line — the same
// signed convention dipsPast uses for the fillet side (fillet_obstacle_detect.go), negated: im.side
// maps the fillet band to signedDist<0, so the host side is where the product is positive.
func onHostSide(im runoutImprint, p math.Point3) bool {
	return im.side*im.boundary.signedDist(im.flat(p)) > 0
}

// circleAngleOf returns p's parameter angle on circle c — the inverse of geom's internal
// pointOnCircle — given c's precomputed binormal (Normal × RefDir).
func circleAngleOf(c geom.Circle, binormal math.Vector3, p math.Point3) float64 {
	d := c.Center.VectorTo(p)
	return stdmath.Atan2(d.Dot(binormal), d.Dot(c.RefDir.AsVector()))
}

// ccwSweep normalizes a raw a1−a0 angle difference to [0, 2π): the CCW sweep from a0 to a1, the
// first of outboardArc's two candidate sub-arcs (its complement, going the other way around the
// circle, is ccw−2π).
func ccwSweep(raw float64) float64 {
	const twoPi = 2 * stdmath.Pi
	ccw := stdmath.Mod(raw, twoPi)
	if ccw < 0 {
		ccw += twoPi
	}
	return ccw
}
