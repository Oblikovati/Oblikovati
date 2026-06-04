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
	"github.com/Oblikovati/oblikovati/kernel/topo"
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

// intersectionCurve3D resolves two face refs and adds their (associative) intersection.
func intersectionCurve3D(part *compdef.PartComponentDefinition, sk *sketch.Sketch3D, in wire.AddSketch3DSurfaceCurveArgs) (json.RawMessage, error) {
	if len(in.FaceRefs) != 2 {
		return nil, fmt.Errorf("sketch3d.addSurfaceCurve: intersection needs 2 face refs, got %d", len(in.FaceRefs))
	}
	if !faceKeyResolves(part, in.FaceRefs[0]) || !faceKeyResolves(part, in.FaceRefs[1]) {
		return surfaceCurveUnhealthy(in.Kind)
	}
	c := sk.AddIntersectionCurve3DRef(newFaceSource(part, in.FaceRefs[0]), newFaceSource(part, in.FaceRefs[1]), gridFromArgs(in))
	return surfaceCurveResult(c.EntityID(), in.Kind)
}

// silhouetteCurve3D resolves one face ref and adds its (associative) silhouette curve.
func silhouetteCurve3D(part *compdef.PartComponentDefinition, sk *sketch.Sketch3D, in wire.AddSketch3DSurfaceCurveArgs) (json.RawMessage, error) {
	if len(in.FaceRefs) != 1 {
		return nil, fmt.Errorf("sketch3d.addSurfaceCurve: silhouette needs 1 face ref, got %d", len(in.FaceRefs))
	}
	if !faceKeyResolves(part, in.FaceRefs[0]) {
		return surfaceCurveUnhealthy(in.Kind)
	}
	dir, err := vector3Arg(in.ViewDir)
	if err != nil {
		return nil, err
	}
	c := sk.AddSilhouetteCurve3DRef(newFaceSource(part, in.FaceRefs[0]), dir, gridFromArgs(in))
	return surfaceCurveResult(c.EntityID(), in.Kind)
}

// faceKeyResolves reports whether a face reference key currently binds to a part face.
func faceKeyResolves(part *compdef.PartComponentDefinition, ref string) bool {
	key := []byte(ref)
	for _, body := range part.SurfaceBodies().All() {
		if _, ok := body.FindFaceByKey(key); ok {
			return true
		}
	}
	return false
}

// faceSurfaceSource adapts a part face to a self-resolving sketch.SurfaceSource: it re-finds
// the face by reference key among the part's current bodies, so a surface-derived curve
// re-evaluates against the rebuilt B-rep on recompute. A lost key yields a zero surface,
// which the kernel tracer treats as no intersection.
type faceSurfaceSource struct {
	ref    string
	bodies func() []*topo.Body
}

func newFaceSource(part *compdef.PartComponentDefinition, ref string) faceSurfaceSource {
	return faceSurfaceSource{ref: ref, bodies: func() []*topo.Body { return part.SurfaceBodies().All() }}
}

func (s faceSurfaceSource) SourceID() string { return s.ref }

func (s faceSurfaceSource) Surface() geom.Surface {
	key := []byte(s.ref)
	for _, b := range s.bodies() {
		if face, ok := b.FindFaceByKey(key); ok {
			return face.Geometry()
		}
	}
	return nil
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
