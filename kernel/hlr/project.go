// SPDX-License-Identifier: GPL-2.0-only

// Package hlr is the hidden-line removal engine behind drawing views (M14-F02 PBI-139,
// #386): it projects a B-rep body's edges orthographically onto a view plane and classifies
// each projected segment as visible or hidden by depth-testing it against the body's mesh.
// Each segment carries its source edge's reference key, so a drawing view stays associative
// across model recompute.
//
// HLR is image-space/sampled, not analytic: the body is tessellated once and each edge
// segment's midpoint is occlusion-tested by casting a ray toward the viewer. This is robust
// for solids whose outline is described by B-rep edges (planar/prismatic parts, holes' rim
// circles); the silhouette outline of a smooth face (a cylinder's side) is not a B-rep edge
// and is a follow-up.
package hlr

import (
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// View is an orthographic projection frame: an origin and an orthonormal basis. XAxis and
// YAxis span the projection plane (screen right and up); ViewDir points INTO the screen (from
// the viewer toward the model), so a point is hidden when a face lies between it and the
// viewer (along -ViewDir). The three axes must be unit and mutually orthogonal.
type View struct {
	Origin                math.Point3
	XAxis, YAxis, ViewDir math.Vector3
}

// NewView builds an orthographic frame looking along viewDir (into the screen), with screen-up
// derived from upHint. viewDir and upHint need not be unit or exactly orthogonal.
func NewView(origin math.Point3, viewDir, upHint math.Vector3) View {
	f := unit(viewDir)
	x := unit(upHint.Cross(f)) // screen right = up × forward (right-handed, y up)
	y := f.Cross(x)            // screen up, already unit (f ⟂ x, both unit)
	return View{Origin: origin, XAxis: x, YAxis: y, ViewDir: f}
}

// Segment is one projected edge segment classified visible/hidden, carrying the source edge's
// reference key for view↔model associativity.
type Segment struct {
	A, B    math.Point2
	Visible bool
	EdgeKey []byte
}

// Project orthographically projects every B-rep edge of body onto view's plane and classifies
// each tessellated segment visible or hidden by depth-testing its midpoint against the body's
// mesh. Segments that project to (near) a point — edges seen end-on — are dropped.
//
//	view := hlr.NewView(center, math.V3(0,1,0), math.V3(0,0,1)) // front view
//	segs := hlr.Project(body, view, ops.DefaultQuality())
func Project(body *topo.Body, view View, q ops.Quality) []Segment {
	mesh, _ := ops.TessellateBody(body, q)
	// The occlusion bias must clear two sources of false self-occlusion at an edge: the
	// faceting error of curved faces (≈ChordTolerance) and a silhouette edge's nearly
	// edge-on adjacent face, which a midpoint ray can graze. Scaling it to the model size
	// (0.5% of the bounding diagonal) makes the classification platform-stable — a grazing
	// face sits within the bias band and is ignored, while a real occluder is far beyond it.
	bias := maxf(2*q.ChordTolerance, 0.005*meshDiagonal(mesh)) + 1e-9
	var segs []Segment
	for _, e := range body.Edges() {
		poly := ops.TessellateEdge(e, q)
		key := e.ReferenceKey()
		for i := 0; i+1 < len(poly); i++ {
			if seg, ok := classifySegment(mesh, view, poly[i], poly[i+1], key, bias); ok {
				segs = append(segs, seg)
			}
		}
	}
	return segs
}

// classifySegment projects one edge sub-segment and classifies it; ok is false for a segment
// that projects to a point (the edge runs along the view direction).
func classifySegment(mesh *ops.Mesh, view View, a, b math.Point3, key []byte, bias float64) (Segment, bool) {
	a2, b2 := project2D(view, a), project2D(view, b)
	if degenerate(a2, b2) {
		return Segment{}, false
	}
	return Segment{A: a2, B: b2, Visible: !occluded(mesh, view, midpoint(a, b), bias), EdgeKey: key}, true
}

// project2D drops a 3D point onto the view plane: (u, v) = ((p-origin)·xAxis, (p-origin)·yAxis).
func project2D(view View, p math.Point3) math.Point2 {
	d := view.Origin.VectorTo(p)
	return math.P2(d.Dot(view.XAxis), d.Dot(view.YAxis))
}

// occluded reports whether a face lies CLEARLY between p and the viewer. The ray starts a hair
// toward the viewer (by bias, to clear p's own face) and casts toward the viewer; only a hit
// beyond a further bias counts, so a silhouette edge's grazing adjacent face (hit within the
// bias band) does not falsely hide the edge — the classification is then platform-stable.
func occluded(mesh *ops.Mesh, view View, p math.Point3, bias float64) bool {
	toViewer := view.ViewDir.Negate()
	origin := p.TranslateBy(toViewer.Scale(math.Scalar(bias)))
	d, hit := ops.RayCastMesh(mesh, origin, toViewer)
	return hit && d > bias
}

func midpoint(a, b math.Point3) math.Point3 {
	return math.P3((a.X+b.X)/2, (a.Y+b.Y)/2, (a.Z+b.Z)/2)
}

func degenerate(a, b math.Point2) bool {
	dx, dy := float64(b.X-a.X), float64(b.Y-a.Y)
	return dx*dx+dy*dy < 1e-12
}

// meshDiagonal is the length of the mesh's bounding-box diagonal — the model size the
// occlusion bias scales to. Zero for an empty mesh.
func meshDiagonal(m *ops.Mesh) float64 {
	if len(m.Positions) == 0 {
		return 0
	}
	lo, hi := m.Positions[0], m.Positions[0]
	for _, p := range m.Positions {
		lo = math.P3(minS(lo.X, p.X), minS(lo.Y, p.Y), minS(lo.Z, p.Z))
		hi = math.P3(maxS(hi.X, p.X), maxS(hi.Y, p.Y), maxS(hi.Z, p.Z))
	}
	return float64(lo.VectorTo(hi).Length())
}

func minS(a, b math.Scalar) math.Scalar {
	if a < b {
		return a
	}
	return b
}

func maxS(a, b math.Scalar) math.Scalar {
	if a > b {
		return a
	}
	return b
}

func maxf(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func unit(v math.Vector3) math.Vector3 {
	l := v.Length()
	if l == 0 {
		return v
	}
	return v.Scale(1 / l)
}
