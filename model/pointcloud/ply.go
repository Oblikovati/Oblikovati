// SPDX-License-Identifier: GPL-2.0-only

package pointcloud

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"strconv"
	"strings"

	omath "oblikovati.org/math"
)

// PLY point reader (M17-F06, #645): decodes the vertex positions of a Stanford PLY file —
// the common export of structured-light / photogrammetry scanners (Artec, etc.). A PLY may carry
// a full mesh (vertices + faces); a point cloud needs only the vertex x/y/z, so the reader parses
// the header, reads the vertex element's positions (skipping any extra per-vertex properties such
// as normals or colour), and ignores the faces. Both ASCII and binary (little/big-endian)
// encodings are handled. Clean-room from the published PLY format.
type plyReader struct{}

// NewPLYReader returns the reader for Stanford .ply scan files.
func NewPLYReader() PointReader { return plyReader{} }

func (plyReader) Extensions() []string { return []string{".ply"} }

// plyProperty is one declared property of an element: its scalar type (empty for a list property,
// which the reader skips) and, for a list, the size of its count and element scalars.
type plyProperty struct {
	name      string
	scalar    string // "" for a list property
	listCount string // list count type (e.g. "uchar"), "" for scalar
	listElem  string // list element type
}

// plyElement is one declared element block: its name, instance count, and properties in order.
type plyElement struct {
	name  string
	count int
	props []plyProperty
}

// Read decodes the PLY's vertex positions into cloud-local points.
func (plyReader) Read(data []byte) ([]omath.Point3, error) {
	format, elements, body, err := parsePLYHeader(data)
	if err != nil {
		return nil, err
	}
	vtx, ok := findElement(elements, "vertex")
	if !ok || vtx.count == 0 {
		return nil, fmt.Errorf("pointcloud: PLY has no vertex element")
	}
	xi, yi, zi, err := xyzIndices(vtx)
	if err != nil {
		return nil, err
	}
	if format == "ascii" {
		return readASCIIVertices(body, vtx, xi, yi, zi)
	}
	return readBinaryVertices(body, vtx, xi, yi, zi, format == "binary_big_endian")
}

// parsePLYHeader reads the ASCII header and returns the format, the element declarations, and the
// body bytes that follow "end_header".
func parsePLYHeader(data []byte) (string, []plyElement, []byte, error) {
	end := bytes.Index(data, []byte("end_header"))
	if !bytes.HasPrefix(data, []byte("ply")) || end < 0 {
		return "", nil, nil, fmt.Errorf("pointcloud: not a PLY file (missing magic / end_header)")
	}
	body := data[end:]
	if nl := bytes.IndexByte(body, '\n'); nl >= 0 {
		body = body[nl+1:]
	}
	format := ""
	var elements []plyElement
	sc := bufio.NewScanner(bytes.NewReader(data[:end]))
	for sc.Scan() {
		f := strings.Fields(sc.Text())
		switch {
		case len(f) >= 2 && f[0] == "format":
			format = f[1]
		case len(f) >= 3 && f[0] == "element":
			n, _ := strconv.Atoi(f[2])
			elements = append(elements, plyElement{name: f[1], count: n})
		case len(f) >= 2 && f[0] == "property" && len(elements) > 0:
			appendProperty(&elements[len(elements)-1], f)
		}
	}
	return format, elements, body, nil
}

// appendProperty adds a property declaration to the element under construction.
func appendProperty(e *plyElement, f []string) {
	if f[1] == "list" && len(f) >= 5 {
		e.props = append(e.props, plyProperty{name: f[4], listCount: f[2], listElem: f[3]})
		return
	}
	e.props = append(e.props, plyProperty{name: f[len(f)-1], scalar: f[1]})
}

// findElement returns the named element.
func findElement(elements []plyElement, name string) (plyElement, bool) {
	for _, e := range elements {
		if e.name == name {
			return e, true
		}
	}
	return plyElement{}, false
}

// xyzIndices returns the property indices of x, y, z in the vertex element, erroring if absent or
// if any precedes a variable-size list property (which the position decode cannot stride past).
func xyzIndices(vtx plyElement) (int, int, int, error) {
	idx := map[string]int{}
	for i, p := range vtx.props {
		if p.scalar == "" {
			return 0, 0, 0, fmt.Errorf("pointcloud: PLY vertex has a list property %q before its positions", p.name)
		}
		idx[p.name] = i
	}
	xi, okx := idx["x"]
	yi, oky := idx["y"]
	zi, okz := idx["z"]
	if !okx || !oky || !okz {
		return 0, 0, 0, fmt.Errorf("pointcloud: PLY vertex has no x/y/z properties")
	}
	return xi, yi, zi, nil
}

// readASCIIVertices reads count whitespace-separated vertex lines, taking x/y/z by column.
func readASCIIVertices(body []byte, vtx plyElement, xi, yi, zi int) ([]omath.Point3, error) {
	out := make([]omath.Point3, 0, vtx.count)
	sc := bufio.NewScanner(bytes.NewReader(body))
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for sc.Scan() && len(out) < vtx.count {
		f := strings.Fields(sc.Text())
		if len(f) <= zi {
			continue
		}
		x, ex := strconv.ParseFloat(f[xi], 64)
		y, ey := strconv.ParseFloat(f[yi], 64)
		z, ez := strconv.ParseFloat(f[zi], 64)
		if ex != nil || ey != nil || ez != nil {
			return nil, fmt.Errorf("pointcloud: PLY ascii vertex %d is not numeric: %q", len(out), sc.Text())
		}
		out = append(out, omath.P3(omath.Scalar(x), omath.Scalar(y), omath.Scalar(z)))
	}
	return out, nil
}

// readBinaryVertices strides over the fixed-size vertex records, decoding x/y/z by their byte
// offset and scalar type. bigEndian selects the byte order.
func readBinaryVertices(body []byte, vtx plyElement, xi, yi, zi int, bigEndian bool) ([]omath.Point3, error) {
	offsets, sizes, stride := vertexLayout(vtx)
	order := binary.ByteOrder(binary.LittleEndian)
	if bigEndian {
		order = binary.BigEndian
	}
	out := make([]omath.Point3, 0, vtx.count)
	for i := 0; i < vtx.count; i++ {
		base := i * stride
		if base+stride > len(body) {
			return nil, fmt.Errorf("pointcloud: PLY truncated at vertex %d of %d", i, vtx.count)
		}
		rec := body[base : base+stride]
		out = append(out, omath.P3(
			omath.Scalar(scalarValue(rec[offsets[xi]:], vtx.props[xi].scalar, sizes[xi], order)),
			omath.Scalar(scalarValue(rec[offsets[yi]:], vtx.props[yi].scalar, sizes[yi], order)),
			omath.Scalar(scalarValue(rec[offsets[zi]:], vtx.props[zi].scalar, sizes[zi], order)),
		))
	}
	return out, nil
}

// vertexLayout returns each property's byte offset and size within a vertex record, plus the
// record stride.
func vertexLayout(vtx plyElement) (offsets, sizes []int, stride int) {
	offsets = make([]int, len(vtx.props))
	sizes = make([]int, len(vtx.props))
	for i, p := range vtx.props {
		offsets[i] = stride
		sizes[i] = plyTypeSize(p.scalar)
		stride += sizes[i]
	}
	return offsets, sizes, stride
}

// scalarValue decodes one numeric value of the given PLY scalar type to a float64.
func scalarValue(b []byte, typ string, size int, order binary.ByteOrder) float64 {
	switch typ {
	case "float", "float32":
		return float64(math.Float32frombits(order.Uint32(b)))
	case "double", "float64":
		return math.Float64frombits(order.Uint64(b))
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
		_ = size
		return 0
	}
}

// plyTypeSize returns the byte size of a PLY scalar type (0 for unknown).
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
