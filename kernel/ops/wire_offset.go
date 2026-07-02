// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Planar wire offsetting (M07-F06, Oblikovati/Oblikovati#629): offset a wire's
// chain in its plane by a distance, joining the corners per the closure type.
// Positive distance offsets toward normal × tangent (the LEFT of travel about
// the given plane normal); negative offsets right. Line and arc edges offset
// exactly (an arc's radius grows or shrinks); other curves offset as sampled
// polylines. Corners where the offsets cross are trimmed at the support-curve
// intersection; gap corners close per WireOffsetCorner. Global
// self-intersection removal of the result (an offset larger than a concave
// feature) is out of scope — the local result is returned as computed.

// WireOffsetCorner selects how a gap corner closes (the reference
// OffsetCornerClosureTypeEnum semantics).
type WireOffsetCorner uint8

const (
	// WireCornerCircular rounds the gap with an arc about the original corner.
	WireCornerCircular WireOffsetCorner = iota
	// WireCornerLinear extends both sides tangentially to their intersection.
	WireCornerLinear
	// WireCornerExtend grows the actual curves (an arc stays an arc) to meet.
	WireCornerExtend
)

// wireOffsetTol is the coincidence/degeneracy tolerance of the offset, scaled to the
// wire's own extent (#1610, ADR-0042 — formerly an absolute 1e-9 cm, which passed
// degenerate radii on metre-scale wires).
func wireOffsetTol(w *topo.Wire) float64 {
	return ResolutionForPoints(wireVertexPoints(w)).Weld()
}

// wireVertexPoints returns the wire's edge endpoints — the extent proxy its
// model-relative tolerances derive from.
func wireVertexPoints(w *topo.Wire) []math.Point3 {
	var pts []math.Point3
	for _, u := range w.Uses() {
		pts = append(pts, u.Edge.StartVertex().Point(), u.Edge.EndVertex().Point())
	}
	return pts
}

// OffsetPlanarWire offsets w by distance in the plane with the given normal,
// returning a new wire-only body. The wire must lie in that plane.
//
// Example: out, err := ops.OffsetPlanarWire(w, math.V3(0, 0, 1), 0.5, ops.WireCornerCircular)
func OffsetPlanarWire(w *topo.Wire, normal math.Vector3, distance float64, corner WireOffsetCorner) (*topo.Body, error) {
	if distance == 0 {
		return nil, fmt.Errorf("ops.OffsetPlanarWire: distance must be non-zero")
	}
	pl, err := newWirePlane(w, normal)
	if err != nil {
		return nil, err
	}
	segs, err := wireSegments2D(w, pl)
	if err != nil {
		return nil, err
	}
	tol := wireOffsetTol(w)
	offs, err := offsetSegments(segs, distance, tol)
	if err != nil {
		return nil, err
	}
	joined, err := joinOffsetCorners(segs, offs, distance, tol, corner, w.IsClosed())
	if err != nil {
		return nil, err
	}
	return buildWireBody(joined, pl, w.IsClosed())
}

// wirePlane is the 2D frame the offset works in: x along u, y along v = n×u.
type wirePlane struct {
	origin  math.Point3
	n, u, v math.Vector3
}

// newWirePlane frames the wire's plane from the caller's normal and verifies
// the wire actually lies in it.
func newWirePlane(w *topo.Wire, normal math.Vector3) (wirePlane, error) {
	l := float64(normal.Length())
	if l == 0 {
		return wirePlane{}, fmt.Errorf("ops.OffsetPlanarWire: zero plane normal")
	}
	n := normal.Scale(math.Scalar(1 / l))
	uses := w.Uses()
	if len(uses) == 0 {
		return wirePlane{}, fmt.Errorf("ops.OffsetPlanarWire: empty wire")
	}
	pl := frameAbout(uses[0].Edge.StartVertex().Point(), n)
	if err := verifyWireInPlane(w, pl); err != nil {
		return wirePlane{}, err
	}
	return pl, nil
}

// frameAbout picks in-plane axes for n through origin o.
func frameAbout(o math.Point3, n math.Vector3) wirePlane {
	ref := math.V3(1, 0, 0)
	if stdmath.Abs(float64(n.X)) > 0.9 {
		ref = math.V3(0, 1, 0)
	}
	u := n.Cross(ref)
	u = u.Scale(math.Scalar(1 / float64(u.Length())))
	return wirePlane{origin: o, n: n, u: u, v: n.Cross(u)}
}

// verifyWireInPlane samples the wire and rejects out-of-plane geometry with
// the offending deviation.
func verifyWireInPlane(w *topo.Wire, pl wirePlane) error {
	tol := ResolutionForPoints(wireVertexPoints(w)).Plane() // model-relative (#1610)
	for _, u := range w.Uses() {
		c := u.Edge.Geometry()
		lo, hi := c.Domain()
		for i := 0; i <= 16; i++ {
			p := c.PointAt(lo + (hi-lo)*float64(i)/16)
			if dev := stdmath.Abs(float64(pl.origin.VectorTo(p).Dot(pl.n))); dev > tol {
				return fmt.Errorf("ops.OffsetPlanarWire: edge %d leaves the plane by %g (max %g)", u.Edge.ID(), dev, tol)
			}
		}
	}
	return nil
}

func (pl wirePlane) to2(p math.Point3) math.Point2 {
	d := pl.origin.VectorTo(p)
	return math.P2(d.Dot(pl.u), d.Dot(pl.v))
}

func (pl wirePlane) to3(p math.Point2) math.Point3 {
	return pl.origin.TranslateBy(pl.u.Scale(p.X).Add(pl.v.Scale(p.Y)))
}

// wireSeg kinds.
const (
	wsLine = iota
	wsArc
	wsPoly
)

// wireSeg is one oriented 2D primitive of the chain. Lines and polys are
// ground-truthed by their points; arcs by center/radius/a0/sweep (a, b derived)
// so trims and extensions stay exactly circular.
type wireSeg struct {
	kind      int
	a, b      math.Point2
	center    math.Point2
	r         float64
	a0, sweep float64 // arc: start angle and signed sweep (positive = CCW)
	poly      []math.Point2
}

// arcPoint evaluates the arc seg at fraction f of its sweep.
func (s *wireSeg) arcPoint(f float64) math.Point2 {
	ang := s.a0 + s.sweep*f
	return math.P2(s.center.X+math.Scalar(s.r*stdmath.Cos(ang)), s.center.Y+math.Scalar(s.r*stdmath.Sin(ang)))
}

// syncArcEnds refreshes the derived a/b endpoints from the arc parameters.
func (s *wireSeg) syncArcEnds() {
	s.a, s.b = s.arcPoint(0), s.arcPoint(1)
}

// startTangent and endTangent return unit travel directions at the seg ends.
func (s *wireSeg) startTangent() math.Vector2 { return s.tangentAt(0) }
func (s *wireSeg) endTangent() math.Vector2   { return s.tangentAt(1) }

func (s *wireSeg) tangentAt(f float64) math.Vector2 {
	switch s.kind {
	case wsArc:
		ang := s.a0 + s.sweep*f
		t := math.V2(math.Scalar(-stdmath.Sin(ang)), math.Scalar(stdmath.Cos(ang)))
		if s.sweep < 0 {
			t = t.Negate()
		}
		return t
	case wsPoly:
		pts := s.poly
		if f == 0 {
			return unit2(pts[0].VectorTo(pts[1]))
		}
		return unit2(pts[len(pts)-2].VectorTo(pts[len(pts)-1]))
	default:
		return unit2(s.a.VectorTo(s.b))
	}
}

func unit2(v math.Vector2) math.Vector2 {
	l := float64(v.Length())
	if l == 0 {
		return v
	}
	return v.Scale(math.Scalar(1 / l))
}

// perpLeft rotates a direction 90° counterclockwise — the positive-offset side.
func perpLeft(v math.Vector2) math.Vector2 { return math.V2(-v.Y, v.X) }

// wireSegments2D converts the wire's oriented edges into 2D segs.
func wireSegments2D(w *topo.Wire, pl wirePlane) ([]wireSeg, error) {
	uses := w.Uses()
	out := make([]wireSeg, len(uses))
	for i, u := range uses {
		seg, err := edgeSegment2D(u, pl)
		if err != nil {
			return nil, err
		}
		out[i] = seg
	}
	return out, nil
}

// edgeSegment2D maps one oriented edge to a 2D seg: lines and arcs exactly,
// anything else as a sampled polyline.
func edgeSegment2D(u topo.Use, pl wirePlane) (wireSeg, error) {
	switch c := u.Edge.Geometry().(type) {
	case geom.LineSegment:
		a, b := pl.to2(c.StartPoint), pl.to2(c.EndPoint)
		if u.Reversed {
			a, b = b, a
		}
		return wireSeg{kind: wsLine, a: a, b: b}, nil
	case geom.Arc3d:
		return arcSegment2D(c, u.Reversed, pl)
	default:
		return polySegment2D(u, pl), nil
	}
}

// arcSegment2D projects an arc into the frame, flipping sweep sign when the
// arc's own normal opposes the offset plane's.
func arcSegment2D(c geom.Arc3d, reversed bool, pl wirePlane) (wireSeg, error) {
	center := pl.to2(c.Center)
	start3 := c.PointAt(0)
	v0 := pl.to2(start3).AsVector().Sub(center.AsVector())
	a0 := stdmath.Atan2(float64(v0.Y), float64(v0.X))
	sweep := c.SweepAngle
	if float64(c.Normal.AsVector().Dot(pl.n)) < 0 {
		sweep = -sweep
	}
	if reversed {
		a0, sweep = a0+sweep, -sweep
	}
	s := wireSeg{kind: wsArc, center: center, r: c.Radius, a0: a0, sweep: sweep}
	s.syncArcEnds()
	return s, nil
}

// polySegment2D samples any other curve at offset density, oriented.
func polySegment2D(u topo.Use, pl wirePlane) wireSeg {
	const samples = 64
	c := u.Edge.Geometry()
	lo, hi := c.Domain()
	pts := make([]math.Point2, samples+1)
	for i := 0; i <= samples; i++ {
		pts[i] = pl.to2(c.PointAt(lo + (hi-lo)*float64(i)/samples))
	}
	if u.Reversed {
		for l, r := 0, len(pts)-1; l < r; l, r = l+1, r-1 {
			pts[l], pts[r] = pts[r], pts[l]
		}
	}
	return wireSeg{kind: wsPoly, a: pts[0], b: pts[len(pts)-1], poly: pts}
}

// offsetSegments offsets each seg by d to its left (positive d).
func offsetSegments(segs []wireSeg, d, tol float64) ([]wireSeg, error) {
	out := make([]wireSeg, len(segs))
	for i := range segs {
		off, err := offsetSegment(segs[i], d, tol)
		if err != nil {
			return nil, fmt.Errorf("ops.OffsetPlanarWire: segment %d: %w", i, err)
		}
		out[i] = off
	}
	return out, nil
}

// offsetSegment offsets one seg: a line translates, an arc re-radiuses
// (CCW: left is toward the center → r-d; CW: away → r+d), a polyline offsets
// per segment with miter joints.
func offsetSegment(s wireSeg, d, tol float64) (wireSeg, error) {
	switch s.kind {
	case wsLine:
		shift := perpLeft(unit2(s.a.VectorTo(s.b))).Scale(math.Scalar(d))
		return wireSeg{kind: wsLine, a: s.a.TranslateBy(shift), b: s.b.TranslateBy(shift)}, nil
	case wsArc:
		r := s.r - d
		if s.sweep < 0 {
			r = s.r + d
		}
		if r <= tol {
			return wireSeg{}, fmt.Errorf("offset %g collapses arc of radius %g", d, s.r)
		}
		off := wireSeg{kind: wsArc, center: s.center, r: r, a0: s.a0, sweep: s.sweep}
		off.syncArcEnds()
		return off, nil
	default:
		return offsetPolySeg(s, d), nil
	}
}

// offsetPolySeg offsets a polyline with miter joints (bevel past the miter
// limit, 4|d|).
func offsetPolySeg(s wireSeg, d float64) wireSeg {
	pts := s.poly
	out := make([]math.Point2, 0, len(pts))
	for i, p := range pts {
		out = append(out, p.TranslateBy(polyOffsetDir(pts, i, d)))
	}
	return wireSeg{kind: wsPoly, a: out[0], b: out[len(out)-1], poly: out}
}

// polyOffsetDir is the offset displacement at polyline vertex i: the segment
// normal at the ends, the (miter-limited) angle bisector normal inside.
func polyOffsetDir(pts []math.Point2, i int, d float64) math.Vector2 {
	last := len(pts) - 1
	switch i {
	case 0:
		return perpLeft(unit2(pts[0].VectorTo(pts[1]))).Scale(math.Scalar(d))
	case last:
		return perpLeft(unit2(pts[last-1].VectorTo(pts[last]))).Scale(math.Scalar(d))
	}
	n1 := perpLeft(unit2(pts[i-1].VectorTo(pts[i])))
	n2 := perpLeft(unit2(pts[i].VectorTo(pts[i+1])))
	bis := n1.Add(n2)
	denom := 1 + float64(n1.Dot(n2))
	if denom < 1.0/8 { // miter longer than 4|d| → bevel via plain average
		return bis.Scale(math.Scalar(d / 2))
	}
	return bis.Scale(math.Scalar(d / denom))
}
