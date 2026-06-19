// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"

	"oblikovati.org/addin/modelaccess"
	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Body, shell and wire enumeration over the wire (M07-F06, #629), plus the
// planar wire offset. Bodies are addressed by index in the active part.

// resolveBody returns the active part's body at the given index.
func resolveBody(s *app.Session, index int) (*topo.Body, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	all := part.SurfaceBodies().All()
	if index < 0 || index >= len(all) {
		return nil, fmt.Errorf("body index %d out of range (part has %d bodies)", index, len(all))
	}
	return all[index], nil
}

// bodyList serves wire.MethodBodyList.
func bodyList(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	var out wire.BodyListResult
	for i, b := range part.SurfaceBodies().All() {
		out.Bodies = append(out.Bodies, bodyInfo(s, i, b))
	}
	return json.Marshal(out)
}

// bodyInfo builds a body's wire summary, including its display name and current visibility (#158).
func bodyInfo(s *app.Session, index int, b *topo.Body) wire.BodyInfo {
	return wire.BodyInfo{
		Index: index, Name: bodyDisplayName(index), Solid: b.IsSolid(), Visible: s.BodyVisible(b),
		Faces: len(b.Faces()), Edges: len(b.Edges()), Vertices: len(b.Vertices()),
		Shells: len(b.Shells()), Wires: len(b.Wires()),
	}
}

// bodyDisplayName is the body's browser name ("Solid1", …), index-derived until bodies carry
// stored, renamable names (the #158 rename follow-up). Matches the model browser.
func bodyDisplayName(index int) string { return fmt.Sprintf("Solid%d", index+1) }

// bodySetVisible shows or hides one body of the active part (#158); the renderer reads the flag.
func bodySetVisible(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.BodySetVisibleArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	b, err := resolveBody(s, in.BodyIndex)
	if err != nil {
		return nil, err
	}
	s.SetBodyVisible(b, in.Visible)
	return json.Marshal(wire.BodyInfoResult{Body: bodyInfo(s, in.BodyIndex, b)})
}

// bodyShells serves wire.MethodBodyShells.
func bodyShells(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.BodyIndexArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	b, err := resolveBody(s, in.BodyIndex)
	if err != nil {
		return nil, err
	}
	q := ops.DefaultQuality()
	var out wire.BodyShellsResult
	for i, sh := range b.Shells() {
		out.Shells = append(out.Shells, shellInfo(i, sh, q))
	}
	return json.Marshal(out)
}

func shellInfo(i int, sh *topo.Shell, q ops.Quality) wire.FaceShellInfo {
	vol := ops.ShellSignedVolume(sh, q)
	box := sh.RangeBox()
	return wire.FaceShellInfo{
		Index: i, Closed: sh.IsClosed(), Void: sh.IsClosed() && vol < 0,
		Faces: len(sh.Faces()), Edges: len(sh.Edges()),
		Volume:   absFloat(vol),
		RangeBox: boxSpan(box),
		Key:      string(sh.ReferenceKey()), TransientKey: sh.ID(),
	}
}

func absFloat(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

func boxSpan(b math.Box) []float64 {
	return []float64{
		float64(b.Min.X), float64(b.Min.Y), float64(b.Min.Z),
		float64(b.Max.X), float64(b.Max.Y), float64(b.Max.Z),
	}
}

// bodyWires serves wire.MethodBodyWires.
func bodyWires(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.BodyIndexArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	b, err := resolveBody(s, in.BodyIndex)
	if err != nil {
		return nil, err
	}
	var out wire.BodyWiresResult
	for i, w := range b.Wires() {
		out.Wires = append(out.Wires, wire.WireInfo{
			Index: i, Closed: w.IsClosed(), Planar: w.IsPlanar(), Edges: len(w.Edges()),
			Key: string(w.ReferenceKey()), TransientKey: w.ID(),
		})
	}
	return json.Marshal(out)
}

// wireOffsetPlanar serves wire.MethodWireOffsetPlanar: the offset result is
// registered as a transient body and returned sampled.
func wireOffsetPlanar(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.OffsetPlanarWireArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	w, err := resolveWireRef(s, in)
	if err != nil {
		return nil, err
	}
	normal, err := xyz(in.Normal, "normal")
	if err != nil {
		return nil, err
	}
	corner, err := offsetCorner(in.CornerClosure)
	if err != nil {
		return nil, err
	}
	res, err := ops.OffsetPlanarWire(w, normal, in.Distance, corner)
	if err != nil {
		return nil, err
	}
	tb := s.TransientBodies().Adopt(res)
	return json.Marshal(wire.OffsetPlanarWireResult{Handle: tb.Handle(), Wires: wirePolylines(res)})
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
	const per = 16
	c := u.Edge.Geometry()
	lo, hi := c.Domain()
	for i := 0; i <= per; i++ {
		t := float64(i) / per
		if u.Reversed {
			t = 1 - t
		}
		p := c.PointAt(lo + (hi-lo)*t)
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
