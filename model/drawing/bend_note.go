// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/hlr"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Bend notes (#1995). A bend note is a feature note reading "<angle>° R<radius> <UP|DOWN>" — a
// sheet-metal bend's angle, inside radius and fold direction. All three are DERIVED from the model:
// the bend face is the cylindrical face on the picked edge (its Radius is the bend radius), the angle
// is the full angle between the two flat faces the cylinder joins, and the direction is whether the
// centre of curvature lies on the outward (concave → UP) or inward (convex → DOWN) side of the
// reference flat (the flat sharing the picked edge). So the note re-resolves with the model.

// AddBendNote adds a bend note on the named base view from an edge of the bend's cylindrical face. It
// errors when the edge does not belong to a cylindrical face joining two flats.
func (as *DrawingAnnotations) AddBendNote(name, viewName string, bendEdge []byte) (*DrawingAnnotation, error) {
	if _, _, _, err := as.annotationBasis(viewName); err != nil {
		return nil, err
	}
	a := &DrawingAnnotation{name: as.uniqueName(name), kind: types.BendNoteAnnotation, viewName: viewName, edgeKey: bendEdge}
	as.recomputeBendNote(a)
	if len(a.labels) == 0 {
		return nil, fmt.Errorf("drawing: edge does not bound a cylindrical bend in view %q", viewName)
	}
	as.items = append(as.items, a)
	return a, nil
}

// recomputeBendNote re-derives the bend callout from the bend edge; with no resolvable cylindrical
// bend it clears the glyph.
func (as *DrawingAnnotations) recomputeBendNote(a *DrawingAnnotation) {
	a.curves, a.labels = nil, nil
	view, body, basis, err := as.annotationBasis(a.viewName)
	if err != nil {
		return
	}
	radiusCm, angleDeg, dir, anchor, ok := bendMetricsFromBody(body, a.edgeKey)
	if !ok {
		return
	}
	text := fmt.Sprintf("%s° R%s %s", trimAngleDeg(angleDeg), holeCoord(radiusCm*cmToMM), dir)
	p := view.place(hlr.ProjectPoint(basis, anchor))
	a.curves, a.labels = featureNoteCallout(float64(p.X), float64(p.Y), text)
}

// bendMetricsFromBody resolves the cylindrical bend face on the edge, the two flats it joins and the
// reference flat, returning the bend radius (cm), angle (deg), direction (UP/DOWN) and 3D anchor.
func bendMetricsFromBody(body *topo.Body, edgeKey []byte) (radiusCm, angleDeg float64, dir string, anchor math.Point3, ok bool) {
	edge, found := body.FindEdgeByKey(edgeKey)
	if !found {
		return
	}
	cylFace, refFlat := bendFaceAndReferenceFlat(edge)
	if cylFace == nil || refFlat == nil {
		return
	}
	cyl := cylFace.Geometry().(geom.Cylinder)
	flats := adjacentFlats(cylFace)
	if len(flats) < 2 {
		return
	}
	angleDeg, ok = flatPairAngleDeg(flats)
	if !ok {
		return
	}
	anchor = edgeMidpoint(edge)
	return cyl.Radius, angleDeg, bendDirection(cyl, anchor, refFlat), anchor, true
}

// bendFaceAndReferenceFlat splits an edge's two faces into its cylindrical bend face and its planar
// reference flat; returns nils when the edge is not a cylinder/flat junction.
func bendFaceAndReferenceFlat(edge *topo.Edge) (cyl, flat *topo.Face) {
	for _, f := range edge.Faces() {
		switch f.Geometry().(type) {
		case geom.Cylinder:
			cyl = f
		case geom.Plane:
			flat = f
		}
	}
	return cyl, flat
}

// adjacentFlats lists the distinct planar faces joined to the bend cylinder along its STRAIGHT edges
// (the bend lines). It ignores the cylinder's arc edges, whose neighbours are the end-cap faces, not
// the flats the bend joins — so a fillet between two faces reports exactly those two.
func adjacentFlats(c *topo.Face) []*topo.Face {
	seen := map[*topo.Face]bool{}
	var flats []*topo.Face
	for _, e := range c.Edges() {
		if _, straight := e.Geometry().(geom.LineSegment); !straight {
			continue
		}
		for _, f := range e.Faces() {
			if f == c || seen[f] {
				continue
			}
			if _, ok := f.Geometry().(geom.Plane); ok {
				seen[f] = true
				flats = append(flats, f)
			}
		}
	}
	return flats
}

// flatPairAngleDeg is the full angle (0..180°) between the outward normals of the first two flats —
// the sheet-metal bend angle (90° for a right-angle bend, 180° for a hem).
func flatPairAngleDeg(flats []*topo.Face) (float64, bool) {
	n1, ok1 := planeOutwardNormal(flats[0])
	n2, ok2 := planeOutwardNormal(flats[1])
	if !ok1 || !ok2 {
		return 0, false
	}
	u1, e1 := math.UnitVector3FromVector(n1)
	u2, e2 := math.UnitVector3FromVector(n2)
	if e1 != nil || e2 != nil {
		return 0, false
	}
	dot := stdmath.Max(-1, stdmath.Min(1, float64(u1.AsVector().Dot(u2.AsVector()))))
	return stdmath.Acos(dot) * 180 / stdmath.Pi, true
}

// bendDirection reports UP when the bend's centre of curvature lies on the outward-normal side of the
// reference flat (a concave/valley bend), DOWN when on the inward side (a convex/mountain bend).
func bendDirection(cyl geom.Cylinder, surfacePoint math.Point3, refFlat *topo.Face) string {
	n, ok := planeOutwardNormal(refFlat)
	if !ok {
		return "UP"
	}
	axis := cyl.AxisDir.AsVector()
	toOrigin := surfacePoint.VectorTo(cyl.Origin)
	toCentre := toOrigin.Sub(axis.Scale(toOrigin.Dot(axis))) // surface point → nearest axis point
	if toCentre.Dot(n) > 0 {
		return "UP"
	}
	return "DOWN"
}

// planeOutwardNormal returns a planar face's outward normal (its plane normal, flipped when the face
// is reversed), or ok=false when the face is not planar.
func planeOutwardNormal(f *topo.Face) (math.Vector3, bool) {
	pl, ok := f.Geometry().(geom.Plane)
	if !ok {
		return math.Vector3{}, false
	}
	n := pl.Normal()
	if f.Reversed() {
		n = n.Negate()
	}
	return n, true
}

// edgeMidpoint is a straight or curved edge's endpoint midpoint (a placement anchor).
func edgeMidpoint(e *topo.Edge) math.Point3 {
	box := e.RangeBox()
	return box.Min.Midpoint(box.Max)
}
