// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"slices"

	"oblikovati.org/math"
)

// chainJoinTol is the distance under which two curve endpoints count as joined when tracing the
// connected chain an offset Loop Select follows.
const chainJoinTol = 1e-6 // tol:calibrated — offset chain endpoint connectivity

// curveEnds returns a curve entity's two endpoints (in sketch space) and whether it is inherently
// closed (a circle, or a projected/closed curve that returns to its start). ok is false for a
// non-curve entity. Projected curves are included, so Loop Select works on a projected perimeter
// that Paths() (which reads only native line/arc geometry) does not report (#2158 follow-up).
func curveEnds(e Entity) (a, b math.Point2, closed, ok bool) {
	switch t := e.(type) {
	case *Line:
		return t.A.Position(), t.B.Position(), false, true
	case *Arc:
		return t.Start.Position(), t.End.Position(), false, true
	case *Circle:
		return t.Center.Position(), t.Center.Position(), true, true
	case *ProjectedCurve:
		p := t.RenderPolyline()
		if len(p) < 2 {
			return math.Point2{}, math.Point2{}, false, false
		}
		return p[0], p[len(p)-1], polylineReturnsToStart(p), true
	case *Spline:
		if len(t.Points) >= 2 {
			return t.Points[0].Position(), t.Points[len(t.Points)-1].Position(), t.Closed, true
		}
	}
	return math.Point2{}, math.Point2{}, false, false
}

// ConnectedChainFrom returns the maximal connected chain of curves reachable from seed by shared
// endpoints, as an ordered Path with per-entity traversal direction — the loop Inventor's Offset
// selects with Loop Select. A single inherently-closed curve (a circle, a closed projection) is its
// own one-element closed chain. ok is false when seed is not a curve.
func (s *Sketch) ConnectedChainFrom(seed Entity) (*Path, bool) {
	sa, sb, sclosed, ok := curveEnds(seed)
	if !ok {
		return nil, false
	}
	if sclosed {
		return &Path{entities: []ProfileEntity{{Entity: seed}}, closed: true}, true
	}
	ends := map[Entity][2]math.Point2{}
	var curves []Entity
	for _, e := range s.ents {
		if e == seed {
			continue
		}
		if a, b, c, ok := curveEnds(e); ok && !c {
			curves = append(curves, e)
			ends[e] = [2]math.Point2{a, b}
		}
	}
	visited := map[Entity]bool{seed: true}
	fwd := traceChain(sb, curves, ends, visited) // curves after the seed, in order
	bwd := traceChain(sa, curves, ends, visited) // curves before the seed, walked outward from sa
	return assembleChain(seed, sa, sb, fwd, bwd), true
}

// traceChain traces curves outward from `start`, each oriented so its traversal start meets the
// running free end, stopping when no unvisited curve continues the chain.
func traceChain(start math.Point2, curves []Entity, ends map[Entity][2]math.Point2, visited map[Entity]bool) []ProfileEntity {
	var chain []ProfileEntity
	cur := start
	for {
		found, reversed, next := nextInChain(cur, curves, ends, visited)
		if found == nil {
			return chain
		}
		visited[found] = true
		chain = append(chain, ProfileEntity{Entity: found, reversed: reversed})
		cur = next
	}
}

// nextInChain finds the first unvisited curve with an endpoint at cur, returning it, whether it is
// traversed reversed (its natural end is at cur), and the free end that continues the chain.
func nextInChain(cur math.Point2, curves []Entity, ends map[Entity][2]math.Point2, visited map[Entity]bool) (Entity, bool, math.Point2) {
	for _, e := range curves {
		if visited[e] {
			continue
		}
		ab := ends[e]
		if ab[0].IsEqualTo(cur, chainJoinTol) {
			return e, false, ab[1]
		}
		if ab[1].IsEqualTo(cur, chainJoinTol) {
			return e, true, ab[0]
		}
	}
	return nil, false, math.Point2{}
}

// assembleChain stitches the backward chain (reversed, its entities flipped so they flow toward the
// seed), the seed, and the forward chain into one ordered Path, marking it closed when the chain's
// two free ends meet.
func assembleChain(seed Entity, sa, sb math.Point2, fwd, bwd []ProfileEntity) *Path {
	entities := make([]ProfileEntity, 0, len(bwd)+1+len(fwd))
	for _, b := range slices.Backward(bwd) {
		entities = append(entities, ProfileEntity{Entity: b.Entity, reversed: !b.reversed})
	}
	entities = append(entities, ProfileEntity{Entity: seed})
	entities = append(entities, fwd...)
	startPt := sa
	if len(bwd) > 0 {
		startPt = traversalStart(bwd[len(bwd)-1], true)
	}
	endPt := sb
	if len(fwd) > 0 {
		endPt = traversalEnd(fwd[len(fwd)-1])
	}
	return &Path{entities: entities, closed: len(entities) > 1 && startPt.IsEqualTo(endPt, chainJoinTol)}
}

// traversalStart / traversalEnd give a profile entity's start/end point honouring its direction.
// flip additionally inverts the reversed flag (used for a backward-walked entity being flipped).
func traversalStart(pe ProfileEntity, flip bool) math.Point2 {
	a, b, _, _ := curveEnds(pe.Entity)
	if pe.reversed != flip {
		return b
	}
	return a
}

func traversalEnd(pe ProfileEntity) math.Point2 {
	a, b, _, _ := curveEnds(pe.Entity)
	if pe.reversed {
		return a
	}
	return b
}
