// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The solids a coaxial ball-and-rod pair bounds (Oblikovati#2036, #2061). Each is assembled from four
// analytic face constructors — a spherical CAP, a spherical BELT, a cylindrical BAND and a planar DISC —
// because the seam circles split each operand's surface and the boolean is only a choice of which parts
// to keep:
//
//	                     rod ENDS in the ball (one seam)     rod passes THROUGH (two seams)
//	∪ join     the ball stud: far cap + stub + tip     belt + a stub and tip at each end
//	− ball−rod a blind spherical bore: far cap +       a bead: belt + the bore wall between
//	           inverted wall + flat pocket bottom      the seams, inverted
//	− rod−ball the dimpled stub: near cap inverted     two separate stubs, one per end, each
//	           + stub + tip                            with its own dimple
//	∩          the plug: near cap + wall + bottom      the core: a cap at each end + the wall
//
// WINDING. Every shared circle is walked by exactly two of these faces, and ops.Validate requires them
// to walk it opposite ways. Only the spherical CAP's direction carries meaning — on a sphere the loop is
// what NAMES which of the two caps survives, and the tessellator reads exactly that (capAxis). A BELT
// needs no such naming: two distinct coaxial circles bound exactly one connected region, since the
// complement is two disjoint caps. So the sphere face fixes the seam directions and the bands and discs
// follow, carrying their material side in the face sense alone. Getting it backwards costs nothing at
// build time and produces a closed, manifold solid of the right volume that ops.Validate rejects on
// orientation.

// CoaxialSphereRodJoin returns a ∪ b for a ball and a coaxial rod — the ball stud when the rod ends
// inside the ball, a ball on a through axle when it clears both sides.
//
// Example:
//
//	ball, _ := brep.SolidSphere(math.P3(0,0,0), 0.5, "ball")
//	rod, _ := brep.SolidCylinder(math.P3(0,0,0), math.V3(0,1,0), 0.3, 1.5)
//	stud, ok := brep.CoaxialSphereRodJoin(ball, rod) // ok: one solid, 3 analytic faces
func CoaxialSphereRodJoin(a, b *topo.Body) (*topo.Body, bool) {
	r, ok := coaxialRodOf(a, b)
	if !ok {
		return nil, false
	}
	if r.through {
		return r.stitch(r.belt(false), r.hiStub(true), r.loStub())
	}
	// The far cap walks the hi seam backwards, so the stub's wall takes it forwards.
	return r.stitch([]curvedFace{r.cap(r.hiSeam, r.out.Scale(-1), false)}, r.hiStub(false))
}

// CoaxialSphereRodCut returns target − tool: the ball bored by the rod when the ball is the target, the
// rod's free length hollowed by the ball when the rod is.
func CoaxialSphereRodCut(target, tool *topo.Body) (*topo.Body, bool) {
	r, ok := coaxialRodOf(target, tool)
	if !ok {
		return nil, false
	}
	if r.ball == target {
		return r.boredBall()
	}
	return r.freeStubs()
}

// boredBall is ball − rod: the ball's surface outside the rod, plus the rod's buried wall turned inward
// as the cavity. A rod that ENDS inside also contributes its buried cap as the flat pocket bottom, whose
// outward normal points along +out into the material that was removed; a rod passing THROUGH leaves an
// open bore at both ends and contributes no cap at all.
func (r coaxialRod) boredBall() (*topo.Body, bool) {
	if r.through {
		return r.stitch(r.belt(false), []curvedFace{r.band(r.loSeam, r.hiSeam, true)})
	}
	bottom, ok := discFace(r.loEnd, true, r.out, r.loLin)
	if !ok {
		return nil, false
	}
	// The far cap walks the hi seam backwards, so the bore wall takes it forwards.
	return r.stitch([]curvedFace{r.cap(r.hiSeam, r.out.Scale(-1), false)},
		[]curvedFace{r.band(r.hiSeam, r.loEnd, true), bottom})
}

// freeStubs is rod − ball: the rod's length outside the ball, its base hollowed by the ball's own
// surface. The cap is INVERTED — the material now sits outside the ball, so the sphere's outward radial
// normal points into the solid. A rod passing through leaves two separate stubs, one per end.
func (r coaxialRod) freeStubs() (*topo.Body, bool) {
	hi := r.hiStub(true)
	hi = append(hi, r.cap(r.hiSeam, r.out, true))
	if !r.through {
		return r.stitch(hi)
	}
	lo := r.loStub()
	return r.stitch(hi, append(lo, r.cap(r.loSeam, r.out.Scale(-1), true)))
}

// CoaxialSphereRodIntersect returns a ∩ b: the material the two share. For a rod that ends inside the
// ball that is the plug — the buried length capped by the ball's dome, carrying the rod's own PLANAR
// buried cap. For a rod passing through it is the core between the two seams, domed at both ends.
func CoaxialSphereRodIntersect(a, b *topo.Body) (*topo.Body, bool) {
	r, ok := coaxialRodOf(a, b)
	if !ok {
		return nil, false
	}
	if r.through {
		return r.stitch([]curvedFace{
			r.cap(r.hiSeam, r.out, false),
			r.cap(r.loSeam, r.out.Scale(-1), false),
			r.band(r.loSeam, r.hiSeam, false),
		})
	}
	bottom, ok := discFace(r.loEnd, false, r.out.Scale(-1), r.loLin)
	if !ok {
		return nil, false
	}
	// The near cap walks the hi seam forwards, so the plug's band takes it backwards.
	return r.stitch([]curvedFace{r.cap(r.hiSeam, r.out, false), r.band(r.loEnd, r.hiSeam, false), bottom})
}

// hiStub is the rod's free length beyond the hi seam: the wall and its end disc. capForward says which
// way the SPHERE face on the other side of that seam walks it; the wall takes the opposite direction,
// and the disc in turn opposes the wall on the rod's own rim. That one bit is the whole winding chain.
func (r coaxialRod) hiStub(capForward bool) []curvedFace {
	near, far := r.hiEnd, r.hiSeam // the cap walked the seam forwards: the wall walks it backwards
	if !capForward {
		near, far = r.hiSeam, r.hiEnd
	}
	tip, ok := discFace(r.hiEnd, !capForward, r.out, r.hiLin)
	if !ok {
		return nil
	}
	return []curvedFace{r.band(near, far, false), tip}
}

// loStub is the mirror at the −out end, present only when the rod passes through the ball. Every sphere
// face that meets the lo seam walks it backwards (the belt by convention, a dimple because it keeps the
// −out cap), so the wall's direction here is fixed and needs no parameter.
func (r coaxialRod) loStub() []curvedFace {
	tip, ok := discFace(r.loEnd, true, r.out.Scale(-1), r.loLin)
	if !ok {
		return nil
	}
	return []curvedFace{r.band(r.loSeam, r.loEnd, false), tip}
}

// belt is the ball's surviving surface when the rod passes through: the zone between the two seams,
// walking the hi seam forwards and the lo seam backwards — the directions every neighbour opposes.
func (r coaxialRod) belt(inverted bool) []curvedFace {
	return []curvedFace{sphereBeltFace(r.sph, r.hiSeam, r.loSeam, inverted, r.ballLin)}
}

// cap is the ball's surviving surface on one side of a seam: the cap toward keep, inverted into a
// dimple when the result's material lies outside the ball.
func (r coaxialRod) cap(seam geom.Circle, keep math.Vector3, inverted bool) curvedFace {
	return sphereCapFace(r.sph, seam, keep, inverted, r.ballLin)
}

// band is the rod's cylindrical wall between two of its coaxial circles. near is walked forwards and far
// backwards (cylinderBandFace's convention), which is how a caller picks the winding a neighbour leaves
// it; inverted turns the wall into a bore.
func (r coaxialRod) band(near, far geom.Circle, inverted bool) curvedFace {
	return cylinderBandFace(r.cyl, near, far, inverted, r.wallLin)
}

// stitch welds the given face groups into one body, declining when any group came back empty — a
// constructor that failed rather than a result with nothing to contribute.
func (r coaxialRod) stitch(groups ...[]curvedFace) (*topo.Body, bool) {
	var faces []curvedFace
	for _, g := range groups {
		if len(g) == 0 {
			return nil, false
		}
		faces = append(faces, g...)
	}
	return curvedStitch(faces), true
}
