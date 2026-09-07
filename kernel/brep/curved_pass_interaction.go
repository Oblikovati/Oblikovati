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
		pb := inflateBox(p.passBox[i])
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

// inflateBox grows a box by the face-pair cull pad on every axis, so a broad-phase reject is only
// taken when the pair is clear by more than the tolerance the narrow phase would decide on.
func inflateBox(b math.Box) math.Box {
	g := math.Scalar(facePairCullPad)
	return math.NewBox(b.Min.TranslateBy(math.V3(-g, -g, -g)), b.Max.TranslateBy(math.V3(g, g, g)))
}

// passPairClear proves one (pass face, other face) pair free of contact, or reports it interacting
// when no closed-form proof applies. The other face must be polygonal-planar (its trim is exact rings);
// the pass face is proven per surface kind.
func passPairClear(pf, of curvedFace) bool {
	// A SURFACE-level separation proof needs no trim at all: when the two surfaces never meet,
	// no patch of one can touch any patch of the other. It is the general form of the
	// parallel-planes clause below, and it is what clears an emboss pad seated on a chamfer cone —
	// the seat is the host cone sunk by the wrap sagitta, i.e. the same cone with a shifted apex,
	// so the two are everywhere |Δ|·sin(halfAngle) apart (#3459).
	if geom.SurfacesApart(pf.surface, of.surface, facePairCullPad) {
		return true
	}
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
		if curveTouchesWallAndTool(cv, of, cyl.AxisDir.AsVector(), band) {
			return false
		}
	}
	return true
}

// curveTouchesWallAndTool reports whether one plane∩cylinder curve enters BOTH trims: the polygonal
// tool face and the wall's axial band. Unknown curve kinds report touching (conservative).
func curveTouchesWallAndTool(cv geom.Curve3, of curvedFace, axis math.Vector3, band coneSideBand_) bool {
	if line, ok := cv.(geom.Line); ok {
		return lineTouchesWallAndTool(line, of, axis, band)
	}
	cf, ok := geom.AsConic(cv)
	if !ok {
		return true // an unrecognised section: report touching, the conservative answer
	}
	return conicTouchesTool(cv, of, axis, band, cf.AxialAmplitude(axis))
}

// lineTouchesWallAndTool tests one ruling line: its intervals inside the tool polygon, with the axial
// span of each interval against the band.
func lineTouchesWallAndTool(l geom.Line, of curvedFace, axis math.Vector3, band coneSideBand_) bool {
	for _, iv := range faceLineIntervals(of, l.Origin, l.Dir.AsVector()) {
		vA := bandV(l.PointAt(iv[0]), axis, band)
		vB := bandV(l.PointAt(iv[1]), axis, band)
		if spanMeetsBand(stdmath.Min(vA, vB), stdmath.Max(vA, vB), band) {
			return true
		}
	}
	return false
}

// conicTouchesTool tests a circle/ellipse section curve: any exact crossing with the polygon's edges
// inside the band, else — when one curve point lies inside the polygon (the conic is enclosed) — the
// curve's axial span against the band. amp is the conic's axial half-amplitude (0 for a circle, which
// lies in a plane perpendicular to the axis).
func conicTouchesTool(cv geom.Curve3, of curvedFace, axis math.Vector3, band coneSideBand_, amp float64) bool {
	pl := facePlane(of)
	pc, ok := toPlaneConic(cv, pl)
	if !ok {
		return true
	}
	if stdmath.IsInf(amp, 0) {
		// An UNBOUNDED branch is not settled by its boundary crossings. It can pass THROUGH the band
		// well inside the trim while crossing that trim's boundary far outside the band, and the scan
		// then reports crossings but none in range — "clear", when the arm sweeps the whole wall
		// (ADR-0062). Ask the direct question instead.
		return conicEntersTrimInBand(cv, of, axis, band)
	}
	if hit, touched := conicPolygonCrossingInBand(pc, of, axis, band); touched {
		return hit
	}
	if !pointInFace2D(to2D(pl, cv.PointAt(0)), of) {
		return false // the conic lies wholly outside the polygon trim
	}
	vc := bandV(to3D(pl, pc.center), axis, band) // the conic centre, back through the tool chart
	return spanMeetsBand(vc-amp, vc+amp, band)
}

// conicPolygonCrossingInBand scans the polygon's edges for exact conic crossings; touched=true when a
// crossing (or a tangency, conservatively) decides the answer, hit reporting whether any crossing lies
// within the wall band.
func conicPolygonCrossingInBand(pc planeConic, of curvedFace, axis math.Vector3, band coneSideBand_) (hit, touched bool) {
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
				if v := bandV(to3D(pl, h.p), axis, band); v >= band.vMin-facePairCullPad && v <= band.vMax+facePairCullPad {
					return true, true
				}
			}
		}
	}
	return false, any // crossings existed but all outside the band → decided, no touch
}

// bandV is a point's axial coordinate in the wall band's OWN v frame — the frame its vMin/vMax are
// expressed in, which is what every caller compares the result against.
//
// The two band builders anchor v differently: a cylinder measures from its bottom rim (vMin=0),
// a cone from its APEX (vMin = apex→bottom), because a cone's radius is a function of apex
// distance. What they agree on is that band.bottom sits at v = vMin, so adding vMin to the offset
// from bottom reads both correctly. Omitting it silently mis-read every CONE band by the
// apex-to-rim distance — a chamfer cone's sections landed at v=4.5 against a band of [10,20] and
// so read as "outside the band", which is why an emboss pad on a chamfer never imprinted (#3459).
// It went unnoticed because the wall-conic path had only ever run against cylinders, where vMin
// is 0 and the two frames coincide.
func bandV(p math.Point3, axis math.Vector3, band coneSideBand_) float64 {
	return band.vMin + float64(band.bottom.VectorTo(p).Dot(axis))
}

// spanMeetsBand reports whether the axial span [lo,hi] comes within the band's own cull margin of the
// band. Every caller asks about one band, so the band is the argument and the margin is read off it.
func spanMeetsBand(lo, hi float64, band coneSideBand_) bool {
	pad := bandCullPad(band)
	return lo <= band.vMax+pad && band.vMin <= hi+pad
}

// bandCullPad is the axial margin a section must clear a wall's rim by to count as strictly inside the
// band — facePairCullPad expressed against the band's OWN extent instead of as an absolute length.
//
// It is ten stitch welds, which is exactly what facePairCullPad is (ten planar stitch grids) at the
// historical ~1-unit part, so no larger part's classification moves. As an absolute length it was a
// margin only at that one scale: a 0.2 mm plate's cap sits exactly one such pad below its bore's rim,
// so the section read as a rim contact and the simplest drill there is declined (ADR-0042, ADR-0061).
func bandCullPad(band coneSideBand_) float64 {
	return cullPadStitches * geom.ResolutionForSize(bandSize(band)).Stitch()
}

// cullPadStitches is the cull margin counted in stitch welds — dimensionless, so it scales with the band.
const cullPadStitches = 10 // tol:numeric — margin in stitch welds, not a length

// bandSize is a wall band's own extent: its diameter at the wider rim plus its axial height.
func bandSize(band coneSideBand_) float64 {
	return 2*stdmath.Max(band.rBot, band.rTop) + (band.vMax - band.vMin)
}

// bandPlacement classifies an axial span against a wall band: strictly inside it, or strictly clear of
// it. Neither means the span straddles a rim, which its callers decline. The span's SOURCE differs — a
// conic's centre and amplitude in closed form, a general crossing walked — the rule does not.
func bandPlacement(lo, hi float64, band coneSideBand_) (inside, clear bool) {
	pad := bandCullPad(band)
	return lo > band.vMin+pad && hi < band.vMax-pad, !spanMeetsBand(lo, hi, band)
}

// conicEntersTrimInBand reports whether an unbounded conic section has a point inside BOTH the wall's
// axial band and the tool face's trim. The band window is inverted to the branch's own parameters
// (geom.AxialWindowParams) and those spans are walked: a branch that reaches the band at all reaches it
// over an interval, so a sampled walk of that interval finds the contact when there is one.
func conicEntersTrimInBand(cv geom.Curve3, of curvedFace, axis math.Vector3, band coneSideBand_) bool {
	base := bandBase(axis, band)
	spans, ok := geom.AxialWindowParams(cv, base, axis, band.vMin, band.vMax)
	if !ok {
		return true // the branch's band window cannot be bracketed: be conservative
	}
	pl := facePlane(of)
	for _, sp := range spans {
		for k := 0; k <= conicBandWalkSamples; k++ {
			t := sp[0] + (sp[1]-sp[0])*float64(k)/conicBandWalkSamples
			if pointInFace2D(to2D(pl, cv.PointAt(t)), of) {
				return true
			}
		}
	}
	return false
}

// conicBandWalkSamples walks the branch's in-band interval finely enough to find a contact with any
// trim a modelled tool face has.
const conicBandWalkSamples = 64

// bandBase is the point the band's axial coordinate is measured from — its own vMin along the axis.
func bandBase(axis math.Vector3, band coneSideBand_) math.Point3 {
	return band.bottom.TranslateBy(axis.Scale(math.Scalar(-bandV(band.bottom, axis, band) + band.vMin)))
}
