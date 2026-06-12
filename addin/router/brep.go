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
			return nil, fmt.Errorf("no transient body with handle %d", ref.Handle)
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

func brepHandleReply(tb *bodyapi.TransientBody) (json.RawMessage, error) {
	return json.Marshal(wire.BrepHandleResult{Handle: tb.Handle(), Stats: brepStats(tb)})
}

// brepCreatePrimitive serves wire.MethodBrepCreatePrimitive.
func brepCreatePrimitive(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.CreatePrimitiveArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	body, err := buildPrimitive(in)
	if err != nil {
		return nil, err
	}
	return brepHandleReply(s.TransientBodies().Adopt(body))
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
func brepBoolean(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.BrepBooleanArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	reg := s.TransientBodies()
	blank, ok := reg.ByHandle(in.BlankHandle)
	if !ok {
		return nil, fmt.Errorf("no transient body with handle %d (the blank must be transient)", in.BlankHandle)
	}
	tool, err := brepSource(s, in.Tool)
	if err != nil {
		return nil, err
	}
	op, ok := types.ParseBooleanType(in.Operation)
	if !ok {
		return nil, fmt.Errorf("unknown boolean operation %q (want union, difference or intersect)", in.Operation)
	}
	if err := s.TransientBodies().DoBoolean(blank, tool, op); err != nil {
		return nil, err
	}
	return brepHandleReply(blank)
}

// brepTransform serves wire.MethodBrepTransform (in place).
func brepTransform(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.BrepTransformArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	tb, ok := s.TransientBodies().ByHandle(in.Handle)
	if !ok {
		return nil, fmt.Errorf("no transient body with handle %d", in.Handle)
	}
	if len(in.Matrix) != 16 {
		return nil, fmt.Errorf("transform needs 16 row-major cells, got %d", len(in.Matrix))
	}
	var m types.Matrix
	copy(m.Cells[:], in.Matrix)
	if err := s.TransientBodies().Transform(tb, m); err != nil {
		return nil, err
	}
	return brepHandleReply(tb)
}

// brepCopy serves wire.MethodBrepCopy.
func brepCopy(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.BrepCopyArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	src, err := brepSource(s, in.Source)
	if err != nil {
		return nil, err
	}
	if in.Source.Handle > 0 {
		copied, err := s.TransientBodies().Copy(src)
		if err != nil {
			return nil, err
		}
		return brepHandleReply(copied.(*bodyapi.TransientBody))
	}
	// A document source was already copied into the registry by brepSource.
	return brepHandleReply(src)
}

// brepSectionWithPlane serves wire.MethodBrepSectionWithPlane.
func brepSectionWithPlane(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.BrepSectionArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	src, err := brepSource(s, in.Source)
	if err != nil {
		return nil, err
	}
	origin, err := xyzPoint(in.PlaneOrigin, "planeOrigin")
	if err != nil {
		return nil, err
	}
	normal, err := xyz(in.PlaneNormal, "planeNormal")
	if err != nil {
		return nil, err
	}
	sec, err := ops.SectionWithPlane(src.Topo(), origin, normal, ops.DefaultQuality())
	if err != nil {
		return nil, err
	}
	tb := s.TransientBodies().Adopt(sec)
	return json.Marshal(wire.BrepWiresResult{Handle: tb.Handle(), Wires: wirePolylines(sec)})
}

// brepDeleteFaces serves wire.MethodBrepDeleteFaces (no healing).
func brepDeleteFaces(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.BrepDeleteFacesArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	tb, ok := s.TransientBodies().ByHandle(in.Handle)
	if !ok {
		return nil, fmt.Errorf("no transient body with handle %d", in.Handle)
	}
	keys := make([][]byte, len(in.FaceKeys))
	for i, k := range in.FaceKeys {
		keys[i] = []byte(k)
	}
	res, err := ops.DropFaces(tb.Topo(), keys, in.KeepInstead)
	if err != nil {
		return nil, err
	}
	if err := s.TransientBodies().Replace(in.Handle, res); err != nil {
		return nil, err
	}
	return brepHandleReply(tb)
}

// brepSilhouette serves wire.MethodBrepSilhouette.
func brepSilhouette(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.BrepSilhouetteArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	src, err := brepSource(s, in.Source)
	if err != nil {
		return nil, err
	}
	f, ok := src.Topo().FindFaceByKey([]byte(in.FaceKey))
	if !ok {
		return nil, fmt.Errorf("no face with key %q on the source body", in.FaceKey)
	}
	dir, err := xyz(in.ViewDirection, "viewDirection")
	if err != nil {
		return nil, err
	}
	sil, err := ops.FaceSilhouetteWires(f, dir, in.IncludeBoundary, ops.DefaultQuality())
	if err != nil {
		return nil, err
	}
	tb := s.TransientBodies().Adopt(sil)
	return json.Marshal(wire.BrepWiresResult{Handle: tb.Handle(), Wires: wirePolylines(sil)})
}

// brepRuledSurface serves wire.MethodBrepRuledSurface.
func brepRuledSurface(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.BrepRuledSurfaceArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	w1, err := brepWireOf(s, in.SectionOne)
	if err != nil {
		return nil, err
	}
	w2, err := brepWireOf(s, in.SectionTwo)
	if err != nil {
		return nil, err
	}
	surf, err := ops.RuledSurfaceBetweenWires(w1, w2)
	if err != nil {
		return nil, err
	}
	return brepHandleReply(s.TransientBodies().Adopt(surf))
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
func brepImprint(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.BrepImprintArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	one, err := brepSource(s, in.BodyOne)
	if err != nil {
		return nil, err
	}
	two, err := brepSource(s, in.BodyTwo)
	if err != nil {
		return nil, err
	}
	ra, rb, err := brep.ImprintBodies(one.Topo(), two.Topo())
	if err != nil {
		return nil, err
	}
	reg := s.TransientBodies()
	ta, tb := reg.Adopt(ra.Body), reg.Adopt(rb.Body)
	return json.Marshal(wire.BrepImprintResult{
		HandleOne: ta.Handle(), HandleTwo: tb.Handle(),
		OneTouchedFaceKeys: faceKeys(ra.TouchedFaces), TwoTouchedFaceKeys: faceKeys(rb.TouchedFaces),
		OneImprintedEdgeKeys: edgeKeys(ra.ImprintedEdge), TwoImprintedEdgeKeys: edgeKeys(rb.ImprintedEdge),
	})
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
func brepIdenticalBodies(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.BrepIdenticalBodiesArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	bodies := make([]*topo.Body, 0, len(in.Sources))
	for _, ref := range in.Sources {
		src, err := brepSource(s, ref)
		if err != nil {
			return nil, err
		}
		bodies = append(bodies, src.Topo())
	}
	opt := ops.IdenticalBodiesOptions{
		Tolerance: in.Tolerance, MatchTopology: in.MatchTopology,
		MatchReflection: in.MatchReflection == nil || *in.MatchReflection,
	}
	groups := ops.GroupIdenticalBodies(bodies, opt, ops.DefaultQuality())
	return json.Marshal(wire.BrepIdenticalBodiesResult{Groups: groups})
}

// brepCreateFromDefinition serves wire.MethodBrepCreateFromDefinition.
func brepCreateFromDefinition(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.BrepCreateFromDefinitionArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	body, issues, err := s.TransientBodies().CreateFromDefinition(in.Definition)
	if err != nil {
		return nil, err
	}
	if len(issues) > 0 {
		return json.Marshal(wire.BrepCreateFromDefinitionResult{Issues: issues})
	}
	tb := body.(*bodyapi.TransientBody)
	return json.Marshal(wire.BrepCreateFromDefinitionResult{Handle: tb.Handle(), Stats: brepStats(tb)})
}

// brepDescribe serves wire.MethodBrepDescribe.
func brepDescribe(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.BrepHandleArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	tb, ok := s.TransientBodies().ByHandle(in.Handle)
	if !ok {
		return nil, fmt.Errorf("no transient body with handle %d", in.Handle)
	}
	return brepHandleReply(tb)
}

// brepList serves wire.MethodBrepList.
func brepList(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	return json.Marshal(wire.BrepListResult{Handles: s.TransientBodies().Handles()})
}

// brepDelete serves wire.MethodBrepDelete.
func brepDelete(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.BrepHandleArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	if !s.TransientBodies().Delete(in.Handle) {
		return nil, fmt.Errorf("no transient body with handle %d", in.Handle)
	}
	return json.Marshal(wire.OKResult{OK: true})
}
