// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Plane sections (M07-F05, Oblikovati/Oblikovati#628): intersect a body with
// a plane and return the section curves as wires on a new wire-only body —
// the reference CreateIntersectionWithPlane. Mesh-accurate: each face mesh's
// triangle×plane segments chain into polylines.

// SectionWithPlane sections the body with the plane (origin, normal).
//
// Example: sec, err := ops.SectionWithPlane(b, math.P3(0,0,1), math.V3(0,0,1), ops.DefaultQuality())
func SectionWithPlane(b *topo.Body, origin math.Point3, normal math.Vector3, q Quality) (*topo.Body, error) {
	l := float64(normal.Length())
	if l == 0 {
		return nil, fmt.Errorf("ops.SectionWithPlane: zero plane normal")
	}
	n := normal.Scale(math.Scalar(1 / l))
	d := float64(origin.AsVector().Dot(n))
	mesh, _ := TessellateBody(b, q)
	segs := planeMeshSegments(mesh, n, d)
	if len(segs) == 0 {
		return nil, fmt.Errorf("ops.SectionWithPlane: the plane through %v misses the body", origin)
	}
	return wiresFromSegments(segs, "section")
}

// planeMeshSegments collects each triangle's crossing segment with the plane.
func planeMeshSegments(mesh *Mesh, n math.Vector3, d float64) [][2]math.Point3 {
	var segs [][2]math.Point3
	for t := 0; t+2 < len(mesh.Indices); t += 3 {
		tri := [3]math.Point3{
			mesh.Positions[mesh.Indices[t]], mesh.Positions[mesh.Indices[t+1]], mesh.Positions[mesh.Indices[t+2]],
		}
		if seg, ok := trianglePlaneSegment(tri, n, d); ok {
			segs = append(segs, seg)
		}
	}
	return segs
}

// trianglePlaneSegment clips one triangle against the plane, returning the
// crossing segment when the triangle genuinely straddles it.
func trianglePlaneSegment(tri [3]math.Point3, n math.Vector3, d float64) ([2]math.Point3, bool) {
	var s [3]float64
	for i, p := range tri {
		s[i] = float64(n.Dot(p.AsVector())) - d
	}
	var pts []math.Point3
	for i := range 3 {
		j := (i + 1) % 3
		if (s[i] > 0) == (s[j] > 0) || s[i] == s[j] {
			continue
		}
		f := math.Scalar(s[i] / (s[i] - s[j]))
		pts = append(pts, tri[i].TranslateBy(tri[i].VectorTo(tri[j]).Scale(f)))
	}
	if len(pts) != 2 || float64(pts[0].DistanceTo(pts[1])) < sectionWeld(pts) {
		return [2]math.Point3{}, false
	}
	return [2]math.Point3{pts[0], pts[1]}, true
}

// sectionWeld is the model-relative tolerance (#1399) for welding section segment endpoints into
// chains, derived from the section points' own extent rather than a cm-anchored grid.
func sectionWeld(pts []math.Point3) float64 { return ResolutionForPoints(pts).Plane() }

// wiresFromSegments chains loose segments into polyline wires on a fresh
// wire-only body (shared endpoints welded on a tolerance grid).
func wiresFromSegments(segs [][2]math.Point3, feat string) (*topo.Body, error) {
	pts := make([]math.Point3, 0, 2*len(segs))
	for _, s := range segs {
		pts = append(pts, s[0], s[1])
	}
	weldTol := sectionWeld(pts)
	chains := chainSegments(segs, weldTol)
	bld := topo.NewBuilder(false, topo.NewLineage(topo.Tok(feat, "body", 0)))
	body := bld.Build()
	for i, chain := range chains {
		curve, err := geom.NewPolyline(chain)
		if err != nil {
			continue // a degenerate (sub-tolerance) chain carries no geometry
		}
		v0 := bld.AddVertex(chain[0], topo.NewLineage(topo.Tok(feat, "vertex", 2*i)))
		v1 := v0
		if float64(chain[0].DistanceTo(chain[len(chain)-1])) > weldTol {
			v1 = bld.AddVertex(chain[len(chain)-1], topo.NewLineage(topo.Tok(feat, "vertex", 2*i+1)))
		}
		e := bld.AddEdge(curve, v0, v1, topo.NewLineage(topo.Tok(feat, "edge", i)))
		body.AttachWire(topo.NewLineage(topo.Tok(feat, "wire", i)), []topo.Use{topo.Fwd(e)})
	}
	if len(body.Wires()) == 0 {
		return nil, fmt.Errorf("ops: section produced no usable chains from %d segments", len(segs))
	}
	return body, nil
}

// chainSegments greedily links segments end-to-end on a weld grid into
// maximal polylines (closed chains end where they start).
func chainSegments(segs [][2]math.Point3, tol float64) [][]math.Point3 {
	used := make([]bool, len(segs))
	var chains [][]math.Point3
	for i := range segs {
		if used[i] {
			continue
		}
		used[i] = true
		chain := []math.Point3{segs[i][0], segs[i][1]}
		chain = extendChain(chain, segs, used, tol)
		chains = append(chains, chain)
	}
	return chains
}

// extendChain grows the chain from both ends until no segment attaches.
func extendChain(chain []math.Point3, segs [][2]math.Point3, used []bool, tol float64) []math.Point3 {
	for grew := true; grew; {
		grew = false
		for j := range segs {
			if used[j] {
				continue
			}
			if next, ok := attachSegment(chain, segs[j], tol); ok {
				chain, used[j], grew = next, true, true
			}
		}
	}
	return chain
}

// attachSegment tries to weld a segment onto either chain end.
func attachSegment(chain []math.Point3, seg [2]math.Point3, tol float64) ([]math.Point3, bool) {
	head, tail := chain[0], chain[len(chain)-1]
	switch {
	case float64(tail.DistanceTo(seg[0])) <= tol:
		return append(chain, seg[1]), true
	case float64(tail.DistanceTo(seg[1])) <= tol:
		return append(chain, seg[0]), true
	case float64(head.DistanceTo(seg[1])) <= tol:
		return append([]math.Point3{seg[0]}, chain...), true
	case float64(head.DistanceTo(seg[0])) <= tol:
		return append([]math.Point3{seg[1]}, chain...), true
	}
	return chain, false
}

// FaceSilhouetteWires computes the silhouette curves of one face as viewed
// from viewDir, clipped to the face's trim, as wires on a new body — the
// reference TransientBRep.CreateSilhouetteCurve. includeBoundary keeps
// silhouette runs that coincide with the face's own edges.
//
// Example: sil, err := ops.FaceSilhouetteWires(f, math.V3(0,0,1), true, ops.DefaultQuality())
func FaceSilhouetteWires(f *topo.Face, viewDir math.Vector3, includeBoundary bool, q Quality) (*topo.Body, error) {
	loops := geom.Silhouette(f.Geometry(), viewDir, faceParamWindow(f))
	if len(loops) == 0 {
		return nil, fmt.Errorf("ops: face %d has no silhouette from %v", f.ID(), viewDir)
	}
	onTol := stdmath.Max(q.tol(), 1e-6) // tol:calibrated — floors the (already model-relative) quality chord tolerance
	var segs [][2]math.Point3
	for _, pl := range loops {
		segs = append(segs, clipPolylineToFace(pl, f, onTol, includeBoundary)...)
	}
	if len(segs) == 0 {
		return nil, fmt.Errorf("ops: face %d's silhouette lies outside its trim", f.ID())
	}
	return wiresFromSegments(segs, "silhouette")
}

// clipPolylineToFace keeps the polyline's segments whose midpoints lie on the
// face's trimmed surface (within onTol, analytic — no tessellation read),
// optionally dropping runs hugging the face's boundary edges.
func clipPolylineToFace(pl []math.Point3, f *topo.Face, onTol float64, includeBoundary bool) [][2]math.Point3 {
	var out [][2]math.Point3
	for i := 0; i+1 < len(pl); i++ {
		mid := pl[i].TranslateBy(pl[i].VectorTo(pl[i+1]).Scale(0.5))
		if !brep.PointOnFace(f, mid, onTol) {
			continue
		}
		if !includeBoundary && pointNearFaceBoundary(mid, f, onTol) {
			continue
		}
		out = append(out, [2]math.Point3{pl[i], pl[i+1]})
	}
	return out
}

// faceParamWindow derives the silhouette tracing window from the face's own
// trim: every boundary sample inverted into (u, v). An unbounded surface
// direction (a cylinder's axial v) gets no window from the surface domain, so
// the trim is the only honest source.
func faceParamWindow(f *topo.Face) geom.SurfaceGrid {
	s := f.Geometry()
	g := geom.SurfaceGrid{UMin: stdmath.Inf(1), UMax: stdmath.Inf(-1), VMin: stdmath.Inf(1), VMax: stdmath.Inf(-1)}
	for _, e := range f.Edges() {
		c := e.Geometry()
		lo, hi := c.Domain()
		for i := 0; i <= 16; i++ {
			u, v := s.ParamAt(c.PointAt(lo + (hi-lo)*float64(i)/16))
			g.UMin, g.UMax = stdmath.Min(g.UMin, u), stdmath.Max(g.UMax, u)
			g.VMin, g.VMax = stdmath.Min(g.VMin, v), stdmath.Max(g.VMax, v)
		}
	}
	if stdmath.IsInf(g.UMin, 1) {
		g = geom.SurfaceGrid{UMin: stdmath.NaN()} // boundary-less face: fill from the domain below
	}
	g.USteps, g.VSteps = 97, 97
	return shiftSeamWindows(s, g)
}

// shiftSeamWindows nudges a full-period tracing window half a cell off the
// seam: a silhouette ruling lying exactly at u = 0 = 2π is invisible to
// marching squares (the field is ~0 on the boundary nodes with no sign change
// across them); shifted, the same ruling falls strictly inside a cell. The
// window still spans one full period — periodic surfaces evaluate past their
// nominal domain naturally.
func shiftSeamWindows(s geom.Surface, g geom.SurfaceGrid) geom.SurfaceGrid {
	uLo, uHi := s.UDomain()
	vLo, vHi := s.VDomain()
	if stdmath.IsNaN(g.UMin) {
		g.UMin, g.UMax, g.VMin, g.VMax = uLo, uHi, vLo, vHi
	}
	if fullFiniteSpan(g.UMin, g.UMax, uLo, uHi) {
		half := (g.UMax - g.UMin) / float64(2*g.USteps)
		g.UMin, g.UMax = g.UMin+half, g.UMax+half
	}
	if fullFiniteSpan(g.VMin, g.VMax, vLo, vHi) {
		half := (g.VMax - g.VMin) / float64(2*g.VSteps)
		g.VMin, g.VMax = g.VMin+half, g.VMax+half
	}
	return g
}

// fullFiniteSpan reports whether [lo, hi] covers the whole finite surface
// domain (the periodic full-turn case).
func fullFiniteSpan(lo, hi, dLo, dHi float64) bool {
	if stdmath.IsInf(dLo, 0) || stdmath.IsInf(dHi, 0) || dHi <= dLo {
		return false
	}
	return lo <= dLo+1e-9 && hi >= dHi-1e-9 // tol:parametric — surface-domain bounds
}

// pointNearFaceBoundary reports whether p hugs one of the face's edges.
func pointNearFaceBoundary(p math.Point3, f *topo.Face, tol float64) bool {
	for _, e := range f.Edges() {
		c := e.Geometry()
		lo, hi := c.Domain()
		for i := 0; i <= 32; i++ {
			if float64(p.DistanceTo(c.PointAt(lo+(hi-lo)*float64(i)/32))) <= tol {
				return true
			}
		}
	}
	return false
}
