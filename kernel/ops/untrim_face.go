// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Untrim (M36-F11) recovers the natural NURBS bounds of a trimmed face: it rebuilds the face on its
// full surface domain, bounded by the four boundary iso-curves, discarding the trim loops. The base
// surface is unchanged, so re-applying the original trim reproduces the face.

// UntrimFace returns a single-face surface body covering the full domain of the given face's NURBS
// surface (its four boundary iso-curves as the outer loop). It errors when the face is not a NURBS
// surface.
func UntrimFace(b *topo.Body, faceKey []byte) (*topo.Body, error) {
	f, ok := b.FindFaceByKey(faceKey)
	if !ok {
		return nil, fmt.Errorf("ops.UntrimFace: no face with key %x", faceKey)
	}
	surf, ok := f.Geometry().(geom.BSplineSurface)
	if !ok {
		return nil, fmt.Errorf("ops.UntrimFace: face is not a NURBS surface (%T)", f.Geometry())
	}
	return fullDomainBody(surf, "untrim"), nil
}

// fullDomainBody builds a one-face surface body over a B-spline surface's whole domain, bounded by
// its four boundary iso-curves.
func fullDomainBody(s geom.BSplineSurface, feat string) *topo.Body {
	bld := topo.NewBuilder(false, topo.NewLineage(topo.Tok(feat, "body", 0)))
	c00 := bld.AddVertex(s.PointAt(0, 0), topo.NewLineage(topo.Tok(feat, "v", 0)))
	c10 := bld.AddVertex(s.PointAt(1, 0), topo.NewLineage(topo.Tok(feat, "v", 1)))
	c11 := bld.AddVertex(s.PointAt(1, 1), topo.NewLineage(topo.Tok(feat, "v", 2)))
	c01 := bld.AddVertex(s.PointAt(0, 1), topo.NewLineage(topo.Tok(feat, "v", 3)))
	eBottom := bld.AddEdge(vIsoCurve(s, false), c00, c10, topo.NewLineage(topo.Tok(feat, "e", 0))) // v=0, along u
	eRight := bld.AddEdge(uIsoCurve(s, true), c10, c11, topo.NewLineage(topo.Tok(feat, "e", 1)))   // u=1, along v
	eTop := bld.AddEdge(vIsoCurve(s, true), c01, c11, topo.NewLineage(topo.Tok(feat, "e", 2)))     // v=1, along u
	eLeft := bld.AddEdge(uIsoCurve(s, false), c00, c01, topo.NewLineage(topo.Tok(feat, "e", 3)))   // u=0, along v
	loop := topo.OuterLoop(topo.Fwd(eBottom), topo.Fwd(eRight), topo.Rev(eTop), topo.Rev(eLeft))
	bld.AddFace(s, topo.NewLineage(topo.Tok(feat, "face", 0)), loop)
	return bld.Build()
}

// uIsoCurve returns the boundary curve at u=0 (or u=1) — the v-curve through that boundary control
// row.
func uIsoCurve(s geom.BSplineSurface, atMax bool) geom.Curve3 {
	i := 0
	if atMax {
		i = len(s.Ctrl) - 1
	}
	c, _ := geom.NewBSplineCurve(s.VDegree, s.Ctrl[i], s.Weights[i], s.VKnots)
	return c
}

// vIsoCurve returns the boundary curve at v=0 (or v=1) — the u-curve through that boundary control
// column.
func vIsoCurve(s geom.BSplineSurface, atMax bool) geom.Curve3 {
	j := 0
	if atMax {
		j = len(s.Ctrl[0]) - 1
	}
	ctrl := make([]math.Point3, len(s.Ctrl))
	w := make([]float64, len(s.Ctrl))
	for i := range s.Ctrl {
		ctrl[i], w[i] = s.Ctrl[i][j], s.Weights[i][j]
	}
	c, _ := geom.NewBSplineCurve(s.UDegree, ctrl, w, s.UKnots)
	return c
}
