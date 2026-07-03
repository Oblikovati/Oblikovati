// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/sketch"
)

// Spline tangency handles over the wire (M06-F11, #626):
// sketch.setSplineHandle and sketch3d.setSplineHandle activate, edit or
// deactivate the handle on one fit point.

// setSplineHandle serves wire.MethodSketchSetSplineHandle.
func setSplineHandle(_ *app.Session, part *compdef.PartComponentDefinition, in wire.SetSplineHandleArgs) (wire.SplineHandleInfo, error) {
	sk, err := sketchAtIndex(part, in.SketchIndex)
	if err != nil {
		return wire.SplineHandleInfo{}, err
	}
	sp, err := splineByID(sk, in.Spline)
	if err != nil {
		return wire.SplineHandleInfo{}, err
	}
	if !in.Active {
		sk.SplineHandles().Deactivate(sp, in.FitPointIndex)
		return wire.SplineHandleInfo{}, nil
	}
	h, err := sk.SplineHandles().Activate(sp, in.FitPointIndex)
	if err != nil {
		return wire.SplineHandleInfo{}, err
	}
	if err := applyHandleEdit(h, in); err != nil {
		return wire.SplineHandleInfo{}, err
	}
	return splineHandleInfo(h), nil
}

// applyHandleEdit applies the optional tangent/weight edit to a 2D handle.
func applyHandleEdit(h *sketch.SplineHandle, in wire.SetSplineHandleArgs) error {
	if len(in.Tangent) == 2 {
		return h.SetTangent(math.V2(math.Scalar(in.Tangent[0]), math.Scalar(in.Tangent[1])), in.Weight)
	}
	if len(in.Tangent) != 0 {
		return fmt.Errorf("a 2D spline handle tangent needs [x, y], got %d components", len(in.Tangent))
	}
	if in.Weight != 0 {
		dir, ok := h.TangentDirection()
		if !ok {
			return fmt.Errorf("spline handle %d is degenerate; set a tangent direction first", h.EntityID())
		}
		return h.SetTangent(dir, in.Weight)
	}
	return nil
}

// splineHandleInfo renders a 2D handle as its wire DTO.
func splineHandleInfo(h *sketch.SplineHandle) wire.SplineHandleInfo {
	info := wire.SplineHandleInfo{HandleID: uint64(h.EntityID()), Weight: h.Weight()}
	if dir, ok := h.TangentDirection(); ok {
		info.Tangent = []float64{float64(dir.X), float64(dir.Y)}
	}
	return info
}

// splineByID resolves a 2D spline entity by session id.
func splineByID(sk *sketch.Sketch, id uint64) (*sketch.Spline, error) {
	e, ok := sk.EntityByID(sketch.ID(id))
	if !ok {
		return nil, fmt.Errorf("entity %d not found", id)
	}
	sp, isSpline := e.(*sketch.Spline)
	if !isSpline {
		return nil, fmt.Errorf("entity %d is %T, want a spline", id, e)
	}
	return sp, nil
}

// setSplineHandle3D serves wire.MethodSketch3DSetSplineHandle.
func setSplineHandle3D(_ *app.Session, part *compdef.PartComponentDefinition, in wire.SetSplineHandleArgs) (wire.SplineHandleInfo, error) {
	sk, err := sketch3DAtIndex(part, in.SketchIndex)
	if err != nil {
		return wire.SplineHandleInfo{}, err
	}
	sp, err := spline3DByID(sk, in.Spline)
	if err != nil {
		return wire.SplineHandleInfo{}, err
	}
	if !in.Active {
		sk.DeactivateSplineHandle3D(sp, in.FitPointIndex)
		return wire.SplineHandleInfo{}, nil
	}
	h, err := sk.ActivateSplineHandle3D(sp, in.FitPointIndex)
	if err != nil {
		return wire.SplineHandleInfo{}, err
	}
	if err := applyHandleEdit3D(h, in); err != nil {
		return wire.SplineHandleInfo{}, err
	}
	return splineHandle3DInfo(h), nil
}

// applyHandleEdit3D applies the optional tangent/weight edit to a 3D handle.
func applyHandleEdit3D(h *sketch.SplineHandle3D, in wire.SetSplineHandleArgs) error {
	if len(in.Tangent) == 3 {
		return h.SetTangent(math.V3(
			math.Scalar(in.Tangent[0]), math.Scalar(in.Tangent[1]), math.Scalar(in.Tangent[2])), in.Weight)
	}
	if len(in.Tangent) != 0 {
		return fmt.Errorf("a 3D spline handle tangent needs [x, y, z], got %d components", len(in.Tangent))
	}
	if in.Weight != 0 {
		dir, ok := h.TangentDirection()
		if !ok {
			return fmt.Errorf("spline handle %d is degenerate; set a tangent direction first", h.EntityID())
		}
		return h.SetTangent(dir, in.Weight)
	}
	return nil
}

// splineHandle3DInfo renders a 3D handle as its wire DTO.
func splineHandle3DInfo(h *sketch.SplineHandle3D) wire.SplineHandleInfo {
	info := wire.SplineHandleInfo{HandleID: uint64(h.EntityID()), Weight: h.Weight()}
	if dir, ok := h.TangentDirection(); ok {
		info.Tangent = []float64{float64(dir.X), float64(dir.Y), float64(dir.Z)}
	}
	return info
}

// spline3DByID resolves a 3D spline entity by session id.
func spline3DByID(sk *sketch.Sketch3D, id uint64) (*sketch.Spline3D, error) {
	e, ok := sk.EntityByID(sketch.ID(id))
	if !ok {
		return nil, fmt.Errorf("entity %d not found", id)
	}
	sp, isSpline := e.(*sketch.Spline3D)
	if !isSpline {
		return nil, fmt.Errorf("entity %d is %T, want a 3D spline", id, e)
	}
	return sp, nil
}
