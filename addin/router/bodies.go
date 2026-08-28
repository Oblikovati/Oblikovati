// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"
	stdmath "math"
	"slices"

	"oblikovati.org/addin/modelaccess"
	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/analysis"
	"oblikovati.org/model/compdef"
)

// Body, shell and wire enumeration over the wire (M07-F06, #629), plus the
// planar wire offset. Bodies are addressed by index in the active part.

// resolveBody returns the active part's body at the given index. It is the session-based
// entry (used by the transient-body/wire-offset paths that resolve the active part inline);
// the typedPart handlers resolve the part via the adapter and call bodyAt directly.
func resolveBody(s *app.Session, index int) (*topo.Body, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	return bodyAt(part, index)
}

// bodyAt returns the part's surface body at index, naming the bound on overflow.
func bodyAt(part *compdef.PartComponentDefinition, index int) (*topo.Body, error) {
	all := part.SurfaceBodies().All()
	if index < 0 || index >= len(all) {
		return nil, fmt.Errorf("body index %d out of range (part has %d bodies)", index, len(all))
	}
	return all[index], nil
}

// bodyList serves wire.MethodBodyList.
func bodyList(s *app.Session, part *compdef.PartComponentDefinition) (wire.BodyListResult, error) {
	return bodyListResult(s, part), nil
}

// bodyListResult builds the active part's current body list (shared by body.list and the
// mutating body.delete, which returns the refreshed list).
func bodyListResult(s *app.Session, part *compdef.PartComponentDefinition) wire.BodyListResult {
	var out wire.BodyListResult
	for i, b := range part.SurfaceBodies().All() {
		out.Bodies = append(out.Bodies, bodyInfo(s, i, b))
	}
	return out
}

// bodyInfo builds a body's wire summary: its display name, visibility (#158), persistent
// reference key, assigned color-style name (#1078) and effective material id (the read-back
// of the write-only assignMaterial, for analysis add-ins).
func bodyInfo(s *app.Session, index int, b *topo.Body) wire.BodyInfo {
	key := string(b.ReferenceKey())
	style, _ := s.BodyColorStyle(key)
	return wire.BodyInfo{
		Index: index, Name: bodyDisplayName(s, index, key), Solid: b.IsSolid(), Visible: s.BodyVisible(b),
		Faces: len(b.Faces()), Edges: len(b.Edges()), Vertices: len(b.Vertices()),
		Shells: len(b.Shells()), Wires: len(b.Wires()),
		Key: key, Style: style, MaterialID: s.BodyMaterialID(key),
	}
}

// bodyDisplayName is the body's browser name: the user's stored name for that body (#1078), or the
// index-derived "Solid{N}" default when it was never renamed. Matches the model browser.
func bodyDisplayName(s *app.Session, index int, key string) string {
	if d := s.ActiveDocument(); d != nil {
		if name, ok := d.BodyName(key); ok {
			return name
		}
	}
	return fmt.Sprintf("Solid%d", index+1)
}

// bodyRename serves wire.MethodBodyRename: store (or, with an empty name, clear) the display name
// of one body of the active part, keyed by its reference key so it survives recompute and
// round-trips in the .obk (#1078).
func bodyRename(s *app.Session, part *compdef.PartComponentDefinition, in wire.BodyRenameArgs) (wire.BodyInfoResult, error) {
	b, err := bodyAt(part, in.BodyIndex)
	if err != nil {
		return wire.BodyInfoResult{}, err
	}
	d := s.ActiveDocument()
	if d == nil {
		return wire.BodyInfoResult{}, fmt.Errorf("body.rename: no active document")
	}
	d.SetBodyName(string(b.ReferenceKey()), in.Name)
	return wire.BodyInfoResult{Body: bodyInfo(s, in.BodyIndex, b)}, nil
}

// bodyPhysicalProperties serves wire.MethodBodyPhysicalProperties: the geometry and mass
// properties of one body — the per-body counterpart of analysis.massProperties, which sums all
// bodies (#1078).
func bodyPhysicalProperties(s *app.Session, part *compdef.PartComponentDefinition, in wire.BodyPhysicalPropertiesArgs) (wire.MassPropertiesResult, error) {
	b, err := bodyAt(part, in.BodyIndex)
	if err != nil {
		return wire.MassPropertiesResult{}, err
	}
	accuracy := types.MassPropertiesMedium
	if in.Accuracy != "" {
		a, ok := types.ParseMassPropertiesAccuracy(in.Accuracy)
		if !ok {
			return wire.MassPropertiesResult{}, fmt.Errorf("body.physicalProperties: unknown accuracy %q (want low|medium|high)", in.Accuracy)
		}
		accuracy = a
	}
	density := in.DensityGCm3
	if density == 0 { // default to the part's assigned material density
		if props, ok := s.PhysicalProperties(); ok && props.Density > 0 {
			density = props.Density
		}
	}
	mp := analysis.MassPropertiesOf([]*topo.Body{b}, density, accuracy)
	return massPropertiesResult(mp), nil
}

// bodySetVisible shows or hides one body of the active part (#158); the renderer reads the flag.
func bodySetVisible(s *app.Session, part *compdef.PartComponentDefinition, in wire.BodySetVisibleArgs) (wire.BodyInfoResult, error) {
	b, err := bodyAt(part, in.BodyIndex)
	if err != nil {
		return wire.BodyInfoResult{}, err
	}
	s.SetBodyVisible(b, in.Visible)
	return wire.BodyInfoResult{Body: bodyInfo(s, in.BodyIndex, b)}, nil
}

// bodyDelete serves wire.MethodBodyDelete: append a delete-body feature that removes the body at
// BodyIndex (anchored to its reference key so it survives recompute), then return the refreshed
// body list (#1078).
func bodyDelete(s *app.Session, part *compdef.PartComponentDefinition, in wire.BodyIndexArgs) (wire.BodyListResult, error) {
	b, err := bodyAt(part, in.BodyIndex)
	if err != nil {
		return wire.BodyListResult{}, err
	}
	pf := part.Features().AddDeleteBody(b.ReferenceKey())
	part.Recompute()
	if !pf.Health().OK() {
		return wire.BodyListResult{}, fmt.Errorf("body.delete: %s", pf.Health().Reason)
	}
	return bodyListResult(s, part), nil
}

// bodyShells serves wire.MethodBodyShells.
func bodyShells(_ *app.Session, part *compdef.PartComponentDefinition, in wire.BodyIndexArgs) (wire.BodyShellsResult, error) {
	b, err := bodyAt(part, in.BodyIndex)
	if err != nil {
		return wire.BodyShellsResult{}, err
	}
	q := ops.DefaultQuality()
	var out wire.BodyShellsResult
	for i, sh := range b.Shells() {
		out.Shells = append(out.Shells, shellInfo(i, sh, q))
	}
	return out, nil
}

func shellInfo(i int, sh *topo.Shell, q ops.Quality) wire.FaceShellInfo {
	vol := ops.ShellSignedVolume(sh, q)
	box := sh.RangeBox()
	return wire.FaceShellInfo{
		Index: i, Closed: sh.IsClosed(), Void: sh.IsClosed() && vol < 0,
		Faces: len(sh.Faces()), Edges: len(sh.Edges()),
		Volume:   stdmath.Abs(vol),
		RangeBox: boxSpan(box),
		Key:      string(sh.ReferenceKey()), TransientKey: sh.ID(),
	}
}

func boxSpan(b math.Box) []float64 {
	return []float64{
		float64(b.Min.X), float64(b.Min.Y), float64(b.Min.Z),
		float64(b.Max.X), float64(b.Max.Y), float64(b.Max.Z),
	}
}

// bodyWires serves wire.MethodBodyWires.
func bodyWires(_ *app.Session, part *compdef.PartComponentDefinition, in wire.BodyIndexArgs) (wire.BodyWiresResult, error) {
	b, err := bodyAt(part, in.BodyIndex)
	if err != nil {
		return wire.BodyWiresResult{}, err
	}
	var out wire.BodyWiresResult
	for i, w := range b.Wires() {
		out.Wires = append(out.Wires, wire.WireInfo{
			Index: i, Closed: w.IsClosed(), Planar: w.IsPlanar(), Edges: len(w.Edges()),
			Key: string(w.ReferenceKey()), TransientKey: w.ID(),
		})
	}
	return out, nil
}

// wireOffsetPlanar serves wire.MethodWireOffsetPlanar: the offset result is
// registered as a transient body and returned sampled. The source wire lives on a document
// body OR a transient body (Handle > 0), so this resolves no active-part context up front.
func wireOffsetPlanar(s *app.Session, in wire.OffsetPlanarWireArgs) (wire.OffsetPlanarWireResult, error) {
	w, err := resolveWireRef(s, in)
	if err != nil {
		return wire.OffsetPlanarWireResult{}, err
	}
	normal, err := xyz(in.Normal, "normal")
	if err != nil {
		return wire.OffsetPlanarWireResult{}, err
	}
	corner, err := offsetCorner(in.CornerClosure)
	if err != nil {
		return wire.OffsetPlanarWireResult{}, err
	}
	res, err := ops.OffsetPlanarWire(w, normal, in.Distance, corner)
	if err != nil {
		return wire.OffsetPlanarWireResult{}, err
	}
	tb := s.TransientBodies().Adopt(res)
	return wire.OffsetPlanarWireResult{Handle: tb.Handle(), Wires: wirePolylines(res)}, nil
}

// resolveWireRef finds the wire on a document body (BodyIndex) or a transient
// body (Handle > 0 — sections and silhouettes land their wires there).
func resolveWireRef(s *app.Session, in wire.OffsetPlanarWireArgs) (*topo.Wire, error) {
	b, err := offsetSourceBody(s, in)
	if err != nil {
		return nil, err
	}
	wires := b.Wires()
	if in.WireIndex < 0 || in.WireIndex >= len(wires) {
		return nil, fmt.Errorf("wire index %d out of range (body has %d wires)", in.WireIndex, len(wires))
	}
	return wires[in.WireIndex], nil
}

func offsetSourceBody(s *app.Session, in wire.OffsetPlanarWireArgs) (*topo.Body, error) {
	if in.Handle > 0 {
		tb, ok := s.TransientBodies().ByHandle(in.Handle)
		if !ok {
			return nil, fmt.Errorf("no transient body with handle %d", in.Handle)
		}
		return tb.Topo(), nil
	}
	return resolveBody(s, in.BodyIndex)
}

// offsetCorner maps the wire spelling (empty ⇒ circular, the reference default).
func offsetCorner(spelling string) (ops.WireOffsetCorner, error) {
	if spelling == "" {
		return ops.WireCornerCircular, nil
	}
	kind, ok := types.ParseOffsetCornerClosureType(spelling)
	if !ok {
		return 0, fmt.Errorf("unknown corner closure %q (want circular, linear or extend)", spelling)
	}
	switch kind {
	case types.LinearCornerClosure:
		return ops.WireCornerLinear, nil
	case types.ExtendCornerClosure:
		return ops.WireCornerExtend, nil
	default:
		return ops.WireCornerCircular, nil
	}
}

// wirePolylines samples a body's wires for the reply payload.
func wirePolylines(b *topo.Body) []wire.WirePolyline {
	var out []wire.WirePolyline
	for _, w := range b.Wires() {
		var pts []float64
		for _, u := range w.Uses() {
			pts = appendUseSamples(pts, u)
		}
		out = append(out, wire.WirePolyline{Points: pts, Closed: w.IsClosed()})
	}
	return out
}

func appendUseSamples(pts []float64, u topo.Use) []float64 {
	samples := geom.SampleCurve3(u.Edge.Geometry(), 16) // shared sampler (ADR-0055)
	if u.Reversed {
		slices.Reverse(samples)
	}
	for _, p := range samples {
		pts = append(pts, float64(p.X), float64(p.Y), float64(p.Z))
	}
	return pts
}

// xyz decodes a [x, y, z] vector argument.
func xyz(v []float64, label string) (math.Vector3, error) {
	if len(v) != 3 {
		return math.Vector3{}, fmt.Errorf("%s needs [x, y, z], got %d values", label, len(v))
	}
	return math.V3(math.Scalar(v[0]), math.Scalar(v[1]), math.Scalar(v[2])), nil
}

// xyzPoint decodes a [x, y, z] point argument.
func xyzPoint(v []float64, label string) (math.Point3, error) {
	w, err := xyz(v, label)
	if err != nil {
		return math.Point3{}, err
	}
	return w.AsPoint(), nil
}
