// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Exact pass-face interaction proofs for the mixed per-face boolean (ADR-0058). A pass-through face
// must not interact with the other operand; the box gate declines whenever boxes merely overlap — a
// tool INSIDE a cylinder always overlaps the wall's box while never touching the wall. Following the
// reference kernels (OCCT IntTools_FaceFace: intersect the SURFACES analytically, then test the curves
// against both TRIMS), an overlapping pair is now cleared by a closed-form proof: plane∩plane line
// intervals against both faces' exact trims, or plane∩cylinder curves (line pair / circle / ellipse)
// against the tool polygon and the wall band. Anything without a closed-form proof stays conservative
// (interacting → the boolean declines to its fallbacks). No sampled decision anywhere.

// passClearOf reports whether every pass-through face of p is provably clear of every face of the
// other operand: padded boxes disjoint, or an exact non-interaction proof.
func passClearOf(p, other facePartition) bool {
	otherFaces, otherBoxes := other.gateFaces()
	for i, pf := range p.pass {
		pb := inflateBox(p.passBox[i], facePairCullPad)
		for j := range otherFaces {
			if !pb.Intersects(otherBoxes[j]) {
				continue
			}
			if !passPairClear(pf, otherFaces[j]) {
				return false
			}
		}
	}
	return true
}

// gateFaces is the partition's full face list with a conservative box per face (the exact loop-point
// box for a polygonal face, the true topo box for a pass face).
func (p facePartition) gateFaces() ([]curvedFace, []math.Box) {
	faces := append(append(append(append([]curvedFace{}, p.planar...), p.uv...), p.wall...), p.pass...)
	boxes := make([]math.Box, 0, len(faces))
	for _, f := range p.planar {
		boxes = append(boxes, paddedFaceBox(f))
	}
	boxes = append(boxes, p.uvBox...)
	boxes = append(boxes, p.wallBox...)
	boxes = append(boxes, p.passBox...)
	return faces, boxes
}

// inflateBox grows a box by pad on every axis.
func inflateBox(b math.Box, pad float64) math.Box {
	g := math.Scalar(pad)
	return math.NewBox(b.Min.TranslateBy(math.V3(-g, -g, -g)), b.Max.TranslateBy(math.V3(g, g, g)))
}

// passPairClear proves one (pass face, other face) pair free of contact, or reports it interacting
// when no closed-form proof applies. The other face must be polygonal-planar (its trim is exact rings);
// the pass face is proven per surface kind.
func passPairClear(pf, of curvedFace) bool {
	if !allStraightFace(of) {
		return false
	}
	switch pf.surface.(type) {
	case geom.Plane:
		return planePassClear(pf, of)
	case geom.Cylinder:
		return cylinderPassClear(pf, of)
	default:
		return false
	}
}

// allStraightFace reports whether every boundary edge of the face is straight (a polygonal trim).
func allStraightFace(f curvedFace) bool {
	for _, l := range f.loops {
		if !straightLoop(l) {
			return false
		}
	}
	return true
}

// planePassClear proves a curved-edged PLANAR pass face clear of a polygonal face: the plane∩plane
// line's intervals inside the polygon (exact) and inside the pass face (exact even-odd over
// closed-form crossings) do not overlap within the cull pad. Parallel planes are clear unless
// coplanar (a flush contact is an interaction the mixed dispatch does not model).
func planePassClear(pf, of curvedFace) bool {
	p0, dir, ok := geom.PlanePlaneLine(facePlane(pf), facePlane(of))
	if !ok {
		return !coplanar(pf, of)
	}
	toolIv := faceLineIntervals(of, p0, dir)
	if len(toolIv) == 0 {
		return true
	}
	passIv, exact := curvedFaceLineIntervals(pf, p0, dir)
	if !exact {
		return false
	}
	return len(intersectIntervals(inflateIntervals(toolIv, facePairCullPad), passIv)) == 0
}

// inflateIntervals grows each interval by pad on both ends (the conservative contact band).
func inflateIntervals(iv [][2]float64, pad float64) [][2]float64 {
	out := make([][2]float64, len(iv))
	for i, v := range iv {
		out[i] = [2]float64{v[0] - pad, v[1] + pad}
	}
	return out
}

// cylinderPassClear proves a full-band cylinder wall clear of a polygonal planar face: every
// plane∩cylinder curve (line pair, circle, or ellipse) misses the polygon's trim or the wall's band.
func cylinderPassClear(pf, of curvedFace) bool {
	cyl, band, ok := fullCylinderSideBand(pf)
	if !ok {
		return false
	}
	res := geom.ResolutionForSize(2*cyl.Radius + (band.vMax - band.vMin))
	curves, handled := geom.IntersectSurfacesAnalytic(facePlane(of), cyl, res)
	if !handled {
		return false
	}
	for _, cv := range curves {
		if curveTouchesWallAndTool(cv, of, cyl, band) {
			return false
		}
	}
	return true
}

// curveTouchesWallAndTool reports whether one plane∩cylinder curve enters BOTH trims: the polygonal
// tool face and the wall's axial band. Unknown curve kinds report touching (conservative).
func curveTouchesWallAndTool(cv geom.Curve3, of curvedFace, cyl geom.Cylinder, band coneSideBand_) bool {
	switch c := cv.(type) {
	case geom.Line:
		return lineTouchesWallAndTool(c, of, cyl, band)
	case geom.Circle:
		return conicTouchesTool(cv, of, cyl, band, 0)
	case geom.EllipseFull:
		amp := ellipseAxialAmplitude(c, cyl)
		return conicTouchesTool(cv, of, cyl, band, amp)
	default:
		return true
	}
}

// lineTouchesWallAndTool tests one ruling line: its intervals inside the tool polygon, with the axial
// span of each interval against the band.
func lineTouchesWallAndTool(l geom.Line, of curvedFace, cyl geom.Cylinder, band coneSideBand_) bool {
	for _, iv := range faceLineIntervals(of, l.Origin, l.Dir.AsVector()) {
		vA := bandV(l.PointAt(iv[0]), cyl, band)
		vB := bandV(l.PointAt(iv[1]), cyl, band)
		if spansOverlap(stdmath.Min(vA, vB), stdmath.Max(vA, vB), band.vMin, band.vMax, facePairCullPad) {
			return true
		}
	}
	return false
}

// conicTouchesTool tests a circle/ellipse section curve: any exact crossing with the polygon's edges
// inside the band, else — when one curve point lies inside the polygon (the conic is enclosed) — the
// curve's axial span against the band. amp is the conic's axial half-amplitude (0 for a circle, which
// lies in a plane perpendicular to the axis).
func conicTouchesTool(cv geom.Curve3, of curvedFace, cyl geom.Cylinder, band coneSideBand_, amp float64) bool {
	pl := facePlane(of)
	pc, ok := toPlaneConic(cv, pl)
	if !ok {
		return true
	}
	if hit, touched := conicPolygonCrossingInBand(pc, of, cyl, band); touched {
		return hit
	}
	if !pointInFace2D(to2D(pl, cv.PointAt(0)), of) {
		return false // the conic lies wholly outside the polygon trim
	}
	vc := bandV(to3D(pl, pc.center), cyl, band) // the conic centre, back through the tool chart
	return spansOverlap(vc-amp, vc+amp, band.vMin, band.vMax, facePairCullPad)
}

// conicPolygonCrossingInBand scans the polygon's edges for exact conic crossings; touched=true when a
// crossing (or a tangency, conservatively) decides the answer, hit reporting whether any crossing lies
// within the wall band.
func conicPolygonCrossingInBand(pc planeConic, of curvedFace, cyl geom.Cylinder, band coneSideBand_) (hit, touched bool) {
	pl := facePlane(of)
	any := false
	for _, ring := range planarRings(of) {
		for i, n := 0, len(ring); i < n; i++ {
			hits, tangent := conicEdgeHits(pc, to2D(pl, ring[i]), to2D(pl, ring[(i+1)%n]), geom.ResolutionForPoints(ring))
			if tangent {
				return true, true // grazing contact: no sound parity, report interacting
			}
			for _, h := range hits {
				any = true
				if v := bandV(to3D(pl, h.p), cyl, band); v >= band.vMin-facePairCullPad && v <= band.vMax+facePairCullPad {
					return true, true
				}
			}
		}
	}
	return false, any // crossings existed but all outside the band → decided, no touch
}

// bandV is a point's axial coordinate in the wall band's frame (distance from the bottom rim centre).
func bandV(p math.Point3, cyl geom.Cylinder, band coneSideBand_) float64 {
	return float64(band.bottom.VectorTo(p).Dot(cyl.AxisDir.AsVector()))
}

// ellipseAxialAmplitude is the axial half-extent of a plane∩cylinder ellipse about its centre.
func ellipseAxialAmplitude(el geom.EllipseFull, cyl geom.Cylinder) float64 {
	axis := cyl.AxisDir.AsVector()
	a := el.MajorRadius * float64(el.MajorAxis.AsVector().Dot(axis))
	minor := el.Normal.AsVector().Cross(el.MajorAxis.AsVector())
	b := el.MinorRadius * float64(minor.Dot(axis))
	return stdmath.Sqrt(a*a + b*b)
}

// spansOverlap reports whether [a0,a1] and [b0,b1] come within pad of each other.
func spansOverlap(a0, a1, b0, b1, pad float64) bool {
	return a0 <= b1+pad && b0 <= a1+pad
}
