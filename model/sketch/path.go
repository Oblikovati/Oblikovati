// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"

	"oblikovati.org/math"
)

// arcSampleStep bounds the angle of each polyline segment an arc path is sampled into
// (~10°), so a curved sweep/loft rail follows the arc instead of collapsing to its chord.
const arcSampleStep = stdmath.Pi / 18

// Path is a connected chain of sketch entities used as a sweep/loft rail or guide.
// It may be open or closed; tangency continuity between consecutive entities is a
// property the feature engine can check at consumption (the chain order is provided
// here).
type Path struct {
	entities []ProfileEntity
	closed   bool
}

// Entities returns the path's ordered entities; Count returns their number.
func (p *Path) Entities() []ProfileEntity {
	out := make([]ProfileEntity, len(p.entities))
	copy(out, p.entities)
	return out
}
func (p *Path) Count() int { return len(p.entities) }

// IsClosed reports whether the path forms a closed loop.
func (p *Path) IsClosed() bool { return p.closed }

// Points returns the path's ordered vertices in sketch space, honoring each segment's
// traversal direction. A sweep maps these through the path's sketch plane to a 3D rail.
// Every curved segment — arc (arcSampleStep), spline (true NURBS samples), elliptical
// arc, equation curve — is sampled into a polyline so a curved rail follows the curve
// instead of collapsing to its chord (M06-F12, Oblikovati/Oblikovati#627); lines
// contribute their endpoints.
func (p *Path) Points() []math.Point2 {
	var pts []math.Point2
	for i, pe := range p.entities {
		seg, ok := segmentPolyline(pe.Entity, pe.reversed)
		if !ok {
			continue
		}
		if i == 0 {
			pts = append(pts, seg...)
		} else {
			pts = append(pts, seg[1:]...) // drop the point shared with the previous segment's end
		}
	}
	return pts
}

// segmentPolyline returns a path segment's points in traversal order (reversed flips it).
// Curved segments are sampled into polylines; straight (and unknown) segments return
// their two endpoints.
func segmentPolyline(e Entity, reversed bool) ([]math.Point2, bool) {
	a, b, ok := segmentEnds(e)
	if !ok {
		return nil, false
	}
	pts := []math.Point2{a.Position(), b.Position()}
	switch t := e.(type) {
	case *Arc:
		pts = arcPolyline(t)
	case *Spline:
		pts = sampleSplineEntity(t)
	case *EllipticalArc:
		pts = sampleEllipticalArcEntity(t)
	case *EquationCurve:
		pts = t.Sample(curveSamples)
	}
	if reversed {
		for l, r := 0, len(pts)-1; l < r; l, r = l+1, r-1 {
			pts[l], pts[r] = pts[r], pts[l]
		}
	}
	return pts, true
}

// arcPolyline samples an arc from its start to its end into a polyline, ~arcSampleStep per
// segment, with the exact stored endpoints at the ends.
func arcPolyline(a *Arc) []math.Point2 {
	c, s, e := a.Center.Position(), a.Start.Position(), a.End.Position()
	r := float64(c.DistanceTo(s))
	angS := stdmath.Atan2(float64(s.Y-c.Y), float64(s.X-c.X))
	angE := stdmath.Atan2(float64(e.Y-c.Y), float64(e.X-c.X))
	sweep := angE - angS
	if a.CounterClockwise {
		if sweep <= 1e-12 {
			sweep += 2 * stdmath.Pi
		}
	} else if sweep >= -1e-12 {
		sweep -= 2 * stdmath.Pi
	}
	n := int(stdmath.Ceil(stdmath.Abs(sweep) / arcSampleStep))
	if n < 1 {
		n = 1
	}
	pts := make([]math.Point2, n+1)
	for i := 0; i <= n; i++ {
		ang := angS + sweep*float64(i)/float64(n)
		pts[i] = math.P2(c.X+math.Scalar(r*stdmath.Cos(ang)), c.Y+math.Scalar(r*stdmath.Sin(ang)))
	}
	pts[0], pts[n] = s, e // exact endpoints (avoid sampling drift)
	return pts
}

// Paths returns every maximal connected chain in the sketch — open and closed —
// from its non-construction geometry, for use as sweep/loft rails.
func (s *Sketch) Paths() []*Path {
	closed, open := detectLoops(s.normalGeometry())
	out := make([]*Path, 0, len(closed)+len(open))
	for _, l := range append(closed, open...) {
		out = append(out, &Path{entities: l.entities, closed: l.closed})
	}
	return out
}

// Path3D is a connected chain in 3D space (a sweep path from a 3D sketch). The 3D
// sketch entity model is built out incrementally; for now a Path3D is constructed
// directly from an ordered point chain, which is what the sweep feature consumes.
type Path3D struct {
	points []*Point3D
	closed bool
}

// NewPath3D builds a 3D path from an ordered chain of points.
func NewPath3D(points []*Point3D, closed bool) *Path3D {
	return &Path3D{points: append([]*Point3D(nil), points...), closed: closed}
}

// Points returns the path's ordered points as positions.
func (p *Path3D) Points() []math.Point3 {
	out := make([]math.Point3, len(p.points))
	for i, pt := range p.points {
		out[i] = pt.Position()
	}
	return out
}

// Count returns the number of points; IsClosed reports closure.
func (p *Path3D) Count() int     { return len(p.points) }
func (p *Path3D) IsClosed() bool { return p.closed }
