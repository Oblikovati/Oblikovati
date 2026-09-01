// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	"fmt"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/internal/retopo"
	"oblikovati.org/kernel/ops/internal/tol"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// draftMod is the draft's [Modification] (ADR-0050 P7, the real #1802 fix): it tilts the selected
// planar faces about their neutral-plane hinge and re-intersects the neighbours, so a face bordering
// a fillet cylinder gets a re-trimmed elliptical edge instead of the plane-only rebuild panicking on
// the curved neighbour. It mirrors OCCT Draft_Modification: all new geometry is precomputed into
// per-entity maps, and NewSurface/NewCurve/NewPoint are lookups.
type draftMod struct {
	selected map[uint64]bool
	res      tol.Resolution
	faceSurf map[uint64]geom.Surface // every face's surface: tilted for selected, original otherwise
	newSurf  map[uint64]geom.Surface // only the tilted (selected) faces
	newPoint map[uint64]math.Point3  // relocated vertices
	newCurve map[uint64]geom.Curve3  // re-intersected edges
}

// newDraftMod precomputes the draft: tilt each selected planar face, relocate every vertex on a
// tilted face to the meeting point of its adjacent (new) surfaces, and re-intersect every edge that
// touches a relocated vertex. Errors if a selected face is not planar (curved-face draft is future
// work — the plane-only tilt does not yet apply to a cylinder/cone selected face).
func newDraftMod(solid *topo.Body, faceKeys [][]byte, pull math.UnitVector3, neutral *geom.Plane, angle float64) (*draftMod, error) {
	sel, err := retopo.ResolveFaceSet(solid, faceKeys)
	if err != nil {
		return nil, err
	}
	d := &draftMod{
		selected: sel, res: geom.ResolutionForBox(solid.RangeBox()),
		faceSurf: map[uint64]geom.Surface{}, newSurf: map[uint64]geom.Surface{},
		newPoint: map[uint64]math.Point3{}, newCurve: map[uint64]geom.Curve3{},
	}
	if err := d.tiltSelectedFaces(solid, pull, neutral, angle); err != nil {
		return nil, err
	}
	moved := d.relocateVertices(solid)
	if err := d.reintersectEdges(solid, moved); err != nil {
		return nil, err
	}
	return d, nil
}

// tiltSelectedFaces rotates each selected planar face about its hinge and records both the tilted
// surface (newSurf) and every face's effective surface (faceSurf) for the re-intersection.
func (d *draftMod) tiltSelectedFaces(solid *topo.Body, pull math.UnitVector3, neutral *geom.Plane, angle float64) error {
	for _, f := range solid.Faces() {
		if !d.selected[f.ID()] {
			d.faceSurf[f.ID()] = f.Geometry()
			continue
		}
		if _, ok := f.Geometry().(geom.Plane); !ok {
			return fmt.Errorf("draft: selected face %d is %T; only planar faces can be drafted (curved-face draft is future work)", f.ID(), f.Geometry())
		}
		tilted := draftedPlane(f, pull, neutral, angle)
		d.faceSurf[f.ID()], d.newSurf[f.ID()] = tilted, tilted
	}
	return nil
}

// relocateVertices moves every vertex lying on a tilted face to the common point of its adjacent
// new surfaces (a 3-surface Gauss-Newton solve from the old position), and returns the set of moved
// vertex IDs so the edge re-intersection knows which edges changed.
func (d *draftMod) relocateVertices(solid *topo.Body) map[uint64]bool {
	vf := retopo.VertexFaceMap(solid)
	moved := map[uint64]bool{}
	for _, v := range solid.Vertices() {
		if !d.vertexOnSelected(vf[v.ID()]) {
			continue
		}
		if p, ok := intersectSurfacesNear(v.Point(), d.surfacesAt(vf[v.ID()]), d.res.Weld()); ok {
			d.newPoint[v.ID()], moved[v.ID()] = p, true
		}
	}
	return moved
}

// reintersectEdges recomputes the curve of every edge touching a moved vertex: the intersection of
// its two faces' (new) surfaces, trimmed to the edge's new endpoints. Errors if a curve cannot be
// rebuilt (an unhandled surface pair).
func (d *draftMod) reintersectEdges(solid *topo.Body, moved map[uint64]bool) error {
	for _, e := range solid.Edges() {
		if !moved[e.StartVertex().ID()] && !moved[e.EndVertex().ID()] {
			continue
		}
		faces := e.Faces()
		if len(faces) != 2 {
			continue
		}
		sA, sB := d.faceSurf[faces[0].ID()], d.faceSurf[faces[1].ID()]
		crv, ok := d.reintersectEdge(e, sA, sB, d.endpoint(e.StartVertex()), d.endpoint(e.EndVertex()))
		if !ok {
			return fmt.Errorf("draft: cannot re-intersect edge %d between %T and %T", e.ID(), sA, sB)
		}
		d.newCurve[e.ID()] = crv
	}
	return nil
}

// vertexOnSelected reports whether any of the faces meeting at a vertex is drafted.
func (d *draftMod) vertexOnSelected(faces []*topo.Face) bool {
	for _, f := range faces {
		if d.selected[f.ID()] {
			return true
		}
	}
	return false
}

// surfacesAt returns the effective (new) surfaces of the faces meeting at a vertex, the tilted
// (drafted) ones first — so the three the Newton relocation pins on always include a moved surface
// even when more than three faces meet (a filleted corner), keeping the vertex on the tilted plane.
func (d *draftMod) surfacesAt(faces []*topo.Face) []geom.Surface {
	out := make([]geom.Surface, 0, len(faces))
	for _, f := range faces {
		if d.selected[f.ID()] {
			out = append(out, d.faceSurf[f.ID()])
		}
	}
	for _, f := range faces {
		if !d.selected[f.ID()] {
			out = append(out, d.faceSurf[f.ID()])
		}
	}
	return out
}

// endpoint returns a vertex's relocated position, or its original point if it did not move.
func (d *draftMod) endpoint(v *topo.Vertex) math.Point3 {
	if p, ok := d.newPoint[v.ID()]; ok {
		return p
	}
	return v.Point()
}

// NewSurface returns the tilted surface of a drafted face.
func (d *draftMod) NewSurface(f *topo.Face) (geom.Surface, bool) {
	s, ok := d.newSurf[f.ID()]
	return s, ok
}

// NewCurve returns the re-intersected curve of an edge touching the draft.
func (d *draftMod) NewCurve(e *topo.Edge) (geom.Curve3, bool) {
	c, ok := d.newCurve[e.ID()]
	return c, ok
}

// NewPoint returns the relocated position of a vertex on a drafted face.
func (d *draftMod) NewPoint(v *topo.Vertex) (math.Point3, bool) {
	p, ok := d.newPoint[v.ID()]
	return p, ok
}
