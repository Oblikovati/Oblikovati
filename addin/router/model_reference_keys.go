// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"

	"oblikovati.org/addin/modelaccess"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// referenceKeys surfaces the active part's topology (faces/edges/vertices) with their
// persistent reference keys. Over the wire there is no viewport pick, so this is how an
// add-in obtains a key to feed back into the key consumers (include, surface curves,
// project geometry, attributes). Each entry carries a representative point so the caller
// can recognise which face/edge/vertex it is.
func referenceKeys(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	var out wire.ReferenceKeysResult
	for _, b := range part.SurfaceBodies().All() {
		out.Bodies = append(out.Bodies, bodyTopology(b))
	}
	return json.Marshal(out)
}

// bodyTopology collects one body's faces/edges/vertices as reference keys + representative
// points (face range-box centre, edge midpoint, vertex position).
func bodyTopology(b *topo.Body) wire.BodyTopology {
	var bt wire.BodyTopology
	for _, f := range b.Faces() {
		bt.Faces = append(bt.Faces, wire.TopologyRef{Key: string(f.ReferenceKey()), Point: topoRefPoint(f.RangeBox().Center()), Kind: surfaceKind(f)})
	}
	for _, e := range b.Edges() {
		bt.Edges = append(bt.Edges, wire.TopologyRef{Key: string(e.ReferenceKey()), Point: topoRefPoint(e.RangeBox().Center())})
	}
	for _, v := range b.Vertices() {
		bt.Vertices = append(bt.Vertices, wire.TopologyRef{Key: string(v.ReferenceKey()), Point: topoRefPoint(v.Point())})
	}
	return bt
}

// surfaceKind classifies a face's surface so a caller can pick by geometry (e.g. the
// cylindrical face to thread).
func surfaceKind(f *topo.Face) string {
	switch f.Geometry().(type) {
	case geom.Plane:
		return "plane"
	case geom.Cylinder:
		return "cylinder"
	case geom.Cone:
		return "cone"
	case geom.Sphere:
		return "sphere"
	case geom.Torus:
		return "torus"
	default:
		return "spline"
	}
}

// topoRefPoint flattens a point to [x,y,z].
func topoRefPoint(p math.Point3) []float64 {
	return []float64{float64(p.X), float64(p.Y), float64(p.Z)}
}
