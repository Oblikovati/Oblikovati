// SPDX-License-Identifier: GPL-2.0-only

package pointcloud

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	stdmath "math"
	"strconv"
	"strings"

	"oblikovati.org/kernel/exchange/plyfmt"
	"oblikovati.org/math"
)

// PLY point reader (M17-F06, #645): a point cloud needs only the vertex positions of a Stanford
// PLY (the common 3D-scanner export), so this reader delegates the format parsing to the shared
// kernel/exchange/plyfmt package and takes its vertices plus any common scan color/intensity
// vertex properties.
type plyReader struct{}

// NewPLYReader returns the reader for Stanford .ply scan files.
func NewPLYReader() PointReader { return plyReader{} }

func (plyReader) Extensions() []string { return []string{".ply"} }

// FileUnitMM: PLY carries no unit, so it follows the same declared millimetre convention as the
// unitless mesh formats (STL/OBJ) — the .ply mesh/cloud symmetry test pins this (#1636).
func (plyReader) FileUnitMM() float64 { return 1 }

// ReadSamples decodes the PLY's vertex positions into cloud-local samples (faces are ignored).
// The vertex list layout comes from the header, so a fault is structural: no per-record warnings.
func (plyReader) ReadSamples(data []byte) ([]PointSample, []string, error) {
	doc, err := plyfmt.Parse(data)
	if err != nil {
		return nil, nil, err
	}
	vtx, ok := vertexElement(doc)
	if !ok {
		return nil, nil, fmt.Errorf("plyfmt: PLY has no vertex element")
	}
	samples, err := decodePLYVertexSamples(doc, vtx)
	return samples, nil, err
}

// Read returns point-only coordinates for callers that do not need channels.
func (r plyReader) Read(data []byte) ([]math.Point3, []string, error) {
	samples, warns, err := r.ReadSamples(data)
	if err != nil {
		return nil, nil, err
	}
	out := make([]math.Point3, len(samples))
	for i, s := range samples {
		out[i] = s.Point
	}
	return out, warns, nil
}

func vertexElement(doc *plyfmt.Document) (plyfmt.Element, bool) {
	for _, e := range doc.Elements {
		if e.Name == "vertex" && e.Count > 0 {
			return e, true
		}
	}
	return plyfmt.Element{}, false
}

func decodePLYVertexSamples(doc *plyfmt.Document, vtx plyfmt.Element) ([]PointSample, error) {
	xi, yi, zi, intensityIdx, rgbIdx, err := plyVertexIndices(vtx)
	if err != nil {
		return nil, err
	}
	if doc.Format == "ascii" {
		return decodePLYAsciiSamples(doc.Body(), vtx, xi, yi, zi, intensityIdx, rgbIdx)
	}
	return decodePLYBinarySamples(doc.Body(), doc.Format, vtx, xi, yi, zi, intensityIdx, rgbIdx)
}

func plyVertexIndices(vtx plyfmt.Element) (xi, yi, zi int, intensityIdx int, rgbIdx [3]int, err error) {
	xi, yi, zi, intensityIdx = -1, -1, -1, -1
	rgbIdx = [3]int{-1, -1, -1}
	// A name→index-slot table keeps the property scan a flat map lookup rather than a wide switch.
	slot := map[string]*int{
		"x": &xi, "y": &yi, "z": &zi, "intensity": &intensityIdx,
		"red": &rgbIdx[0], "green": &rgbIdx[1], "blue": &rgbIdx[2],
	}
	for i, p := range vtx.Props {
		if p.Scalar == "" {
			return 0, 0, 0, 0, [3]int{}, fmt.Errorf("plyfmt: vertex has a list property %q before its positions", p.Name)
		}
		if dst, ok := slot[p.Name]; ok {
			*dst = i
		}
	}
	if xi < 0 || yi < 0 || zi < 0 {
		return 0, 0, 0, 0, [3]int{}, fmt.Errorf("plyfmt: vertex has no x/y/z properties")
	}
	return xi, yi, zi, intensityIdx, rgbIdx, nil
}

func decodePLYAsciiSamples(body []byte, vtx plyfmt.Element, xi, yi, zi, intensityIdx int, rgbIdx [3]int) ([]PointSample, error) {
	out := make([]PointSample, 0, vtx.Count)
	sc := bufio.NewScanner(bytes.NewReader(body))
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for sc.Scan() && len(out) < vtx.Count {
		f := strings.Fields(sc.Text())
		if len(f) == 0 {
			continue
		}
		s, ok := plyAsciiSample(f, vtx.Props, xi, yi, zi, intensityIdx, rgbIdx)
		if !ok {
			return nil, fmt.Errorf("plyfmt: ascii vertex %d is malformed: %q", len(out), sc.Text())
		}
		out = append(out, s)
	}
	if len(out) < vtx.Count {
		return nil, fmt.Errorf("plyfmt: truncated at vertex %d of %d", len(out), vtx.Count)
	}
	return out, nil
}

func plyAsciiSample(f []string, props []plyfmt.Property, xi, yi, zi, intensityIdx int, rgbIdx [3]int) (PointSample, bool) {
	if len(f) <= plyMaxIndex(xi, yi, zi) {
		return PointSample{}, false
	}
	pt, ok := plyAsciiPoint(f, props, xi, yi, zi)
	if !ok {
		return PointSample{}, false
	}
	s := PointSample{Point: pt}
	if intensityIdx >= 0 {
		if v, ok := scalarFieldValue(f, props[intensityIdx].Scalar, intensityIdx); ok {
			s.HasIntensity = true
			s.Intensity = v
		}
	}
	if rgbIdx[0] >= 0 && rgbIdx[1] >= 0 && rgbIdx[2] >= 0 {
		if rgb, ok := plyAsciiRGB(f, props, rgbIdx); ok {
			s.HasRGB = true
			s.RGB = rgb
		}
	}
	return s, true
}

// plyMaxIndex returns the largest of the three position column indices — the minimum field count an
// ascii vertex line must have to carry x, y and z.
func plyMaxIndex(xi, yi, zi int) int {
	m := xi
	if yi > m {
		m = yi
	}
	if zi > m {
		m = zi
	}
	return m
}

// plyAsciiPoint parses the x/y/z columns of one ascii vertex line, reporting false if any is missing
// or malformed.
func plyAsciiPoint(f []string, props []plyfmt.Property, xi, yi, zi int) (math.Point3, bool) {
	var pt math.Point3
	var ok bool
	if pt.X, ok = scalarFieldValue(f, props[xi].Scalar, xi); !ok {
		return math.Point3{}, false
	}
	if pt.Y, ok = scalarFieldValue(f, props[yi].Scalar, yi); !ok {
		return math.Point3{}, false
	}
	if pt.Z, ok = scalarFieldValue(f, props[zi].Scalar, zi); !ok {
		return math.Point3{}, false
	}
	return pt, true
}

func plyAsciiRGB(f []string, props []plyfmt.Property, rgbIdx [3]int) ([3]float32, bool) {
	var rgb [3]float32
	for i := 0; i < 3; i++ {
		v, ok := scalarFieldValue(f, props[rgbIdx[i]].Scalar, rgbIdx[i])
		if !ok {
			return [3]float32{}, false
		}
		rgb[i] = float32(v)
	}
	return rgb, true
}

func scalarFieldValue(fields []string, typ string, idx int) (float64, bool) {
	if idx >= len(fields) {
		return 0, false
	}
	v, err := strconv.ParseFloat(fields[idx], 64)
	if err != nil {
		return 0, false
	}
	switch typ {
	case "char", "int8", "uchar", "uint8", "short", "int16", "ushort", "uint16", "int", "int32", "uint", "uint32", "float", "float32", "double", "float64":
		return v, true
	default:
		return 0, false
	}
}

func decodePLYBinarySamples(body []byte, format string, vtx plyfmt.Element, xi, yi, zi, intensityIdx int, rgbIdx [3]int) ([]PointSample, error) {
	offsets, sizes, stride := plyLayout(vtx)
	order := plyByteOrder(format)
	out := make([]PointSample, 0, vtx.Count)
	for i := 0; i < vtx.Count; i++ {
		base := i * stride
		if base+stride > len(body) {
			return nil, fmt.Errorf("plyfmt: truncated at vertex %d of %d", i, vtx.Count)
		}
		rec := body[base : base+stride]
		out = append(out, plyBinarySample(rec, vtx, offsets, sizes, order, xi, yi, zi, intensityIdx, rgbIdx))
	}
	return out, nil
}

// plyByteOrder maps the PLY header's binary format string to its byte order.
func plyByteOrder(format string) binary.ByteOrder {
	if format == "binary_big_endian" {
		return binary.BigEndian
	}
	return binary.LittleEndian
}

// plyBinarySample decodes one fixed-layout binary vertex record into a sample, reading position and
// any intensity / RGB channels at their prototype offsets. A single val closure removes the repeated
// offset/size/order threading each channel would otherwise carry.
func plyBinarySample(rec []byte, vtx plyfmt.Element, offsets, sizes []int, order binary.ByteOrder, xi, yi, zi, intensityIdx int, rgbIdx [3]int) PointSample {
	val := func(idx int) float64 {
		return plyScalarValue(rec[offsets[idx]:], vtx.Props[idx].Scalar, sizes[idx], order)
	}
	s := PointSample{Point: math.P3(math.Scalar(val(xi)), math.Scalar(val(yi)), math.Scalar(val(zi)))}
	if intensityIdx >= 0 {
		s.HasIntensity = true
		s.Intensity = val(intensityIdx)
	}
	if rgbIdx[0] >= 0 && rgbIdx[1] >= 0 && rgbIdx[2] >= 0 {
		s.HasRGB = true
		s.RGB = [3]float32{float32(val(rgbIdx[0])), float32(val(rgbIdx[1])), float32(val(rgbIdx[2]))}
	}
	return s
}

func plyLayout(e plyfmt.Element) (offsets, sizes []int, stride int) {
	offsets = make([]int, len(e.Props))
	sizes = make([]int, len(e.Props))
	for i, p := range e.Props {
		offsets[i] = stride
		sizes[i] = plyTypeSize(p.Scalar)
		stride += sizes[i]
	}
	return offsets, sizes, stride
}

func plyScalarValue(b []byte, typ string, _ int, order binary.ByteOrder) float64 {
	switch typ {
	case "float", "float32":
		return float64(stdmath.Float32frombits(order.Uint32(b)))
	case "double", "float64":
		return stdmath.Float64frombits(order.Uint64(b))
	case "char", "int8":
		return float64(int8(b[0]))
	case "uchar", "uint8":
		return float64(b[0])
	case "short", "int16":
		return float64(int16(order.Uint16(b)))
	case "ushort", "uint16":
		return float64(order.Uint16(b))
	case "int", "int32":
		return float64(int32(order.Uint32(b)))
	case "uint", "uint32":
		return float64(order.Uint32(b))
	default:
		return 0
	}
}

func plyTypeSize(typ string) int {
	switch typ {
	case "char", "uchar", "int8", "uint8":
		return 1
	case "short", "ushort", "int16", "uint16":
		return 2
	case "int", "uint", "int32", "uint32", "float", "float32":
		return 4
	case "double", "float64":
		return 8
	default:
		return 0
	}
}
