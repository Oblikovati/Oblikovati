// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"

	"github.com/Oblikovati/api/types"
	"github.com/Oblikovati/api/wire"

	"github.com/Oblikovati/oblikovati/app"
	"github.com/Oblikovati/oblikovati/model/sketch"
)

// addSketch3DConstraint is the discriminated 3D geometric-constraint constructor.
func addSketch3DConstraint(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.AddSketch3DConstraintArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	sk, err := activeSketch3DAt(s, in.SketchIndex)
	if err != nil {
		return nil, err
	}
	c, err := buildSketch3DConstraint(sk, types.Geometric3DConstraintKind(in.Kind), in.Entities)
	if err != nil {
		return nil, err
	}
	sk.GeometricConstraints3D().Add(c)
	return json.Marshal(wire.AddSketch3DConstraintResult{
		Index: sk.GeometricConstraints3D().Count() - 1, Kind: in.Kind, DOF: sk.DegreesOfFreedom(),
	})
}

// buildSketch3DConstraint resolves the operands and applies the matching constraint factory.
func buildSketch3DConstraint(sk *sketch.Sketch3D, kind types.Geometric3DConstraintKind, refs []uint64) (sketch.Constraint, error) {
	switch kind {
	case types.Geo3DCoincident, types.Geo3DCollinear, types.Geo3DConcentric:
		return pointConstraint3D(sk, kind, refs)
	case types.Geo3DParallel, types.Geo3DPerpendicular:
		return linePairConstraint3D(sk, kind, refs)
	case types.Geo3DMidpoint:
		return midpointConstraint3D(sk, refs)
	case types.Geo3DGround:
		return groundConstraint3D(sk, refs)
	default:
		return orientationConstraint3D(sk, kind, refs)
	}
}

// pointConstraint3D builds the point-operand constraints (coincident/collinear/concentric).
func pointConstraint3D(sk *sketch.Sketch3D, kind types.Geometric3DConstraintKind, refs []uint64) (sketch.Constraint, error) {
	switch kind {
	case types.Geo3DCoincident:
		p, err := points3D(sk, refs, 2)
		if err != nil {
			return nil, err
		}
		return sketch.NewCoincident3D(p[0], p[1]), nil
	case types.Geo3DCollinear:
		p, err := points3D(sk, refs, 3)
		if err != nil {
			return nil, err
		}
		return sketch.NewCollinear3D(p[0], p[1], p[2]), nil
	default: // concentric
		p, err := points3D(sk, refs, 2)
		if err != nil {
			return nil, err
		}
		return sketch.NewConcentric3D(p[0], p[1]), nil
	}
}

// linePairConstraint3D builds the two-line constraints (parallel/perpendicular).
func linePairConstraint3D(sk *sketch.Sketch3D, kind types.Geometric3DConstraintKind, refs []uint64) (sketch.Constraint, error) {
	l, err := lines3D(sk, refs, 2)
	if err != nil {
		return nil, err
	}
	if kind == types.Geo3DParallel {
		return sketch.NewParallel3D(l[0], l[1]), nil
	}
	return sketch.NewPerpendicular3D(l[0], l[1]), nil
}

// midpointConstraint3D builds the point-on-line-midpoint constraint.
func midpointConstraint3D(sk *sketch.Sketch3D, refs []uint64) (sketch.Constraint, error) {
	if len(refs) != 2 {
		return nil, fmt.Errorf("sketch3d.addConstraint: midpoint needs a point + line, got %d refs", len(refs))
	}
	p, err := pointRef3D(sk, refs[0])
	if err != nil {
		return nil, err
	}
	l, err := lineRef3D(sk, refs[1])
	if err != nil {
		return nil, err
	}
	return sketch.NewMidpoint3D(p, l), nil
}

// groundConstraint3D builds the fix-a-point constraint.
func groundConstraint3D(sk *sketch.Sketch3D, refs []uint64) (sketch.Constraint, error) {
	p, err := points3D(sk, refs, 1)
	if err != nil {
		return nil, err
	}
	return sketch.NewGround3D(p[0]), nil
}

// orientationConstraint3D builds the parallel-to-axis/plane constraints.
func orientationConstraint3D(sk *sketch.Sketch3D, kind types.Geometric3DConstraintKind, refs []uint64) (sketch.Constraint, error) {
	l, err := lines3D(sk, refs, 1)
	if err != nil {
		return nil, err
	}
	switch kind {
	case types.Geo3DParallelToXAxis:
		return sketch.NewParallelToXAxis3D(l[0]), nil
	case types.Geo3DParallelToYAxis:
		return sketch.NewParallelToYAxis3D(l[0]), nil
	case types.Geo3DParallelToZAxis:
		return sketch.NewParallelToZAxis3D(l[0]), nil
	case types.Geo3DParallelToXYPlane:
		return sketch.NewParallelToXYPlane3D(l[0]), nil
	case types.Geo3DParallelToXZPlane:
		return sketch.NewParallelToXZPlane3D(l[0]), nil
	case types.Geo3DParallelToYZPlane:
		return sketch.NewParallelToYZPlane3D(l[0]), nil
	default:
		return nil, fmt.Errorf("sketch3d.addConstraint: unsupported kind %q", kind)
	}
}

// deleteSketch3DConstraint removes a geometric constraint by index.
func deleteSketch3DConstraint(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.DeleteSketch3DConstraintArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	sk, err := activeSketch3DAt(s, in.SketchIndex)
	if err != nil {
		return nil, err
	}
	g := sk.GeometricConstraints3D()
	if in.ConstraintIndex < 0 || in.ConstraintIndex >= g.Count() {
		return nil, fmt.Errorf("sketch3d.deleteConstraint: index %d out of range (%d constraints)", in.ConstraintIndex, g.Count())
	}
	g.Delete(g.Item(in.ConstraintIndex))
	return json.Marshal(wire.OKResult{OK: true})
}

// points3D resolves exactly n point refs.
func points3D(sk *sketch.Sketch3D, refs []uint64, n int) ([]*sketch.Point3D, error) {
	if len(refs) != n {
		return nil, fmt.Errorf("sketch3d.addConstraint: need %d point refs, got %d", n, len(refs))
	}
	out := make([]*sketch.Point3D, n)
	for i, id := range refs {
		p, err := pointRef3D(sk, id)
		if err != nil {
			return nil, err
		}
		out[i] = p
	}
	return out, nil
}

// lines3D resolves exactly n line refs.
func lines3D(sk *sketch.Sketch3D, refs []uint64, n int) ([]*sketch.Line3D, error) {
	if len(refs) != n {
		return nil, fmt.Errorf("sketch3d.addConstraint: need %d line refs, got %d", n, len(refs))
	}
	out := make([]*sketch.Line3D, n)
	for i, id := range refs {
		l, err := lineRef3D(sk, id)
		if err != nil {
			return nil, err
		}
		out[i] = l
	}
	return out, nil
}

// pointRef3D resolves a session id to a 3D point (a standalone point or a curve endpoint).
func pointRef3D(sk *sketch.Sketch3D, id uint64) (*sketch.Point3D, error) {
	if p, ok := sk.PointByID(sketch.ID(id)); ok {
		return p, nil
	}
	return nil, fmt.Errorf("sketch3d: no 3D point with id %d", id)
}

// lineRef3D resolves a session id to a 3D line entity.
func lineRef3D(sk *sketch.Sketch3D, id uint64) (*sketch.Line3D, error) {
	e, ok := sk.EntityByID(sketch.ID(id))
	if !ok {
		return nil, fmt.Errorf("sketch3d: no entity with id %d", id)
	}
	l, ok := e.(*sketch.Line3D)
	if !ok {
		return nil, fmt.Errorf("sketch3d: entity %d is %T, want a 3D line", id, e)
	}
	return l, nil
}
