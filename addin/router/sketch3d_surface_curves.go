// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"

	"oblikovati/api/types"
	"oblikovati/api/wire"

	"oblikovati/addin/modelaccess"
	"oblikovati/app"
	"oblikovati/kernel/geom"
	"oblikovati/math"
	"oblikovati/model/compdef"
	"oblikovati/model/sketch"
)

// addSketch3DSurfaceCurve adds a surface-derived curve (intersection/silhouette) to a 3D
// sketch, resolving the referenced part faces by reference key to their surfaces and
// wrapping the F10 surface-intersection kernel (M22-F11).
//
//nolint:funlen // one-case-per-kind dispatch (intersection/silhouette/onFace/projectToSurface/offset); length is the dispatch, like the serialize codecs.
func addSketch3DSurfaceCurve(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	var in wire.AddSketch3DSurfaceCurveArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	sk, err := sketch3DAtIndex(part, in.SketchIndex)
	if err != nil {
		return nil, err
	}
	switch types.Sketch3DEntityKind(in.Kind) {
	case types.Sketch3DEntityIntersection:
		return intersectionCurve3D(part, sk, in)
	case types.Sketch3DEntitySilhouette:
		return silhouetteCurve3D(part, sk, in)
	case types.Sketch3DEntityOnFace:
		return onFaceCurve3D(part, sk, in)
	case types.Sketch3DEntityProjectToSurface:
		return projectToSurfaceCurve3D(part, sk, in)
	case types.Sketch3DEntityOffset:
		return offsetCurve3(sk, in)
	default:
		return nil, fmt.Errorf("sketch3d.addSurfaceCurve: unsupported kind %q (want intersection|silhouette|onFace|projectToSurface|offset)", in.Kind)
	}
}

// projectToSurfaceCurve3D resolves an in-sketch source curve + a target face (associative)
// and adds its perpendicular projection onto the face's surface.
func projectToSurfaceCurve3D(part *compdef.PartComponentDefinition, sk *sketch.Sketch3D, in wire.AddSketch3DSurfaceCurveArgs) (json.RawMessage, error) {
	if len(in.FaceRefs) != 1 {
		return nil, fmt.Errorf("sketch3d.addSurfaceCurve: projectToSurface needs 1 face ref, got %d", len(in.FaceRefs))
	}
	src, err := sk.SourceCurve3(sketch.ID(in.SourceEntityID))
	if err != nil {
		return nil, err
	}
	if !part.FaceKeyResolves(in.FaceRefs[0]) {
		return surfaceCurveUnhealthy(in.Kind)
	}
	c := sk.AddProjectToSurfaceCurve3DRef(src, compdef.NewFaceRefSource(part, in.FaceRefs[0]))
	return surfaceCurveResult(c.EntityID(), in.Kind)
}

// offsetCurve3 resolves an in-sketch source curve and adds its offset by OffsetDistance in
// the plane with the given Normal.
func offsetCurve3(sk *sketch.Sketch3D, in wire.AddSketch3DSurfaceCurveArgs) (json.RawMessage, error) {
	src, err := sk.SourceCurve3(sketch.ID(in.SourceEntityID))
	if err != nil {
		return nil, err
	}
	normal, err := vector3Arg(in.Normal)
	if err != nil {
		return nil, err
	}
	c := sk.AddOffsetCurve3(src, in.OffsetDistance, normal)
	return surfaceCurveResult(c.EntityID(), in.Kind)
}

// onFaceCurve3D resolves one face ref and adds an (associative) curve in its parameter
// space from the request's flat UV polyline.
func onFaceCurve3D(part *compdef.PartComponentDefinition, sk *sketch.Sketch3D, in wire.AddSketch3DSurfaceCurveArgs) (json.RawMessage, error) {
	if len(in.FaceRefs) != 1 {
		return nil, fmt.Errorf("sketch3d.addSurfaceCurve: onFace needs 1 face ref, got %d", len(in.FaceRefs))
	}
	uv, err := uvPairs(in.UV) // validate the request shape before resolving the reference
	if err != nil {
		return nil, err
	}
	if !part.FaceKeyResolves(in.FaceRefs[0]) {
		return surfaceCurveUnhealthy(in.Kind)
	}
	c := sk.AddOnFaceCurve3DRef(compdef.NewFaceRefSource(part, in.FaceRefs[0]), uv)
	return surfaceCurveResult(c.EntityID(), in.Kind)
}

// uvPairs converts a flat [u0,v0,u1,v1,…] slice into parameter-space points (≥ 2 points).
func uvPairs(flat []float64) ([]math.Point2, error) {
	if len(flat) < 4 || len(flat)%2 != 0 {
		return nil, fmt.Errorf("sketch3d.addSurfaceCurve: onFace uv needs an even count ≥ 4, got %d", len(flat))
	}
	out := make([]math.Point2, len(flat)/2)
	for i := range out {
		out[i] = math.P2(math.Scalar(flat[2*i]), math.Scalar(flat[2*i+1]))
	}
	return out, nil
}

// intersectionCurve3D resolves two face refs and adds their (associative) intersection.
func intersectionCurve3D(part *compdef.PartComponentDefinition, sk *sketch.Sketch3D, in wire.AddSketch3DSurfaceCurveArgs) (json.RawMessage, error) {
	if len(in.FaceRefs) != 2 {
		return nil, fmt.Errorf("sketch3d.addSurfaceCurve: intersection needs 2 face refs, got %d", len(in.FaceRefs))
	}
	if !part.FaceKeyResolves(in.FaceRefs[0]) || !part.FaceKeyResolves(in.FaceRefs[1]) {
		return surfaceCurveUnhealthy(in.Kind)
	}
	c := sk.AddIntersectionCurve3DRef(compdef.NewFaceRefSource(part, in.FaceRefs[0]), compdef.NewFaceRefSource(part, in.FaceRefs[1]), gridFromArgs(in))
	return surfaceCurveResult(c.EntityID(), in.Kind)
}

// silhouetteCurve3D resolves one face ref and adds its (associative) silhouette curve.
func silhouetteCurve3D(part *compdef.PartComponentDefinition, sk *sketch.Sketch3D, in wire.AddSketch3DSurfaceCurveArgs) (json.RawMessage, error) {
	if len(in.FaceRefs) != 1 {
		return nil, fmt.Errorf("sketch3d.addSurfaceCurve: silhouette needs 1 face ref, got %d", len(in.FaceRefs))
	}
	if !part.FaceKeyResolves(in.FaceRefs[0]) {
		return surfaceCurveUnhealthy(in.Kind)
	}
	dir, err := vector3Arg(in.ViewDir)
	if err != nil {
		return nil, err
	}
	c := sk.AddSilhouetteCurve3DRef(compdef.NewFaceRefSource(part, in.FaceRefs[0]), dir, gridFromArgs(in))
	return surfaceCurveResult(c.EntityID(), in.Kind)
}

// gridFromArgs builds the tracer grid window from the request fields.
func gridFromArgs(in wire.AddSketch3DSurfaceCurveArgs) geom.SurfaceGrid {
	return geom.SurfaceGrid{
		UMin: in.GridUMin, UMax: in.GridUMax, VMin: in.GridVMin, VMax: in.GridVMax,
		USteps: in.GridUSteps, VSteps: in.GridVSteps,
	}
}

// surfaceCurveResult marshals a created surface-curve's id and kind (healthy).
func surfaceCurveResult(id sketch.ID, kind string) (json.RawMessage, error) {
	return json.Marshal(wire.AddSketch3DSurfaceCurveResult{EntityID: uint64(id), Kind: kind, Healthy: true})
}

// surfaceCurveUnhealthy reports a lost face reference (nothing created), not an error.
func surfaceCurveUnhealthy(kind string) (json.RawMessage, error) {
	return json.Marshal(wire.AddSketch3DSurfaceCurveResult{Kind: kind, Healthy: false})
}
