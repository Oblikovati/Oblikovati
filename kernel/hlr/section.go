// SPDX-License-Identifier: GPL-2.0-only

package hlr

import (
	stdmath "math"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Section views (M14-F02 PBI-140, #387) are a clipped-HLR cut-away. The view's ViewDir is the
// cutting-plane normal and its Origin lies on the plane; ProjectSection removes the half of the
// model in front of the plane (between the viewer and the plane), projects the retained half's
// edges with hidden-line removal, draws the body∩plane intersection as the bold cut outline, and
// fills the exposed cut faces with 45° hatch lines — a true cut-away, not a full-body projection
// with an outline drawn over it. The kept half is the side ViewDir points INTO (you look into
// the cut), i.e. points p with (p−Origin)·ViewDir ≥ 0.

// ProjectSection projects a section cut of body through the plane (view.Origin, view.ViewDir).
// It returns the retained half's edge segments (KindEdge, visible/hidden), the cut outline
// (KindCut, always visible) and hatch lines (KindHatch) filling the cut cross-section.
func ProjectSection(body *topo.Body, view View, q ops.Quality) []Segment {
	mesh, _ := ops.TessellateBody(body, q)
	bias := max(2*q.ChordTolerance, 0.005*meshDiagonal(mesh)) + 1e-9
	keep := clipMeshToHalfSpace(mesh, view) // occlusion tests against the retained half only
	segs := sectionEdges(body, view, keep, q, bias)
	loops, cut := sectionOutline(body, view, q)
	segs = append(segs, cut...)
	return append(segs, hatchLoops(loops, meshDiagonal(mesh)/30)...)
}

// side is the signed distance of p from the cut plane along the keep direction (ViewDir):
// positive on the retained side, negative on the removed (near) side.
func side(view View, p math.Point3) float64 {
	return float64(view.Origin.VectorTo(p).Dot(view.ViewDir))
}

// clipMeshToHalfSpace keeps the triangles whose centroid is on the retained side, so the
// occlusion mesh is the cut-away half — the removed near half can no longer hide edges. It
// shares the original positions and only filters the index triples.
func clipMeshToHalfSpace(mesh *ops.Mesh, view View) *ops.Mesh {
	kept := &ops.Mesh{Positions: mesh.Positions}
	for t := 0; t+2 < len(mesh.Indices); t += 3 {
		i0, i1, i2 := mesh.Indices[t], mesh.Indices[t+1], mesh.Indices[t+2]
		c := centroid3(mesh.Positions[i0], mesh.Positions[i1], mesh.Positions[i2])
		if side(view, c) >= 0 {
			kept.Indices = append(kept.Indices, i0, i1, i2)
		}
	}
	return kept
}

// sectionEdges projects every B-rep edge, clipped to the retained half, with hidden-line
// classification against the cut-away mesh.
func sectionEdges(body *topo.Body, view View, keep *ops.Mesh, q ops.Quality, bias float64) []Segment {
	var out []Segment
	for _, e := range body.Edges() {
		poly := ops.TessellateEdge(e, q)
		key := e.ReferenceKey()
		for i := 0; i+1 < len(poly); i++ {
			a, b, ok := clipSegmentToHalfSpace(view, poly[i], poly[i+1])
			if !ok {
				continue
			}
			if seg, ok := classifySegment(keep, view, a, b, key, bias); ok {
				out = append(out, seg)
			}
		}
	}
	return out
}

// clipSegmentToHalfSpace returns the portion of segment a→b on the retained side of the cut
// plane; ok is false when the whole segment lies on the removed near side.
func clipSegmentToHalfSpace(view View, a, b math.Point3) (math.Point3, math.Point3, bool) {
	sa, sb := side(view, a), side(view, b)
	switch {
	case sa < 0 && sb < 0:
		return a, b, false
	case sa >= 0 && sb >= 0:
		return a, b, true
	}
	cross := a.TranslateBy(a.VectorTo(b).Scale(math.Scalar(sa / (sa - sb))))
	if sa >= 0 {
		return a, cross, true
	}
	return cross, b, true
}

// sectionOutline intersects the body with the cut plane and projects the resulting loops: the
// bold cut-outline segments (KindCut) and the 2D loops the hatch fills.
func sectionOutline(body *topo.Body, view View, q ops.Quality) (loops [][]math.Point2, cut []Segment) {
	sec, err := ops.SectionWithPlane(body, view.Origin, view.ViewDir, q)
	if err != nil {
		return nil, nil
	}
	for _, w := range sec.Wires() {
		var pts []math.Point2
		for _, e := range w.Edges() {
			for _, p := range ops.TessellateEdge(e, q) {
				pts = append(pts, project2D(view, p))
			}
		}
		for i := 0; i+1 < len(pts); i++ {
			if !degenerate(pts[i], pts[i+1]) {
				cut = append(cut, Segment{A: pts[i], B: pts[i+1], Visible: true, Kind: KindCut})
			}
		}
		loops = append(loops, pts)
	}
	return loops, cut
}

func centroid3(a, b, c math.Point3) math.Point3 {
	return math.P3((a.X+b.X+c.X)/3, (a.Y+b.Y+c.Y)/3, (a.Z+b.Z+c.Z)/3)
}

const hatch45 = 0.70710678118654752440 // cos45° = sin45°, the 45° hatch rotation

// hatchLoops fills the projected cut loops with parallel 45° lines spaced `spacing` apart.
// It rotates into a frame where the hatch lines are horizontal scan lines, fills each scan line
// across the loops by the even-odd rule (so holes stay unhatched), and rotates back.
func hatchLoops(loops [][]math.Point2, spacing float64) []Segment {
	if spacing <= 0 {
		return nil
	}
	edges := rotatedLoopEdges(loops)
	if len(edges) == 0 {
		return nil
	}
	lo, hi := scanYBounds(edges)
	var out []Segment
	for y := lo + spacing*0.5; y < hi; y += spacing {
		xs := scanCrossings(edges, y)
		for i := 0; i+1 < len(xs); i += 2 {
			a, b := rotateBack(xs[i], y), rotateBack(xs[i+1], y)
			if !degenerate(a, b) {
				out = append(out, Segment{A: a, B: b, Visible: true, Kind: KindHatch})
			}
		}
	}
	return out
}

// rotatedLoopEdges rotates every loop edge by −45° (so 45° hatch becomes horizontal) and returns
// them as endpoint pairs in the rotated frame.
func rotatedLoopEdges(loops [][]math.Point2) [][2]math.Point2 {
	var edges [][2]math.Point2
	for _, loop := range loops {
		for i := 0; i+1 < len(loop); i++ {
			edges = append(edges, [2]math.Point2{rotateMinus45(loop[i]), rotateMinus45(loop[i+1])})
		}
	}
	return edges
}

// rotateMinus45 rotates a 2D point by −45°.
func rotateMinus45(p math.Point2) math.Point2 {
	x, y := float64(p.X), float64(p.Y)
	return math.P2(math.Scalar(x*hatch45+y*hatch45), math.Scalar(-x*hatch45+y*hatch45))
}

// rotateBack rotates a rotated-frame point (x, y) back by +45° into the view plane.
func rotateBack(x, y float64) math.Point2 {
	return math.P2(math.Scalar(x*hatch45-y*hatch45), math.Scalar(x*hatch45+y*hatch45))
}

// scanCrossings returns the sorted x-coordinates where the horizontal line at y crosses the
// loop edges (each crossing toggles inside/outside under the even-odd rule).
func scanCrossings(edges [][2]math.Point2, y float64) []float64 {
	var xs []float64
	for _, e := range edges {
		y0, y1 := float64(e[0].Y), float64(e[1].Y)
		if (y0 <= y) == (y1 <= y) {
			continue // both on the same side: no crossing
		}
		x0, x1 := float64(e[0].X), float64(e[1].X)
		xs = append(xs, x0+(x1-x0)*(y-y0)/(y1-y0))
	}
	sortFloats(xs)
	return xs
}

// scanYBounds returns the min/max y over the rotated edges.
func scanYBounds(edges [][2]math.Point2) (lo, hi float64) {
	lo, hi = stdmath.Inf(1), stdmath.Inf(-1)
	for _, e := range edges {
		for _, p := range e {
			lo, hi = stdmath.Min(lo, float64(p.Y)), stdmath.Max(hi, float64(p.Y))
		}
	}
	return lo, hi
}

// sortFloats sorts in place (insertion sort — crossing counts per scan line are tiny).
func sortFloats(xs []float64) {
	for i := 1; i < len(xs); i++ {
		for j := i; j > 0 && xs[j-1] > xs[j]; j-- {
			xs[j-1], xs[j] = xs[j], xs[j-1]
		}
	}
}
