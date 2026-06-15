// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/math"
)

// MoveData is the serialized form of a MoveFeature: the target body index and either a
// single transform (the 16 row-major cells, the legacy form) or an ordered operation
// list (M20-F20). Exactly one of Matrix/Ops is populated; an op list takes precedence
// when present so the separately driven operations survive the round trip.
type MoveData struct {
	Body   int          `yaml:"body"`
	Matrix []float64    `yaml:"matrix,omitempty"`
	Ops    []MoveOpData `yaml:"ops,omitempty"`
}

// MoveOpData is one serialized move operation. Only the fields its kind uses are written.
type MoveOpData struct {
	Kind   string    `yaml:"kind"`
	Offset []float64 `yaml:"offset,omitempty"` // freeDrag [x,y,z]
	Dir    []float64 `yaml:"dir,omitempty"`    // alongRay / rotate direction [x,y,z]
	Dist   float64   `yaml:"dist,omitempty"`   // alongRay distance
	Point  []float64 `yaml:"point,omitempty"`  // rotate axis point [x,y,z]
	Angle  float64   `yaml:"angle,omitempty"`  // rotate angle (radians)
}

// serializeMove projects a move recipe to its persisted form (op list when present,
// else the baked transform).
func serializeMove(def *MoveDefinition) *MoveData {
	if len(def.Ops) > 0 {
		return &MoveData{Body: def.BodyIndex, Ops: encodeMoveOps(def.Ops)}
	}
	cells := def.Transform.Cells()
	return &MoveData{Body: def.BodyIndex, Matrix: cells[:]}
}

// encodeMoveOps freezes each operation's parametric scalars to their current value.
func encodeMoveOps(operations []MoveOperation) []MoveOpData {
	out := make([]MoveOpData, len(operations))
	for i, op := range operations {
		out[i] = encodeMoveOp(op)
	}
	return out
}

// encodeMoveOp serializes the fields the operation's kind uses.
func encodeMoveOp(op MoveOperation) MoveOpData {
	switch op.Kind {
	case types.MoveFreeDrag:
		return MoveOpData{Kind: string(op.Kind), Offset: []float64{evalFloat(op.X), evalFloat(op.Y), evalFloat(op.Z)}}
	case types.MoveAlongRay:
		return MoveOpData{Kind: string(op.Kind), Dir: moveVec3Cells(op.Dir), Dist: evalFloat(op.Dist)}
	default: // MoveRotateAboutLine
		return MoveOpData{Kind: string(op.Kind), Dir: moveVec3Cells(op.AxisDir), Point: movePoint3Cells(op.AxisPoint), Angle: evalFloat(op.Angle)}
	}
}

// restoreMove rebuilds a MoveFeature, erroring on a missing payload or a malformed move.
func restoreMove(fs *PartFeatures, d *MoveData) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("move feature is missing its payload")
	}
	m := NewModifyFeatures(fs)
	if len(d.Ops) > 0 {
		operations, err := decodeMoveOps(d.Ops)
		if err != nil {
			return nil, err
		}
		return m.AddMoveOps(d.Body, operations), nil
	}
	if len(d.Matrix) != 16 {
		return nil, fmt.Errorf("move feature matrix has %d cells, want 16", len(d.Matrix))
	}
	var cells [16]float64
	copy(cells[:], d.Matrix)
	return m.AddMove(d.Body, math.Matrix4FromCells(cells)), nil
}

// decodeMoveOps rebuilds the operation list with constant closures (a reloaded file's
// scalars are frozen values; an in-session edit re-parametrizes them).
func decodeMoveOps(data []MoveOpData) ([]MoveOperation, error) {
	out := make([]MoveOperation, len(data))
	for i, od := range data {
		op, err := decodeMoveOp(od)
		if err != nil {
			return nil, err
		}
		out[i] = op
	}
	return out, nil
}

// decodeMoveOp rebuilds one operation, erroring on an unknown kind.
func decodeMoveOp(od MoveOpData) (MoveOperation, error) {
	kind := types.MoveOperationType(od.Kind)
	switch kind {
	case types.MoveFreeDrag:
		o := moveVec3(od.Offset)
		return FreeDragOp(constFloat(o.X), constFloat(o.Y), constFloat(o.Z)), nil
	case types.MoveAlongRay:
		return AlongRayOp(moveVec3(od.Dir), constFloat(od.Dist)), nil
	case types.MoveRotateAboutLine:
		return RotateAboutLineOp(movePoint3(od.Point), moveVec3(od.Dir), constFloat(od.Angle)), nil
	default:
		return MoveOperation{}, fmt.Errorf("move operation has unknown kind %q (want freeDrag/alongRay/rotateAboutLine)", od.Kind)
	}
}

// moveVec3Cells / movePoint3Cells flatten a vector/point to [x,y,z]; moveVec3 /
// movePoint3 are their inverses, tolerating a short/nil slice as the zero vector/point.
func moveVec3Cells(v math.Vector3) []float64  { return []float64{v.X, v.Y, v.Z} }
func movePoint3Cells(p math.Point3) []float64 { return []float64{p.X, p.Y, p.Z} }

func moveVec3(a []float64) math.Vector3 {
	if len(a) < 3 {
		return math.Vector3{}
	}
	return math.V3(a[0], a[1], a[2])
}

func movePoint3(a []float64) math.Point3 {
	if len(a) < 3 {
		return math.Point3{}
	}
	return math.P3(a[0], a[1], a[2])
}
