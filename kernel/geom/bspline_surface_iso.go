// SPDX-License-Identifier: GPL-2.0-only

package geom

import "oblikovati.org/math"

// The boundary iso-curves of a B-spline surface. They are a property of the surface, not of any
// one operation, so they live here rather than in a caller: untrimming, opening-fill and every
// other consumer must read the SAME curve for the same boundary, or the edges they build will not
// meet (kernel ground rules: behaviour many operations need is a method on the geometry).

// BoundaryUIso returns the boundary curve at u=0, or at u=1 when atMax — the v-direction curve
// through that boundary control row.
//
// Example: left := s.BoundaryUIso(false) // the u=0 edge of the surface's full domain
func (s BSplineSurface) BoundaryUIso(atMax bool) BSplineCurve {
	i := 0
	if atMax {
		i = len(s.Ctrl) - 1
	}
	c, _ := NewBSplineCurve(s.VDegree, s.Ctrl[i], s.Weights[i], s.VKnots)
	return c
}

// BoundaryVIso returns the boundary curve at v=0, or at v=1 when atMax — the u-direction curve
// through that boundary control column.
//
// Example: bottom := s.BoundaryVIso(false) // the v=0 edge of the surface's full domain
func (s BSplineSurface) BoundaryVIso(atMax bool) BSplineCurve {
	j := 0
	if atMax {
		j = len(s.Ctrl[0]) - 1
	}
	ctrl := make([]math.Point3, len(s.Ctrl))
	w := make([]float64, len(s.Ctrl))
	for i := range s.Ctrl {
		ctrl[i], w[i] = s.Ctrl[i][j], s.Weights[i][j]
	}
	c, _ := NewBSplineCurve(s.UDegree, ctrl, w, s.UKnots)
	return c
}
