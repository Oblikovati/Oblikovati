// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"

	"github.com/Oblikovati/api/types"
	"github.com/Oblikovati/api/wire"

	"github.com/Oblikovati/oblikovati/addin/modelaccess"
	"github.com/Oblikovati/oblikovati/app"
	"github.com/Oblikovati/oblikovati/kernel/geom"
	"github.com/Oblikovati/oblikovati/model/compdef"
	"github.com/Oblikovati/oblikovati/model/sketch"
)

// addSketch3DSurfaceCurve adds a surface-derived curve (intersection/silhouette) to a 3D
// sketch, resolving the referenced part faces by reference key to their surfaces and
// wrapping the F10 surface-intersection kernel (M22-F11).
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
	default:
		return nil, fmt.Errorf("sketch3d.addSurfaceCurve: unsupported kind %q (want intersection|silhouette)", in.Kind)
	}
}

// intersectionCurve3D resolves two face refs and adds their intersection curve.
func intersectionCurve3D(part *compdef.PartComponentDefinition, sk *sketch.Sketch3D, in wire.AddSketch3DSurfaceCurveArgs) (json.RawMessage, error) {
	if len(in.FaceRefs) != 2 {
		return nil, fmt.Errorf("sketch3d.addSurfaceCurve: intersection needs 2 face refs, got %d", len(in.FaceRefs))
	}
	a, okA := faceSurfaceByKey(part, in.FaceRefs[0])
	b, okB := faceSurfaceByKey(part, in.FaceRefs[1])
	if !okA || !okB {
		return surfaceCurveUnhealthy(in.Kind)
	}
	c := sk.AddIntersectionCurve3D(a, b, gridFromArgs(in))
	return surfaceCurveResult(c.EntityID(), in.Kind)
}

// silhouetteCurve3D resolves one face ref and adds its silhouette curve.
func silhouetteCurve3D(part *compdef.PartComponentDefinition, sk *sketch.Sketch3D, in wire.AddSketch3DSurfaceCurveArgs) (json.RawMessage, error) {
	if len(in.FaceRefs) != 1 {
		return nil, fmt.Errorf("sketch3d.addSurfaceCurve: silhouette needs 1 face ref, got %d", len(in.FaceRefs))
	}
	face, ok := faceSurfaceByKey(part, in.FaceRefs[0])
	if !ok {
		return surfaceCurveUnhealthy(in.Kind)
	}
	dir, err := vector3Arg(in.ViewDir)
	if err != nil {
		return nil, err
	}
	c := sk.AddSilhouetteCurve3D(face, dir, gridFromArgs(in))
	return surfaceCurveResult(c.EntityID(), in.Kind)
}

// faceSurfaceByKey resolves a face reference key to its surface among the part's bodies.
func faceSurfaceByKey(part *compdef.PartComponentDefinition, ref string) (geom.Surface, bool) {
	key := []byte(ref)
	for _, body := range part.SurfaceBodies().All() {
		if face, ok := body.FindFaceByKey(key); ok {
			return face.Geometry(), true
		}
	}
	return nil, false
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
