// SPDX-License-Identifier: GPL-2.0-only

package clientgraphics

import (
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/math"
)

// DecodeGroup converts a wire submit request into a kernel-typed Group, validating the
// flat-array shapes once at the boundary so Build can assume them. Lane defaults to
// persistent; Visible defaults to true.
func DecodeGroup(args wire.SetClientGraphicsArgs) (Group, error) {
	if args.ClientId == "" {
		return Group{}, fmt.Errorf("clientGraphics: empty clientId in %+v", args)
	}
	lane, err := decodeLane(args.Lane, LanePersistent)
	if err != nil {
		return Group{}, err
	}
	nodes, err := decodeNodes(args.Nodes)
	if err != nil {
		return Group{}, err
	}
	return Group{clientID: args.ClientId, lane: lane, visible: args.Visible == nil || *args.Visible, nodes: nodes}, nil
}

// decodeLane maps a wire lane string to a Lane, applying fallback for an empty value.
func decodeLane(s string, fallback Lane) (Lane, error) {
	switch lane := Lane(s); lane {
	case "":
		return fallback, nil
	case LanePersistent, LaneOverlay, LanePreview:
		return lane, nil
	default:
		return "", fmt.Errorf("clientGraphics: unknown lane %q, want persistent|overlay|preview", s)
	}
}

// decodeNodes decodes each wire node, failing on the first malformed one.
func decodeNodes(in []wire.GraphicsNode) ([]Node, error) {
	out := make([]Node, len(in))
	for i, n := range in {
		node, err := decodeNode(n)
		if err != nil {
			return nil, fmt.Errorf("node[%d]: %w", i, err)
		}
		out[i] = node
	}
	return out, nil
}

// decodeNode decodes a node's transform/visibility and its primitives.
func decodeNode(n wire.GraphicsNode) (Node, error) {
	xf, has, err := decodeTransform(n.Transform)
	if err != nil {
		return Node{}, err
	}
	prims := make([]Primitive, len(n.Primitives))
	for i, p := range n.Primitives {
		prim, err := decodePrimitive(p)
		if err != nil {
			return Node{}, fmt.Errorf("primitive[%d]: %w", i, err)
		}
		prims[i] = prim
	}
	return Node{Transform: xf, HasTransform: has, Visible: n.Visible, Opacity: n.Opacity, Primitives: prims}, nil
}

// decodeTransform builds a Matrix4 from a 16-element row-major slice; an empty slice
// means "no transform" (identity), any other length is an error.
func decodeTransform(t []float64) (math.Matrix4, bool, error) {
	if len(t) == 0 {
		return math.Identity4(), false, nil
	}
	if len(t) != 16 {
		return math.Matrix4{}, false, fmt.Errorf("transform has %d cells, want 16 (row-major 4x4)", len(t))
	}
	var cells [16]math.Scalar
	copy(cells[:], t)
	return math.Matrix4FromCells(cells), true, nil
}

// decodePrimitive validates and converts one wire primitive into kernel types.
func decodePrimitive(p wire.GraphicsPrimitive) (Primitive, error) {
	kind := types.GraphicsPrimitiveKind(p.Kind)
	out := Primitive{
		Kind: kind, Indices: p.Indices, Scalars: p.Scalars,
		ColorBinding:  types.GraphicsColorBinding(p.ColorBinding),
		NormalBinding: types.GraphicsNormalBinding(p.NormalBinding),
		LineType:      types.GraphicsLineType(p.LineType), LineWeight: p.LineWeight,
		PointStyle: types.GraphicsPointStyle(p.PointStyle), PointSize: p.PointSize,
		Text: p.Text, FontSize: p.FontSize, Opacity: p.Opacity, OnTop: p.OnTop,
	}
	var err error
	if out.Coords, err = decodePoints("coordinates", p.Coordinates); err != nil {
		return Primitive{}, err
	}
	if out.Normals, err = decodeVectors("normals", p.Normals); err != nil {
		return Primitive{}, err
	}
	if out.Colors, err = decodeColors("colors", p.Colors); err != nil {
		return Primitive{}, err
	}
	if out.Color, err = decodeColor(p.Color); err != nil {
		return Primitive{}, err
	}
	if out.Mapper, err = decodeMapper(p.ColorMapper); err != nil {
		return Primitive{}, err
	}
	out.Anchor = anchorPoint(p.Anchor)
	return out, nil
}

// decodePoints converts an xyz-triple flat array into points.
func decodePoints(field string, a []float64) ([]math.Point3, error) {
	if len(a)%3 != 0 {
		return nil, fmt.Errorf("%s has %d values, want a multiple of 3 (xyz triples)", field, len(a))
	}
	out := make([]math.Point3, len(a)/3)
	for i := range out {
		out[i] = math.P3(a[i*3], a[i*3+1], a[i*3+2])
	}
	return out, nil
}

// decodeVectors converts an xyz-triple flat array into vectors.
func decodeVectors(field string, a []float64) ([]math.Vector3, error) {
	if len(a)%3 != 0 {
		return nil, fmt.Errorf("%s has %d values, want a multiple of 3 (xyz triples)", field, len(a))
	}
	out := make([]math.Vector3, len(a)/3)
	for i := range out {
		out[i] = math.V3(a[i*3], a[i*3+1], a[i*3+2])
	}
	return out, nil
}

// decodeColors converts an rgba-quad flat array into colors.
func decodeColors(field string, a []float32) ([][4]float32, error) {
	if len(a)%4 != 0 {
		return nil, fmt.Errorf("%s has %d values, want a multiple of 4 (rgba quads)", field, len(a))
	}
	out := make([][4]float32, len(a)/4)
	for i := range out {
		out[i] = [4]float32{a[i*4], a[i*4+1], a[i*4+2], a[i*4+3]}
	}
	return out, nil
}

// decodeColor reads an optional overall rgba color; empty means opaque white.
func decodeColor(a []float32) ([4]float32, error) {
	if len(a) == 0 {
		return [4]float32{1, 1, 1, 1}, nil
	}
	if len(a) != 4 {
		return [4]float32{}, fmt.Errorf("color has %d values, want 4 (rgba)", len(a))
	}
	return [4]float32{a[0], a[1], a[2], a[3]}, nil
}

// decodeMapper validates and converts a color mapper (len(Colors) == 4*len(Values)).
func decodeMapper(m *wire.GraphicsColorMapper) (*ColorMapper, error) {
	if m == nil {
		return nil, nil
	}
	if len(m.Values) == 0 || len(m.Colors) != 4*len(m.Values) {
		return nil, fmt.Errorf("colorMapper has %d values and %d color components, want components == 4*values", len(m.Values), len(m.Colors))
	}
	colors := make([][4]float32, len(m.Values))
	for i := range colors {
		colors[i] = [4]float32{m.Colors[i*4], m.Colors[i*4+1], m.Colors[i*4+2], m.Colors[i*4+3]}
	}
	return &ColorMapper{Values: m.Values, Colors: colors}, nil
}

// anchorPoint reads a text primitive's xyz anchor (short/empty → origin).
func anchorPoint(a []float64) math.Point3 {
	if len(a) < 3 {
		return math.P3(0, 0, 0)
	}
	return math.P3(a[0], a[1], a[2])
}
