// SPDX-License-Identifier: GPL-2.0-only

// Package plyfmt is a clean-room low-level reader for the Stanford PLY format (ASCII and binary,
// little/big-endian), shared by the point-cloud scan reader (vertices only) and the mesh-exchange
// importer (vertices + faces) so the header/scalar parsing lives in one place (#645). It parses
// the header into typed element declarations and reads the vertex positions and face index lists
// on demand; it deliberately does not weld or triangulate — callers build their own point set or
// triangle soup.
package plyfmt

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

// Property is one declared property of an element: its scalar type (empty for a list property)
// and, for a list, the count and element scalar types.
type Property struct {
	Name      string
	Scalar    string
	ListCount string
	ListElem  string
}

// Element is one declared element block: its name, instance count, and properties in order.
type Element struct {
	Name  string
	Count int
	Props []Property
}

// Document is a parsed PLY: its encoding, element declarations, and the body bytes (everything
// after end_header).
type Document struct {
	Format   string
	Elements []Element
	body     []byte
}

// Parse reads the ASCII header and captures the body, erroring on a non-PLY input.
func Parse(data []byte) (*Document, error) {
	end := bytes.Index(data, []byte("end_header"))
	if !bytes.HasPrefix(data, []byte("ply")) || end < 0 {
		return nil, fmt.Errorf("plyfmt: not a PLY file (missing magic / end_header)")
	}
	body := data[end:]
	if nl := bytes.IndexByte(body, '\n'); nl >= 0 {
		body = body[nl+1:]
	}
	d := &Document{body: body}
	sc := bufio.NewScanner(bytes.NewReader(data[:end]))
	for sc.Scan() {
		f := strings.Fields(sc.Text())
		switch {
		case len(f) >= 2 && f[0] == "format":
			d.Format = f[1]
		case len(f) >= 3 && f[0] == "element":
			n, _ := strconv.Atoi(f[2])
			d.Elements = append(d.Elements, Element{Name: f[1], Count: n})
		case len(f) >= 2 && f[0] == "property" && len(d.Elements) > 0:
			d.appendProperty(f)
		}
	}
	return d, nil
}

func (d *Document) appendProperty(f []string) {
	e := &d.Elements[len(d.Elements)-1]
	if f[1] == "list" && len(f) >= 5 {
		e.Props = append(e.Props, Property{Name: f[4], ListCount: f[2], ListElem: f[3]})
		return
	}
	e.Props = append(e.Props, Property{Name: f[len(f)-1], Scalar: f[1]})
}

// Body returns the bytes after end_header, ready for vertex/face decoding.
func (d *Document) Body() []byte { return d.body }

func (d *Document) element(name string) (Element, bool) {
	for _, e := range d.Elements {
		if e.Name == name {
			return e, true
		}
	}
	return Element{}, false
}

func (d *Document) byteOrder() binary.ByteOrder {
	if d.Format == "binary_big_endian" {
		return binary.BigEndian
	}
	return binary.LittleEndian
}

// Vertices reads the vertex element's x/y/z positions. It is the position-only projection of Scan
// for callers (the mesh importer) that do not need colour or intensity.
func (d *Document) Vertices() ([]omath.Point3, error) {
	scan, err := d.Scan()
	if err != nil {
		return nil, err
	}
	return scan.Points, nil
}

// Faces reads the face element's vertex-index lists (variable length); nil when there is no face
// element. It is read after the vertices, so positions and faces decode from one pass each.
func (d *Document) Faces() ([][]int, error) {
	face, ok := d.element("face")
	if !ok || face.Count == 0 {
		return nil, nil
	}
	if d.Format == "ascii" {
		return d.asciiFaces(face)
	}
	return d.binaryFaces(face)
}

// asciiFaces reads the face lines after the vertex lines: "N i0 i1 … i(N-1)".
func (d *Document) asciiFaces(face Element) ([][]int, error) {
	vtx, _ := d.element("vertex")
	sc := bufio.NewScanner(bytes.NewReader(d.body))
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for skipped := 0; skipped < vtx.Count && sc.Scan(); {
		if len(strings.Fields(sc.Text())) > 0 {
			skipped++
		}
	}
	out := make([][]int, 0, face.Count)
	for sc.Scan() && len(out) < face.Count {
		f := strings.Fields(sc.Text())
		if len(f) == 0 {
			continue
		}
		n, err := strconv.Atoi(f[0])
		if err != nil || len(f) < n+1 {
			return nil, fmt.Errorf("plyfmt: malformed ascii face %d: %q", len(out), sc.Text())
		}
		idx := make([]int, n)
		for k := 0; k < n; k++ {
			idx[k], _ = strconv.Atoi(f[k+1])
		}
		out = append(out, idx)
	}
	return out, nil
}

// binaryFaces walks the variable-length face records that follow the fixed-size vertex records.
func (d *Document) binaryFaces(face Element) ([][]int, error) {
	vtx, _ := d.element("vertex")
	_, _, vStride := layout(vtx)
	pos := vtx.Count * vStride
	prop := face.Props[0] // a face's single list property (vertex_indices)
	out := make([][]int, 0, face.Count)
	for i := 0; i < face.Count; i++ {
		idx, next, err := d.readBinaryFace(pos, prop, i)
		if err != nil {
			return nil, err
		}
		out = append(out, idx)
		pos = next
	}
	return out, nil
}

// readBinaryFace reads one binary face record at pos: a list-count value then that many index
// values. It returns the indices and the position after the record.
func (d *Document) readBinaryFace(pos int, prop Property, i int) ([]int, int, error) {
	order := d.byteOrder()
	countSize := plyTypeSize(prop.ListCount)
	elemSize := plyTypeSize(prop.ListElem)
	if pos+countSize > len(d.body) {
		return nil, 0, fmt.Errorf("plyfmt: truncated reading face %d count", i)
	}
	n := int(scalarValue(d.body[pos:], prop.ListCount, countSize, order))
	pos += countSize
	if pos+n*elemSize > len(d.body) {
		return nil, 0, fmt.Errorf("plyfmt: truncated in face %d indices", i)
	}
	idx := make([]int, n)
	for k := 0; k < n; k++ {
		idx[k] = int(scalarValue(d.body[pos:], prop.ListElem, elemSize, order))
		pos += elemSize
	}
	return idx, pos, nil
}

// layout returns each property's byte offset and size in a fixed-size record, plus the stride.
func layout(e Element) (offsets, sizes []int, stride int) {
	offsets = make([]int, len(e.Props))
	sizes = make([]int, len(e.Props))
	for i, p := range e.Props {
		offsets[i] = stride
		sizes[i] = plyTypeSize(p.Scalar)
		stride += sizes[i]
	}
	return offsets, sizes, stride
}

// scalarValue decodes one numeric value of the given PLY scalar type to a float64.
func scalarValue(b []byte, typ string, _ int, order binary.ByteOrder) float64 {
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
