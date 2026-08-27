// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"fmt"

	"oblikovati.org/math"
)

// Codec registrations for the non-curve 2D entities: annotations (image/text/
// fillRegion), block instances, frozen projections, and the M21 derived curves
// (equation/fixed/offset spline). Bodies are the former switch cases of
// serializeEntity/serializeDerivedCurve and restoreEntity/restoreDerivedCurve,
// paired per kind (#1624) — including the derived-curve default: branch that
// used to fail a save at runtime.

func init() {
	registerEntityCodec(BlockInstanceKind, entityCodec{
		encode: func(e Entity) (EntityData, error) {
			v := e.(*BlockInstance)
			return EntityData{ID: int(v.id), Block: v.def.name, Transform: matrixCells(v.transform)}, nil
		},
		decode: func(r *sketchRestorer, ed EntityData) (Entity, error) {
			def, ok := r.blockDefs.ByName(ed.Block)
			if !ok {
				return nil, fmt.Errorf("block instance references unknown definition %q", ed.Block)
			}
			t, err := matrixFromCells(ed.Transform)
			if err != nil {
				return nil, err
			}
			return r.s.Blocks().Insert(def, t), nil
		},
	})
	registerEntityCodec(ImageKind, entityCodec{
		encode: func(e Entity) (EntityData, error) {
			v := e.(*SketchImage)
			return EntityData{
				ID: int(v.id), ImageRef: v.Ref,
				Anchor:   []float64{float64(v.Anchor.X), float64(v.Anchor.Y)},
				Size:     []float64{float64(v.Width), float64(v.Height)},
				Rotation: float64(v.Rotation), Opacity: v.Opacity,
			}, nil
		},
		decode: func(r *sketchRestorer, ed EntityData) (Entity, error) {
			if len(ed.Anchor) != 2 || len(ed.Size) != 2 {
				return nil, fmt.Errorf("image needs a 2-component anchor and size")
			}
			anchor := math.P2(math.Scalar(ed.Anchor[0]), math.Scalar(ed.Anchor[1]))
			return r.s.images.Add(ed.ImageRef, anchor, math.Scalar(ed.Size[0]), math.Scalar(ed.Size[1]), math.Scalar(ed.Rotation), ed.Opacity), nil
		},
	})
	registerEntityCodec(TextKind, entityCodec{
		encode: func(e Entity) (EntityData, error) {
			v := e.(*TextBox)
			return EntityData{
				ID: int(v.id), Text: v.Text,
				Anchor:     []float64{float64(v.Anchor.X), float64(v.Anchor.Y)},
				TextHeight: float64(v.Height), Rotation: float64(v.Rotation),
				Justify: int(v.Justify), VJustify: int(v.VJustify),
				FontFamily: v.Family, FontResource: v.FontResource, FontSize: float64(v.FontSize),
			}, nil
		},
		decode: func(r *sketchRestorer, ed EntityData) (Entity, error) {
			if len(ed.Anchor) != 2 {
				return nil, fmt.Errorf("text needs a 2-component anchor")
			}
			anchor := math.P2(math.Scalar(ed.Anchor[0]), math.Scalar(ed.Anchor[1]))
			tb := r.s.texts.AddStyled(anchor, ed.Text, math.Scalar(ed.TextHeight), math.Scalar(ed.Rotation),
				TextHJustify(ed.Justify), TextVJustify(ed.VJustify), ed.FontFamily, math.Scalar(ed.FontSize))
			tb.FontResource = ed.FontResource
			return tb, nil
		},
	})
	registerEntityCodec(FillRegionKind, entityCodec{
		encode: func(e Entity) (EntityData, error) {
			v := e.(*FillRegion)
			return EntityData{ID: int(v.id), Seed: []float64{float64(v.Seed.X), float64(v.Seed.Y)}, Style: v.Style}, nil
		},
		decode: func(r *sketchRestorer, ed EntityData) (Entity, error) {
			if len(ed.Seed) != 2 {
				return nil, fmt.Errorf("fillRegion needs a 2-component seed")
			}
			return r.s.fills.Add(math.P2(math.Scalar(ed.Seed[0]), math.Scalar(ed.Seed[1])), ed.Style), nil
		},
	})
	registerProjectionCodecs()
	registerDerivedCurveCodecs()
}

// registerProjectionCodecs pairs the frozen projected reference entities (#1268):
// they pin their own ids and re-attach to their model source on recompute.
func registerProjectionCodecs() {
	registerEntityCodec(ProjectedPointKind, entityCodec{
		encode: func(e Entity) (EntityData, error) {
			v := e.(*ProjectedPoint)
			kind, id := v.SourceDescriptor()
			return EntityData{
				ID: int(v.anchor.id), Points: []int{int(v.anchor.id)},
				Anchor: []float64{float64(v.anchor.X), float64(v.anchor.Y)}, Source: id, SourceKind: kind,
			}, nil
		},
		decode: func(r *sketchRestorer, ed EntityData) (Entity, error) {
			return r.restoreProjectedPoint(ed)
		},
	})
	registerEntityCodec(ProjectedCurveKind, entityCodec{
		encode: func(e Entity) (EntityData, error) {
			v := e.(*ProjectedCurve)
			kind, id := v.SourceDescriptor()
			// Store the analytic curve (a few floats) not the 17-point polyline (ADR-0055); a
			// non-analytic projection still stores its coords.
			if curve, ok := v.AnalyticCurve(); ok {
				if shape, params, ok := analyticCurveData(curve); ok {
					return EntityData{ID: int(v.id), ProjShape: shape, ProjParams: params, Source: id, SourceKind: kind}, nil
				}
			}
			return EntityData{ID: int(v.id), Coords: flattenPoints(v.points), Source: id, SourceKind: kind}, nil
		},
		decode: func(r *sketchRestorer, ed EntityData) (Entity, error) {
			var c *ProjectedCurve
			if curve, ok := analyticCurveFromData(ed.ProjShape, ed.ProjParams); ok {
				c = r.s.RestoreProjectedCurveAnalytic(ID(ed.ID), curve, ed.SourceKind, ed.Source)
			} else {
				c = r.s.RestoreProjectedCurve(ID(ed.ID), unflattenPoints(ed.Coords), ed.SourceKind, ed.Source)
			}
			r.note(ed.ID)
			return c, nil
		},
	})
}

// registerDerivedCurveCodecs pairs the M21 derived curves. An offset spline's
// parent must already be restored (entities restore in recipe order).
func registerDerivedCurveCodecs() {
	registerEntityCodec(EquationCurveKind, entityCodec{
		encode: func(e Entity) (EntityData, error) {
			v := e.(*EquationCurve)
			return EntityData{ID: int(v.id), XExpr: v.XExpr, YExpr: v.YExpr, T0: v.T0, T1: v.T1}, nil
		},
		decode: func(r *sketchRestorer, ed EntityData) (Entity, error) {
			return r.s.eqCurves.Add(ed.XExpr, ed.YExpr, ed.T0, ed.T1)
		},
	})
	registerEntityCodec(FixedSplineKind, entityCodec{
		encode: func(e Entity) (EntityData, error) {
			v := e.(*FixedSpline)
			return EntityData{ID: int(v.id), Coords: flattenPoints(v.Pts)}, nil
		},
		decode: func(r *sketchRestorer, ed EntityData) (Entity, error) {
			return r.s.fixedSpl.Add(unflattenPoints(ed.Coords)), nil
		},
	})
	registerEntityCodec(OffsetSplineKind, entityCodec{
		encode: func(e Entity) (EntityData, error) {
			v := e.(*OffsetSpline)
			return EntityData{ID: int(v.id), ParentID: int(v.Parent.id), OffsetDist: v.Dist}, nil
		},
		decode: func(r *sketchRestorer, ed EntityData) (Entity, error) {
			parent, ok := r.entityMap[ed.ParentID].(*Spline)
			if !ok {
				return nil, fmt.Errorf("offsetSpline parent %d is not a spline", ed.ParentID)
			}
			return r.s.offSpl.Add(parent, ed.OffsetDist), nil
		},
	})
}
