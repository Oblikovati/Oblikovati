// SPDX-License-Identifier: GPL-2.0-only

package ipt

// The DC feature graph — PER-NODE PAYLOAD READERS (M48 #2228 split of featuregraph.go). Given a
// resolved node, these decode ONE value from its payload: a Parameter's f64, a Boolean's byte, an
// enum node's u16, a DirectionAxis's unit vector — plus the feature-relative readers that resolve a
// property slot to its node and read that value (featureParameter/featureBoolean/featureEnum,
// extrudeDirection/extrudeDepth/extrudeOperation/extrudeThroughAll). The node-list decode and the
// dependency resolution live in featuregraph.go and featuregraph_deps.go.

import (
	"encoding/binary"
	"math"
)

// Property indices on a feature node, from InventorLoader's Create_FxExtrude_New. A feature whose
// property 2 is a DirectionAxis is an extrude, and its property 4 is the Parameter holding the
// depth (dimLength1).
const (
	propReversed  = 3 // Boolean: grow against the profile normal
	propDirection = 2
	propDistance  = 4
	propMidplane  = 7    // Boolean: grow half each way about the sketch plane
	propDistance2 = 0x1B // Parameter: the second direction's length (a two-sided extrude)
)

// booleanNodeType is InventorLoader's Read_90874D28 "Boolean".
const booleanNodeType = 0x90874D28

// directionVecOffset is where a DirectionAxis holds its unit direction. InventorLoader
// Read_CE52DF40: header, an f64, the >2017 4-byte gap, then the f64x3 'dir'. Self-validating —
// every one of the corpus's 252 extrude directions reads back with |d| = 1 here.
const directionVecOffset = 50

// booleanNameOffset is where a Boolean node's length-prefixed UTF-16 name begins: the content
// header runs to 34, exactly as on a Parameter.
const booleanNameOffset = 34

// parameterValue reads a Parameter node's nominal value. Layout after the content header: a
// length-prefixed UTF-16 name at 34, a 4-byte gap, the unit and formula refs, then the f64.
func parameterValue(pay []byte) (float64, bool) {
	const nameLenOffset = 34
	if nameLenOffset+4 > len(pay) {
		return 0, false
	}
	n := int(binary.LittleEndian.Uint32(pay[nameLenOffset:]))
	if n < 0 || n > 256 {
		return 0, false
	}
	at := nameLenOffset + 4 + 2*n + 12 // name, gap(4), unit ref(4), formula ref(4)
	if at+8 > len(pay) {
		return 0, false
	}
	v := math.Float64frombits(binary.LittleEndian.Uint64(pay[at:]))
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, false
	}
	return v, true
}

// booleanValue reads a Boolean node's value. Layout after the content header (InventorLoader
// Read_90874D28): a length-prefixed UTF-16 name, a 4-byte gap, then ONE byte — NOT the u16 an enum
// node carries at 36. Reading it at the enum offset lands on the name-length field, which is 0 on
// every node here, so the whole corpus silently reported false.
func booleanValue(pay []byte) (bool, bool) {
	if booleanNameOffset+4 > len(pay) {
		return false, false
	}
	n := int(binary.LittleEndian.Uint32(pay[booleanNameOffset:]))
	if n < 0 || n > 256 {
		return false, false
	}
	at := booleanNameOffset + 4 + 2*n + 4 // name, gap(4)
	if at >= len(pay) {
		return false, false
	}
	return pay[at] != 0, true
}

// featureBoolean reads the Boolean a feature's property i references (false when absent).
func featureBoolean(nodes []dcNode, props []int, i int) bool {
	if i >= len(props) {
		return false
	}
	n, ok := nodeAt(nodes, props[i])
	if !ok || n.typ != booleanNodeType {
		return false
	}
	v, ok := booleanValue(n.payload)
	return ok && v
}

// enumValueOf reads a node's 16-bit enum value (an operation, extent, or hole-kind enum), and whether
// the payload is long enough to hold one.
func enumValueOf(n dcNode) (uint16, bool) {
	if len(n.payload) < enumValueOffset+2 {
		return 0, false
	}
	return binary.LittleEndian.Uint16(n.payload[enumValueOffset:]), true
}

// featureEnum reads the enum value the feature's i-th property names, and whether that property
// resolves to a node carrying one — the generic form of extrudeExtentEnum for any property slot
// (a hole's extent sits at a different index than an extrude's).
func featureEnum(nodes []dcNode, props []int, i int) (uint16, bool) {
	if i >= len(props) {
		return 0, false
	}
	n, ok := nodeAt(nodes, props[i])
	if !ok {
		return 0, false
	}
	return enumValueOf(n)
}

// extrudeDirection reads the unit direction the feature's DirectionAxis names, and whether it read
// at all. This is the vector the extrude actually grows along; `reversed` flips THIS, not the
// profile normal — so reversed cannot be interpreted without it (see Extrude.Dir).
func extrudeDirection(nodes []dcNode, props []int) ([3]float64, bool) {
	var d [3]float64
	if propDirection >= len(props) {
		return d, false
	}
	n, ok := nodeAt(nodes, props[propDirection])
	if !ok || n.typ != directionAxisType || len(n.payload) < directionVecOffset+24 {
		return d, false
	}
	for i := 0; i < 3; i++ {
		d[i] = math.Float64frombits(binary.LittleEndian.Uint64(n.payload[directionVecOffset+8*i:]))
	}
	l := math.Sqrt(d[0]*d[0] + d[1]*d[1] + d[2]*d[2])
	if math.IsNaN(l) || math.Abs(l-1) > 1e-6 {
		return [3]float64{}, false // not a unit vector ⇒ not this layout; decline rather than guess
	}
	return d, true
}

// featureParameter reads the Parameter value a feature's property i references (0 when absent).
func featureParameter(nodes []dcNode, props []int, i int) float64 {
	if i >= len(props) {
		return 0
	}
	n, ok := nodeAt(nodes, props[i])
	if !ok || n.typ != parameterNodeType {
		return 0
	}
	v, ok := parameterValue(n.payload)
	if !ok {
		return 0
	}
	return v
}

// extrudeDepth returns the depth a feature node references, and whether that node is an extrude at
// all: an extrude is the feature whose direction property is a DirectionAxis (a revolve, hole or
// fillet has something else there), and its distance property must resolve to a Parameter.
func extrudeDepth(nodes []dcNode, pay []byte) (float64, bool) {
	props, ok := featureProperties(pay)
	if !ok || len(props) <= propDistance {
		return 0, false
	}
	dir, ok := nodeAt(nodes, props[propDirection])
	if !ok || dir.typ != directionAxisType {
		return 0, false
	}
	dist, ok := nodeAt(nodes, props[propDistance])
	if !ok || dist.typ != parameterNodeType {
		return 0, false
	}
	return parameterValue(dist.payload)
}

// partFeatureOperationEnumType is the enum node an extrude names as its FIRST property: the boolean
// operation it applies (InventorLoader Read_729ABE28 "PartFeatureOperationEnum", whose values —
// 1 NewBody, 2 Cut, 3 Join, 4 Intersection — are the OpNewBody..OpIntersect constants).
const partFeatureOperationEnumType = 0x729ABE28

// propOperation is the extrude's operation property (InventorLoader Create_FxExtrude reads
// `getPropertyValue(properties, 0x00, 'value')`).
const propOperation = 0

// enumValueOffset is where an enum node holds its value: the content header runs to 34, then a
// 2-byte type discriminator.
const enumValueOffset = 36

// extrudeOperation returns the boolean operation a feature node NAMES, and whether it names one.
// The predecessor took the i-th enum node found by scanning the segment's bytes, which pairs
// operations with extrudes by position — the same guess that made every extrude take the wrong
// depth, and just as wrong on a part whose features are not authored in scan order.
func extrudeOperation(nodes []dcNode, pay []byte) (int, bool) {
	props, ok := featureProperties(pay)
	if !ok || len(props) <= propOperation {
		return 0, false
	}
	n, ok := nodeAt(nodes, props[propOperation])
	if !ok || n.typ != partFeatureOperationEnumType || len(n.payload) < enumValueOffset+2 {
		return 0, false
	}
	return int(binary.LittleEndian.Uint16(n.payload[enumValueOffset:])), true
}

// propExtent is the extrude's termination property (InventorLoader Create_FxExtrude_New reads
// `extend = getPropertyValue(properties, 0x06, 'value')`).
const propExtent = 6

// extentAll is the `extend` value meaning the extrude runs through ALL the material, taking its
// length from the body rather than from a parameter. Verified on BigChunkyPlate: its four
// distance-less extrudes read 5 here while every measured one reads 1.
const extentAll = 5

// extentEnumNodeType is the PartFeatureExtentEnum node (InventorLoader Read_92637D29).
const extentEnumNodeType = 0x92637D29

// extrudeThroughAll reports whether a feature terminates by running through all the material. Such
// an extrude has NO distance: its depth parameter decodes as 0, and building that as a length makes
// a zero-thickness body.
func extrudeThroughAll(nodes []dcNode, pay []byte) bool {
	props, ok := featureProperties(pay)
	if !ok {
		return false
	}
	v, ok := extrudeExtentEnum(nodes, props)
	return ok && v == extentAll
}
