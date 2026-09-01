// SPDX-License-Identifier: GPL-2.0-only

package hlr

import (
	stdmath "math"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/tessellate"
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

// SectionOptions modifies which material a section keeps (#1982). Reverse keeps the opposite half
// (the same cut plane, the far side retained). Depth > 0 limits the retained half to a slab of
// that thickness (model units) measured from the cut plane along the keep direction, so only
// geometry within Depth of the plane participates. The zero value is a full-depth forward cut,
// matching a bare section.
type SectionOptions struct {
	Reverse bool
	Depth   float64
}

// ProjectSection projects a full-depth forward section cut of body through the plane
// (view.Origin, view.ViewDir). It returns the retained half's edge segments (KindEdge,
// visible/hidden), the cut outline (KindCut, always visible) and hatch lines (KindHatch) filling
// the cut cross-section.
func ProjectSection(body *topo.Body, view View, q ops.Quality) []Segment {
	return ProjectSectionOpts(body, view, q, SectionOptions{})
}

// ProjectSectionOpts is ProjectSection with reverse-direction and limited-depth options (#1982).
// The cut plane, its bold outline and hatch are unchanged by the options; only the retained
// half — the material kept behind the plane — narrows (Depth) or flips (Reverse).
func ProjectSectionOpts(body *topo.Body, view View, q ops.Quality, o SectionOptions) []Segment {
	mesh, _ := tessellate.TessellateBody(body, q)
	bias := max(2*q.ChordTolerance, 0.005*meshDiagonal(mesh)) + 1e-9
	clip := newSectionClip(view, o)
	keep := clip.clipMesh(mesh) // occlusion tests against the retained half only
	segs := sectionEdges(body, view, clip, keep, q, bias)
	loops, cut := sectionOutline(body, view, q)
	segs = append(segs, cut...)
	return append(segs, hatchLoops(loops, meshDiagonal(mesh)/30)...)
}

// sectionClip is the retained-material test for one section: the origin on the cut plane, the
// unit keep direction (the retained half is the side it points into) and an optional slab depth
// (0 ⇒ unbounded). Reverse flips keep; a limited depth adds a parallel far bound.
type sectionClip struct {
	origin math.Point3
	keep   math.Vector3
	depth  float64
}

// newSectionClip resolves the options into a keep direction and slab depth. Reverse negates the
// keep direction (the projection basis is untouched, so the view does not mirror — only the
// removed half swaps).
func newSectionClip(view View, o SectionOptions) sectionClip {
	keep := view.ViewDir
	if o.Reverse {
		keep = keep.Negate()
	}
	return sectionClip{origin: view.Origin, keep: keep, depth: o.Depth}
}

// side is the signed distance of p from the cut plane along the keep direction: positive on the
// retained side, negative on the removed (near) side.
func (c sectionClip) side(p math.Point3) float64 {
	return float64(c.origin.VectorTo(p).Dot(c.keep))
}

// clipMesh keeps the triangles whose centroid lies in the retained slab, so the occlusion mesh is
// the cut-away half — removed material can no longer hide edges. It shares the original positions
// and only filters the index triples.
func (c sectionClip) clipMesh(mesh *ops.Mesh) *ops.Mesh {
	kept := &ops.Mesh{Positions: mesh.Positions}
	for t := 0; t+2 < len(mesh.Indices); t += 3 {
		i0, i1, i2 := mesh.Indices[t], mesh.Indices[t+1], mesh.Indices[t+2]
		s := c.side(centroid3(mesh.Positions[i0], mesh.Positions[i1], mesh.Positions[i2]))
		if s >= 0 && (c.depth <= 0 || s <= c.depth) {
			kept.Indices = append(kept.Indices, i0, i1, i2)
		}
	}
	return kept
}

// sectionEdges projects every B-rep edge, clipped to the retained slab, with hidden-line
// classification against the cut-away mesh.
func sectionEdges(body *topo.Body, view View, clip sectionClip, keep *ops.Mesh, q ops.Quality, bias float64) []Segment {
	var out []Segment
	for _, e := range body.Edges() {
		poly := tessellate.TessellateEdge(e, q)
		key := e.ReferenceKey()
		for i := 0; i+1 < len(poly); i++ {
			a, b, ok := clip.clipSegment(poly[i], poly[i+1])
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

// clipSegment returns the portion of segment a→b inside the retained slab (near side, and — when
// depth > 0 — nearer than depth); ok is false when nothing survives.
func (c sectionClip) clipSegment(a, b math.Point3) (math.Point3, math.Point3, bool) {
	a, b, ok := clipHalfSpace(a, b, c.side(a), c.side(b)) // near cut plane: keep side ≥ 0
	if !ok || c.depth <= 0 {
		return a, b, ok
	}
	return clipHalfSpace(a, b, c.depth-c.side(a), c.depth-c.side(b)) // far bound: depth − side ≥ 0
}

// clipHalfSpace returns the portion of segment a→b where the per-endpoint signed values sa,sb are
// ≥ 0 (interpolating the crossing); ok is false when both endpoints are below.
func clipHalfSpace(a, b math.Point3, sa, sb float64) (math.Point3, math.Point3, bool) {
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
			for _, p := range tessellate.TessellateEdge(e, q) {
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
