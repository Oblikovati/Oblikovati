// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Assembling a coaxial ball-and-rod result (Oblikovati#2036, #2061). The spans file reduces the problem
// to runs of constant membership along the axis; this one turns the surviving runs into faces and welds
// them. Because that is the WHOLE classification, one engine covers every extent — the rod ending inside
// the ball, clearing it entirely, stopping part way through its shoulder, or buried whole — and every
// operation, with no case table:
//
//	∪  keep each operand's runs OUTSIDE the other, both facing out
//	∩  keep each operand's runs INSIDE the other, both facing out
//	−  keep the target's runs OUTSIDE the tool, plus the tool's runs INSIDE the target, INVERTED
//
// The one thing spans do not settle is WINDING. Every rim is shared by exactly two faces and
// ops.Validate requires them to walk it opposite ways, but which way a given face walks is not a local
// property — flipping one face forces its neighbour, and so on down the chain. Only the spherical CAPS
// are pinned: on a sphere the loop direction is what NAMES which of the two caps survives, and the
// tessellator reads exactly that (capAxis). So the caps are fixed and settleWindings propagates the
// rest, declining if a chain cannot be satisfied rather than emitting a body that measures right and
// fails orientation.

// coaxialSelection says, for one operand, which of its runs survive and whether they are inverted (the
// result's material lies on the far side of that operand's own surface — a cut's tool).
type coaxialSelection struct {
	keepInside bool
	invert     bool
}

// coaxialSelections maps an operation to what it keeps of the ball and of the rod. ballIsTarget only
// matters for a difference, which is the one asymmetric operation.
func coaxialSelections(op Op, ballIsTarget bool) (ball, rod coaxialSelection) {
	switch op {
	case Union:
		return coaxialSelection{}, coaxialSelection{}
	case Intersection:
		return coaxialSelection{keepInside: true}, coaxialSelection{keepInside: true}
	default: // Difference: the target keeps what is outside, the tool contributes what is inside, inverted
		if ballIsTarget {
			return coaxialSelection{}, coaxialSelection{keepInside: true, invert: true}
		}
		return coaxialSelection{keepInside: true, invert: true}, coaxialSelection{}
	}
}

// CoaxialSphereRodJoin returns a ∪ b for a ball and a coaxial rod — the ball stud when the rod ends
// inside the ball, a ball on a through axle when it clears both sides.
//
// Example:
//
//	ball, _ := brep.SolidSphere(math.P3(0,0,0), 0.5, "ball")
//	rod, _ := brep.SolidCylinder(math.P3(0,0,0), math.V3(0,1,0), 0.3, 1.5)
//	stud, ok := brep.CoaxialSphereRodJoin(ball, rod) // ok: one solid, 3 analytic faces
func CoaxialSphereRodJoin(a, b *topo.Body) (*topo.Body, bool) {
	return coaxialResult(a, b, Union, false)
}

// CoaxialSphereRodCut returns target − tool: the ball bored by the rod when the ball is the target, the
// rod's length outside the ball when the rod is.
func CoaxialSphereRodCut(target, tool *topo.Body) (*topo.Body, bool) {
	return coaxialResult(target, tool, Difference, true)
}

// CoaxialSphereRodIntersect returns a ∩ b: the material the two share.
func CoaxialSphereRodIntersect(a, b *topo.Body) (*topo.Body, bool) {
	return coaxialResult(a, b, Intersection, false)
}

// coaxialResult recognises the pair and assembles op's result. firstIsTarget marks the asymmetric case
// (a difference), where which body was passed first decides what is subtracted from what.
func coaxialResult(a, b *topo.Body, op Op, firstIsTarget bool) (*topo.Body, bool) {
	r, ok := coaxialRodOf(a, b)
	if !ok {
		return nil, false
	}
	ballSel, rodSel := coaxialSelections(op, !firstIsTarget || r.ball == a)
	pieces := append(r.ballPieces(ballSel), r.rodPieces(rodSel)...)
	faces, ok := settleWindings(pieces)
	if !ok {
		return nil, false
	}
	return curvedStitch(faces), true
}

// coaxialPiece is one face of the result before its winding is settled: a builder taking the flip, the
// rims it bounds with the direction it walks each when unflipped, and whether it may flip at all.
type coaxialPiece struct {
	build func(flip bool) (curvedFace, bool)
	rims  []coaxialRim
	fixed bool // a spherical cap: its winding names the surviving region
}

// coaxialRim identifies a shared circle by the axial station and radius it sits at, with the direction
// the piece walks it before any flip.
type coaxialRim struct {
	station, radius float64
	forward         bool
}

// ballPieces turns the ball's surviving runs into faces: a run reaching a pole is a CAP (whose winding
// is pinned, since it names which cap), any other run a BELT between two rims.
func (r coaxialRod) ballPieces(sel coaxialSelection) []coaxialPiece {
	var out []coaxialPiece
	R := r.sph.Radius
	for _, sp := range r.ballSpans() {
		switch {
		case sp.inside != sel.keepInside:
		case sp.lo <= -R && sp.hi >= R:
			out = append(out, r.wholeBallPiece(sel.invert))
		case sp.lo <= -R:
			out = append(out, r.ballCapPiece(sp.hi, r.out.Scale(-1), sel.invert))
		case sp.hi >= R:
			out = append(out, r.ballCapPiece(sp.lo, r.out, sel.invert))
		default:
			out = append(out, r.ballBeltPiece(sp.lo, sp.hi, sel.invert))
		}
	}
	return out
}

// wholeBallPiece is the untouched sphere — the whole ball survives when the rod removes nothing of it.
func (r coaxialRod) wholeBallPiece(invert bool) coaxialPiece {
	return coaxialPiece{fixed: true, build: func(bool) (curvedFace, bool) {
		return curvedFace{surface: r.sph, reversed: invert, lineage: r.ballLin}, true
	}}
}

// ballCapPiece is the ball's surface from the rim at `station` round to the pole on the `keep` side. Its
// loop direction is what names that cap, so it is fixed.
func (r coaxialRod) ballCapPiece(station float64, keep math.Vector3, invert bool) coaxialPiece {
	rim := r.circleAt(station, r.ballRadiusAt(station))
	forward := keep.Dot(r.out) >= 0
	return coaxialPiece{
		fixed: true,
		rims:  []coaxialRim{{station, rim.Radius, forward}},
		build: func(bool) (curvedFace, bool) {
			return sphereCapFace(r.sph, rim, keep, invert, r.ballLin), true
		},
	}
}

// ballBeltPiece is the ball's surface between two rims. Unlike a cap it needs no winding to name it —
// two distinct coaxial circles bound exactly one connected region — so it takes whatever its neighbours
// leave it.
func (r coaxialRod) ballBeltPiece(lo, hi float64, invert bool) coaxialPiece {
	loRim, hiRim := r.circleAt(lo, r.ballRadiusAt(lo)), r.circleAt(hi, r.ballRadiusAt(hi))
	return coaxialPiece{
		rims: []coaxialRim{{hi, hiRim.Radius, true}, {lo, loRim.Radius, false}},
		build: func(flip bool) (curvedFace, bool) {
			outer, inner := hiRim, loRim
			if flip {
				outer, inner = loRim, hiRim
			}
			return sphereBeltFace(r.sph, outer, inner, invert, r.ballLin), true
		},
	}
}

// rodPieces turns the rod's surviving runs into faces: its wall between two rims, and the annular part
// of each end cap that survives.
func (r coaxialRod) rodPieces(sel coaxialSelection) []coaxialPiece {
	var out []coaxialPiece
	for _, sp := range r.wallSpans() {
		if sp.inside == sel.keepInside {
			out = append(out, r.wallPiece(sp.lo, sp.hi, sel.invert))
		}
	}
	out = append(out, r.capPieces(r.sLo, r.out.Scale(-1), r.loLin, sel)...)
	return append(out, r.capPieces(r.sHi, r.out, r.hiLin, sel)...)
}

// wallPiece is the rod's cylindrical side between two stations. cylinderBandFace walks its near circle
// forwards and its far one backwards, so flipping is a swap.
func (r coaxialRod) wallPiece(lo, hi float64, invert bool) coaxialPiece {
	rc := r.cyl.Radius
	loRim, hiRim := r.circleAt(lo, rc), r.circleAt(hi, rc)
	return coaxialPiece{
		rims: []coaxialRim{{lo, rc, true}, {hi, rc, false}},
		build: func(flip bool) (curvedFace, bool) {
			near, far := loRim, hiRim
			if flip {
				near, far = hiRim, loRim
			}
			return cylinderBandFace(r.cyl, near, far, invert, r.wallLin), true
		},
	}
}

// capPieces turns one rod end cap's surviving radial runs into faces — a full disc when the run reaches
// the axis, an annulus when the ball's own circle crosses the cap and only its outside survives.
func (r coaxialRod) capPieces(station float64, outward math.Vector3, lin topo.Lineage,
	sel coaxialSelection) []coaxialPiece {
	var out []coaxialPiece
	for _, sp := range r.capSpans(station) {
		if sp.inside != sel.keepInside {
			continue
		}
		if sp.lo <= 0 {
			out = append(out, r.discPiece(station, sp.hi, outward, lin, sel.invert))
			continue
		}
		out = append(out, r.annulusPiece(station, sp.lo, sp.hi, outward, lin, sel.invert))
	}
	return out
}

// discPiece is a full planar disc closing the rod at `station`.
func (r coaxialRod) discPiece(station, radius float64, outward math.Vector3, lin topo.Lineage,
	invert bool) coaxialPiece {
	rim := r.circleAt(station, radius)
	face := func(forward bool) (curvedFace, bool) {
		return discFace(rim, forward, r.discOutward(outward, invert), lin)
	}
	return coaxialPiece{
		rims:  []coaxialRim{{station, radius, true}},
		build: func(flip bool) (curvedFace, bool) { return face(!flip) },
	}
}

// annulusPiece is the ring of a rod cap left when the ball's surface crosses it: bounded outside by the
// rod's own rim and inside by the ball's circle at that station.
func (r coaxialRod) annulusPiece(station, inner, outer float64, outward math.Vector3,
	lin topo.Lineage, invert bool) coaxialPiece {
	outerRim, innerRim := r.circleAt(station, outer), r.circleAt(station, inner)
	face := func(forward bool) (curvedFace, bool) {
		return annulusFace(outerRim, innerRim, forward, r.discOutward(outward, invert), lin)
	}
	return coaxialPiece{
		rims:  []coaxialRim{{station, outer, true}, {station, inner, false}},
		build: func(flip bool) (curvedFace, bool) { return face(!flip) },
	}
}

// discOutward is the direction a rod cap's material faces. discFace and annulusFace place the sense from
// it, so an inverted piece (a cut's tool, bounding a cavity) hands them the opposite direction.
func (r coaxialRod) discOutward(outward math.Vector3, invert bool) math.Vector3 {
	if invert {
		return outward.Scale(-1)
	}
	return outward
}

// settleWindings decides each piece's flip so that the two faces meeting on any rim walk it opposite
// ways — the invariant ops.Validate checks. The spherical caps are fixed, so it seeds from those (or,
// for a component with none, from an arbitrary piece) and propagates along the shared rims. ok=false
// when a component cannot be satisfied, which is the signal to decline rather than build a solid that
// measures right and fails orientation.
func settleWindings(pieces []coaxialPiece) ([]curvedFace, bool) {
	state := newWindingState(pieces)
	for i, p := range pieces { // the fixed pieces seed first: their direction is the one that cannot move
		if p.fixed && !state.settled[i] {
			state.propagateFrom(i)
		}
	}
	for i := range pieces {
		if !state.settled[i] {
			state.propagateFrom(i)
		}
	}
	if !state.consistent() {
		return nil, false
	}
	return state.faces()
}
