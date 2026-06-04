// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"fmt"

	"github.com/Oblikovati/oblikovati/math"
	"github.com/Oblikovati/oblikovati/model/param"
)

// DimensionConstraint3D sizes a 3D sketch, backed by a model parameter like its 2D
// counterpart. It implements [Constraint], so the dimension-agnostic Newton core
// (F05) solves 3D sketches unchanged.
type DimensionConstraint3D struct {
	constraintBase
	kind    DimKind
	driven  bool
	param   *param.Parameter
	measure func() float64
	vars    []*math.Scalar
}

// Parameter returns the backing parameter; Measured returns the live value.
func (d *DimensionConstraint3D) Parameter() *param.Parameter { return d.param }
func (d *DimensionConstraint3D) Measured() float64           { return d.measure() }

// KindName returns the wire/types name of this 3D dimension's kind
// ([github.com/Oblikovati/api/types.Dimension3DConstraintKind]). The set grows with the
// 3D dimension factories (M22-F06); unmapped kinds report "unknown".
func (d *DimensionConstraint3D) KindName() string {
	switch d.kind {
	case DistanceDim:
		return "distance"
	case RadiusDim:
		return "radius"
	case AngleDim:
		return "twoLineAngle"
	default:
		return "unknown"
	}
}

// Driven reports/sets whether the dimension only reports rather than constrains.
func (d *DimensionConstraint3D) Driven() bool          { return d.driven }
func (d *DimensionConstraint3D) SetDriven(driven bool) { d.driven = driven }

// Residuals returns measured-minus-target when driving, nil when driven.
func (d *DimensionConstraint3D) Residuals() []float64 {
	if d.driven {
		return nil
	}
	return []float64{d.measure() - d.param.ModelValue()}
}

// Variables returns the 3D DOFs this dimension constrains.
func (d *DimensionConstraint3D) Variables() []*math.Scalar {
	if d.driven {
		return nil
	}
	return d.vars
}

// DimensionConstraints3D owns a 3D sketch's dimensions and their parameters.
type DimensionConstraints3D struct {
	params *param.Parameters
	items  []*DimensionConstraint3D
	seq    int
}

// NewDimensionConstraints3D creates a 3D dimension collection backed by params.
func NewDimensionConstraints3D(params *param.Parameters) *DimensionConstraints3D {
	return &DimensionConstraints3D{params: params}
}

// All returns the 3D dimensions; Count/Item index them.
func (dc *DimensionConstraints3D) All() []*DimensionConstraint3D {
	out := make([]*DimensionConstraint3D, len(dc.items))
	copy(out, dc.items)
	return out
}
func (dc *DimensionConstraints3D) Count() int                        { return len(dc.items) }
func (dc *DimensionConstraints3D) Item(i int) *DimensionConstraint3D { return dc.items[i] }

// AddDistance dimensions the distance between two 3D points.
func (dc *DimensionConstraints3D) AddDistance(a, b *Point3D, expression string) (*DimensionConstraint3D, error) {
	p, err := dc.params.AddModelParameter(dc.nextName(), expression)
	if err != nil {
		return nil, fmt.Errorf("sketch: 3D dimension parameter: %w", err)
	}
	d := &DimensionConstraint3D{
		constraintBase: newConstraint(),
		kind:           DistanceDim,
		param:          p,
		measure:        func() float64 { return a.Position().DistanceTo(b.Position()) },
		vars:           []*math.Scalar{&a.X, &a.Y, &a.Z, &b.X, &b.Y, &b.Z},
	}
	dc.items = append(dc.items, d)
	return d, nil
}

func (dc *DimensionConstraints3D) nextName() string {
	for {
		name := fmt.Sprintf("d3_%d", dc.seq)
		dc.seq++
		if _, taken := dc.params.ByName(name); !taken {
			return name
		}
	}
}
