// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/math"
	"oblikovati.org/model/param"
	"oblikovati.org/model/sketch"
)

// Bend Part feature (M20-F17, #651): fold a solid part body around a sketch bend line, the
// only solid-deformation feature in the part-modeling set (distinct from sheet-metal bends,
// which need the sheet-metal environment). The geometry is the planar-faceted bend in
// bend_part_geom.go; this is the Definition→Add→Feature triangle over it.

// BendPartDefinition is the bend recipe: the sketch + the index of its bend line, the bend
// type (which two of radius/angle/arc-length drive it), the three parametric scalars, and
// Flip to fold toward the opposite side of the sketch plane.
type BendPartDefinition struct {
	Sketch    *sketch.Sketch
	LineIndex int
	BendType  types.BendPartType
	Radius    func() float64
	Angle     func() float64
	ArcLength func() float64
	Flip      bool
}

// BendPartFeature folds the running solid each recompute.
type BendPartFeature struct {
	def      *BendPartDefinition
	featName string
}

// Definition returns the bend recipe.
func (b *BendPartFeature) Definition() *BendPartDefinition { return b.def }

// Kind implements [Feature].
func (b *BendPartFeature) Kind() string { return "bend-part" }

// Recompute folds the running body about the bend line and replaces it with the result.
func (b *BendPartFeature) Recompute(in Input) (Output, error) {
	body, err := lastBody(in, "bend-part")
	if err != nil {
		return Output{}, err
	}
	point, dir, up, err := b.bendLine()
	if err != nil {
		return Output{}, err
	}
	radius, angle, err := bendRadiusAngle(b.def)
	if err != nil {
		return Output{}, err
	}
	bent, err := bendSolid(body, point, dir, up, radius, angle, featOr(b.featName, "bend"))
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: replaceBody(in.Bodies, body, bent)}, nil
}

// bendLine resolves the bend line's model-space point, direction, and the up-normal (the
// sketch plane normal, flipped when Flip is set) the moving flange folds toward.
func (b *BendPartFeature) bendLine() (point math.Point3, dir, up math.Vector3, err error) {
	sk := b.def.Sketch
	if sk == nil {
		return point, dir, up, fmt.Errorf("bend-part: no sketch set")
	}
	lines := sk.Lines()
	if b.def.LineIndex < 0 || b.def.LineIndex >= lines.Count() {
		return point, dir, up, fmt.Errorf("bend-part: line index %d out of range (%d lines)", b.def.LineIndex, lines.Count())
	}
	l := lines.Item(b.def.LineIndex)
	pl := sk.Plane()
	p0, p1 := pl.ToModel(l.StartPoint().Position()), pl.ToModel(l.EndPoint().Position())
	up = pl.Normal().AsVector()
	if b.def.Flip {
		up = up.Scale(-1)
	}
	return p0, p0.VectorTo(p1), up, nil
}

// bendRadiusAngle derives the bend radius and angle from the two driving inputs of the bend
// type (the third is computed): angle = arcLength/radius, or radius = arcLength/angle.
func bendRadiusAngle(def *BendPartDefinition) (radius, angle float64, err error) {
	switch def.BendType {
	case types.RadiusAndAngleBend:
		radius, angle = callOrZero(def.Radius), callOrZero(def.Angle)
	case types.RadiusAndArcLengthBend:
		radius = callOrZero(def.Radius)
		if radius <= 0 {
			return 0, 0, fmt.Errorf("bend-part: radius %.4g must be > 0", radius)
		}
		angle = callOrZero(def.ArcLength) / radius
	case types.ArcLengthAndAngleBend:
		angle = callOrZero(def.Angle)
		if angle <= 0 {
			return 0, 0, fmt.Errorf("bend-part: angle %.4g must be > 0", angle)
		}
		radius = callOrZero(def.ArcLength) / angle
	default:
		return 0, 0, fmt.Errorf("bend-part: unknown bend type %v", def.BendType)
	}
	if radius <= 0 || angle <= 0 {
		return 0, 0, fmt.Errorf("bend-part: radius %.4g and angle %.4g must both be > 0", radius, angle)
	}
	return radius, angle, nil
}

// EditableParams exposes the two scalars that drive the bend type for features.edit.
func (b *BendPartFeature) EditableParams() []EditableParam {
	switch b.def.BendType {
	case types.RadiusAndArcLengthBend:
		return []EditableParam{
			scalarParam("Radius", param.Length, &b.def.Radius),
			scalarParam("Arc Length", param.Length, &b.def.ArcLength),
		}
	case types.ArcLengthAndAngleBend:
		return []EditableParam{
			scalarParam("Arc Length", param.Length, &b.def.ArcLength),
			scalarParam("Angle", param.Angle, &b.def.Angle),
		}
	default: // RadiusAndAngleBend
		return []EditableParam{
			scalarParam("Radius", param.Length, &b.def.Radius),
			scalarParam("Angle", param.Angle, &b.def.Angle),
		}
	}
}

// BendPartFeatures adds bend features into the engine.
type BendPartFeatures struct{ engine *PartFeatures }

// NewBendPartFeatures binds the collection to a feature engine.
func NewBendPartFeatures(engine *PartFeatures) *BendPartFeatures { return &BendPartFeatures{engine} }

// Add places a bend feature, naming it Bend1, Bend2, ….
func (c *BendPartFeatures) Add(def *BendPartDefinition) *PartFeature {
	f := &BendPartFeature{def: def}
	pf := c.engine.Add(f)
	pf.SetName(c.engine.UniqueName("Bend"))
	f.featName = pf.Name()
	return pf
}
