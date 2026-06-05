// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"fmt"
	stdmath "math"

	"github.com/Oblikovati/oblikovati/math"
)

// Polyline is a connected chain of straight 3D segments through Vertices
// (contract: Polyline). It is parameterized t∈[0,1] uniformly across segments
// (each segment spans an equal 1/(n−1) slice of t, regardless of its length).
type Polyline struct {
	Vertices []math.Point3
}

// NewPolyline builds a polyline from at least two vertices. It copies the slice
// so the value stays immutable, and errors when given fewer than two points.
func NewPolyline(vertices []math.Point3) (Polyline, error) {
	if len(vertices) < 2 {
		return Polyline{}, fmt.Errorf("geom: polyline needs >= 2 vertices, got %d", len(vertices))
	}
	return Polyline{Vertices: append([]math.Point3(nil), vertices...)}, nil
}

// PointAt returns the position at parameter t.
func (p Polyline) PointAt(t float64) math.Point3 {
	seg, local := locateSegment(len(p.Vertices), t)
	a, b := p.Vertices[seg], p.Vertices[seg+1]
	return a.TranslateBy(a.VectorTo(b).Scale(local))
}

// TangentAt returns the derivative dP/dt on the active segment (its start→end
// vector scaled by the segment count, the chain factor for uniform t).
func (p Polyline) TangentAt(t float64) math.Vector3 {
	seg, _ := locateSegment(len(p.Vertices), t)
	return p.Vertices[seg].VectorTo(p.Vertices[seg+1]).Scale(float64(len(p.Vertices) - 1))
}

// Domain returns [0, 1].
func (p Polyline) Domain() (lo, hi float64) { return 0, 1 }

// Length returns the total length summed over all segments.
func (p Polyline) Length() float64 {
	total := 0.0
	for i := 0; i+1 < len(p.Vertices); i++ {
		total += p.Vertices[i].DistanceTo(p.Vertices[i+1])
	}
	return total
}

// Polyline2d is the 2D analogue of [Polyline] (contract: Polyline2d).
type Polyline2d struct {
	Vertices []math.Point2
}

// NewPolyline2d builds a 2D polyline from at least two vertices (copied).
func NewPolyline2d(vertices []math.Point2) (Polyline2d, error) {
	if len(vertices) < 2 {
		return Polyline2d{}, fmt.Errorf("geom: polyline needs >= 2 vertices, got %d", len(vertices))
	}
	return Polyline2d{Vertices: append([]math.Point2(nil), vertices...)}, nil
}

// PointAt returns the position at parameter t.
func (p Polyline2d) PointAt(t float64) math.Point2 {
	seg, local := locateSegment(len(p.Vertices), t)
	a, b := p.Vertices[seg], p.Vertices[seg+1]
	return a.TranslateBy(a.VectorTo(b).Scale(local))
}

// TangentAt returns the derivative dP/dt on the active segment.
func (p Polyline2d) TangentAt(t float64) math.Vector2 {
	seg, _ := locateSegment(len(p.Vertices), t)
	return p.Vertices[seg].VectorTo(p.Vertices[seg+1]).Scale(float64(len(p.Vertices) - 1))
}

// Domain returns [0, 1].
func (p Polyline2d) Domain() (lo, hi float64) { return 0, 1 }

// Length returns the total length summed over all segments.
func (p Polyline2d) Length() float64 {
	total := 0.0
	for i := 0; i+1 < len(p.Vertices); i++ {
		total += p.Vertices[i].DistanceTo(p.Vertices[i+1])
	}
	return total
}

// PolylineFromCurve3 tessellates a 3D curve into a polyline of `segments` straight chords
// (contract: CreatePolyline3dFromCurve), sampling the curve at segments+1 equally-spaced
// parameters across its domain. It errors when segments < 1 or the curve's domain is
// unbounded (an infinite line has no finite polyline form).
func PolylineFromCurve3(c Curve3, segments int) (Polyline, error) {
	ts, err := sampleParams(c.Domain, segments)
	if err != nil {
		return Polyline{}, err
	}
	verts := make([]math.Point3, len(ts))
	for i, t := range ts {
		verts[i] = c.PointAt(t)
	}
	return NewPolyline(verts)
}

// PolylineFromCurve2 is the 2D analogue of [PolylineFromCurve3] (contract:
// CreatePolyline2dFromCurve).
func PolylineFromCurve2(c Curve2, segments int) (Polyline2d, error) {
	ts, err := sampleParams(c.Domain, segments)
	if err != nil {
		return Polyline2d{}, err
	}
	verts := make([]math.Point2, len(ts))
	for i, t := range ts {
		verts[i] = c.PointAt(t)
	}
	return NewPolyline2d(verts)
}

// sampleParams returns segments+1 equally-spaced parameters spanning the [lo, hi] domain,
// validating the count and rejecting an unbounded domain (which has no finite tessellation).
func sampleParams(domain func() (lo, hi float64), segments int) ([]float64, error) {
	if segments < 1 {
		return nil, fmt.Errorf("geom: polyline-from-curve needs >= 1 segment, got %d", segments)
	}
	lo, hi := domain()
	if stdmath.IsInf(lo, 0) || stdmath.IsInf(hi, 0) {
		return nil, fmt.Errorf("geom: cannot tessellate an unbounded curve (domain [%g, %g])", lo, hi)
	}
	ts := make([]float64, segments+1)
	for i := range ts {
		ts[i] = lo + (hi-lo)*float64(i)/float64(segments)
	}
	return ts, nil
}

// locateSegment maps parameter t∈[0,1] to a segment index and the local
// fraction within it, clamping out-of-range t to the endpoints. vertexCount is
// assumed >= 2 (enforced by the constructors).
func locateSegment(vertexCount int, t float64) (seg int, local float64) {
	segs := vertexCount - 1
	s := t * float64(segs)
	seg = int(stdmath.Floor(s))
	if seg < 0 {
		return 0, 0
	}
	if seg >= segs {
		return segs - 1, 1
	}
	return seg, s - float64(seg)
}
