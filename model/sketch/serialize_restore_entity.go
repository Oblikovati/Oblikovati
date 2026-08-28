// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/math"
)

// Sketch recipe restore — POINTS and ENTITIES (M48 #2244 split of serialize_restore.go). Rebuilds the
// sketch points and entities (lines/arcs/circles/splines, projected points/curves) from their recipe
// rows, re-binding each to its constituent points by id. The shared reference re-binding helpers live in
// serialize_restore.go; the constraints/dimensions in serialize_restore_constraint.go.

func (r *sketchRestorer) restorePoints(points []PointData) {
	for _, pd := range points {
		r.pointMap[pd.ID] = r.restorePoint(pd)
	}
}

// restorePoint recreates one point (standalone SketchPoint or a curve-owned point) and
// pins its persisted local id.
func (r *sketchRestorer) restorePoint(pd PointData) *Point {
	pos := math.P2(pd.X, pd.Y)
	p := r.newPointFor(pd.Standalone, pos)
	p.SetCenterPoint(pd.CenterPoint) // hole-centre marker (#2015)
	r.pin(p, pd.ID)
	return p
}

// newPointFor creates a standalone SketchPoint or a curve-owned solver point, exactly one.
func (r *sketchRestorer) newPointFor(standalone bool, pos math.Point2) *Point {
	if standalone {
		return r.s.points.Add(pos)
	}
	return r.s.newPoint(pos)
}

func (r *sketchRestorer) restoreEntities(entities []EntityData) error {
	for _, ed := range entities {
		e, err := r.restoreEntity(ed)
		if err != nil {
			return err
		}
		// Projected reference entities (#1268) carry no construction flag and pin their own ids,
		// so each post-step is optional rather than a hard type-assert.
		if sc, ok := e.(interface{ SetConstruction(bool) }); ok {
			sc.SetConstruction(ed.Construction)
		}
		if ed.Centerline {
			if cl, ok := e.(interface{ SetCenterline(bool) }); ok {
				cl.SetCenterline(true)
			}
		}
		if ic, ok := e.(idCarrier); ok {
			r.pin(ic, ed.ID)
		}
		r.s.readEntityFormat(ed, e.EntityID()) // per-entity format overrides (#2015)
		r.entityMap[ed.ID] = e
	}
	return nil
}

// restoreEntity rebuilds one entity through its kind's registered codec — the
// same pairing its serializeEntity encode came from, so the two can never
// drift (#1624). An unknown kind is a hard error: a recipe never restores
// partially.
func (r *sketchRestorer) restoreEntity(ed EntityData) (Entity, error) {
	c, ok := entityCodecs2D[EntityKind(ed.Kind)]
	if !ok {
		return nil, fmt.Errorf("unknown entity kind %q", ed.Kind)
	}
	return c.decode(r, ed)
}

// restoreProjectedPoint rebuilds a frozen projected point and registers its anchor in the point
// map so constraints referencing the anchor restore, pinning the anchor id (#1268).
func (r *sketchRestorer) restoreProjectedPoint(ed EntityData) (Entity, error) {
	if len(ed.Points) < 1 || len(ed.Anchor) < 2 {
		return nil, fmt.Errorf("projectedPoint needs an anchor id and a 2-component anchor")
	}
	pos := math.P2(math.Scalar(ed.Anchor[0]), math.Scalar(ed.Anchor[1]))
	pp := r.s.RestoreProjectedPoint(ID(ed.Points[0]), pos, ed.SourceKind, ed.Source)
	r.pointMap[ed.Points[0]] = pp.Anchor()
	r.note(ed.Points[0])
	r.note(ed.ID)
	return pp, nil
}

// restoreProjectedCurve rebuilds a projected curve as its concrete grounded reference entity
// (ADR-0055 phase 3): it re-creates the reference Line/Circle/Arc/Ellipse/Spline from the persisted
// analytic form (or polyline), re-registers the frozen Projection that drives it, and pins the
// entity's defining points to their saved ids in the point map so constraints referencing them
// restore. The entity's own id is pinned by restoreEntities' generic post-step.
func (r *sketchRestorer) restoreProjectedCurve(ed EntityData) (Entity, error) {
	ent, ok := r.s.buildReferenceEntity(ed.ProjShape, ed.ProjParams, ed.Coords)
	if !ok {
		return nil, fmt.Errorf("projectedCurve %d: cannot rebuild reference entity (shape %q, %d params, %d coords)",
			ed.ID, ed.ProjShape, len(ed.ProjParams), len(ed.Coords))
	}
	r.s.RestoreProjection(ent, ed.SourceKind, ed.Source)
	dps := DefiningPoints(ent)
	if len(ed.Points) != len(dps) {
		return nil, fmt.Errorf("projectedCurve %d: %d point ids for a %d-point reference entity", ed.ID, len(ed.Points), len(dps))
	}
	for i, p := range dps {
		p.setID(ID(ed.Points[i]))
		r.pointMap[ed.Points[i]] = p
		r.note(ed.Points[i])
	}
	r.note(ed.ID)
	return ent, nil
}

// restoreSplineExtras rebuilds a spline's fit method and active tangency
// handles (M06-F11, #626).
func restoreSplineExtras(s *Sketch, sp *Spline, ed EntityData) error {
	if ed.FitMethod != "" {
		m, ok := types.ParseSplineFitMethod(ed.FitMethod)
		if !ok {
			return fmt.Errorf("unknown spline fit method %q (want smooth|sweet|chord)", ed.FitMethod)
		}
		sp.FitMethod = m
	}
	for _, hd := range ed.Handles {
		h, err := s.splineHandles.Activate(sp, hd.FitIndex)
		if err != nil {
			return err
		}
		h.End.SetPosition(math.P2(math.Scalar(hd.EndX), math.Scalar(hd.EndY)))
	}
	return nil
}

// restorePlane rebuilds a sketch plane from its serialized origin and axes.
func restorePlane(pd PlaneData) (Plane, error) {
	x, err := math.UnitVector3FromVector(vector3(pd.XAxis))
	if err != nil {
		return Plane{}, fmt.Errorf("plane x-axis: %w", err)
	}
	y, err := math.UnitVector3FromVector(vector3(pd.YAxis))
	if err != nil {
		return Plane{}, fmt.Errorf("plane y-axis: %w", err)
	}
	return NewPlane(math.P3(pd.Origin[0], pd.Origin[1], pd.Origin[2]), x, y)
}
