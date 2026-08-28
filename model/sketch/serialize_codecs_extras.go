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
	// A projected curve is a concrete grounded reference entity driven by a Projection (ADR-0055
	// phase 3), not an entity with a Kind(); it is encoded from the sketch's projection list by
	// serializeProjection, so this codec's encode half is an unreachable programming-error guard,
	// and its decode rebuilds the reference entity and re-registers the frozen projection.
	registerEntityCodec(ProjectedCurveKind, entityCodec{
		encode: func(_ Entity) (EntityData, error) {
			return EntityData{}, fmt.Errorf("projectedCurve is serialized via serializeProjection, not the entity codec (ADR-0055)")
		},
		decode: func(r *sketchRestorer, ed EntityData) (Entity, error) {
			return r.restoreProjectedCurve(ed)
		},
	})
}

// projectionOwnedIDs is the set of entity ids owned by a projection — the concrete reference
// entities serializeSketch skips in the normal entity loop because they persist through their
// Projection instead (ADR-0055 phase 3).
func projectionOwnedIDs(s *Sketch) map[ID]bool {
	out := make(map[ID]bool, len(s.projections))
	for _, pr := range s.projections {
		out[pr.Entity().EntityID()] = true
	}
	return out
}

// serializeProjection encodes one curve projection as a projectedCurve row: the driven reference
// entity's id and defining-point ids (so constraints referencing them round-trip), the source
// descriptor (for rebind), and the entity's exact analytic form — line/circle/arc/ellipse — as a
// compact shape+params, or its polyline for a non-analytic reference spline (ADR-0055).
func serializeProjection(pr *Projection) EntityData {
	ent := pr.Entity()
	kind, id := pr.SourceDescriptor()
	ed := EntityData{
		ID:         int(ent.EntityID()),
		Kind:       string(ProjectedCurveKind),
		Points:     pointIDsOf(DefiningPoints(ent)),
		Source:     id,
		SourceKind: kind,
	}
	if c2, ok := entityCurve2(ent); ok {
		if shape, params, ok := analyticCurveData(c2); ok {
			ed.ProjShape, ed.ProjParams = shape, params
			return ed
		}
	}
	ed.Coords = flattenPoints(definingPointPositions(ent))
	return ed
}

// definingPointPositions returns the positions of an entity's defining points, in DefiningPoints
// order — the polyline of a reference spline for persistence.
func definingPointPositions(ent Entity) []math.Point2 {
	dps := DefiningPoints(ent)
	out := make([]math.Point2, len(dps))
	for i, p := range dps {
		out[i] = p.Position()
	}
	return out
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
