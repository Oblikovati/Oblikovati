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
}

// SelfCrossingFaceLoops returns every loop of b whose boundary, developed onto its own face's surface,
// is not a simple polygon — the faces whose trimmed region is undefined. Faces whose surface has no
// usable development (a fitted patch), and loops that wrap a periodic seam (where the development is
// not a polygon at all), are skipped rather than guessed at, so a report here is always a real defect.
//
// Example: SelfCrossingFaceLoops(d8Body, PropertyQuality()) returns the two corner-round walls whose
// far-end trim curve runs 0.2527 rad past their own u=0 ruling, each pinching off Area ≈ 1.2111.
func SelfCrossingFaceLoops(b *topo.Body, q Quality) []SelfCrossingLoop {
	var out []SelfCrossingLoop
	for _, f := range b.Faces() {
		loops, ok := developedFaceLoops(f, q)
		if !ok {
			continue
		}
		for i, l := range loops {
			if area, crosses := loopSelfCrossing(l); crosses {
				out = append(out, SelfCrossingLoop{Face: f, Loop: i, Area: area})
			}
		}
	}
	return out
}

// developedLoop is one boundary loop in its surface's METRIC chart — u and v scaled to arc length, so
// an area in it is an area on the surface.
type developedLoop struct{ pts []math.Point2 }

// developedFaceLoops develops every loop of f into the metric chart of f's own surface: the plane's
// own frame for a plane, the arc-length-scaled (u,v) for an analytic curved surface. ok=false for a
// surface with no such chart, or when any loop wraps the seam.
func developedFaceLoops(f *topo.Face, q Quality) ([]developedLoop, bool) {
	s := f.Geometry()
	outer3D := faceOuterBoundary(f, q)
	if s == nil || len(outer3D) < 3 {
		return nil, false
	}
	holes3D := faceHoleBoundaries(f, q)
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
// defect, and exactly the quantity a shoelace of the whole loop is wrong by.
func loopSelfCrossing(l developedLoop) (float64, bool) {
	n := len(l.pts)
	if n < 4 {
		return 0, false
	}
	for i := 0; i < n; i++ {
		a, b := xy(l.pts[i]), xy(l.pts[(i+1)%n])
		for j := i + 2; j < n; j++ {
			if i == 0 && j == n-1 {
				continue // edges n-1→0 and 0→1 are adjacent (share vertex 0)
			}
			c, d := xy(l.pts[j]), xy(l.pts[(j+1)%n])
			if !segmentsCross(a, b, c, d) {
				continue
			}
			return pinchedOffArea(l.pts, i, j), true
		}
	}
	return 0, false
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
