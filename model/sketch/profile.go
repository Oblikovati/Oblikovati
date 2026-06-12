// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"

	"oblikovati.org/math"
)

// Profiles, loops and paths are the only things a sketch exports to the feature
// engine (parametric-cad §10, modeling/00): solved geometry is turned into closed
// regions (with outer/inner loops) and connected chains. The feature engine never
// sees the solver or raw sketch entities beyond these.

// ProfileEntity is one curve in a loop or path, with the orientation in which it is
// traversed (Reversed when its natural direction opposes the traversal).
type ProfileEntity struct {
	Entity   Entity
	reversed bool
}

// Reversed reports whether the entity is traversed against its natural direction.
func (pe ProfileEntity) Reversed() bool { return pe.reversed }

// Loop is an ordered chain of entities. It is closed when its traversal returns to
// the start point.
type Loop struct {
	entities []ProfileEntity
	polygon  []math.Point2
	closed   bool
}

// Entities returns the loop's ordered entities.
func (l Loop) Entities() []ProfileEntity { return l.entities }

// IsClosed reports whether the loop forms a closed region.
func (l Loop) IsClosed() bool { return l.closed }

// Polygon returns the loop's representative polyline (arcs/circles sampled), used
// for containment tests.
func (l Loop) Polygon() []math.Point2 { return l.polygon }

// Profile is a region features extrude/revolve: an outer boundary loop and the
// inner loops (holes) it contains. An open profile (IsClosed false) has only its
// chain as the outer loop; features reject open profiles for solids but accept them
// for surfaces (enforced at the feature, not here).
type Profile struct {
	outer Loop
	inner []Loop
}

// OuterLoop returns the profile's outer boundary loop.
func (p *Profile) OuterLoop() Loop { return p.outer }

// InnerLoops returns the profile's hole loops.
func (p *Profile) InnerLoops() []Loop {
	out := make([]Loop, len(p.inner))
	copy(out, p.inner)
	return out
}

// IsClosed reports whether the profile encloses a region (its outer loop is closed).
func (p *Profile) IsClosed() bool { return p.outer.closed }

// Area returns the profile's enclosed area: the outer loop's area minus its holes' (0 for
// an open profile). Uses the loops' representative polygons (curves sampled), so it is the
// tessellated area, accurate to the sampling density.
func (p *Profile) Area() float64 {
	if !p.outer.closed {
		return 0
	}
	area := p.outer.Area()
	for _, h := range p.inner {
		area -= h.Area()
	}
	return area
}

// Area returns the loop's absolute enclosed area (shoelace over its polygon).
func (l Loop) Area() float64 { return stdmath.Abs(signedPolygonArea(l.polygon)) }

// signedPolygonArea returns the signed area of a polygon (positive for CCW winding).
func signedPolygonArea(poly []math.Point2) float64 {
	if len(poly) < 3 {
		return 0
	}
	var sum float64
	for i := range poly {
		j := (i + 1) % len(poly)
		sum += float64(poly[i].X*poly[j].Y - poly[j].X*poly[i].Y)
	}
	return sum / 2
}

// Contains reports whether the sketch-plane point q lies in the profile's region: inside
// the (closed) outer loop and outside every inner-loop hole. Used to hit-test a click on
// a profile so it can be picked for extrude/revolve. An open profile contains nothing.
func (p *Profile) Contains(q math.Point2) bool {
	if !p.outer.closed || !pointInPolygon(q, p.outer.polygon) {
		return false
	}
	for _, h := range p.inner {
		if pointInPolygon(q, h.polygon) {
			return false
		}
	}
	return true
}

// Profiles is the set of profiles detected in a sketch.
type Profiles struct {
	items []*Profile
}

// Count returns the number of profiles; Item returns the i-th.
func (ps *Profiles) Count() int          { return len(ps.items) }
func (ps *Profiles) Item(i int) *Profile { return ps.items[i] }

// All returns the detected profiles.
func (ps *Profiles) All() []*Profile {
	out := make([]*Profile, len(ps.items))
	copy(out, ps.items)
	return out
}

// Profiles detects regions in the sketch from its non-construction geometry: it builds
// a planar arrangement of the (faceted) curves and extracts the minimal closed cells —
// so a dividing curve splits a shape into several regions (arrangement.go / regions.go).
// Cells are classified into outer boundaries and holes by even–odd nesting; geometry
// that bounds no cell (a connected but unclosed chain) becomes an open profile.
func (s *Sketch) Profiles() *Profiles {
	ents := s.normalGeometry()
	loops := detectRegions(ents)
	ps := &Profiles{}
	ps.items = append(ps.items, groupRegions(loops)...)
	for _, chain := range openChainsOutside(ents, loops) {
		ps.items = append(ps.items, &Profile{outer: chain})
	}
	return ps
}

// openChainsOutside returns the open chains formed by entities that bound no detected
// region — the geometry an extrude rightly rejects but a surface may consume. Entities
// already used by a region are excluded so a dividing curve is not also reported open.
func openChainsOutside(ents []Entity, loops []Loop) []Loop {
	used := map[Entity]bool{}
	for _, l := range loops {
		for _, pe := range l.entities {
			used[pe.Entity] = true
		}
	}
	var leftover []Entity
	for _, e := range ents {
		if !used[e] {
			leftover = append(leftover, e)
		}
	}
	_, open := detectLoops(leftover)
	return open
}

// normalGeometry returns the sketch's non-construction curve entities.
func (s *Sketch) normalGeometry() []Entity {
	var out []Entity
	for _, e := range s.ents {
		if cg, ok := e.(interface{ IsConstruction() bool }); ok && cg.IsConstruction() {
			continue
		}
		out = append(out, e)
	}
	return out
}

// groupRegions builds profiles from closed loops by even–odd nesting: a loop
// contained in an even number of others is an outer boundary; loops one level
// deeper inside it are its holes.
func groupRegions(loops []Loop) []*Profile {
	depth := make([]int, len(loops))
	for i := range loops {
		for j := range loops {
			if i != j && containsLoop(loops[j], loops[i]) {
				depth[i]++
			}
		}
	}
	var profiles []*Profile
	for i, l := range loops {
		if depth[i]%2 != 0 {
			continue // a hole; attached to its outer loop below
		}
		profiles = append(profiles, &Profile{outer: l, inner: holesOf(i, loops, depth)})
	}
	return profiles
}

// holesOf returns the loops one nesting level inside outer (index oi) — its holes.
func holesOf(oi int, loops []Loop, depth []int) []Loop {
	var holes []Loop
	for j, l := range loops {
		if j != oi && depth[j] == depth[oi]+1 && containsLoop(loops[oi], l) {
			holes = append(holes, l)
		}
	}
	return holes
}

// containsLoop reports whether outer contains inner: every vertex of inner lies
// inside outer's polygon. (Testing all vertices, not just the centroid, avoids a
// false positive when a hole is centered on the region's centroid.)
func containsLoop(outer, inner Loop) bool {
	if len(inner.polygon) == 0 {
		return false
	}
	for _, v := range inner.polygon {
		if !pointInPolygon(v, outer.polygon) {
			return false
		}
	}
	return true
}

// pointInPolygon is the even–odd ray-casting test.
func pointInPolygon(p math.Point2, poly []math.Point2) bool {
	inside := false
	n := len(poly)
	for i, j := 0, n-1; i < n; j, i = i, i+1 {
		yi, yj := poly[i].Y, poly[j].Y
		if (yi > p.Y) != (yj > p.Y) {
			x := poly[i].X + (p.Y-yi)/(yj-yi)*(poly[j].X-poly[i].X)
			if p.X < x {
				inside = !inside
			}
		}
	}
	return inside
}

// circleSamples is how many points a closed curve is sampled into for containment.
const circleSamples = 24

// sampleCircle returns a polygon approximating a circle.
func sampleCircle(c *Circle) []math.Point2 {
	return sampleCircleN(c, circleSamples)
}

// sampleCircleN is sampleCircle at caller-chosen density (region properties
// scale it with the requested accuracy — M06-F08, #623).
func sampleCircleN(c *Circle, n int) []math.Point2 {
	pts := make([]math.Point2, n)
	for i := range pts {
		a := 2 * stdmath.Pi * float64(i) / float64(n)
		pts[i] = math.P2(c.Center.X+c.Radius*stdmath.Cos(a), c.Center.Y+c.Radius*stdmath.Sin(a))
	}
	return pts
}
