// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
)

// Settling the loop directions of a coaxial ball-and-rod result (Oblikovati#2036, #2061). Every rim in
// such a result is a circle shared by exactly two faces, and ops.Validate requires them to walk it
// opposite ways. That is not decidable face by face: a face walks all of ITS rims one way or the other
// together (flipping a band swaps both its circles), so one choice propagates down the whole chain.
//
// The chain is seeded by the SPHERE faces, whose direction is not free — on a closed surface the loop is
// what NAMES the region, and kernel/ops reads exactly that (capAxis in sphere_cap_mesh.go) to pick the
// pole it fans toward. Caps and belts alike are therefore fixed (Oblikovati/Oblikovati#3447: a belt
// walked the other way names the sphere's complement of it, the two disjoint caps). Bands, discs and
// annuli lie on OPEN surfaces, whose bounded trim is the region either way, so they take whatever the
// sphere faces leave them.
//
// Getting this wrong is cheap to miss and expensive to ship: the result is still a closed, manifold solid
// of the RIGHT volume, and only ops.Validate's per-edge orientation check catches it.

// windingState is the propagation in progress: one flip bit per piece, plus which pieces are settled.
type windingState struct {
	pieces  []coaxialPiece
	flip    []bool
	settled []bool
	byRim   map[coaxialRimKey][]int // the piece indices meeting on each rim
}

// coaxialRimKey identifies a shared rim. Stations and radii are compared at a quantised precision, so
// two independently computed copies of the same circle land in the same bucket.
type coaxialRimKey struct {
	station, radius int64
}

// rimKeyScale quantises a rim's station and radius into an integer key. It is deliberately coarse
// relative to double precision and fine relative to any real geometry: the two faces meeting on a rim
// compute it from the same span endpoint, so they agree to the last bit or not at all.
const rimKeyScale = 1e9

func newWindingState(pieces []coaxialPiece) *windingState {
	s := &windingState{pieces: pieces, flip: make([]bool, len(pieces)),
		settled: make([]bool, len(pieces)), byRim: map[coaxialRimKey][]int{}}
	for i, p := range pieces {
		for _, rim := range p.rims {
			k := rimKey(rim)
			s.byRim[k] = append(s.byRim[k], i)
		}
	}
	return s
}

func rimKey(rim coaxialRim) coaxialRimKey {
	return coaxialRimKey{
		station: int64(stdmath.Round(rim.station * rimKeyScale)),
		radius:  int64(stdmath.Round(rim.radius * rimKeyScale)),
	}
}

// propagateFrom settles the component containing seed, breadth-first: each unsettled neighbour takes
// whichever flip makes it oppose the piece it was reached from. A fixed piece (a spherical cap) never
// flips; if the seed is unsettled and free, it starts unflipped.
func (s *windingState) propagateFrom(seed int) {
	s.settled[seed] = true
	queue := []int{seed}
	for len(queue) > 0 {
		i := queue[0]
		queue = queue[1:]
		for _, rim := range s.pieces[i].rims {
			for _, j := range s.byRim[rimKey(rim)] {
				if j == i || s.settled[j] {
					continue
				}
				s.flip[j] = s.walk(i, rim) == s.baseWalk(j, rim)
				s.settled[j] = true
				queue = append(queue, j)
			}
		}
	}
}

// walk is the direction piece i actually walks the given rim, once its flip is applied.
func (s *windingState) walk(i int, rim coaxialRim) bool {
	return s.baseWalk(i, rim) != s.flip[i]
}

// baseWalk is the direction piece i walks the rim before any flip. A piece that does not carry the rim
// reports false; callers only ask about rims the piece has.
func (s *windingState) baseWalk(i int, rim coaxialRim) bool {
	want := rimKey(rim)
	for _, r := range s.pieces[i].rims {
		if rimKey(r) == want {
			return r.forward
		}
	}
	return false
}

// consistent reports whether every rim is now walked once each way, and whether propagation respected
// the fixed pieces. A rim used by other than two pieces is not this family's topology at all.
func (s *windingState) consistent() bool {
	for i, p := range s.pieces {
		if p.fixed && s.flip[i] {
			return false // a cap was flipped: its loop would name the other cap
		}
	}
	for _, at := range s.byRim {
		if len(at) != 2 {
			return false
		}
		if s.walkOn(at[0], at[1]) {
			return false
		}
	}
	return true
}

// walkOn reports whether the two pieces walk their shared rim the SAME way — the violation.
func (s *windingState) walkOn(i, j int) bool {
	for _, rim := range s.pieces[i].rims {
		for _, other := range s.pieces[j].rims {
			if rimKey(rim) == rimKey(other) {
				return s.walk(i, rim) == s.walk(j, other)
			}
		}
	}
	return false
}

// faces builds every piece at its settled flip.
func (s *windingState) faces() ([]curvedFace, bool) {
	out := make([]curvedFace, 0, len(s.pieces))
	for i, p := range s.pieces {
		f, ok := p.build(s.flip[i])
		if !ok {
			return nil, false
		}
		out = append(out, f)
	}
	return out, true
}
