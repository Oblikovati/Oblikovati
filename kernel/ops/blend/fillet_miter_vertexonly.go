// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Vertex-only miter (OCCT tests/blend tolblend_simple/D4): two equal-radius convex arms meeting at
// a 4-valent planar vertex WITHOUT a shared face (opposite pyramid edges). The two rolling-ball
// cylinders still trim each other — their axes both pass through the corner's equidistant point, so
// the intersection is the ellipse arc in the arms' mirror plane — but the seam's ends lie on the
// two SHARP edges (each shared by one host of each arm), not on a shared face. Verified against
// DRAWEXE on D4: the band end arc lies on BOTH cylinders (|d−r| < 2e-5·r at every station) and its
// endpoints split the sharp edges exactly where the host rails end.

// solveVertexOnlyMiter builds the seam where two filleted edges meet ONLY at vertex v: each sharp
// edge at v yields one seam endpoint (its crossing with both cylinders — certified against cylA,
// cylB, and the host rails), and the seam samples cylA ∩ cylB between them. Declines honestly when
// the corner is not the symmetric configuration whose seam closes on the sharp edges.
func solveVertexOnlyMiter(v *topo.Vertex, ps []filletPick, r float64) (*cornerMiter, error) {
	c0, c1, err := vertexOnlyArms(v, ps, r)
	if err != nil {
		return nil, err
	}
	sharps := sharpEdgesAt(v, ps)
	if len(sharps) != 2 {
		return nil, fmt.Errorf("fillet: two filleted edges meeting only at a %d-edge vertex need exactly 2 sharp edges for a vertex-only miter (got %d)", len(v.Edges()), len(sharps))
	}
	weld := seamWeld(r)
	p0, err := seamEndOnSharpEdge(v, sharps[0], c0, c1, weld)
	if err != nil {
		return nil, err
	}
	p1, err := seamEndOnSharpEdge(v, sharps[1], c0, c1, weld)
	if err != nil {
		return nil, err
	}
	seam, err := sampleVertexOnlySeam(c0, c1, p0, p1)
	if err != nil {
		return nil, err
	}
	return &cornerMiter{vertex: v, sBot: seam[len(seam)-1], seam: seam}, nil
}

// vertexOnlyArms solves both arms' rolling-ball cylinders at v from their own host-face pairs (no
// shared face exists, so each arm's frame comes from its two planes alone).
func vertexOnlyArms(v *topo.Vertex, ps []filletPick, r float64) (miterCyl, miterCyl, error) {
	c0, err := vertexOnlyArmCyl(v, ps[0], r)
	if err != nil {
		return miterCyl{}, miterCyl{}, err
	}
	c1, err := vertexOnlyArmCyl(v, ps[1], r)
	return c0, c1, err
}

// vertexOnlyArmCyl is one arm's rolling-ball cylinder: axis point v + offDir·r from the arm's two
// planar hosts, axis along the edge. nF is unused by the vertex-only seam (no outer-face terminus)
// and left zero.
func vertexOnlyArmCyl(v *topo.Vertex, p filletPick, r float64) (miterCyl, error) {
	faces := p.edge.Faces()
	if len(faces) != 2 {
		return miterCyl{}, fmt.Errorf("fillet: vertex-only miter edge %d bounds %d faces, need 2", p.edge.ID(), len(faces))
	}
	nA, okA := planeNormal(faces[0])
	nB, okB := planeNormal(faces[1])
	if !okA || !okB {
		return miterCyl{}, fmt.Errorf("fillet: vertex-only miter arm faces must be planar")
	}
	axis, err := math.UnitVector3FromVector(p.edge.StartVertex().Point().VectorTo(p.edge.EndVertex().Point()))
	if err != nil {
		return miterCyl{}, fmt.Errorf("fillet: degenerate vertex-only miter edge")
	}
	offDir := nA.Add(nB).Scale(-1 / (1 + nA.Dot(nB)))
	return miterCyl{cen: v.Point().TranslateBy(offDir.Scale(r)), axis: axis.AsVector(), r: r}, nil
}

// sharpEdgesAt returns the edges at v that are NOT picked (the corner's surviving sharp edges).
func sharpEdgesAt(v *topo.Vertex, ps []filletPick) []*topo.Edge {
	picked := map[uint64]bool{}
	for _, p := range ps {
		picked[p.edge.ID()] = true
	}
	var out []*topo.Edge
	for _, e := range v.Edges() {
		if !picked[e.ID()] {
			out = append(out, e)
		}
	}
	return out
}

// seamWeld is the vertex-only miter's coincidence tolerance: model-scaled from the corner radius
// (the seam geometry's own length scale), never a bare epsilon (ADR-0042).
func seamWeld(r float64) float64 {
	return 1e-6 * r
}

// seamEndOnSharpEdge finds the seam endpoint on one sharp edge: the edge's near-vertex crossing
// with cylinder cA, certified to lie on cylinder cB too (the seam end is a point of BOTH arm
// cylinders). ok=false when the edge misses a cylinder or the crossing is not mutual — the corner
// is not the symmetric vertex-only configuration and the miter declines.
func seamEndOnSharpEdge(v *topo.Vertex, e *topo.Edge, cA, cB miterCyl, weld float64) (math.Point3, error) {
	from, to := e.StartVertex().Point(), e.EndVertex().Point()
	dir := from.VectorTo(to)
	roots := grazeTolerantCylRoots(cA, from, dir, weld)
	p, found := math.Point3{}, false
	bestD := stdmath.Inf(1)
	for _, t := range roots {
		if t < -1e-9 || t > 1+1e-9 {
			continue
		}
		q := from.TranslateBy(dir.Scale(t))
		if d := float64(q.DistanceTo(v.Point())); d < bestD {
			p, bestD, found = q, d, true
		}
	}
	if !found {
		return math.Point3{}, fmt.Errorf("fillet: vertex-only miter — sharp edge %d does not cross the fillet cylinder", e.ID())
	}
	if off := stdmath.Abs(distToCylAxis(cB, p) - cB.r); off > weld {
		return math.Point3{}, fmt.Errorf("fillet: vertex-only miter — seam end %v is off the second cylinder by %g (weld %g); the corner is not the mutual-trim configuration", p, off, weld)
	}
	return p, nil
}

// grazeTolerantCylRoots is cylLineRoots plus the tangency case: at the symmetric vertex-only corner
// the sharp edge GRAZES the cylinder (line-line distance exactly r), so the discriminant sits at 0
// and float noise flips it negative, dropping the double root. When no real root exists, the
// closest-approach parameter is accepted iff the line actually reaches within weld of the surface
// there — a certified graze, never a fabricated crossing.
func grazeTolerantCylRoots(c miterCyl, lp math.Point3, ld math.Vector3, weld float64) []float64 {
	if roots := cylLineRoots(c, lp, ld); len(roots) > 0 {
		return roots
	}
	e := c.cen.VectorTo(lp)
	la, ea := ld.Dot(c.axis), e.Dot(c.axis)
	a := ld.LengthSquared() - la*la
	b := 2 * (e.Dot(ld) - ea*la)
	if a <= 0 {
		return nil // axis-parallel line: no graze point either
	}
	t := -b / (2 * a)
	p := lp.TranslateBy(ld.Scale(t))
	if stdmath.Abs(distToCylAxis(c, p)-c.r) > weld {
		return nil
	}
	return []float64{t}
}

// distToCylAxis is the perpendicular distance from p to cylinder c's axis line.
func distToCylAxis(c miterCyl, p math.Point3) float64 {
	w := c.cen.VectorTo(p)
	return w.Sub(c.axis.Scale(w.Dot(c.axis))).Length()
}

// sampleVertexOnlySeam samples cylA ∩ cylB from p0 to p1 by walking cylA's contact direction and
// solving the axial station on cylB at each step — the sampleAsymmetricMiterSeam pattern with both
// ends pinned to the sharp-edge crossings (every interior station lies on BOTH cylinders exactly).
func sampleVertexOnlySeam(cA, cB miterCyl, p0, p1 math.Point3) ([]math.Point3, error) {
	d0, d1 := miterContactDir(cA, p0), miterContactDir(cA, p1)
	k := seamChordCount(d0, d1)
	prev := cA.cen.VectorTo(p0).Dot(cA.axis)
	out := make([]math.Point3, k+1)
	out[0], out[k] = p0, p1
	for j := 1; j < k; j++ {
		d := slerpVec(d0, d1, float64(j)/float64(k))
		p, lambda, ok := cylContactPointOnCyl(cA, d, cB, prev)
		if !ok {
			return nil, fmt.Errorf("fillet: vertex-only miter seam station %d has no point on both cylinders", j)
		}
		out[j], prev = p, lambda
	}
	return out, nil
}

// vertexOnlyMiterTangents orients a vertex-only miter's corner: ta is the seam end lying on a plane
// of face a, tb the end on face b (each seam end sits on the sharp edge bounding exactly one host
// of this arm), and the seam is returned running ta→tb.
func vertexOnlyMiterTangents(in cornerInputs, m *cornerMiter) (ta, tb math.Point3, seam []math.Point3) {
	first, last := m.seam[0], m.seam[len(m.seam)-1]
	if pointOnFacePlane(in.a, first) {
		return first, last, m.seam
	}
	return last, first, reversePoints(m.seam)
}

// pointOnFacePlane reports whether p lies on face f's plane within a radius-free, plane-scaled
// tolerance (1e-6 of the point's offset magnitude floor 1e-9) — used only to ORIENT the seam, where
// exactly one of the two candidate ends lies on the plane by construction.
func pointOnFacePlane(f *topo.Face, p math.Point3) bool {
	pl, ok := f.Geometry().(geom.Plane)
	if !ok {
		return false
	}
	n := outwardPlaneNormal(f, pl)
	off := stdmath.Abs(n.Dot(p.AsVector()) - n.Dot(pl.Origin.AsVector()))
	scale := stdmath.Max(1e-9, 1e-6*float64(p.DistanceTo(pl.Origin)))
	return off <= scale
}
