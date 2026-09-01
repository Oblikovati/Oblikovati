// SPDX-License-Identifier: GPL-2.0-only

// Package shotscene holds the small scene-building helpers shared by the head's throwaway
// live-capture drivers (previewshot, filletgateshot, …): building a base box, adding a square
// sketch, finding a box's vertical edge, and framing the camera on an edge. Extracted so the
// drivers do not each re-declare the same geometry scaffolding.
package shotscene

import (
	stdmath "math"

	"oblikovati.org/app"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
	"oblikovati.org/scene"
)

// NewPart adds and activates an empty part named name, returning its definition.
func NewPart(s *app.Session, name string) *compdef.PartComponentDefinition {
	pd, err := compdef.AddPart(s.Workspace(), name, true)
	if err != nil {
		panic(err)
	}
	_ = s.Workspace().SetActiveDocument(pd)
	return pd.Content().(*compdef.PartComponentDefinition)
}

// BuildBox commits a side×side×height base solid in a fresh part (its corner at the origin).
func BuildBox(s *app.Session, name string, side, height float64) *compdef.PartComponentDefinition {
	def := NewPart(s, name)
	sk := AddSquare(def, 0, 0, side)
	feature.NewExtrudeFeatures(def.Features()).AddByDistanceExtent(sk, 0, ops.NewBody, func() float64 { return height })
	def.Recompute()
	return def
}

// AddSquare adds a side×side square sketch on the XY plane with its corner at (ox,oy).
func AddSquare(def *compdef.PartComponentDefinition, ox, oy, side float64) *sketch.Sketch {
	sk := def.Sketches().Add(sketch.XYPlane())
	c0 := sk.Points().Add(math.P2(ox, oy))
	c1 := sk.Points().Add(math.P2(ox+side, oy))
	c2 := sk.Points().Add(math.P2(ox+side, oy+side))
	c3 := sk.Points().Add(math.P2(ox, oy+side))
	sk.Lines().Add(c0, c1)
	sk.Lines().Add(c1, c2)
	sk.Lines().Add(c2, c3)
	sk.Lines().Add(c3, c0)
	return sk
}

// AimCameraAtEdge frames body from outside edge e (looking at its midpoint along the outward
// direction, tilted down), so a thin edge wedge faces the camera rather than presenting edge-on.
func AimCameraAtEdge(s *app.Session, body *topo.Body, e *topo.Edge) {
	pts := tessellate.TessellateEdge(e, ops.DefaultQuality())
	mid := pts[len(pts)/2]
	rb := body.RangeBox()
	cx, cy := (rb.Min.X+rb.Max.X)/2, (rb.Min.Y+rb.Max.Y)/2
	ox, oy := mid.X-cx, mid.Y-cy
	n := stdmath.Hypot(ox, oy)
	if n == 0 {
		n = 1
	}
	dist := (rb.Max.X - rb.Min.X) * 2.4
	cam := scene.NewCamera(1280, 800)
	cam.Target = mid
	cam.Eye = math.P3(mid.X+ox/n*dist, mid.Y+oy/n*dist, mid.Z+dist*0.5)
	cam.Up = math.V3(0, 0, 1)
	s.SetCamera(cam)
}

// VerticalEdge returns the first edge running mostly along Z (a box's vertical corner).
func VerticalEdge(edges []*topo.Edge) *topo.Edge {
	for _, e := range edges {
		pts := tessellate.TessellateEdge(e, ops.DefaultQuality())
		if len(pts) < 2 {
			continue
		}
		a, b := pts[0], pts[len(pts)-1]
		dz := stdmath.Abs(a.Z - b.Z)
		if dz > stdmath.Abs(a.X-b.X) && dz > stdmath.Abs(a.Y-b.Y) {
			return e
		}
	}
	return edges[0]
}
