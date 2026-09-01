// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/internal/probe"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// mixedTorusRadiusTol is the largest |R−2r|/r a mixed corner's pivot radius may deviate from the
// derived 2r before the pass declines. R and r are both model lengths so the ratio is
// DIMENSIONLESS and scale-invariant (needs no bbox factor). R=2r is the rolling-ball pivot of a
// mixed-sense trihedral corner: at an orthogonal corner one arm's fillet axis sits r INSIDE a wall
// and the odd-sense arm's r OUTSIDE it, so the two spines are 2r apart (geometry-derivation §3).
// That holds for BOTH mixed signatures — 2-concave+1-convex and 1-concave+2-convex — because it is
// a statement about the two senses, not about which is the majority. A pair that fails this
// is not the box-corner torus the derivation covers (a non-orthogonal or curved mixed corner shifts
// R off 2r) so it is left to the honest-reject baseline rather than mis-modelled.
const mixedTorusRadiusTol = 1e-6

// The mixed-sense trihedral (torus) corner treatment (OCCT tests/blend/simple K9/M2/L6 and B5/C4/D7) is accumulated
// by the unified pass (fillet_corner_setback_unified.go): accumulateMixedTorus builds one corner's
// torus via buildMixedTorusCorner and emits its band retracts (railWrites), the dropped sphere vertex,
// the torus patch (extraPatches), and the synthetic host end-corner (hostEnds); weldMixedTorusFaces
// then assembles the mixed body directly. The gate + geometry helpers below (buildMixedTorusCorner,
// solveMixedTorusCorner, splitMixedSense, mixedCornerGeom, mixedTorusPatch, …) are the reused helpers
// the classifier + accumulate own; the old adoptMixedTorusCorner entrypoint was folded into that pass.

// cornerBand is one filleted edge converging on a shared trihedral corner: its slot in the fils
// slice (fi + which end), its two host faces, its constant cylinder, and its sense (concave = ef.flip).
type cornerBand struct {
	fi      int
	atC1    bool
	a, b    *topo.Face
	cyl     geom.Cylinder
	concave bool
}

// bandRetract is one band's rewritten corner cross-section: the new tangent point on each of its two
// faces (keyed by face id) and the section arc's midpoint. Applied in place on a fils copy so the band
// rail (cylinderFace), the host re-trim (filletMaps) and the torus patch all read the retracted station.
type bandRetract struct {
	fi   int
	atC1 bool
	pt   map[uint64]math.Point3
	mid  math.Point3
}

// mixedTorusCorner is the OCCT toroidal corner for one mixed-sense trihedral vertex: the shared host
// plane and the synthetic end corner that re-trims it along the torus top-contact arc, the finished
// torus patch face, and the three retracted band cross-sections.
type mixedTorusCorner struct {
	vertexID  uint64
	topFace   *topo.Face
	topCorner corner
	patch     filletFace
	bands     [3]bandRetract
}

// buildMixedTorusCorner classifies the corner at vid and, when it is the mixed-sense orthogonal-planar
// trihedral this pass owns, solves its torus treatment. Any other config (same-sense, non-orthogonal,
// curved-host, wrong valence) returns ok=false so the corner keeps its baseline sphere.
func buildMixedTorusCorner(vid uint64, cb *cornerBlend, fils []edgeFillet) (mixedTorusCorner, bool) {
	if cb == nil || cb.vertex == nil {
		return mixedTorusCorner{}, false
	}
	pivot, pair, ok := splitMixedSense(cornerBandsAt(vid, fils))
	if !ok {
		return mixedTorusCorner{}, false
	}
	faces := mixedCornerFaces(pivot, pair)
	if len(faces) != 3 || !orthogonalPlanarTriple(faces) {
		return mixedTorusCorner{}, false
	}
	return solveMixedTorusCorner(vid, cb.vertex, pivot, pair)
}

// solveMixedTorusCorner builds the torus geometry, patch face, host-plane re-trim corner and the three
// band retractions once the gate has accepted the corner. ok=false on a degenerate torus frame or a
// pivot radius off 2r (mixedCornerGeom's guard).
func solveMixedTorusCorner(vid uint64, v *topo.Vertex, pivot cornerBand, pair []cornerBand) (mixedTorusCorner, bool) {
	g, ok := mixedCornerGeom(pivot, pair)
	if !ok {
		return mixedTorusCorner{}, false
	}
	patch, ok := mixedTorusPatch(g)
	if !ok {
		return mixedTorusCorner{}, false
	}
	return mixedTorusCorner{
		vertexID: vid, topFace: g.top, topCorner: mixedTopCorner(g, v),
		patch: patch, bands: mixedBandRetracts(g, pivot),
	}, true
}

// cornerBandsAt collects every filleted edge whose blend corner meets vid — the (up to three) bands
// converging on the trihedral vertex.
func cornerBandsAt(vid uint64, fils []edgeFillet) []cornerBand {
	var out []cornerBand
	for i := range fils {
		for _, atC1 := range []bool{false, true} {
			c := fils[i].c0
			if atC1 {
				c = fils[i].c1
			}
			if c.blend && c.vertex != nil && c.vertex.ID() == vid {
				out = append(out, cornerBand{fi: i, atC1: atC1, a: fils[i].a, b: fils[i].b, cyl: fils[i].cyl, concave: fils[i].flip})
			}
		}
	}
	return out
}

// splitMixedSense partitions three converging bands into the single MINORITY-sense PIVOT band and the
// two majority-sense bands. Both mixed signatures are the same torus up to which sense is the odd one
// out — the pivot always supplies the axis and the inner-equator contact, the pair always supplies the
// two tube arcs and the shared plane the top-contact arc lies on:
//
//   - 2 concave + 1 convex (K9/M2/L6): the CONVEX band pivots, the two concave bands share the top plane;
//   - 1 concave + 2 convex (B5/C4/D7, the 270°-sector corner): the CONCAVE band pivots, the two convex
//     bands share the top plane. DRAWEXE 8.0.0 on `pcylinder s 50 100 270` + `blend 10 s_9 s_6 s_5`
//     dumps this corner as exactly geom.Torus(centre on the concave spine, axis = concave edge, R=2r,
//     r) — same surface, mirrored roles.
//
// ok=false for any valence other than 3 or a SAME-sense triple (3 concave = the void sphere octant,
// 3 convex = the run-off clip — neither is a torus).
func splitMixedSense(bands []cornerBand) (pivot cornerBand, pair []cornerBand, ok bool) {
	if len(bands) != 3 {
		return cornerBand{}, nil, false
	}
	var cvx, cc []cornerBand
	for _, b := range bands {
		if b.concave {
			cc = append(cc, b)
		} else {
			cvx = append(cvx, b)
		}
	}
	switch {
	case len(cvx) == 1 && len(cc) == 2:
		return cvx[0], cc, true
	case len(cc) == 1 && len(cvx) == 2:
		return cc[0], cvx, true
	}
	return cornerBand{}, nil, false
}

// mixedCornerFaces returns the distinct host faces of the three bands (the trihedral's three faces).
func mixedCornerFaces(pivot cornerBand, pair []cornerBand) []*topo.Face {
	seen := map[uint64]*topo.Face{}
	for _, b := range append([]cornerBand{pivot}, pair...) {
		seen[b.a.ID()], seen[b.b.ID()] = b.a, b.b
	}
	out := make([]*topo.Face, 0, len(seen))
	for _, f := range seen {
		out = append(out, f)
	}
	return out
}

// sharedBandFace returns the host face both bands touch (the shared plane the pair's torus
// top-contact arc lies on), or ok=false when they share none.
func sharedBandFace(x, y cornerBand) (*topo.Face, bool) {
	for _, fx := range []*topo.Face{x.a, x.b} {
		if fx == y.a || fx == y.b {
			return fx, true
		}
	}
	return nil, false
}

// otherBandFace returns the band's face that is not top (its wall — the face it shares with the
// pivot band).
func otherBandFace(b cornerBand, top *topo.Face) *topo.Face {
	if b.a == top {
		return b.b
	}
	return b.a
}

// closestPointsBetweenLines returns the mutually-nearest points of two lines (o1,d1) and (o2,d2), d1/d2
// unit — the common-perpendicular feet. For the torus corner: the point on the PIVOT band's fillet axis
// (p1 = centre C) and on one paired band's spine (p2) closest to each other. Assumes the lines are not
// parallel (the pivot and paired fillet axes are orthogonal at a box corner); a parallel pair divides by
// ~0 and is screened out upstream by the orthogonal-planar gate.
func closestPointsBetweenLines(o1 math.Point3, d1 math.Vector3, o2 math.Point3, d2 math.Vector3) (p1, p2 math.Point3) {
	w := o2.VectorTo(o1)
	b := d1.Dot(d2)
	den := 1 - b*b
	t := (b*d2.Dot(w) - d1.Dot(w)) / den
	s := (d2.Dot(w) - b*d1.Dot(w)) / den
	return o1.TranslateBy(d1.Scale(t)), o2.TranslateBy(d2.Scale(s))
}

// footOnLine projects p onto the line (o, d), d unit — the nearest point on the line to p.
func footOnLine(p, o math.Point3, d math.Vector3) math.Point3 {
	return o.TranslateBy(d.Scale(o.VectorTo(p).Dot(d)))
}

// dropOntoPlane returns c projected onto the plane of top along the plane's outward normal (the torus
// axis is parallel to that normal at an orthogonal corner, so this is the axis∩plane top-circle centre).
func dropOntoPlane(c math.Point3, top *topo.Face, nTop math.Vector3) math.Point3 {
	pl := top.Geometry().(geom.Plane)
	return c.TranslateBy(nTop.Scale(-pl.Origin.VectorTo(c).Dot(nTop)))
}

// arcMidOnCircle returns the on-circle midpoint of the 90° arc from→to: the circle centre pushed radius
// along the bisector of the two radial directions (the bulge point Arc3dByThreePoints needs).
func arcMidOnCircle(center, from, to math.Point3, radius float64) math.Point3 {
	bis := center.VectorTo(from).Add(center.VectorTo(to))
	return center.TranslateBy(probe.Unit(bis).Scale(radius))
}
