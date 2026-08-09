// SPDX-License-Identifier: GPL-2.0-only

package ipt

import (
	"encoding/binary"
	"math"
)

// The revolve axis is a SketchLine3D (node type revolveAxisNodeType 0x8EF06C89, named at the
// Revolution feature's property 2). InventorLoader's Read_8EF06C89 decodes its tail as pos(3 f64) +
// dir(3 f64) + a u8 — a point and a direction in MODEL space. The heuristic revolveAxisIndex finds
// the centreline only when it is an isolated or construction line in the profile sketch; when the
// axis is instead an ordinary profile EDGE (CapstainMotorCap turns about its y=0 top edge, which is
// neither isolated nor construction) the heuristic returns nothing and this reference supplies the
// true axis.

// axisLine3DTail is the fixed byte count of a SketchLine3D's pos(3)+dir(3)+u8 payload tail.
const axisLine3DTail = 6*8 + 1

// RevolveAxis2D returns the revolve's axis as a point and a direction in the PROFILE SKETCH's 2D
// plane (the plane RevolveProfileSketch names), by projecting the decoded model-space SketchLine3D
// onto that plane's axes. ok=false when the part has no revolve axis node, no resolvable profile
// sketch, or that sketch states no readable plane — callers then fall back to the geometric
// heuristic. The direction is not normalised; callers use its orientation, not its length.
func RevolveAxis2D(d *Document) (ox, oy, dx, dy float64, ok bool) {
	nodes := dcNodes(d)
	pos, dir, ok := revolveAxisLine3D(nodes)
	if !ok {
		return 0, 0, 0, 0, false
	}
	idx, ok := RevolveProfileSketch(d)
	if !ok {
		return 0, 0, 0, 0, false
	}
	g := GraphSketches(d)
	if idx < 0 || idx >= len(g) || !g[idx].PlaneOK {
		return 0, 0, 0, 0, false
	}
	pl := g[idx].Plane
	ox, oy = projectOntoPlane(pos, pl)
	// A direction is a free vector: project the displacement (pos+dir) minus the projected origin,
	// i.e. project dir against the plane axes without the plane origin offset.
	dx = dot(dir, pl.XAxis)
	dy = dot(dir, pl.YAxis)
	if math.Hypot(dx, dy) < 1e-9 {
		return 0, 0, 0, 0, false // axis is normal to the profile plane — not an in-plane centreline
	}
	return ox, oy, dx, dy, true
}

// revolveAxisLine3D decodes the Revolution feature's axis SketchLine3D into its model-space point and
// direction, reading the pos/dir pair anchored at the payload tail (robust to the variable header).
func revolveAxisLine3D(nodes []dcNode) (pos, dir [3]float64, ok bool) {
	for _, n := range nodes {
		if n.typ != featureNodeType {
			continue
		}
		if _, isExtrude := extrudeDepth(nodes, n.payload); isExtrude {
			continue
		}
		props, ok := featureProperties(n.payload)
		if !ok || len(props) <= propDirection || !isRevolveFeature(nodes, props) {
			continue
		}
		ax, ok := nodeAt(nodes, props[propDirection])
		if !ok || len(ax.payload) < axisLine3DTail {
			return pos, dir, false
		}
		b := len(ax.payload) - axisLine3DTail
		f := func(o int) float64 { return math.Float64frombits(binary.LittleEndian.Uint64(ax.payload[b+o:])) }
		return [3]float64{f(0), f(8), f(16)}, [3]float64{f(24), f(32), f(40)}, true
	}
	return pos, dir, false
}

// projectOntoPlane maps a model-space point to the plane's 2D coordinates (distance along X and Y
// from the plane origin).
func projectOntoPlane(p [3]float64, pl SketchPlacement) (x, y float64) {
	rel := [3]float64{p[0] - pl.Origin[0], p[1] - pl.Origin[1], p[2] - pl.Origin[2]}
	return dot(rel, pl.XAxis), dot(rel, pl.YAxis)
}

func dot(a, b [3]float64) float64 { return a[0]*b[0] + a[1]*b[1] + a[2]*b[2] }
