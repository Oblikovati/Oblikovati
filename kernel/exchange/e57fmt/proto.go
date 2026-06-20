// SPDX-License-Identifier: GPL-2.0-only

package e57fmt

import (
	"encoding/xml"
	"fmt"
	"strconv"
)

// fieldKind is how one prototype field's records are encoded in a CompressedVector bytestream.
type fieldKind int

const (
	kindScaledInteger fieldKind = iota // bit-packed integer; value = raw*scale + offset
	kindInteger                        // bit-packed integer; value = raw
	kindFloat                          // byte-aligned IEEE-754 (single or double)
)

// protoField describes one field of the points prototype (e.g. cartesianX). Min/Max bound the
// stored integer for the bit-packed kinds; Scale/Offset map a ScaledInteger raw value to a real;
// doublePrec selects float64 over float32 for the Float kind.
type protoField struct {
	name       string
	kind       fieldKind
	min        int64
	max        int64
	scale      float64
	offset     float64
	doublePrec bool
}

// pointsSection is the decoded <points> CompressedVector: where its binary section starts
// (a physical offset), how many records it holds, and the prototype field layout.
type pointsSection struct {
	fileOffset  uint64
	recordCount uint64
	fields      []protoField
}

// xmlField captures one prototype child element generically: its element name is the field name
// (cartesianX, intensity, …) and its attributes describe the encoding.
type xmlField struct {
	XMLName   xml.Name
	Type      string `xml:"type,attr"`
	Minimum   string `xml:"minimum,attr"`
	Maximum   string `xml:"maximum,attr"`
	Scale     string `xml:"scale,attr"`
	Offset    string `xml:"offset,attr"`
	Precision string `xml:"precision,attr"`
}

// xmlPrototype collects every child element of <prototype> generically (their tag names are the
// field names), so a prototype with extra fields (intensity, colour, invalid-state) parses too.
type xmlPrototype struct {
	Fields []xmlField `xml:",any"`
}

type xmlPoints struct {
	FileOffset  uint64       `xml:"fileOffset,attr"`
	RecordCount uint64       `xml:"recordCount,attr"`
	Prototype   xmlPrototype `xml:"prototype"`
}

// xmlRoot matches just enough of e57Root to reach the first scan's points; namespace prefixes are
// ignored because the tags carry no namespace (Go matches on local name).
type xmlRoot struct {
	Children []xmlPoints `xml:"data3D>vectorChild>points"`
}

// parsePointsSection unmarshals the E57 XML and returns the first scan's points descriptor.
func parsePointsSection(doc []byte) (pointsSection, error) {
	var root xmlRoot
	if err := xml.Unmarshal(doc, &root); err != nil {
		return pointsSection{}, fmt.Errorf("e57fmt: parsing E57 XML: %w", err)
	}
	if len(root.Children) == 0 {
		return pointsSection{}, fmt.Errorf("e57fmt: E57 has no data3D points section")
	}
	p := root.Children[0]
	fields, err := protoFields(p.Prototype.Fields)
	if err != nil {
		return pointsSection{}, err
	}
	return pointsSection{fileOffset: p.FileOffset, recordCount: p.RecordCount, fields: fields}, nil
}

// protoFields converts the XML field elements to typed prototype fields.
func protoFields(xs []xmlField) ([]protoField, error) {
	out := make([]protoField, 0, len(xs))
	for _, x := range xs {
		f, err := protoFieldFrom(x)
		if err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, nil
}

// protoFieldFrom maps one xmlField to a protoField, defaulting scale to 1 and offset to 0 (the
// E57 defaults when the attributes are absent) and recognising the three storable node types.
func protoFieldFrom(x xmlField) (protoField, error) {
	f := protoField{name: x.XMLName.Local, min: atoiDefault(x.Minimum, 0), max: atoiDefault(x.Maximum, 0)}
	f.scale = atofDefault(x.Scale, 1)
	f.offset = atofDefault(x.Offset, 0)
	switch x.Type {
	case "ScaledInteger":
		f.kind = kindScaledInteger
	case "Integer":
		f.kind = kindInteger
	case "Float":
		f.kind = kindFloat
		f.doublePrec = x.Precision != "single"
	default:
		return protoField{}, fmt.Errorf("e57fmt: prototype field %q has unsupported type %q", f.name, x.Type)
	}
	return f, nil
}

func atoiDefault(s string, def int64) int64 {
	if s == "" {
		return def
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return def
	}
	return n
}

func atofDefault(s string, def float64) float64 {
	if s == "" {
		return def
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return def
	}
	return v
}
