// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"oblikovati.org/api/types"
	"oblikovati.org/math"
	"oblikovati.org/model/param"
)

// Edit Feature scalar coverage for the surfacing/freeform/M09 features (#704): the
// features.get/edit wire surface and the head's Edit Feature dialog render whatever
// EditableParams exposes, so a feature without an implementation cannot be edited
// after placement at all.

// floatParam builds a scalar param over a plain float64 field (its address) — the
// counterpart of scalarParam for definitions that store a value, not a closure.
func floatParam(label string, u param.Unit, field *float64) EditableParam {
	return EditableParam{
		Label: label, Unit: u,
		Get: func() float64 { return *field },
		Set: func(v float64) { *field = v },
	}
}

// EditableParams exposes the ruled band's distance.
func (r *RuledSurfaceFeature) EditableParams() []EditableParam {
	return []EditableParam{scalarParam("Distance", param.Length, &r.def.Distance)}
}

// EditableParams exposes the surface-offset distance.
func (o *SurfaceOffsetFeature) EditableParams() []EditableParam {
	return []EditableParam{scalarParam("Distance", param.Length, &o.def.Distance)}
}

// EditableParams exposes the mid-surface wall-pairing threshold.
func (m *MidSurfaceFeature) EditableParams() []EditableParam {
	return []EditableParam{floatParam("Max thickness", param.Length, &m.def.MaxThickness)}
}

// EditableParams exposes the stitch weld tolerance.
func (s *StitchFeature) EditableParams() []EditableParam {
	return []EditableParam{floatParam("Tolerance", param.Length, &s.def.Tolerance)}
}

// EditableParams exposes the sculpt enclosure tolerance.
func (s *SculptFeature) EditableParams() []EditableParam {
	return []EditableParam{floatParam("Tolerance", param.Length, &s.def.Tolerance)}
}

// EditableParams exposes the extend distance.
func (e *ExtendFeature) EditableParams() []EditableParam {
	return []EditableParam{scalarParam("Distance", param.Length, &e.def.Distance)}
}

// EditableParams exposes the freeform body's subdivision level (clamped at 0 by SetLevel).
func (f *FreeformFeature) EditableParams() []EditableParam {
	return []EditableParam{{
		Label: "Level", Unit: param.Unitless, Integer: true,
		Get: func() float64 { return float64(f.body.Level()) },
		Set: func(v float64) { f.body.SetLevel(int(v + 0.5)) },
	}}
}

// EditableParams exposes the mold parting position and shrinkage allowance.
func (m *CoreCavityFeature) EditableParams() []EditableParam {
	return []EditableParam{
		scalarParam("Parting position", param.Length, &m.def.Position),
		floatParam("Shrinkage", param.Unitless, &m.def.Shrinkage),
	}
}

// EditableParams exposes the boss stud's diameter and height.
func (b *BossFeature) EditableParams() []EditableParam {
	return []EditableParam{
		scalarParam("Diameter", param.Length, &b.def.Diameter),
		scalarParam("Height", param.Length, &b.def.Height),
	}
}

// EditableParams exposes the thicken wall thickness.
func (f *ThickenFeature) EditableParams() []EditableParam {
	return []EditableParam{scalarParam("Thickness", param.Length, &f.thickness)}
}

// EditableParams exposes the face-offset distance.
func (f *FaceOffsetFeature) EditableParams() []EditableParam {
	return []EditableParam{scalarParam("Distance", param.Length, &f.distance)}
}

// EditableParams exposes a move-face's displacement magnitude (translate mode) or its
// rotation angle (rotate mode, #331).
func (f *MoveFaceFeature) EditableParams() []EditableParam {
	if _, _, _, rotating := f.Rotation(); rotating {
		return []EditableParam{scalarParam("Angle", param.Angle, &f.angle)}
	}
	return []EditableParam{spacingParam("Distance", &f.translation, math.V3(0, 0, 1))}
}

// EditableParams exposes the direct edit's scalar by operation: the push/pull distance
// (size), the rotation angle, the scale factor, or the move displacement magnitude.
// Delete has no scalar input.
func (f *DirectEditFeature) EditableParams() []EditableParam {
	switch f.def.Operation {
	case types.DirectEditSizeOperation:
		return []EditableParam{scalarParam("Distance", param.Length, &f.def.Distance)}
	case types.DirectEditRotateOperation:
		return []EditableParam{scalarParam("Angle", param.Angle, &f.def.Angle)}
	case types.DirectEditScaleOperation:
		return []EditableParam{scalarParam("Scale factor", param.Unitless, &f.def.ScaleFactor)}
	case types.DirectEditMoveOperation:
		return []EditableParam{spacingParam("Distance", &f.def.Translation, math.V3(0, 0, 1))}
	default:
		return nil
	}
}
