// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// fluxFace is one face prepared for repeated winding queries: its surface, its trim loops projected into
// (u, v) ONCE (loopToUV inverts ParamAt per sample — costly for a NURBS patch — so it must not run per
// query), the quadrature rectangle, and the orientation sign.
type fluxFace struct {
	surface geom.Surface
	polys   [][]math.Point2
	u0, u1  float64
	v0, v1  float64
	sign    float64
}

// fluxQuery is the universally-robust point-in-solid classifier: the generalized winding number
// (Jacobson) computed as the total solid angle each face subtends at p, integrated as the exact flux
// Ω = ∫∫_D (r·(S_u×S_v))/|r|³ du dv over the face's trimmed (u, v) domain (r = S(u,v) − p). For a closed
// body Σ_faces Ω is +4π when p is inside and 0 when outside (an orientation-reversed face contributes the
// opposite sign). Unlike ray parity it is not fooled by a non-watertight import (the flux is a continuous
// field, ≈4π well inside and ≈0 well outside regardless of a seam gap); unlike the nearest-point normal
// test it does not depend on any single face's orientation being reliable. It integrates the analytic
// surface directly (no stored mesh is read — the quadrature refines to the classifier's own tolerance).
//
// A body is prepared once for repeated queries — the analytic analog of a reused tessellation. Projecting
// the trim polylines once (loopToUV inverts ParamAt per sample, costly for a NURBS patch) is what makes
// the flux path affordable inside the boolean/fillet loops that classify thousands of points against one
// body.
type fluxQuery struct {
	faces []fluxFace
}

// newFluxQuery projects every face's trim into (u, v), captures its quadrature rectangle, and derives a
// consistent outward orientation sign per face — see orientFaceSigns. The sign is derived from the loop
// geometry, NOT the stored Face.Reversed flag, so an imported B-rep with inconsistent normal-side flags
// still yields Σ Ω = 4π inside.
func newFluxQuery(faces []curvedFace) *fluxQuery {
	q := &fluxQuery{faces: make([]fluxFace, 0, len(faces))}
	for _, f := range faces {
		uPer, vPer := surfacePeriodic(f.surface)
		polys := trimPolys(f, uPer, vPer)
		u0, u1, v0, v1, ok := fluxDomain(f, polys)
		if !ok {
			continue
		}
		q.faces = append(q.faces, fluxFace{f.surface, polys, u0, u1, v0, v1, 1})
	}
	for i, s := range orientFaceSigns(q.faces) {
		q.faces[i].sign = s
	}
	return q
}

// inside sums each prepared face's signed solid angle at p; a closed body gives ≈4π inside, ≈0 outside.
// The per-face sign is the orientation-normalized outward sign (orientFaceSigns), so the winding is 4π
// inside even when the stored Reversed flags are inconsistent.
func (q *fluxQuery) inside(p math.Point3) bool {
	total := 0.0
	for i := range q.faces {
		f := &q.faces[i]
		// Halfway between the two attractors (0 outside, 4π inside): robust to the O(tolerance) residual the
		// quadrature leaves on a non-watertight import, where the field lands near but not exactly on 4π/0.
		total += f.sign * integrateFluxCell(f.surface, p, f.polys, f.u0, f.u1, f.v0, f.v1, 0)
	}
	return stdmath.Abs(total) > 2*stdmath.Pi
}

// trimPolys projects every loop of the face into one continuous (u, v) polyline (reusing loopToUV's seam
// unwrapping). A boundaryless face returns nil — its whole finite domain is integrated.
func trimPolys(f curvedFace, uPer, vPer bool) [][]math.Point2 {
	if len(f.loops) == 0 {
		return nil
	}
	polys := make([][]math.Point2, 0, len(f.loops))
	for _, loop := range f.loops {
		if poly := loopToUV(f.surface, loop, uPer, vPer); len(poly) >= 3 {
			polys = append(polys, poly)
		}
	}
	return polys
}

// fluxDomain is the (u, v) rectangle the quadrature covers: the loops' unwrapped bounding box when the
// face is trimmed, else the surface's finite domain. It fails (ok=false) only for a face whose domain is
// unbounded AND untrimmed, which a closed body never has.
func fluxDomain(f curvedFace, polys [][]math.Point2) (u0, u1, v0, v1 float64, ok bool) {
	if len(polys) > 0 {
		return polyBounds(polys)
	}
	ul, uh := f.surface.UDomain()
	vl, vh := f.surface.VDomain()
	if stdmath.IsInf(ul, 0) || stdmath.IsInf(uh, 0) || stdmath.IsInf(vl, 0) || stdmath.IsInf(vh, 0) {
		return 0, 0, 0, 0, false
	}
	return ul, uh, vl, vh, true
}

// polyBounds is the axis-aligned (u, v) bounding box of every polyline vertex.
func polyBounds(polys [][]math.Point2) (u0, u1, v0, v1 float64, ok bool) {
	u0, v0 = stdmath.Inf(1), stdmath.Inf(1)
	u1, v1 = stdmath.Inf(-1), stdmath.Inf(-1)
	for _, poly := range polys {
		for _, q := range poly {
			u0, u1 = stdmath.Min(u0, q.X), stdmath.Max(u1, q.X)
			v0, v1 = stdmath.Min(v0, q.Y), stdmath.Max(v1, q.Y)
		}
	}
	return u0, u1, v0, v1, u1 > u0 && v1 > v0
}

// maxFluxDepth caps the adaptive subdivision near p, where the 1/|r|³ integrand peaks. The cap stops the
// recursion when p sits essentially on the surface (the integrand is singular there — that point is
// classified ON by the caller before the winding test).
const maxFluxDepth = 12

// baseFluxDepth is the minimum uniform subdivision every face gets (4^d cells) so the smooth far-field of
// the flux is captured before adaptive refinement begins. The winding test only needs the total within π
// of 0 or 4π, so a small base grid suffices; the near-p peak is resolved above it by fluxRefineRatio.
const baseFluxDepth = 4 // 16×16 base cells per face

// fluxRefineRatio splits a cell whose 3D extent is still large relative to its distance from p: there the
// 1/|r|³ integrand varies steeply across the cell and a single sample under-resolves it. Far from p the
// integrand is flat and the midpoint rule already converges, so those cells are not split — this is what
// keeps the flux affordable: refinement (of both the peak AND the trim boundary) follows p, not a fixed
// depth, and a far trim-straddle cell is left as a fractionally-weighted leaf whose small flux tolerates
// the half-cell trim uncertainty.
const fluxRefineRatio = 0.35 // tol:numeric — cell 3D span : distance-to-p ratio (dimensionless)

// integrateFluxCell integrates the solid-angle flux over one (u, v) cell, subdividing adaptively. A cell
// wholly outside the trim contributes nothing; a cell is split while it is below the base grid or its
// near-p integrand peak is unresolved; a resolved leaf contributes its midpoint estimate weighted by the
// fraction of the cell inside the trim (1 for an interior leaf, a sampled fraction for a boundary leaf).
func integrateFluxCell(s geom.Surface, p math.Point3, polys [][]math.Point2, u0, u1, v0, v1 float64, depth int) float64 {
	um, vm := 0.5*(u0+u1), 0.5*(v0+v1)
	frac := cellTrimFraction(polys, u0, u1, v0, v1)
	if frac == 0 {
		return 0
	}
	if depth < baseFluxDepth || (depth < maxFluxDepth && cellNeedsSplit(s, p, u0, u1, v0, v1)) {
		return integrateFluxCell(s, p, polys, u0, um, v0, vm, depth+1) +
			integrateFluxCell(s, p, polys, um, u1, v0, vm, depth+1) +
			integrateFluxCell(s, p, polys, u0, um, vm, v1, depth+1) +
			integrateFluxCell(s, p, polys, um, u1, vm, v1, depth+1)
	}
	return fluxIntegrand(s, p, um, vm) * (u1 - u0) * (v1 - v0) * frac
}

// fluxIntegrand evaluates (r·(S_u×S_v))/|r|³ at (u, v); the cross product carries the area Jacobian, so the
// midpoint value times the cell area is the cell's solid-angle contribution. It is 0 at a degenerate
// (pole/apex) sample, where S_u×S_v vanishes.
func fluxIntegrand(s geom.Surface, p math.Point3, u, v float64) float64 {
	du, dv := s.DerivativesAt(u, v)
	n := du.Cross(dv)
	r := p.VectorTo(s.PointAt(u, v)) // r = S(u,v) − p
	d := float64(r.Length())
	if d == 0 {
		return 0
	}
	return float64(r.Dot(n)) / (d * d * d)
}

// cellNeedsSplit reports whether the integrand's near-p peak is unresolved in this cell: its 3D span
// (corner spread) is still an appreciable fraction of the nearest corner's distance to p.
func cellNeedsSplit(s geom.Surface, p math.Point3, u0, u1, v0, v1 float64) bool {
	c00, c11 := s.PointAt(u0, v0), s.PointAt(u1, v1)
	c01, c10 := s.PointAt(u0, v1), s.PointAt(u1, v0)
	span := c00.DistanceTo(c11)
	if s := c01.DistanceTo(c10); s > span {
		span = s
	}
	near := minDist(p, c00, c01, c10, c11)
	return span > fluxRefineRatio*near
}

// minDist is the smallest distance from p to any of the given points.
func minDist(p math.Point3, pts ...math.Point3) float64 {
	best := stdmath.Inf(1)
	for _, q := range pts {
		if d := float64(p.DistanceTo(q)); d < best {
			best = d
		}
	}
	return best
}

// cellTrimFraction estimates the fraction of a (u, v) cell inside the trim loops from a 5-point sample
// (centre + four corners): 1 for a fully-interior cell, 0 for a fully-exterior one, and the sampled ratio
// for a boundary cell. An untrimmed face (no polys) is entirely inside. A boundary cell near p is split
// further (its fraction is then re-sampled on the smaller children, converging on the true boundary); a
// boundary cell far from p is left as a leaf, where its small flux tolerates the coarse fraction.
func cellTrimFraction(polys [][]math.Point2, u0, u1, v0, v1 float64) float64 {
	if len(polys) == 0 {
		return 1
	}
	um, vm := 0.5*(u0+u1), 0.5*(v0+v1)
	pts := [5]math.Point2{math.P2(um, vm), math.P2(u0, v0), math.P2(u1, v0), math.P2(u0, v1), math.P2(u1, v1)}
	inCount := 0
	for _, q := range pts {
		if pointInLoops2D(polys, q) {
			inCount++
		}
	}
	return float64(inCount) / float64(len(pts))
}

// pointInLoops2D is the even-odd point-in-region test in the (u, v) plane: a +u ray from q counts
// crossings of every loop segment (outer and holes together), inside on an odd total. The loops are the
// already-unwrapped polylines, so a straight 2D crossing count is exact — holes subtract naturally
// (a point inside the outer loop and inside a hole gets two crossings → even → outside).
func pointInLoops2D(polys [][]math.Point2, q math.Point2) bool {
	crossings := 0
	for _, poly := range polys {
		for i := range poly {
			if segCrossesURay(q, poly[i], poly[(i+1)%len(poly)]) {
				crossings++
			}
		}
	}
	return crossings%2 == 1
}

// segCrossesURay reports whether a ray from q toward +u crosses segment ab (standard crossing rule,
// straddle on v then the interpolated u lies beyond q).
func segCrossesURay(q, a, b math.Point2) bool {
	if (a.Y > q.Y) == (b.Y > q.Y) {
		return false
	}
	t := (q.Y - a.Y) / (b.Y - a.Y)
	return a.X+t*(b.X-a.X) > q.X
}
