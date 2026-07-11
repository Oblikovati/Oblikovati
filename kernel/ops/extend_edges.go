// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"errors"
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Multi-edge and extend-to-plane surface extend (#1878). ExtendByEdge grows one boundary edge of a
// planar face by a distance; these grow SEVERAL boundary edges of the same planar face in one
// feature, and can grow each edge until it reaches a target plane (Inventor's extend-to-object).
// A vertex shared by two extended edges takes both displacements. Multi-face / curved (NURBS)
// surfaces remain phase C, exactly as ExtendByEdge.

// ExtendEdgesByDistance extends each selected boundary edge of a single planar surface face
// outward by distance, growing the face.
func ExtendEdgesByDistance(body *topo.Body, edgeKeys [][]byte, distance float64, feat string) (*topo.Body, error) {
	return extendEdges(body, edgeKeys, feat, func(pl geom.Plane, a, b, c math.Point3) (math.Vector3, math.Vector3) {
		s := extendDir(pl.Normal(), a, b, c).Scale(distance)
		return s, s
	})
}

// ExtendEdgesToPlane extends each selected boundary edge until its endpoints reach target: each
// endpoint slides along the in-plane extend direction to the plane. An endpoint whose extend
// direction is parallel to the plane, or that would move backward to reach it, stays put.
func ExtendEdgesToPlane(body *topo.Body, edgeKeys [][]byte, target geom.Plane, feat string) (*topo.Body, error) {
	return extendEdges(body, edgeKeys, feat, func(pl geom.Plane, a, b, c math.Point3) (math.Vector3, math.Vector3) {
		dir := extendDir(pl.Normal(), a, b, c)
		return reachPlane(a, dir, target), reachPlane(b, dir, target)
	})
}

// extendEdges resolves the selected boundary edges of one planar face, accumulates each edge's
// endpoint displacement (from shift) at the face's polygon vertices, and rebuilds the grown sheet.
func extendEdges(body *topo.Body, edgeKeys [][]byte, feat string, shift func(pl geom.Plane, a, b, c math.Point3) (math.Vector3, math.Vector3)) (*topo.Body, error) {
	face, err := singleExtendFace(body, edgeKeys)
	if err != nil {
		return nil, err
	}
	pl := face.Geometry().(geom.Plane)
	poly := facePolygon(face)
	c := centroid(poly)
	disp := make([]math.Vector3, len(poly))
	for _, key := range edgeKeys {
		edge, _ := body.FindEdgeByKey(key) // resolved in singleExtendFace
		a, b := edge.StartVertex().Point(), edge.EndVertex().Point()
		da, db := shift(pl, a, b, c)
		addDispAt(poly, disp, a, da)
		addDispAt(poly, disp, b, db)
	}
	moved := make([]math.Point3, len(poly))
	for i, p := range poly {
		moved[i] = p.TranslateBy(disp[i])
	}
	return buildSheet([]sheetPatch{{poly: moved, normal: pl.Normal()}}, feat), nil
}

// singleExtendFace resolves every edge key and returns the single planar face they all bound, or an
// error (a lost key, a non-boundary/shared edge, edges on different faces, or a curved face — the
// phase-C cases, matching ExtendByEdge).
func singleExtendFace(body *topo.Body, edgeKeys [][]byte) (*topo.Face, error) {
	if len(edgeKeys) == 0 {
		return nil, errors.New("extend: no edges")
	}
	var face *topo.Face
	for _, key := range edgeKeys {
		edge, ok := body.FindEdgeByKey(key)
		if !ok {
			return nil, errors.New("extend: edge reference lost")
		}
		faces := edge.Faces()
		if len(faces) != 1 {
			return nil, errors.New("extend: edge is not a single-face boundary edge")
		}
		if face == nil {
			face = faces[0]
		} else if faces[0] != face {
			return nil, errors.New("extend: edges lie on different faces")
		}
	}
	if _, planar := face.Geometry().(geom.Plane); !planar {
		return nil, errors.New("extend: curved surface not supported")
	}
	return face, nil
}

// addDispAt adds shift to the displacement of the polygon vertex coincident with p.
func addDispAt(poly []math.Point3, disp []math.Vector3, p math.Point3, shift math.Vector3) {
	for i, q := range poly {
		if coincidentPt(q, p) {
			disp[i] = disp[i].Add(shift)
		}
	}
}

// reachPlane returns the displacement that slides p along dir until it lies on target, or the zero
// vector when dir is parallel to the plane or the plane is behind p (no forward reach).
func reachPlane(p math.Point3, dir math.Vector3, target geom.Plane) math.Vector3 {
	den := float64(dir.Dot(target.Normal()))
	if stdmath.Abs(den) < 1e-12 {
		return math.Vector3{}
	}
	t := float64(target.Normal().Dot(p.VectorTo(target.Origin))) / den
	if t <= 0 {
		return math.Vector3{}
	}
	return dir.Scale(t)
}
