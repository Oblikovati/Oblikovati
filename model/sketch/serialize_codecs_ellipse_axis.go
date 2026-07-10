// SPDX-License-Identifier: GPL-2.0-only

package sketch

import "fmt"

// Codecs for the ellipse-axis relation constraints (#1879). Each stores its operand entity ids
// in Curves and their per-operand major-axis flags in AxisMajor (whether an operand is an
// ellipse is re-derived from the resolved entity's type); the horizontal/vertical forms store
// only the ellipse operand — the world axis is implied by the kind.

// registerEllipseAxisConstraintCodecs is called from the constraint-codec init() (not its own,
// to keep registration off the import-linkage path — archguard B6/#1617).
func registerEllipseAxisConstraintCodecs() {
	registerConstraintCodec(EllipseParallelKind, axisRelationCodec(func(g *GeometricConstraints, a, b AxisOperand) {
		g.AddEllipseParallel(a, b)
	}))
	registerConstraintCodec(EllipsePerpendicularKind, axisRelationCodec(func(g *GeometricConstraints, a, b AxisOperand) {
		g.AddEllipsePerpendicular(a, b)
	}))
	registerConstraintCodec(EllipseCollinearKind, axisRelationCodec(func(g *GeometricConstraints, a, b AxisOperand) {
		g.AddEllipseCollinear(a, b)
	}))
}

// encodeAxisOperands stores each real operand's entity id + major flag (a world axis names no
// entity and is skipped — the kind implies it).
func encodeAxisOperands(c Constraint) (ConstraintData, error) {
	v := c.(*AxisRelationConstraint)
	var curves []int
	var majors []bool
	for _, op := range []AxisOperand{v.a, v.b} {
		if e := op.operandEntity(); e != nil {
			curves = append(curves, int(e.EntityID()))
			majors = append(majors, op.axisMajor())
		}
	}
	return ConstraintData{Curves: curves, AxisMajor: majors}, nil
}

// axisRelationCodec pairs a two-operand ellipse-axis relation with its factory.
func axisRelationCodec(add func(*GeometricConstraints, AxisOperand, AxisOperand)) constraintCodec {
	return constraintCodec{
		encode: encodeAxisOperands,
		decode: func(r *sketchRestorer, cd ConstraintData) error {
			a, err := r.axisOperand(cd, 0)
			if err != nil {
				return err
			}
			b, err := r.axisOperand(cd, 1)
			if err != nil {
				return err
			}
			add(r.s.geomCons, a, b)
			return nil
		},
	}
}

// axisOperand resolves the i-th stored operand back to a line or ellipse-axis operand, reading
// its major flag from AxisMajor.
func (r *sketchRestorer) axisOperand(cd ConstraintData, i int) (AxisOperand, error) {
	e, err := r.entity(cd.Curves, i)
	if err != nil {
		return nil, err
	}
	major := i < len(cd.AxisMajor) && cd.AxisMajor[i]
	if op, ok := EllipseAxisOf(e, major); ok {
		return op, nil
	}
	l, ok := e.(*Line)
	if !ok {
		return nil, fmt.Errorf("axis-relation operand %d (%T) is neither a line nor an ellipse", i, e)
	}
	return LineAxis(l), nil
}
