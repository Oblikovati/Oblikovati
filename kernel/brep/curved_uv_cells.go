// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/math"
)

// Phase 3 of the ruled-side (u,v) arrangement (#2212): subdivide the band along the imprint
// and classify each resulting cell as kept or dropped.
//
// The subdivision itself is the shared planar engine (Arrange, arrange2d.go) — this phase only
// feeds it and reads its cells back. Classification asks the material predicate at ONE interior
// point per cell, so finding a point that is genuinely inside the cell (and not hugging an edge,
// where the predicate is ill-conditioned) is most of the work here.

// arrangeBand runs the planar subdivision on the assembled band, returning the cells as (u,v) polygons
// (outer loop + nested holes). It is the periodic band flattened to a rectangle; cross-seam adjacency is
// reconciled later when the kept region's boundary is walked.
func arrangeBand(segs []uvSeg) []Face2D {
	in := make([][2]math.Point2, 0, len(segs))
	for _, s := range segs {
		in = append(in, [2]math.Point2{s.a, s.b})
	}
	return Arrange(in)
}

// keptCells returns the arrangement cells whose interior is on the material side of the imprint, each cell
// classified by evaluating the predicate at a point strictly inside its outer loop. A cell straddling the
// imprint cannot occur (the imprint is an arrangement edge), so any interior sample decides the whole cell.
func keptCells(cells []Face2D, material materialPredicate) []Face2D {
	var kept []Face2D
	for _, cell := range cells {
		if p, ok := interiorOfCell(cell); ok && material(p) {
			kept = append(kept, cell)
		}
	}
	return kept
}

// interiorOfCell returns the DEEPEST interior point of the cell — the one maximising distance to its outer
// and hole edges — so the material classification is sampled well clear of the imprint. Any strictly interior
// point decides the whole cell (the imprint is an arrangement edge, so the interior has one material), but the
// vertex-average centroid of a CONCAVE cell whose boundary is densely sampled along the imprint (the
// Steinmetz pinch's two lobes carve deep concavities into the surrounding region) is dragged onto the imprint
// itself, where the membership test flickers — the deepest-point sampler avoids the boundary entirely
// (#1403). It is a pole-of-inaccessibility grid search: a coarse scan of the bbox then a local refine.
func interiorOfCell(cell Face2D) (math.Point2, bool) {
	best, bestDist := math.Point2{}, -1.0
	scan := func(lo, hi math.Point2, n int) {
		for i := 0; i <= n; i++ {
			for j := 0; j <= n; j++ {
				p := math.P2(math.Lerp(float64(lo.X), float64(hi.X), float64(i)/float64(n)),
					math.Lerp(float64(lo.Y), float64(hi.Y), float64(j)/float64(n)))
				if d := cellEdgeClearance(p, cell); d > bestDist && insideCell(p, cell) {
					best, bestDist = p, d
				}
			}
		}
	}
	lo, hi := cellBounds(cell)
	scan(lo, hi, 24)
	if bestDist < 0 {
		return centroidOf(cell.Outer), false // no interior sample found (a degenerate sliver cell)
	}
	// Refine within one coarse cell of the best to sharpen the deepest point on a thin lobe.
	step := math.P2((hi.X-lo.X)/24, (hi.Y-lo.Y)/24)
	scan(math.P2(best.X-step.X, best.Y-step.Y), math.P2(best.X+step.X, best.Y+step.Y), 8)
	return best, true
}

// cellEdgeClearance is the distance from p to the nearest edge of the cell (outer loop and every hole) — the
// objective the interior-point search maximises so the sample sits as far from the imprint as the cell allows.
func cellEdgeClearance(p math.Point2, cell Face2D) float64 {
	d := loopClearance(p, cell.Outer)
	for _, h := range cell.Holes {
		if dh := loopClearance(p, h); dh < d {
			d = dh
		}
	}
	return d
}

// loopClearance is the minimum distance from p to any edge of a closed (u,v) polygon.
func loopClearance(p math.Point2, poly []math.Point2) float64 {
	d := stdmath.MaxFloat64
	for i, n := 0, len(poly); i < n; i++ {
		if e := perpDistToSeg(p, poly[i], poly[(i+1)%n]); e < d {
			d = e
		}
	}
	return d
}

// cellBounds returns the (u,v) bounding box of a cell's outer loop.
func cellBounds(cell Face2D) (lo, hi math.Point2) {
	lo, hi = math.P2(stdmath.MaxFloat64, stdmath.MaxFloat64), math.P2(-stdmath.MaxFloat64, -stdmath.MaxFloat64)
	for _, p := range cell.Outer {
		lo = math.P2(stdmath.Min(float64(lo.X), float64(p.X)), stdmath.Min(float64(lo.Y), float64(p.Y)))
		hi = math.P2(stdmath.Max(float64(hi.X), float64(p.X)), stdmath.Max(float64(hi.Y), float64(p.Y)))
	}
	return lo, hi
}

// insideCell reports whether p is inside the cell's outer loop and outside all its holes.
func insideCell(p math.Point2, cell Face2D) bool {
	if !pointInPolygon2D(p, cell.Outer) {
		return false
	}
	for _, h := range cell.Holes {
		if pointInPolygon2D(p, h) {
			return false
		}
	}
	return true
}

// interiorPointOf returns a point strictly inside the simple polygon (and ok). The centroid serves for a
// convex cell; for a concave one it may fall outside, so the fallback steps a short way from each edge
// midpoint toward the centroid until a point lands inside — robust for the arrangement's simple cells.
func interiorPointOf(poly []math.Point2) (math.Point2, bool) {
	if len(poly) < 3 {
		return math.Point2{}, false
	}
	c := centroidOf(poly)
	if pointInPolygon2D(c, poly) {
		return c, true
	}
	for i := range poly {
		a, b := poly[i], poly[(i+1)%len(poly)]
		mid := math.P2((float64(a.X)+float64(b.X))/2, (float64(a.Y)+float64(b.Y))/2)
		for _, f := range []float64{1e-3, 1e-2, 0.1, 0.5} {
			p := math.P2(math.Lerp(float64(mid.X), float64(c.X), f), math.Lerp(float64(mid.Y), float64(c.Y), f))
			if pointInPolygon2D(p, poly) {
				return p, true
			}
		}
	}
	return c, false
}

// centroidOf returns the vertex average of a polygon (a cheap interior estimate, exact for the convex case).
func centroidOf(poly []math.Point2) math.Point2 {
	var sx, sy float64
	for _, p := range poly {
		sx, sy = sx+float64(p.X), sy+float64(p.Y)
	}
	n := float64(len(poly))
	return math.P2(sx/n, sy/n)
}
