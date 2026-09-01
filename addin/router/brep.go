// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/model/bodyapi"
)

// The transient B-rep factory over the wire (M07-F05, #628): ownerless
// bodies in the session registry, addressed by handle. Operations on a
// document body always COPY it into transient space first.

// brepSource resolves a BrepBodyRef onto a kernel body: a transient handle
// directly, a document body as a fresh registry copy.
func brepSource(s *app.Session, ref wire.BrepBodyRef) (*bodyapi.TransientBody, error) {
	reg := s.TransientBodies()
	if ref.Handle > 0 && ref.BodyIndex != nil {
		return nil, fmt.Errorf("a body ref must set handle OR bodyIndex, got both")
	}
	if ref.Handle > 0 {
		tb, ok := reg.ByHandle(ref.Handle)
		if !ok {
			return nil, fmt.Errorf(errNoTransientBody, ref.Handle)
		}
		return tb, nil
	}
	if ref.BodyIndex == nil {
		return nil, fmt.Errorf("a body ref must set handle or bodyIndex")
	}
	doc, err := resolveBody(s, *ref.BodyIndex)
	if err != nil {
		return nil, err
	}
	clone, err := bodyapi.CopyTopoBody(doc, len(reg.Handles())+1)
	if err != nil {
		return nil, err
	}
	return reg.Adopt(clone), nil
}

// brepStats summarizes a transient body for the reply.
func brepStats(tb *bodyapi.TransientBody) wire.BrepBodyStats {
	return wire.BrepBodyStats{
		Solid: tb.IsSolid(), Faces: tb.FaceCount(), Edges: tb.EdgeCount(),
		Vertices: tb.VertexCount(), Shells: tb.ShellCount(), Wires: tb.WireCount(),
		Volume: tb.Volume(),
	}
}

func brepHandleReply(tb *bodyapi.TransientBody) wire.BrepHandleResult {
	return wire.BrepHandleResult{Handle: tb.Handle(), Stats: brepStats(tb)}
}

// brepCreatePrimitive serves wire.MethodBrepCreatePrimitive.
func brepCreatePrimitive(s *app.Session, in wire.CreatePrimitiveArgs) (wire.BrepHandleResult, error) {
	body, err := buildPrimitive(in)
	if err != nil {
		return wire.BrepHandleResult{}, err
	}
	return brepHandleReply(s.TransientBodies().Adopt(body)), nil
}

func buildPrimitive(in wire.CreatePrimitiveArgs) (*topo.Body, error) {
	switch in.Kind {
	case "block":
		return primitiveBlock(in)
	case "cylinderCone":
		return primitiveCylinderCone(in)
	case "sphere":
		return primitiveSphere(in)
	case "torus":
		return primitiveTorus(in)
	default:
		return nil, fmt.Errorf("unknown primitive kind %q (want block, cylinderCone, sphere or torus)", in.Kind)
	}
}

func primitiveBlock(in wire.CreatePrimitiveArgs) (*topo.Body, error) {
	min, err := xyzPoint(in.Min, "min")
	if err != nil {
		return nil, err
	}
	max, err := xyzPoint(in.Max, "max")
	if err != nil {
		return nil, err
	}
	return brep.SolidBlock(min, max, "transient")
}

func primitiveCylinderCone(in wire.CreatePrimitiveArgs) (*topo.Body, error) {
	bottom, err := xyzPoint(in.Bottom, "bottom")
	if err != nil {
		return nil, err
	}
	top, err := xyzPoint(in.Top, "top")
	if err != nil {
		return nil, err
	}
	return brep.SolidCylinderCone(bottom, top, in.BottomRadius, in.TopRadius, "transient")
}

func primitiveSphere(in wire.CreatePrimitiveArgs) (*topo.Body, error) {
	center, err := xyzPoint(in.Center, "center")
	if err != nil {
		return nil, err
	}
	return brep.SolidSphere(center, in.Radius, "transient")
}

func primitiveTorus(in wire.CreatePrimitiveArgs) (*topo.Body, error) {
	center, err := xyzPoint(in.Center, "center")
	if err != nil {
		return nil, err
	}
	axis, err := xyz(in.Axis, "axis")
	if err != nil {
		return nil, err
	}
	return brep.SolidTorus(center, axis, in.MajorRadius, in.MinorRadius, "transient")
}

// brepBoolean serves wire.MethodBrepBoolean: blank modified in place.
func brepBoolean(s *app.Session, in wire.BrepBooleanArgs) (wire.BrepHandleResult, error) {
	reg := s.TransientBodies()
	blank, ok := reg.ByHandle(in.BlankHandle)
	if !ok {
		return wire.BrepHandleResult{}, fmt.Errorf("no transient body with handle %d (the blank must be transient)", in.BlankHandle)
	}
	tool, err := brepSource(s, in.Tool)
	if err != nil {
		return wire.BrepHandleResult{}, err
	}
	op, ok := types.ParseBooleanType(in.Operation)
	if !ok {
		return wire.BrepHandleResult{}, fmt.Errorf("unknown boolean operation %q (want union, difference or intersect)", in.Operation)
	}
	if err := s.TransientBodies().DoBoolean(blank, tool, op); err != nil {
		return wire.BrepHandleResult{}, err
	}
	return brepHandleReply(blank), nil
}

// brepTransform serves wire.MethodBrepTransform (in place).
func brepTransform(s *app.Session, in wire.BrepTransformArgs) (wire.BrepHandleResult, error) {
	tb, ok := s.TransientBodies().ByHandle(in.Handle)
	if !ok {
		return wire.BrepHandleResult{}, fmt.Errorf(errNoTransientBody, in.Handle)
	}
	if len(in.Matrix) != 16 {
		return wire.BrepHandleResult{}, fmt.Errorf("transform needs 16 row-major cells, got %d", len(in.Matrix))
	}
	var m types.Matrix
	copy(m.Cells[:], in.Matrix)
	if err := s.TransientBodies().Transform(tb, m); err != nil {
		return wire.BrepHandleResult{}, err
	}
	return brepHandleReply(tb), nil
}

// brepCopy serves wire.MethodBrepCopy.
func brepCopy(s *app.Session, in wire.BrepCopyArgs) (wire.BrepHandleResult, error) {
	src, err := brepSource(s, in.Source)
	if err != nil {
		return wire.BrepHandleResult{}, err
	}
	if in.Source.Handle > 0 {
		copied, err := s.TransientBodies().Copy(src)
		if err != nil {
			return wire.BrepHandleResult{}, err
		}
		return brepHandleReply(copied.(*bodyapi.TransientBody)), nil
	}
	// A document source was already copied into the registry by brepSource.
	return brepHandleReply(src), nil
}

// brepSectionWithPlane serves wire.MethodBrepSectionWithPlane.
func brepSectionWithPlane(s *app.Session, in wire.BrepSectionArgs) (wire.BrepWiresResult, error) {
	src, err := brepSource(s, in.Source)
	if err != nil {
		return wire.BrepWiresResult{}, err
	}
	origin, err := xyzPoint(in.PlaneOrigin, "planeOrigin")
	if err != nil {
		return wire.BrepWiresResult{}, err
	}
	normal, err := xyz(in.PlaneNormal, "planeNormal")
	if err != nil {
		return wire.BrepWiresResult{}, err
	}
	sec, err := ops.SectionWithPlane(src.Topo(), origin, normal, ops.DefaultQuality())
	if err != nil {
		return wire.BrepWiresResult{}, err
	}
	tb := s.TransientBodies().Adopt(sec)
	return wire.BrepWiresResult{Handle: tb.Handle(), Wires: wirePolylines(sec)}, nil
}

// brepDeleteFaces serves wire.MethodBrepDeleteFaces (no healing).
func brepDeleteFaces(s *app.Session, in wire.BrepDeleteFacesArgs) (wire.BrepHandleResult, error) {
	tb, ok := s.TransientBodies().ByHandle(in.Handle)
	if !ok {
		return wire.BrepHandleResult{}, fmt.Errorf(errNoTransientBody, in.Handle)
	}
	keys := make([][]byte, len(in.FaceKeys))
	for i, k := range in.FaceKeys {
		keys[i] = []byte(k)
	}
	res, err := ops.DropFaces(tb.Topo(), keys, in.KeepInstead)
	if err != nil {
		return wire.BrepHandleResult{}, err
	}
	if err := s.TransientBodies().Replace(in.Handle, res); err != nil {
		return wire.BrepHandleResult{}, err
	}
	return brepHandleReply(tb), nil
}

// brepSilhouette serves wire.MethodBrepSilhouette.
func brepSilhouette(s *app.Session, in wire.BrepSilhouetteArgs) (wire.BrepWiresResult, error) {
	src, err := brepSource(s, in.Source)
	if err != nil {
		return wire.BrepWiresResult{}, err
	}
	f, ok := src.Topo().FindFaceByKey([]byte(in.FaceKey))
	if !ok {
		return wire.BrepWiresResult{}, fmt.Errorf("no face with key %q on the source body", in.FaceKey)
	}
	dir, err := xyz(in.ViewDirection, "viewDirection")
	if err != nil {
		return wire.BrepWiresResult{}, err
	}
	sil, err := ops.FaceSilhouetteWires(f, dir, in.IncludeBoundary, ops.DefaultQuality())
	if err != nil {
		return wire.BrepWiresResult{}, err
	}
	tb := s.TransientBodies().Adopt(sil)
	return wire.BrepWiresResult{Handle: tb.Handle(), Wires: wirePolylines(sil)}, nil
}

// brepOffsetFaces serves wire.MethodBrepOffsetFaces: the named faces of the source, offset by
// Distance along their normals, come back as a new transient surface body to sample.
func brepOffsetFaces(s *app.Session, in wire.BrepOffsetFacesArgs) (wire.BrepHandleResult, error) {
	src, err := brepSource(s, in.Source)
	if err != nil {
		return wire.BrepHandleResult{}, err
	}
	keys := make([][]byte, len(in.FaceKeys))
	for i, k := range in.FaceKeys {
		keys[i] = []byte(k)
	}
	off, err := ops.OffsetFaceSurfaces(src.Topo(), keys, in.Distance, in.Reverse)
	if err != nil {
		return wire.BrepHandleResult{}, err
	}
	return brepHandleReply(s.TransientBodies().Adopt(off)), nil
}

// brepRuledSurface serves wire.MethodBrepRuledSurface.
func brepRuledSurface(s *app.Session, in wire.BrepRuledSurfaceArgs) (wire.BrepHandleResult, error) {
	w1, err := brepWireOf(s, in.SectionOne)
	if err != nil {
		return wire.BrepHandleResult{}, err
	}
	w2, err := brepWireOf(s, in.SectionTwo)
	if err != nil {
		return wire.BrepHandleResult{}, err
	}
	surf, err := ops.RuledSurfaceBetweenWires(w1, w2)
	if err != nil {
		return wire.BrepHandleResult{}, err
	}
	return brepHandleReply(s.TransientBodies().Adopt(surf)), nil
}

func brepWireOf(s *app.Session, ref wire.BrepWireRef) (*topo.Wire, error) {
	src, err := brepSource(s, ref.Body)
	if err != nil {
		return nil, err
	}
	wires := src.Topo().Wires()
	if ref.WireIndex < 0 || ref.WireIndex >= len(wires) {
		return nil, fmt.Errorf("wire index %d out of range (body has %d wires)", ref.WireIndex, len(wires))
	}
	return wires[ref.WireIndex], nil
}

// brepImprint serves wire.MethodBrepImprint.
func brepImprint(s *app.Session, in wire.BrepImprintArgs) (wire.BrepImprintResult, error) {
	one, err := brepSource(s, in.BodyOne)
	if err != nil {
		return wire.BrepImprintResult{}, err
	}
	two, err := brepSource(s, in.BodyTwo)
	if err != nil {
		return wire.BrepImprintResult{}, err
	}
	ra, rb, err := brep.ImprintBodies(one.Topo(), two.Topo())
	if err != nil {
		return wire.BrepImprintResult{}, err
	}
	reg := s.TransientBodies()
	ta, tb := reg.Adopt(ra.Body), reg.Adopt(rb.Body)
	return wire.BrepImprintResult{
		HandleOne: ta.Handle(), HandleTwo: tb.Handle(),
		OneTouchedFaceKeys: faceKeys(ra.TouchedFaces), TwoTouchedFaceKeys: faceKeys(rb.TouchedFaces),
		OneImprintedEdgeKeys: edgeKeys(ra.ImprintedEdge), TwoImprintedEdgeKeys: edgeKeys(rb.ImprintedEdge),
	}, nil
}

func faceKeys(faces []*topo.Face) []string {
	out := make([]string, len(faces))
	for i, f := range faces {
		out[i] = string(f.ReferenceKey())
	}
	return out
}

func edgeKeys(edges []*topo.Edge) []string {
	out := make([]string, len(edges))
	for i, e := range edges {
		out[i] = string(e.ReferenceKey())
	}
	return out
}

// brepIdenticalBodies serves wire.MethodBrepIdenticalBodies.
func brepIdenticalBodies(s *app.Session, in wire.BrepIdenticalBodiesArgs) (wire.BrepIdenticalBodiesResult, error) {
	bodies := make([]*topo.Body, 0, len(in.Sources))
	for _, ref := range in.Sources {
		src, err := brepSource(s, ref)
		if err != nil {
			return wire.BrepIdenticalBodiesResult{}, err
		}
		bodies = append(bodies, src.Topo())
	}
	opt := query.IdenticalBodiesOptions{
		Tolerance: in.Tolerance, MatchTopology: in.MatchTopology,
		MatchReflection: in.MatchReflection == nil || *in.MatchReflection,
	}
	groups := query.GroupIdenticalBodies(bodies, opt, ops.DefaultQuality())
	return wire.BrepIdenticalBodiesResult{Groups: groups}, nil
}

// brepCreateFromDefinition serves wire.MethodBrepCreateFromDefinition.
func brepCreateFromDefinition(s *app.Session, in wire.BrepCreateFromDefinitionArgs) (wire.BrepCreateFromDefinitionResult, error) {
	body, issues, err := s.TransientBodies().CreateFromDefinition(in.Definition)
	if err != nil {
		return wire.BrepCreateFromDefinitionResult{}, err
	}
	if len(issues) > 0 {
		return wire.BrepCreateFromDefinitionResult{Issues: issues}, nil
	}
	tb := body.(*bodyapi.TransientBody)
	return wire.BrepCreateFromDefinitionResult{Handle: tb.Handle(), Stats: brepStats(tb)}, nil
}

// brepDescribe serves wire.MethodBrepDescribe.
func brepDescribe(s *app.Session, in wire.BrepHandleArgs) (wire.BrepHandleResult, error) {
	tb, ok := s.TransientBodies().ByHandle(in.Handle)
	if !ok {
		return wire.BrepHandleResult{}, fmt.Errorf(errNoTransientBody, in.Handle)
	}
	return brepHandleReply(tb), nil
}

// brepList serves wire.MethodBrepList. It reads no args and no active-model context (the transient
// registry is session-scoped), so it stays a raw handler.
func brepList(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	return json.Marshal(wire.BrepListResult{Handles: s.TransientBodies().Handles()})
}

// brepDelete serves wire.MethodBrepDelete.
func brepDelete(s *app.Session, in wire.BrepHandleArgs) (wire.OKResult, error) {
	if !s.TransientBodies().Delete(in.Handle) {
		return wire.OKResult{}, fmt.Errorf(errNoTransientBody, in.Handle)
	}
	return wire.OKResult{OK: true}, nil
}
