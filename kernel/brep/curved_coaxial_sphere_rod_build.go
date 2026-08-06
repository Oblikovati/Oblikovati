// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The four solids a coaxial ball-and-rod pair bounds (Oblikovati#2036). Every one of them is THREE
// analytic faces off the frame curved_coaxial_sphere_rod.go recognised — a spherical cap, a
// cylindrical band, and a planar disc — because the single seam circle splits each operand's surface
// in two and the boolean is only a choice of which halves to keep:
//
//	           ball cap kept | rod band       | disc          | inverted faces
//	∪ join     far  (−out)   | seam → free    | free,  +out   | none
//	− ball−rod far  (−out)   | buried → seam  | buried, +out  | the band, turned into the bore
//	− rod−ball near (+out)   | seam → free    | free,  +out   | the ball cap, turned into a dimple
//	∩ intersect near (+out)  | buried → seam  | buried, −out  | none
//
// WINDING. The seam circle is shared by the ball cap and the band, and the rod rim by the band and the
// disc, so one decision fixes all three: ops.Validate requires an edge's two uses to run opposite ways.
// Only the ball cap's direction carries meaning — on a sphere the loop is what NAMES which cap survives
// and the tessellator reads exactly that (capAxis) — while a cylindrical band and a planar disc cover
// the same region either way. So the cap chooses, the other two follow, and which side holds material
// is carried by the face sense alone. Getting this backwards costs nothing at build time and produces a
// closed, manifold solid of the right volume that ops.Validate rejects on orientation.

// CoaxialSphereRodJoin returns a ∪ b for a ball and a coaxial rod that ends inside it — the ball stud.
// The result is the ball minus the cap the rod covers, the rod's free stub, and the stub's end cap.
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
	return r.assemble(rodResult{keepNearCap: false, rim: r.free, rimLin: r.freeLin, outward: r.out})
}

// CoaxialSphereRodCut returns target − tool for a ball and a coaxial rod that ends inside it: a blind
// spherical bore when the ball is the target, the dimpled free stub when the rod is.
func CoaxialSphereRodCut(target, tool *topo.Body) (*topo.Body, bool) {
	r, ok := coaxialRodOf(target, tool)
	if !ok {
		return nil, false
	}
	if r.ball == target {
		// ball − rod: the ball's surface outside the rod, the rod's buried wall turned inward as the
		// bore, and the rod's buried cap as the flat pocket bottom — whose outward normal points along
		// +out, into the material that was removed.
		return r.assemble(rodResult{rim: r.buried, rimLin: r.buriedLin, outward: r.out, bandInverted: true})
	}
	// rod − ball: the free stub, its base hollowed by the ball's own surface. The ball cap is the NEAR
	// one and inverted — the material now sits outside the ball, so the sphere's outward radial normal
	// points into the solid.
	return r.assemble(rodResult{keepNearCap: true, capInverted: true,
		rim: r.free, rimLin: r.freeLin, outward: r.out})
}

// CoaxialSphereRodIntersect returns a ∩ b: the plug, the buried length of the rod capped by the ball's
// own dome. Unlike a full crossing's intersect it carries the rod's PLANAR buried cap, which lies
// inside the ball and so survives into the common material.
func CoaxialSphereRodIntersect(a, b *topo.Body) (*topo.Body, bool) {
	r, ok := coaxialRodOf(a, b)
	if !ok {
		return nil, false
	}
	return r.assemble(rodResult{keepNearCap: true,
		rim: r.buried, rimLin: r.buriedLin, outward: r.out.Scale(-1)})
}

// rodResult is one of the four results as a choice of halves: which ball cap survives, which rod rim
// the band runs to, which way that rim's disc faces, and which of the two faces is inverted (its
// material on the far side of its own surface).
type rodResult struct {
	keepNearCap  bool         // keep the small cap the stub protrudes toward, not the rest of the ball
	capInverted  bool         // the ball cap is a dimple: material outside the ball
	bandInverted bool         // the rod wall is a bore: material outside the cylinder
	rim          geom.Circle  // the rod rim the band runs to from the seam
	rimLin       topo.Lineage // that rim's cap face, so its reference key survives (ADR-0043 K1a)
	outward      math.Vector3 // which way the disc closing rim faces out of the solid
}

// assemble welds the three faces. capForward is the pivot: it is the direction the ball cap walks the
// seam circle, so the band must take the seam the other way (hence the near/far swap), and the disc
// must in turn oppose the band on the rim — which lands on capForward again, since the band walks its
// near circle forward and its far circle backward.
func (r coaxialRod) assemble(res rodResult) (*topo.Body, bool) {
	keep := r.out
	if !res.keepNearCap {
		keep = r.out.Scale(-1)
	}
	near, far := r.seam, res.rim
	if res.keepNearCap { // the cap already walked the seam forward
		near, far = res.rim, r.seam
	}
	disc, ok := discFace(res.rim, !res.keepNearCap, res.outward, res.rimLin)
	if !ok {
		return nil, false
	}
	return curvedStitch([]curvedFace{
		sphereCapFace(r.sph, r.seam, keep, res.capInverted, r.ballLin),
		cylinderBandFace(r.cyl, near, far, res.bandInverted, r.wallLin),
		disc,
	}), true
}
