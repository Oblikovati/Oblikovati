// SPDX-License-Identifier: GPL-2.0-only

package sketch

import "oblikovati/math"

// This file detects connected chains in a 3D sketch (M22-F09): the open paths a sweep/
// loft consumes as a rail, and the planar closed loops that bound a Profile3D. Unlike a
// 2D sketch — where entities share point pointers — 3D-sketch segments connect by
// endpoint *coincidence in space*, so chaining matches endpoints by position within a
// small tolerance.

// chainTol is the endpoint-coincidence tolerance for 3D chain detection (database units).
const chainTol = 1e-7

// seg3D is a two-endpoint 3D segment (a line or arc) participating in a chain.
type seg3D struct {
	a, b *Point3D
}

// segmentEnds3D returns the two endpoints of a chainable 3D entity (line or arc), or
// ok=false for self-closed or point-like entities.
func segmentEnds3D(e Entity) (a, b *Point3D, ok bool) {
	switch v := e.(type) {
	case *Line3D:
		return v.A, v.B, true
	case *Arc3D:
		return v.Start, v.End, true
	default:
		return nil, nil, false
	}
}

// chain3D is an ordered run of points formed by connected segments; closed reports
// whether its head and tail coincide.
type chain3D struct {
	points []*Point3D
	closed bool
}

// chainSegments collects the non-construction line/arc segments of the sketch.
func (s *Sketch3D) chainSegments() []seg3D {
	var segs []seg3D
	for _, e := range s.ents {
		if c, ok := e.(interface{ IsConstruction() bool }); ok && c.IsConstruction() {
			continue
		}
		if a, b, ok := segmentEnds3D(e); ok {
			segs = append(segs, seg3D{a: a, b: b})
		}
	}
	return segs
}

// detectChains3D returns every maximal connected chain of the segments, ordered, with
// closed loops flagged.
func detectChains3D(segs []seg3D) []chain3D {
	used := make([]bool, len(segs))
	var out []chain3D
	for i := range segs {
		if used[i] {
			continue
		}
		out = append(out, walkChain3D(segs, used, i))
	}
	return out
}

// walkChain3D grows a chain from segment i, extending both ends by endpoint coincidence.
func walkChain3D(segs []seg3D, used []bool, i int) chain3D {
	used[i] = true
	pts := []*Point3D{segs[i].a, segs[i].b}
	extendChain3D(segs, used, &pts, true)  // forward from the tail
	extendChain3D(segs, used, &pts, false) // backward from the head
	closed := len(pts) > 2 && coincident3D(pts[0], pts[len(pts)-1])
	return chain3D{points: pts, closed: closed}
}

// extendChain3D repeatedly attaches an unused segment that shares the chain's free end,
// appending (forward) or prepending (backward) the segment's far endpoint.
func extendChain3D(segs []seg3D, used []bool, pts *[]*Point3D, forward bool) {
	for {
		end := tailOf(*pts, forward)
		j, far := matchSegment3D(segs, used, end)
		if j < 0 {
			return
		}
		used[j] = true
		if forward {
			*pts = append(*pts, far)
		} else {
			*pts = append([]*Point3D{far}, *pts...)
		}
	}
}

// tailOf returns the chain's free end (last point when extending forward, first when back).
func tailOf(pts []*Point3D, forward bool) *Point3D {
	if forward {
		return pts[len(pts)-1]
	}
	return pts[0]
}

// matchSegment3D finds an unused segment with an endpoint coincident with end, returning
// its index and far endpoint (-1 when none matches).
func matchSegment3D(segs []seg3D, used []bool, end *Point3D) (int, *Point3D) {
	for j := range segs {
		if used[j] {
			continue
		}
		switch {
		case coincident3D(segs[j].a, end):
			return j, segs[j].b
		case coincident3D(segs[j].b, end):
			return j, segs[j].a
		}
	}
	return -1, nil
}

// coincident3D reports whether two points share a location within the chain tolerance.
func coincident3D(a, b *Point3D) bool {
	return a.Position().IsEqualTo(b.Position(), chainTol)
}

// Paths3D returns every maximal connected chain of line/arc segments in the sketch — the
// sweep/loft rails — as point-ordered paths (open and closed).
func (s *Sketch3D) Paths3D() []*Path3D {
	chains := detectChains3D(s.chainSegments())
	out := make([]*Path3D, 0, len(chains))
	for _, ch := range chains {
		out = append(out, NewPath3D(ch.points, ch.closed))
	}
	return out
}

// Profile3D is a closed, planar chain of a 3D sketch — a region a planar-section feature
// can consume. It carries the bounding loop's ordered points, the loop plane normal, and
// the enclosed area.
type Profile3D struct {
	points []*Point3D
	normal math.Vector3
	area   float64
}

// Points returns the profile loop's ordered vertices.
func (p *Profile3D) Points() []math.Point3 {
	out := make([]math.Point3, len(p.points))
	for i, pt := range p.points {
		out[i] = pt.Position()
	}
	return out
}

// Normal returns the loop plane's (unnormalized would be wrong) unit normal; Area returns
// the enclosed area; IsClosed always reports true (a Profile3D is a closed loop).
func (p *Profile3D) Normal() math.Vector3 { return p.normal }
func (p *Profile3D) Area() float64        { return p.area }
func (p *Profile3D) IsClosed() bool       { return true }

// Profiles3D returns the closed, planar loops of the sketch as profiles. A closed chain
// whose vertices are not coplanar (a non-planar loop) is a valid path but not a profile,
// so it is excluded here.
func (s *Sketch3D) Profiles3D() []*Profile3D {
	var out []*Profile3D
	for _, ch := range detectChains3D(s.chainSegments()) {
		if !ch.closed {
			continue
		}
		if prof, ok := profileFromLoop3D(ch.points); ok {
			out = append(out, prof)
		}
	}
	return out
}

// profileFromLoop3D builds a Profile3D from a closed loop's points when they are coplanar,
// computing the loop plane normal and enclosed area from the polygon area vector.
func profileFromLoop3D(points []*Point3D) (*Profile3D, bool) {
	// Drop the duplicated closing point so the loop is its distinct vertices.
	verts := points
	if len(verts) > 1 && coincident3D(verts[0], verts[len(verts)-1]) {
		verts = verts[:len(verts)-1]
	}
	if len(verts) < 3 {
		return nil, false
	}
	areaVec := polygonAreaVector3D(verts)
	mag := float64(areaVec.Length())
	if mag < chainTol {
		return nil, false // degenerate / collinear loop
	}
	normal := areaVec.Scale(math.Scalar(1 / mag))
	if !coplanar3D(verts, normal) {
		return nil, false
	}
	return &Profile3D{points: append([]*Point3D(nil), verts...), normal: normal, area: mag / 2}, true
}

// polygonAreaVector3D returns Σ (vᵢ × vᵢ₊₁) for the loop vertices; its magnitude is twice
// the planar area and its direction is the loop normal (right-hand rule).
func polygonAreaVector3D(verts []*Point3D) math.Vector3 {
	sum := math.V3(0, 0, 0)
	for i := range verts {
		a := verts[i].Position().AsVector()
		b := verts[(i+1)%len(verts)].Position().AsVector()
		sum = sum.Add(a.Cross(b))
	}
	return sum
}

// coplanar3D reports whether every vertex lies in the plane through the first vertex with
// the given unit normal, within the chain tolerance.
func coplanar3D(verts []*Point3D, normal math.Vector3) bool {
	origin := verts[0].Position()
	for _, v := range verts[1:] {
		if d := origin.VectorTo(v.Position()).Dot(normal); float64(d) > chainTol || float64(d) < -chainTol {
			return false
		}
	}
	return true
}
