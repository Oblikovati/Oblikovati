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
		loops, ok := developedFaceLoops(f)
		rings := faceCornerRings(f)
		if !ok {
			continue
		}
		for i, l := range loops {
			area, i0, j0, crosses := loopSelfCrossing(l)
			if !crosses {
				continue
			}
			out = append(out, SelfCrossingLoop{Face: f, Loop: i, Area: area,
				ChartChordRatio: crossingPairChartChordRatio(l, ringAt(rings, i, len(l.pts)), i0, j0)})
		}
	}
	return out
}

// ChartFaithful reports whether the development rendered the crossing pair faithfully, so Area is an
// area ON the surface rather than a number read off a chart that does not represent it.
func (s SelfCrossingLoop) ChartFaithful() bool {
	return s.ChartChordRatio <= selfCrossChartFaithfulRatio
}

// ringAt returns the 3D ring matching developed loop i, or nil when the rings could not be paired with
// the developed loops point-for-point (in which case no ratio can be measured).
func ringAt(rings [][]math.Point3, i, n int) []math.Point3 {
	if i >= len(rings) || len(rings[i]) != n {
		return nil
	}
	return rings[i]
}

// crossingPairChartChordRatio is the worst chart-length ÷ 3D-chord over the two segments that cross.
// It returns 1 (nominally faithful) when the 3D ring is unavailable, so an unmeasurable pair is never
// silently promoted into the unfaithful population.
func crossingPairChartChordRatio(l developedLoop, ring []math.Point3, i, j int) float64 {
	if ring == nil {
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
// an area in it is an area on the surface.
type developedLoop struct{ pts []math.Point2 }

// developedFaceLoops develops every loop of f into the metric chart of f's own surface: the plane's
// own frame for a plane, the arc-length-scaled (u,v) for an analytic curved surface. ok=false for a
// surface with no such chart, or when any loop wraps the seam. The loops it develops are the exact
// edge-corner rings, so the development carries no facet density (#3476).
func developedFaceLoops(f *topo.Face) ([]developedLoop, bool) {
	s := f.Geometry()
	outer3D := faceOuterCorners(f)
	if s == nil || len(outer3D) < 3 {
		return nil, false
	}
	holes3D := faceHoleCorners(f)
	if pl, planar := s.(geom.Plane); planar {
		flat := planeProjector(pl.NormalAt(0, 0))
		return unitScaledLoops(append([][]math.Point2{project2D(outer3D, flat)}, project2DLoops(holes3D, flat)...)), true
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

// unitScaledLoops wraps already-metric (planar) loops.
func unitScaledLoops(loops [][]math.Point2) []developedLoop {
	return scaledLoops(loops, 1, 1)
}

// scaledLoops scales each loop into the surface's metric chart.
func scaledLoops(loops [][]math.Point2, su, sv float64) []developedLoop {
	out := make([]developedLoop, len(loops))
	for i, l := range loops {
		pts := make([]math.Point2, len(l))
		for j, p := range l {
			pts[j] = math.P2(float64(p.X)*su, float64(p.Y)*sv)
		}
		out[i] = developedLoop{pts: pts}
	}
	return out
}

// loopSelfCrossing returns the area pinched off by the loop's FIRST proper self-crossing (non-adjacent
// edges crossing with strict signs, the same predicate simpleLoop2D uses) — the honest magnitude of the
// defect, and exactly the quantity a shoelace of the whole loop is wrong by — plus the INDICES of the
// two segments that cross, so the caller can ask the 3D boundary how faithfully the chart rendered that
// pair (SelfCrossingLoop.ChartChordRatio). The predicate itself is unchanged: the indices are reported,
// never acted on.
func loopSelfCrossing(l developedLoop) (area float64, i, j int, crosses bool) {
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
			if !segmentsCross(a, b, c, d) {
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
