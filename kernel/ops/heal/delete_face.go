// SPDX-License-Identifier: GPL-2.0-only

package heal

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/internal/probe"
	"oblikovati.org/kernel/ops/internal/retopo"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// DeleteFaces removes the selected faces from a planar solid and heals the openings by
// extending the neighbouring faces until they meet again. Each vertex of a deleted face
// slides along the line of its two surviving neighbour planes to the nearest OTHER neighbour
// plane (the face the heal extends to meet); coincident results weld, collapsing the deleted
// face's loop — e.g. deleting a chamfer face restores the sharp edge. It errors when the
// heal does not produce a valid closed solid (a non-healable selection), so the feature can
// go Sick rather than ship an open body.
func DeleteFaces(solid *topo.Body, faceKeys [][]byte) (*topo.Body, error) {
	del, err := retopo.ResolveFaceSet(solid, faceKeys)
	if err != nil {
		return nil, err
	}
	moved := healedPositions(solid, del)
	body := retopo.BuildSolidFromLoops(survivingLoops(solid, del, moved))
	if r := Validate(body); !r.Valid || !body.IsSolid() {
		return nil, fmt.Errorf("delete-face: heal did not close the body %v", r.Issues)
	}
	// Provenance (ADR-0043): a surviving edge the heal carried through unchanged keeps its original
	// identity rather than the rebuild's build-order name; the new healed edges keep the fallback.
	body.InheritOriginalEdges(solid.Edges())
	return body, nil
}

// healedPositions returns, per vertex touching a deleted face, the position it heals to.
func healedPositions(solid *topo.Body, del map[uint64]bool) map[uint64]math.Point3 {
	vf := retopo.VertexFaceMap(solid)
	ring := ringPlanes(solid, del)
	moved := map[uint64]math.Point3{}
	for _, v := range solid.Vertices() {
		if touchesDeleted(vf[v.ID()], del) {
			moved[v.ID()] = healedVertex(v, vf[v.ID()], del, ring)
		}
	}
	return moved
}

// touchesDeleted reports whether any face at the vertex is being deleted.
func touchesDeleted(faces []*topo.Face, del map[uint64]bool) bool {
	for _, f := range faces {
		if del[f.ID()] {
			return true
		}
	}
	return false
}

// ringPlanes returns the planes of every face that neighbours a deleted face (shares an
// edge) but is not itself deleted — the surfaces the heal extends to.
func ringPlanes(solid *topo.Body, del map[uint64]bool) []geom.Plane {
	seen := map[uint64]bool{}
	var out []geom.Plane
	for _, f := range solid.Faces() {
		if !del[f.ID()] {
			continue
		}
		for _, e := range f.Edges() {
			for _, nb := range e.Faces() {
				if del[nb.ID()] || seen[nb.ID()] {
					continue
				}
				seen[nb.ID()] = true
				out = append(out, nb.Geometry().(geom.Plane))
			}
		}
	}
	return out
}

// healedVertex returns where a vertex on a deleted face moves: the meet of its surviving
// face planes when ≥3 still pin it; otherwise it slides along the line of its two surviving
// planes to the nearest ring plane (the neighbour the heal extends to meet).
func healedVertex(v *topo.Vertex, faces []*topo.Face, del map[uint64]bool, ring []geom.Plane) math.Point3 {
	survivors := survivingPlanes(faces, del)
	if len(survivors) >= 3 {
		if p, ok := probe.MeetOfPlanes(survivors); ok {
			return p
		}
	}
	if len(survivors) == 2 {
		if p, ok := slideToNearest(survivors[0], survivors[1], ring, v.Point()); ok {
			return p
		}
	}
	return v.Point()
}

// survivingPlanes returns the planes of the vertex's faces that are not deleted.
func survivingPlanes(faces []*topo.Face, del map[uint64]bool) []geom.Plane {
	var out []geom.Plane
	for _, f := range faces {
		if !del[f.ID()] {
			out = append(out, f.Geometry().(geom.Plane))
		}
	}
	return out
}

// slideToNearest intersects the line of planes a,b with each ring plane and returns the
// intersection nearest the original vertex (the face the heal extends to meet).
func slideToNearest(a, b geom.Plane, ring []geom.Plane, v math.Point3) (math.Point3, bool) {
	p0, dir, ok := probe.TwoPlaneLine(a, b)
	if !ok {
		return math.Point3{}, false
	}
	best, bestD, found := math.Point3{}, stdmath.Inf(1), false
	for _, c := range ring {
		t, hit := lineHitsPlane(p0, dir, c)
		if !hit {
			continue
		}
		p := p0.TranslateBy(dir.Scale(t))
		if d := p.DistanceTo(v); d < bestD {
			best, bestD, found = p, d, true
		}
	}
	return best, found
}

// lineHitsPlane returns the parameter t where line p0+t·dir meets plane c.
func lineHitsPlane(p0 math.Point3, dir math.Vector3, c geom.Plane) (float64, bool) {
	n := c.Normal()
	den := dir.Dot(n)
	if stdmath.Abs(den) < probe.SingularSolveTol {
		return 0, false
	}
	return (n.Dot(c.Origin.AsVector()) - n.Dot(p0.AsVector())) / den, true
}

// survivingLoops returns every non-deleted face as a point-ring loop with its vertices
// remapped to their healed positions.
func survivingLoops(solid *topo.Body, del map[uint64]bool, moved map[uint64]math.Point3) []retopo.PlanarLoop {
	var out []retopo.PlanarLoop
	for _, f := range solid.Faces() {
		if del[f.ID()] {
			continue
		}
		pl := retopo.PlanarLoop{Normal: f.Geometry().NormalAt(0, 0), Lineage: f.Lineage()}
		for _, l := range f.Loops() {
			pl.Rings = append(pl.Rings, healedRing(l, moved))
		}
		out = append(out, pl)
	}
	return out
}

// healedRing returns a loop's vertices with healed positions substituted.
func healedRing(l *topo.Loop, moved map[uint64]math.Point3) []math.Point3 {
	var pts []math.Point3
	for _, u := range l.EdgeUses() {
		v := u.Edge().StartVertex()
		if u.Reversed() {
			v = u.Edge().EndVertex()
		}
		if p, ok := moved[v.ID()]; ok {
			pts = append(pts, p)
		} else {
			pts = append(pts, v.Point())
		}
	}
	return pts
}
