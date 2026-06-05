// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Oblikovati/api/types"
	"github.com/Oblikovati/api/wire"

	"github.com/Oblikovati/oblikovati/addin/modelaccess"
	"github.com/Oblikovati/oblikovati/app"
	"github.com/Oblikovati/oblikovati/math"
	"github.com/Oblikovati/oblikovati/model/compdef"
	"github.com/Oblikovati/oblikovati/model/feature"
	"github.com/Oblikovati/oblikovati/model/param"
)

// listWorkPlanes enumerates the active part's datum planes (origin frame first, then
// user planes) with their current geometry and health.
func listWorkPlanes(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	planes := part.WorkPlanes()
	out := make([]wire.WorkPlaneInfo, planes.Count())
	for i := 0; i < planes.Count(); i++ {
		wp := planes.Item(i)
		out[i] = wire.WorkPlaneInfo{
			Index:    i,
			Name:     wp.Name(),
			Ref:      string(wp.Key()),
			Origin:   point3Slice(wp.Plane().Origin()),
			Normal:   vector3Slice(wp.Plane().Normal().AsVector()),
			IsOrigin: wp.IsCoordinateSystemElement(),
			Healthy:  wp.Health().OK(),
		}
	}
	return json.Marshal(wire.ListWorkPlanesResult{Planes: out})
}

// createWorkPlanes adds a datum plane of the requested kind to the active part and
// recomputes. An unsatisfiable definition still creates the plane (reported healthy=false)
// — the call only fails on bad arguments (unknown kind, wrong reference count, …).
func createWorkPlanes(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	var in wire.CreateWorkPlaneArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	wp, err := buildWorkPlane(part, in)
	if err != nil {
		return nil, err
	}
	part.Recompute()
	return json.Marshal(wire.CreateWorkPlaneResult{
		Index:   part.WorkPlanes().Count() - 1,
		Ref:     string(wp.Key()),
		Name:    wp.Name(),
		Healthy: wp.Health().OK(),
	})
}

// refPlaneCtor builds a work plane from exactly its references (no scalar parameters);
// refPlaneCtors is the table of the kinds that need only references, keeping
// buildWorkPlane's body to the kinds with extra inputs (offset, angle, fixed frame).
type refPlaneCtor struct {
	arity int
	build func(*feature.WorkPlanes, []feature.WorkRef) *feature.WorkPlane
}

var refPlaneCtors = map[types.WorkPlaneKind]refPlaneCtor{
	types.WorkPlaneThreePoints: {3, func(p *feature.WorkPlanes, r []feature.WorkRef) *feature.WorkPlane {
		return p.AddByThreePoints(r[0], r[1], r[2])
	}},
	types.WorkPlanePlaneAndPoint: {2, func(p *feature.WorkPlanes, r []feature.WorkRef) *feature.WorkPlane {
		return p.AddByPlaneAndPoint(r[0], r[1])
	}},
	types.WorkPlaneTwoPlanes: {2, func(p *feature.WorkPlanes, r []feature.WorkRef) *feature.WorkPlane {
		return p.AddByTwoPlanes(r[0], r[1])
	}},
	types.WorkPlaneTwoLines: {2, func(p *feature.WorkPlanes, r []feature.WorkRef) *feature.WorkPlane {
		return p.AddByTwoLines(r[0], r[1])
	}},
	types.WorkPlaneNormalToCurve: {2, func(p *feature.WorkPlanes, r []feature.WorkRef) *feature.WorkPlane {
		return p.AddByNormalToCurve(r[0], r[1])
	}},
	types.WorkPlaneTorusMidPlane: {1, func(p *feature.WorkPlanes, r []feature.WorkRef) *feature.WorkPlane { return p.AddByTorusMidPlane(r[0]) }},
	types.WorkPlanePointAndTangent: {2, func(p *feature.WorkPlanes, r []feature.WorkRef) *feature.WorkPlane {
		return p.AddByPointAndTangent(r[0], r[1])
	}},
	types.WorkPlanePlaneAndTangent: {2, func(p *feature.WorkPlanes, r []feature.WorkRef) *feature.WorkPlane {
		return p.AddByPlaneAndTangent(r[0], r[1])
	}},
	types.WorkPlaneLineAndTangent: {2, func(p *feature.WorkPlanes, r []feature.WorkRef) *feature.WorkPlane {
		return p.AddByLineAndTangent(r[0], r[1])
	}},
}

// buildWorkPlane dispatches a create request to the matching model constructor.
func buildWorkPlane(part *compdef.PartComponentDefinition, in wire.CreateWorkPlaneArgs) (*feature.WorkPlane, error) {
	planes := part.WorkPlanes()
	refs := toWorkRefs(in.Refs)
	kind := types.WorkPlaneKind(in.Kind)
	if c, ok := refPlaneCtors[kind]; ok {
		if len(refs) != c.arity {
			return nil, fmt.Errorf("workPlanes.create: %s needs %d references, got %d", in.Kind, c.arity, len(refs))
		}
		return c.build(planes, refs), nil
	}
	switch kind {
	case types.WorkPlaneOffset:
		return addOffsetPlane(part, refs, in)
	case types.WorkPlaneLinePlaneAngle:
		return addAnglePlane(part, refs, in)
	case types.WorkPlaneFixed:
		return addFixedWorkPlane(part.WorkPlanes(), in)
	default:
		return nil, fmt.Errorf("workPlanes.create: unknown kind %q (see api/types WorkPlane*)", in.Kind)
	}
}

// addOffsetPlane builds the plane-offset kind: one base plane reference + a distance.
func addOffsetPlane(part *compdef.PartComponentDefinition, refs []feature.WorkRef, in wire.CreateWorkPlaneArgs) (*feature.WorkPlane, error) {
	if len(refs) != 1 {
		return nil, fmt.Errorf("workPlanes.create: %s needs 1 reference, got %d", in.Kind, len(refs))
	}
	d, err := modelLength(part, in.Offset)
	if err != nil {
		return nil, err
	}
	return part.WorkPlanes().AddByPlaneAndOffset(refs[0], func() float64 { return d }), nil
}

// addAnglePlane builds the line-plane-angle kind: a line + a plane reference + an angle.
func addAnglePlane(part *compdef.PartComponentDefinition, refs []feature.WorkRef, in wire.CreateWorkPlaneArgs) (*feature.WorkPlane, error) {
	if len(refs) != 2 {
		return nil, fmt.Errorf("workPlanes.create: %s needs 2 references, got %d", in.Kind, len(refs))
	}
	a, err := modelAngle(part, in.Angle)
	if err != nil {
		return nil, err
	}
	return part.WorkPlanes().AddByLinePlaneAndAngle(refs[0], refs[1], func() float64 { return a }), nil
}

// addFixedWorkPlane builds an AddFixed plane from its origin point and two axis vectors.
func addFixedWorkPlane(planes *feature.WorkPlanes, in wire.CreateWorkPlaneArgs) (*feature.WorkPlane, error) {
	origin, err := parseCoords(in.Origin, "origin")
	if err != nil {
		return nil, err
	}
	x, err := parseAxisVector(in.XAxis, "xaxis")
	if err != nil {
		return nil, err
	}
	y, err := parseAxisVector(in.YAxis, "yaxis")
	if err != nil {
		return nil, err
	}
	o := math.P3(origin[0], origin[1], origin[2])
	return planes.AddFixed(func() math.Point3 { return o }, x, y), nil
}

// toWorkRefs converts the request's reference strings to model work references.
// toWorkRefs wraps reference strings as work refs. An origin reference ("origin/plane/xy",
// "origin/axis/z", …) and a user work plane/axis name are plain plane/axis refs; any other
// string is a B-rep topology reference key (from model.referenceKeys) and is tagged as a
// FaceRef so the resolver builds the plane on that body face — the way a work plane lands on a
// surface a feature created. (Edge/vertex-based kinds over the wire are a known limitation.)
func toWorkRefs(refs []string) []feature.WorkRef {
	out := make([]feature.WorkRef, len(refs))
	for i, r := range refs {
		if strings.HasPrefix(r, "origin/") {
			out[i] = feature.WorkRef(r)
		} else {
			out[i] = feature.FaceRef([]byte(r))
		}
	}
	return out
}

// modelLength parses a unit-bearing distance ("10 mm") into a model value (cm).
func modelLength(part *compdef.PartComponentDefinition, expr string) (float64, error) {
	q, err := part.Units().Parse(expr, param.Length)
	if err != nil {
		return 0, fmt.Errorf("workPlanes.create: offset %q: %w", expr, err)
	}
	return q.Value, nil
}

// modelAngle parses a unit-bearing angle ("45 deg") into a model value (radians).
func modelAngle(part *compdef.PartComponentDefinition, expr string) (float64, error) {
	q, err := part.Units().Parse(expr, param.Angle)
	if err != nil {
		return 0, fmt.Errorf("workPlanes.create: angle %q: %w", expr, err)
	}
	return q.Value, nil
}

// parseCoords requires a 3-component coordinate slice, naming what for errors.
func parseCoords(s []float64, what string) ([]float64, error) {
	if len(s) != 3 {
		return nil, fmt.Errorf("workPlanes.create: %s needs 3 components, got %d", what, len(s))
	}
	return s, nil
}

// parseAxisVector requires a 3-component direction and normalizes it to a unit vector.
func parseAxisVector(s []float64, what string) (math.UnitVector3, error) {
	if _, err := parseCoords(s, what); err != nil {
		return math.UnitVector3{}, err
	}
	return math.NewUnitVector3(s[0], s[1], s[2])
}

// point3Slice / vector3Slice render geometry as [x,y,z] for the wire DTOs.
func point3Slice(p math.Point3) []float64 {
	return []float64{float64(p.X), float64(p.Y), float64(p.Z)}
}

func vector3Slice(v math.Vector3) []float64 {
	return []float64{float64(v.X), float64(v.Y), float64(v.Z)}
}
