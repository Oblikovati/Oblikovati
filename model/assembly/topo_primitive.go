// SPDX-License-Identifier: GPL-2.0-only

package assembly

import (
	"fmt"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The topology boundary: turn a picked face, edge, or vertex of a component's body into a
// constraint [Primitive] in that component's definition space. The host (router) resolves a
// reference key to the topology entity, then calls these; the engine itself stays
// topology-agnostic. A planar face becomes a plane, a cylindrical face an axis with radius,
// a straight edge an axis, a vertex a point — the geometry constraints actually consume.

// PrimitiveFromFace extracts a constraint input from a face: a plane (planar face) or an
// axis carrying the radius (cylindrical face). It errors for surface kinds a constraint
// cannot yet consume, naming the kind.
func PrimitiveFromFace(f *topo.Face) (Primitive, error) {
	switch surf := f.Geometry().(type) {
	case geom.Plane:
		n, err := math.UnitVector3FromVector(surf.Normal())
		if err != nil {
			return Primitive{}, fmt.Errorf("assembly: planar face has a zero normal: %w", err)
		}
		// Carry the plane's U axis as the part-tied secondary so rigid/slider joint origins
		// built from this face can lock the roll about the normal (M12-F02).
		return PlanePrimitive(surf.Origin, n).withSecondary(surf.UAxis), nil
	case geom.Cylinder:
		// The cylinder's reference axis is the part-tied secondary for joint roll locking.
		return CylinderPrimitive(surf.Origin, surf.AxisDir, surf.Radius).withSecondary(surf.Ref), nil
	default:
		return Primitive{}, fmt.Errorf("assembly: unsupported face surface %T for a constraint input", surf)
	}
}

// PrimitiveFromEdge extracts an axis input from a straight edge (line or line segment),
// erroring for curved edges, which a positioning constraint does not yet take.
func PrimitiveFromEdge(e *topo.Edge) (Primitive, error) {
	switch c := e.Geometry().(type) {
	case geom.Line:
		return LinePrimitive(c.Origin, c.Dir), nil
	case geom.LineSegment:
		dir, err := math.UnitVector3FromVector(c.StartPoint.VectorTo(c.EndPoint))
		if err != nil {
			return Primitive{}, fmt.Errorf("assembly: degenerate edge of zero length: %w", err)
		}
		return LinePrimitive(c.StartPoint, dir), nil
	default:
		return Primitive{}, fmt.Errorf("assembly: unsupported edge curve %T for a constraint input", c)
	}
}

// PrimitiveFromVertex extracts a point input from a vertex.
func PrimitiveFromVertex(v *topo.Vertex) Primitive {
	return PointPrimitive(v.Point())
}
