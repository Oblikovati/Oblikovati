// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Crossing-rod abstraction (M2 Phase 2, Oblikovati/Oblikovati#1335). The cut/join stub builders
// (curved_crossing_cut.go, curved_crossing_join.go) assemble a fat cylinder breached by a "rod" crossing it.
// The rod is a cylinder when two cylinders cross, but a CONE when a tapered rod crosses a cylinder — and the
// only places the surface type matters are the band/stub faces (the rod's analytic surface), its axis (for
// angle math and the two end-cap centres), and the radius of its end circle at a given cap. crossRod exposes
// exactly those, so the same drill/join/stub assembly serves a cone rod without duplication. This mirrors the
// crossAxis split that let the imprint/classify stage classify a cone rod (curved_crossing_intersect.go).

// crossRod is the rod operand of a crossing boolean — a cylinder or a cone — exposed through just what the
// stub builders need: its analytic surface (for the band faces), its axis (direction + the crossAxis for
// angle math), and the radius of its end circle at a given cap centre.
type crossRod interface {
	surface() geom.Surface
	axisUnit() math.UnitVector3
	axisVec() math.Vector3
	axisOf() crossAxis
	endRadius(center math.Point3) float64
}

// cylinderRod adapts a geom.Cylinder as a crossRod: its end circle has the cylinder's constant radius.
type cylinderRod struct{ c geom.Cylinder }

func (r cylinderRod) surface() geom.Surface         { return r.c }
func (r cylinderRod) axisUnit() math.UnitVector3    { return r.c.AxisDir }
func (r cylinderRod) axisVec() math.Vector3         { return r.c.AxisDir.AsVector() }
func (r cylinderRod) axisOf() crossAxis             { return cylAxis(r.c) }
func (r cylinderRod) endRadius(math.Point3) float64 { return r.c.Radius }

// coneRod adapts a geom.Cone as a crossRod: its end circle radius is v·tan(HalfAngle), where v is the cap
// centre's distance from the apex along the axis — so a frustum's two ends carry different radii.
type coneRod struct{ c geom.Cone }

func (r coneRod) surface() geom.Surface      { return r.c }
func (r coneRod) axisUnit() math.UnitVector3 { return r.c.AxisDir }
func (r coneRod) axisVec() math.Vector3      { return r.c.AxisDir.AsVector() }
func (r coneRod) axisOf() crossAxis          { return coneAxis(r.c) }

func (r coneRod) endRadius(center math.Point3) float64 {
	v := float64(r.c.Apex.VectorTo(center).Dot(r.c.AxisDir.AsVector()))
	return v * stdmath.Tan(r.c.HalfAngle)
}

// rodCirclePoint returns the point on the rod's circle at a given end centre and axial angle: ref·cos u +
// binormal·sin u at the end's radius — the frame convention both geom.Cylinder and geom.Cone use, so it
// lands on the rod surface for either. Used to seat a seam vertex on the rod and to sample an end disc.
func rodCirclePoint(rod crossRod, center math.Point3, angle float64) math.Point3 {
	ax := rod.axisOf()
	radial := ax.ref.Scale(stdmath.Cos(angle)).Add(ax.dir.Cross(ax.ref).Scale(stdmath.Sin(angle)))
	return center.TranslateBy(radial.Scale(rod.endRadius(center)))
}
