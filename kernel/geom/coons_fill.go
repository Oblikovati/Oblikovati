// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"fmt"

	"oblikovati.org/math"
)

// Boundary fill (M36-F07) — fill an opening bounded by four curves with a single clean NURBS. The
// foundation is the discrete bilinearly-blended Coons patch: a tensor-product B-spline whose control
// net blends the four boundary control polygons (at their Greville abscissae) minus the bilinear
// corner term, so the surface interpolates all four boundaries exactly (G0). Tangent/curvature (G1/
// G2) continuity to the surfaces across each edge is then imposed by matching the fill's boundary
// rows to the neighbours (MatchSurface), and an N-sided opening is filled by grouping its edges onto
// the four logical sides.
//
// The four curves must pair-wise share degree and knots (c0/c1 in u, d0/d1 in v — use F01's
// make-compatible) and meet at consistent corners.

// FillSide describes how one of a fill's four sides meets the surface across it: the adjacent
// surface, which of its edges is shared, and the continuity order to impose (0 = position only/G0).
type FillSide struct {
	Adjacent BSplineSurface
	AdjEdge  Boundary
	Order    int
}

// FillSurface fills the opening bounded by the four curves with a single NURBS that interpolates
// them (Coons, G0) and then meets each adjacent surface at the requested continuity (G1/G2/G3) by
// matching the fill's boundary rows to the neighbour. Sides with Order ≤ 0 stay at G0. The side
// order is c0 (v=0), c1 (v=1), d0 (u=0), d1 (u=1).
func FillSurface(c0, c1, d0, d1 BSplineCurve, sides [4]FillSide) (BSplineSurface, error) {
	fill, err := CoonsFill(c0, c1, d0, d1)
	if err != nil {
		return BSplineSurface{}, err
	}
	fillEdges := [4]Boundary{VMinEdge, VMaxEdge, UMinEdge, UMaxEdge}
	for i, s := range sides {
		if s.Order <= 0 {
			continue
		}
		fill, err = MatchSurface(fill, s.Adjacent, fillEdges[i], s.AdjEdge, s.Order)
		if err != nil {
			return BSplineSurface{}, fmt.Errorf("geom: fill side %d: %w", i, err)
		}
	}
	return fill, nil
}

// CoonsFill returns the discrete Coons surface bounded by the u-curves c0 (at v=0) and c1 (at v=1)
// and the v-curves d0 (at u=0) and d1 (at u=1). It errors when the pairs are incompatible or the
// corners do not meet.
//
// Example: fill, _ := CoonsFill(bottom, top, left, right) // interpolates all four edges (G0).
func CoonsFill(c0, c1, d0, d1 BSplineCurve) (BSplineSurface, error) {
	if err := compatiblePair(c0, c1, "u"); err != nil {
		return BSplineSurface{}, err
	}
	if err := compatiblePair(d0, d1, "v"); err != nil {
		return BSplineSurface{}, err
	}
	if err := consistentCorners(c0, c1, d0, d1); err != nil {
		return BSplineSurface{}, err
	}
	nu, nv := len(c0.Ctrl), len(d0.Ctrl)
	gu := grevilleAbscissae(c0.Knots, c0.Degree)
	gv := grevilleAbscissae(d0.Knots, d0.Degree)
	ctrl := make([][]math.Point3, nu)
	for i := range nu {
		ctrl[i] = make([]math.Point3, nv)
		for j := range nv {
			ctrl[i][j] = coonsControl(c0, c1, d0, d1, i, j, gu[i], gv[j])
		}
	}
	return NewBSplineSurface(c0.Degree, d0.Degree, ctrl, unitNet(nu, nv), c0.Knots, d0.Knots)
}

// coonsControl blends the four boundary control polygons at (a,b) = Greville abscissae of control
// (i,j), minus the bilinear corner term — the discrete Coons formula.
func coonsControl(c0, c1, d0, d1 BSplineCurve, i, j int, a, b float64) math.Point3 {
	nu := len(c0.Ctrl)
	lc := c0.Ctrl[i].Lerp(c1.Ctrl[i], b)        // (1−b)c0[i] + b·c1[i]
	ld := d0.Ctrl[j].Lerp(d1.Ctrl[j], a)        // (1−a)d0[j] + a·d1[j]
	bottom := c0.Ctrl[0].Lerp(c0.Ctrl[nu-1], a) // bilinear corner blend along v=0
	top := c1.Ctrl[0].Lerp(c1.Ctrl[nu-1], a)    // along v=1
	corner := bottom.Lerp(top, b)
	return math.P3(lc.X+ld.X-corner.X, lc.Y+ld.Y-corner.Y, lc.Z+ld.Z-corner.Z)
}

// grevilleAbscissae returns each control point's Greville abscissa (mean of its p covering knots),
// normalized to [0,1] over the curve domain — the parameter position the discrete Coons blend uses.
func grevilleAbscissae(knots []float64, p int) []float64 {
	nctrl := len(knots) - p - 1
	lo, hi := knots[p], knots[len(knots)-1-p]
	span := hi - lo
	g := make([]float64, nctrl)
	for i := range nctrl {
		sum := 0.0
		for k := 1; k <= p; k++ {
			sum += knots[i+k]
		}
		gi := lo
		if p > 0 {
			gi = sum / float64(p)
		}
		if span > 0 {
			g[i] = (gi - lo) / span
		}
	}
	return g
}

// compatiblePair checks two boundary curves share degree, control count and knots (so the discrete
// Coons net is well-defined).
func compatiblePair(a, b BSplineCurve, dir string) error {
	if a.Degree != b.Degree || len(a.Ctrl) != len(b.Ctrl) || len(a.Knots) != len(b.Knots) {
		return fmt.Errorf("geom: Coons %s-boundary curves must share degree/size (%d/%d vs %d/%d)", dir, a.Degree, len(a.Ctrl), b.Degree, len(b.Ctrl))
	}
	for i := range a.Knots {
		if a.Knots[i] != b.Knots[i] {
			return fmt.Errorf("geom: Coons %s-boundary curves must share knots", dir)
		}
	}
	return nil
}

// consistentCorners checks the four boundary curves meet at the four shared corners.
func consistentCorners(c0, c1, d0, d1 BSplineCurve) error {
	nu, nv := len(c0.Ctrl), len(d0.Ctrl)
	corners := [][2]math.Point3{
		{c0.Ctrl[0], d0.Ctrl[0]},       // u=0,v=0
		{c0.Ctrl[nu-1], d1.Ctrl[0]},    // u=1,v=0
		{c1.Ctrl[0], d0.Ctrl[nv-1]},    // u=0,v=1
		{c1.Ctrl[nu-1], d1.Ctrl[nv-1]}, // u=1,v=1
	}
	for k, c := range corners {
		if !c[0].IsEqualTo(c[1], 1e-9) {
			return fmt.Errorf("geom: Coons boundary corner %d does not meet: %v vs %v", k, c[0], c[1])
		}
	}
	return nil
}
