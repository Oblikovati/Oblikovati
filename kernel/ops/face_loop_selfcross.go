// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// A face's boundary, DEVELOPED onto its own surface, must be a SIMPLE polygon.
//
// WHY THIS IS AN INVARIANT AND NOT A PREFERENCE. A trimmed face is the region its loops bound in the
// surface's own parameter chart. If that polygon crosses itself the region is not defined: it has no
// area, no inside, and no correct triangulation — so every consumer downstream (the mesher, mass
// properties, export, the next boolean) is answering a question that has none. conformingPlaneMesh has
// refused such an input since it "once shrank a correct 8475 face to 675" (simpleLoop2D), but nothing
// ever asked whether the bodies the kernel SHIPS contain one.
//
// WHAT THE CORPUS SWEEP FOUND. 17 of the 1144 faces on the OCCT blend-parity corpus's shipped bodies
// have a self-crossing developed boundary, across 11 cases. It is exactly the population the
// conformance re-mesh kept stumbling into without being able to name: all 7 of the 10 cyl/cone
// conformance trims that fail their coverage certificate are in it, and all 3 that pass are not
// (cdt_coverage.go). The damage is wildly out of proportion to the defect — complex/D8's two MIRROR
// corner rounds carry the IDENTICAL crossing, pinching off 1.2111 of a 3307.1168 face (0.037%), and
// the constrained Delaunay answers −0.048% on one and −38.941% on the other purely because the
// crossing falls at a different index in the loop.
//
// SelfCrossingFaceLoops is the detector, so the class is measurable and can be ratcheted down. It is
// NOT wired into the mesher: the preceding slice measured that declining a non-simple domain is net
// harmful (simple/Q5's fillet face goes −5.85% → −19.3% against DRAWEXE), so the repair belongs
// upstream in whatever produced the boundary, not here.

// SelfCrossingLoop names one face loop whose developed boundary crosses itself.
type SelfCrossingLoop struct {
	Face *topo.Face
	Loop int     // index into Face.Loops()
	Area float64 // the area the crossing pinches off, in the surface's own metric chart
	// ChartChordRatio is the worst chart length ÷ 3D chord over the TWO segments that actually cross —
	// how faithfully the development renders the pair the report is about. 1 means the chart measured
	// the boundary; a large value means it did not, and Area is then a chart quantity that must NOT be
	// quoted as an area on the surface. See selfCrossChartFaithfulRatio for why it is REPORTED and
	// never used to filter.
	ChartChordRatio float64
}

// selfCrossChartFaithfulRatio is the largest chart-length ÷ 3D-chord a FAITHFUL development can produce
// for one boundary segment, and hence the cut between "this crossing was measured on the surface" and
// "this crossing was measured on a chart that is not the surface".
//
// WHERE IT COMES FROM (a closed form, not a calibration). A chart segment spanning angle θ about a
// periodic direction has chart length Rθ and 3D chord 2R·sin(θ/2), so the ratio is θ / (2 sin(θ/2)) —
// monotone in θ, and exactly π/2 at θ = π. Past a half turn the chord starts SHRINKING while the chart
// keeps growing, so the development has folded back on itself and the chart no longer even orders the
// boundary correctly. Measured on the corpus the two populations sit either side of it with no
// contest: the three chart-faithful crossings measure 1.000000 (a plane's development is exact, and
// simple/Q5's two cylinder segments are axial rulings), and the two unfaithful ones measure 2.771 and
// 77.06 — θ ≈ 4.42 and 6.20 rad.
//
// ★ IT IS A LABEL, NOT A FILTER, AND THAT IS THE WHOLE POINT. The obvious next move — mirror what the
// retrace detector does and DISCARD an unfaithful pair on a 3D check — is wrong here, and the corpus
// says so outright. A retrace is a claim that two strands resolve to the SAME 3D points (which is why
// that detector now asks the curves directly, #3475), so a 3D disagreement there can only be the
// chart's fault, and it is sound to reject on it. A self-crossing carries no
// such expectation: complex/F2's two unfaithful crossings are unfaithful because the boundary points
// themselves lie 9.125 and 9.818 OFF the radius-10 cylinder they bound — 9.87026 at worst, 0.05596 of
// that model's 176.4 diagonal, which is knownOffSurfaceDebt's complex/F2 entry read by a second ruler,
// and it is on the same face as the larger crossing. Filtering on the ratio would have retired two REAL
// defects as "chart artefacts" — measured, complex/F2 drops from 2 reported loops to 0 and every gate in
// the parity harness stays green — which is precisely the laundering the ratchet exists to prevent.
const selfCrossChartFaithfulRatio = stdmath.Pi / 2

// SelfCrossingFaceLoops returns every loop of b whose boundary, developed onto its own face's surface,
// is not a simple polygon — the faces whose trimmed region is undefined. Faces whose surface has no
// usable development (a fitted patch), and loops that wrap a periodic seam (where the development is
// not a polygon at all), are skipped rather than guessed at, so a report here is always a real defect.
// Each report carries ChartChordRatio so its Area can be read for what it is (see the field).
//
// The boundary it develops is the EXACT edge-corner ring (face_loop_corners.go), not a tessellation:
// a topology verdict must not move with facet density (M48/C3, Oblikovati/Oblikovati#3476). The
// Quality argument is therefore unused, and kept only so the call sites that thread one through do not
// churn — the same reason [FillInternalVoids] ignores its own.
//
// Example: SelfCrossingFaceLoops(d8Body, PropertyQuality()) returns the two corner-round walls whose
// far-end trim curve runs 0.2527 rad past their own u=0 ruling, each pinching off Area ≈ 1.2111.
func SelfCrossingFaceLoops(b *topo.Body, _ Quality) []SelfCrossingLoop {
	var out []SelfCrossingLoop
	for _, f := range b.Faces() {
		rings := faceCornerRings(f)
		loops, ok := developedFaceLoops(f, rings)
		if !ok {
			continue
		}
		out = append(out, crossingsOfFace(f, loops, rings)...)
	}
	return out
}

// crossingsOfFace reports each developed loop of f that crosses itself. Every candidate the chart
// produces is certified against the edges it is drawn from before it is reported, so a chart segment
// that cuts where its own edge does not is never a defect (crossingIsCertified).
func crossingsOfFace(f *topo.Face, loops []developedLoop, rings []cornerRing) []SelfCrossingLoop {
	var out []SelfCrossingLoop
	for i, l := range loops {
		r := ringAt(rings, i, len(l.pts))
		area, i0, j0, crosses := loopSelfCrossing(l, func(a, b int) bool {
			return crossingIsCertified(l, r, a, b)
		})
		if !crosses {
			continue
		}
		out = append(out, SelfCrossingLoop{Face: f, Loop: i, Area: area,
			ChartChordRatio: crossingPairChartChordRatio(l, r.pts, i0, j0)})
	}
	return out
}

// crossingIsCertified corroborates a chart crossing against the boundary the chart came from: the two
// segments must lie on DIFFERENT edges, and each must be a FAITHFUL development of its edge — the
// edge's own mid-parameter point, developed, must sit on the straight chart segment its ends span.
//
// One chart segment per edge renders a curved edge as a straight line, and a straight line can cut
// across ground the edge keeps clear of. Measured on the OCCT blend-parity corpus that invents
// crossings on six planar faces (whose chart is an exact isometry, so the only unfaithfulness there is
// the segment itself). The test is scale-free and reads no tessellation: it asks the edge for one more
// of its own points and checks the chart still agrees with it.
func crossingIsCertified(l developedLoop, r cornerRing, i, j int) bool {
	if i >= len(r.owners) || j >= len(r.owners) || r.owners[i] == r.owners[j] {
		return false // an unpairable ring, or one edge crossing itself, certifies nothing
	}
	return segmentDevelopsItsEdge(l, i) && segmentDevelopsItsEdge(l, j)
}

// segmentDevelopsItsEdge reports whether chart segment i renders its edge straight: the developed
// mid-parameter point must lie on the segment, within chartFaithfulFraction of the segment's own
// length. A zero-length chart segment develops nothing and certifies nothing.
func segmentDevelopsItsEdge(l developedLoop, i int) bool {
	if i >= len(l.mids) {
		return false
	}
	seg := chartSegAt(l.pts, i)
	span := seg.length()
	if span == 0 {
		return false
	}
	return pointToSegment2D(xy(l.mids[i]), seg.a, seg.b) <= chartFaithfulFraction*span
}

// chartFaithfulFraction is how far a segment's own mid may sit off it and still count as a straight
// development — a FRACTION of the segment's chart length, so it carries no model scale (ADR-0042). It
// is many decades above double-precision noise on a developed coordinate and many below the sagitta of
// any edge that actually bows: a 1° arc already bows by 1e-3 of its chord, and the population this
// separates on the corpus bows by 0.02 to 0.2 of theirs.
const chartFaithfulFraction = 1e-6 // tol:relative — dimensionless fraction of the chart segment's length

// pointToSegment2D is the distance from p to the segment ab, in chart coordinates.
func pointToSegment2D(p, a, b [2]float64) float64 {
	dx, dy := b[0]-a[0], b[1]-a[1]
	den := dx*dx + dy*dy
	if den == 0 {
		return stdmath.Hypot(p[0]-a[0], p[1]-a[1])
	}
	t := stdmath.Max(0, stdmath.Min(1, ((p[0]-a[0])*dx+(p[1]-a[1])*dy)/den))
	return stdmath.Hypot(p[0]-(a[0]+t*dx), p[1]-(a[1]+t*dy))
}

// ChartFaithful reports whether the development rendered the crossing pair faithfully, so Area is an
// area ON the surface rather than a number read off a chart that does not represent it.
func (s SelfCrossingLoop) ChartFaithful() bool {
	return s.ChartChordRatio <= selfCrossChartFaithfulRatio
}

// ringAt returns the corner ring matching developed loop i, or an EMPTY ring when the rings could not
// be paired with the developed loops point-for-point (in which case nothing can be measured on it).
func ringAt(rings []cornerRing, i, n int) cornerRing {
	if i >= len(rings) || len(rings[i].pts) != n {
		return cornerRing{}
	}
	return rings[i]
}

// crossingPairChartChordRatio is the worst chart-length ÷ 3D-chord over the two segments that cross.
// It returns 1 (nominally faithful) when the 3D ring is unavailable, so an unmeasurable pair is never
// silently promoted into the unfaithful population.
func crossingPairChartChordRatio(l developedLoop, ring []math.Point3, i, j int) float64 {
	if len(ring) == 0 {
		return 1
	}
	return stdmath.Max(segChartChordRatio(l, ring, i), segChartChordRatio(l, ring, j))
}

// segChartChordRatio is segment i's chart length over the 3D chord it develops; a zero-length chord
// under a positive chart length is infinitely unfaithful.
func segChartChordRatio(l developedLoop, ring []math.Point3, i int) float64 {
	n := len(ring)
	chart := chartSegAt(l.pts, i).length()
	chord := float64(ring[i].DistanceTo(ring[(i+1)%n]))
	if chord == 0 {
		if chart == 0 {
			return 1
		}
		return stdmath.Inf(1)
	}
	return chart / chord
}

// chartSeg is one directed segment of a developed loop, in the surface's metric chart.
type chartSeg struct{ a, b [2]float64 }

// chartSegAt is the segment leaving vertex i of a closed developed loop.
func chartSegAt(pts []math.Point2, i int) chartSeg {
	return chartSeg{a: xy(pts[i]), b: xy(pts[(i+1)%len(pts)])}
}

// length is the segment's extent in the metric chart.
func (s chartSeg) length() float64 { return stdmath.Hypot(s.b[0]-s.a[0], s.b[1]-s.a[1]) }

// developedLoop is one boundary loop in its surface's METRIC chart — u and v scaled to arc length, so
// an area in it is an area on the surface. mids carries each segment's own mid-parameter point through
// the SAME development, which is what certifies that a chart segment renders its edge at all.
type developedLoop struct {
	pts  []math.Point2
	mids []math.Point2
}

// developedFaceLoops develops every loop of f into the metric chart of f's own surface: the plane's
// own frame for a plane, the arc-length-scaled (u,v) for an analytic curved surface. ok=false for a
// surface with no such chart, or when any loop wraps the seam. The loops it develops are the exact
// edge-corner rings, so the development carries no facet density (#3476).
func developedFaceLoops(f *topo.Face, rings []cornerRing) ([]developedLoop, bool) {
	s := f.Geometry()
	if s == nil || len(rings) == 0 || len(rings[0].pts) < 3 {
		return nil, false
	}
	woven := wovenRings(rings)
	charts, ok := developRings(s, woven[0], woven[1:])
	if !ok {
		return nil, false
	}
	return unweaveLoops(charts), true
}

// developRings maps the boundary rings into the surface's metric chart: the plane's own frame for a
// plane, the arc-length-scaled (u,v) for an analytic curved surface.
func developRings(s geom.Surface, outer3D []math.Point3, holes3D [][]math.Point3) ([][]math.Point2, bool) {
	if pl, planar := s.(geom.Plane); planar {
		flat := planeProjector(pl.NormalAt(0, 0))
		return scaledLoops(append([][]math.Point2{project2D(outer3D, flat)}, project2DLoops(holes3D, flat)...), 1, 1), true
	}
	if !developableSurface(s) {
		return nil, false
	}
	outerUV, holesUV, ok := toUVLoops(s, outer3D, holes3D)
	if !ok {
		return nil, false
	}
	su, sv := metricScale(s)
	return scaledLoops(append([][]math.Point2{outerUV}, holesUV...), su, sv), true
}

// wovenRings interleaves each ring as corner, segment-mid, corner, segment-mid … so the certifying
// points pass through the SAME development and the SAME periodic unwrap as the corners they certify —
// developing them separately would re-open the wrap ambiguity the weave closes.
func wovenRings(rings []cornerRing) [][]math.Point3 {
	out := make([][]math.Point3, len(rings))
	for i, r := range rings {
		woven := make([]math.Point3, 0, 2*len(r.pts))
		for k, p := range r.pts {
			woven = append(woven, p, r.mids[k])
		}
		out[i] = woven
	}
	return out
}

// unweaveLoops splits each developed woven ring back into its corners and its per-segment mids.
func unweaveLoops(charts [][]math.Point2) []developedLoop {
	out := make([]developedLoop, len(charts))
	for i, c := range charts {
		loop := developedLoop{pts: make([]math.Point2, 0, len(c)/2), mids: make([]math.Point2, 0, len(c)/2)}
		for k, p := range c {
			if k%2 == 0 {
				loop.pts = append(loop.pts, p)
				continue
			}
			loop.mids = append(loop.mids, p)
		}
		out[i] = loop
	}
	return out
}

// developableSurface reports whether a curved surface has an analytic inversion whose (u,v) chart is a
// faithful development — the quadrics and the torus. A fitted patch is excluded: its ParamAt clamps to
// the patch box, so a "crossing" there would be an artefact of the inversion, not of the boundary.
func developableSurface(s geom.Surface) bool {
	switch s.(type) {
	case geom.Cylinder, geom.EllipticalCylinder, geom.Cone, geom.Sphere, geom.Torus:
		return true
	}
	return false
}

// scaledLoops scales each loop into the surface's metric chart.
func scaledLoops(loops [][]math.Point2, su, sv float64) [][]math.Point2 {
	out := make([][]math.Point2, len(loops))
	for i, l := range loops {
		pts := make([]math.Point2, len(l))
		for j, p := range l {
			pts[j] = math.P2(float64(p.X)*su, float64(p.Y)*sv)
		}
		out[i] = pts
	}
	return out
}

// loopSelfCrossing returns the area pinched off by the loop's first ACCEPTED proper self-crossing
// (non-adjacent edges crossing with strict signs, the same predicate simpleLoop2D uses) — the honest
// magnitude of the defect, and exactly the quantity a shoelace of the whole loop is wrong by — plus the
// INDICES of the two segments that cross, so the caller can ask the 3D boundary how faithfully the
// chart rendered that pair (SelfCrossingLoop.ChartChordRatio). The chart predicate itself is unchanged;
// accept is the caller's corroboration on the exact curves, and a candidate it rejects is skipped
// rather than ending the scan, so a chord artefact never hides a real crossing behind it.
func loopSelfCrossing(l developedLoop, accept func(i, j int) bool) (area float64, i, j int, crosses bool) {
	n := len(l.pts)
	if n < 4 {
		return 0, -1, -1, false
	}
	for i := range n {
		a, b := xy(l.pts[i]), xy(l.pts[(i+1)%n])
		for j := i + 2; j < n; j++ {
			if i == 0 && j == n-1 {
				continue // edges n-1→0 and 0→1 are adjacent (share vertex 0)
			}
			c, d := xy(l.pts[j]), xy(l.pts[(j+1)%n])
			if !segmentsCross(a, b, c, d) || !accept(i, j) {
				continue
			}
			return pinchedOffArea(l.pts, i, j), i, j, true
		}
	}
	return 0, -1, -1, false
}

// segmentsCrossPoint is the intersection of two segments already known to cross properly.
func segmentsCrossPoint(a, b, c, d [2]float64) math.Point2 {
	r := [2]float64{b[0] - a[0], b[1] - a[1]}
	s := [2]float64{d[0] - c[0], d[1] - c[1]}
	den := r[0]*s[1] - r[1]*s[0]
	if den == 0 {
		return math.P2(a[0], a[1])
	}
	t := ((c[0]-a[0])*s[1] - (c[1]-a[1])*s[0]) / den
	return math.P2(a[0]+t*r[0], a[1]+t*r[1])
}

// pinchedOffArea is the |shoelace| of the sub-loop the crossing cuts off: vertices i+1 … j, closed
// through the crossing point. That is exactly the amount the full loop's shoelace misreports, because
// the sub-loop is traversed with the opposite orientation to the rest.
func pinchedOffArea(pts []math.Point2, i, j int) float64 {
	n := len(pts)
	sub := []math.Point2{segmentsCrossPoint(xy(pts[i]), xy(pts[(i+1)%n]), xy(pts[j]), xy(pts[(j+1)%n]))}
	for k := i + 1; k <= j; k++ {
		sub = append(sub, pts[k%n])
	}
	var twice float64
	for k := range sub {
		p, q := sub[k], sub[(k+1)%len(sub)]
		twice += float64(p.X)*float64(q.Y) - float64(q.X)*float64(p.Y)
	}
	return stdmath.Abs(twice) / 2
}
