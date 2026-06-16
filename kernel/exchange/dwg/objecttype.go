// SPDX-License-Identifier: GPL-2.0-only

package dwg

// ObjectType is a DWG object/entity type code. Codes below 500 are fixed (the
// same in every file); codes >= 500 are file-specific and resolved through the
// class section (ODA §2.13). Only the fixed types relevant to sketch geometry are
// named here; others decode but stay Unknown for now.
type ObjectType int

// Fixed object type codes (ODA / reference enum). Geometry types the sketch
// converter consumes are the priority; structural types (BLOCK/SEQEND/VERTEX) are
// included because they bound polyline/insert decoding.
const (
	TypeUnused      ObjectType = 0x00
	TypeText        ObjectType = 0x01
	TypeAttrib      ObjectType = 0x02
	TypeAttdef      ObjectType = 0x03
	TypeBlock       ObjectType = 0x04
	TypeEndblk      ObjectType = 0x05
	TypeSeqend      ObjectType = 0x06
	TypeInsert      ObjectType = 0x07
	TypeVertex2D    ObjectType = 0x0A
	TypeVertex3D    ObjectType = 0x0B
	TypePolyline2D  ObjectType = 0x0F
	TypePolyline3D  ObjectType = 0x10
	TypeArc         ObjectType = 0x11
	TypeCircle      ObjectType = 0x12
	TypeLine        ObjectType = 0x13
	TypePoint       ObjectType = 0x1B
	TypeEllipse     ObjectType = 0x23
	TypeSpline      ObjectType = 0x24
	TypeRay         ObjectType = 0x28
	TypeXline       ObjectType = 0x29
	TypeMtext       ObjectType = 0x2C
	TypeBlockHeader ObjectType = 0x31
	TypeLayer       ObjectType = 0x33
	TypeLwpolyline  ObjectType = 0x4D
)

// objectTypeNames maps the fixed codes to their DXF-style names, for diagnostics
// and for matching the oracle's tallies during validation.
var objectTypeNames = map[ObjectType]string{
	TypeUnused: "UNUSED", TypeText: "TEXT", TypeAttrib: "ATTRIB", TypeAttdef: "ATTDEF",
	TypeBlock: "BLOCK", TypeEndblk: "ENDBLK", TypeSeqend: "SEQEND", TypeInsert: "INSERT",
	TypeVertex2D: "VERTEX_2D", TypeVertex3D: "VERTEX_3D",
	TypePolyline2D: "POLYLINE_2D", TypePolyline3D: "POLYLINE_3D",
	TypeArc: "ARC", TypeCircle: "CIRCLE", TypeLine: "LINE", TypePoint: "POINT",
	TypeEllipse: "ELLIPSE", TypeSpline: "SPLINE", TypeRay: "RAY", TypeXline: "XLINE",
	TypeMtext: "MTEXT", TypeBlockHeader: "BLOCK_HEADER", TypeLayer: "LAYER",
	TypeLwpolyline: "LWPOLYLINE",
}

// Name returns the DXF-style name of a fixed type, or "" for an unnamed/custom
// code (the latter resolved via the class section).
func (t ObjectType) Name() string { return objectTypeNames[t] }

// IsSketchGeometry reports whether the type carries 2D/3D curve geometry the
// sketch converter can map (the rest are structural or unsupported for import).
func (t ObjectType) IsSketchGeometry() bool {
	switch t {
	case TypeLine, TypePoint, TypeCircle, TypeArc, TypeEllipse, TypeSpline,
		TypeLwpolyline, TypePolyline2D, TypePolyline3D:
		return true
	default:
		return false
	}
}
