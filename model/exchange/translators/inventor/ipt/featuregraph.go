// SPDX-License-Identifier: GPL-2.0-only

package ipt

// The DC feature graph: a feature node names its operands (profile, direction, depth) by
// REFERENCE, rather than storing them inline. Decoding those references is what binds an extrude
// to its own depth — the positional "i-th model parameter" guess this replaces silently picked an
// unrelated sketch dimension on any part with more than one parameter (BigChunkyPlate extruded a
// 3 cm plate 40 cm, 26x its true volume).
//
// Node types and the property layout are grounded in the GPL InventorLoader reference
// (_iptformat/reference-InventorLoader): Read_90874D91 "Feature", Read_90874D26 "Parameter", and
// Create_FxExtrude_New's property indices. Cross-checked against files: 14_box_two's two extrudes
// resolve to Parameter nodes holding exactly 1 cm and 2 cm.

import (
	"encoding/binary"
	"math"
)

const (
	featureNodeType   = 0x90874D91 // InventorLoader Read_90874D91 — the generic "Fx" feature
	parameterNodeType = 0x90874D26 // Read_90874D26 — a named parameter with its value
	directionAxisType = 0xCE52DF40 // an extrude's direction; absent on revolve/other features
)

// Property indices on a feature node, from InventorLoader's Create_FxExtrude_New. Only the two we
// need are named: a feature whose property 2 is a DirectionAxis is an extrude, and its property 4
// is the Parameter holding the depth (dimLength1).
const (
	propDirection = 2
	propDistance  = 4
)

// Byte layout of a feature node's payload, for this Inventor generation (block size 0, and the
// post-2023 4-byte gap in the content header). The property list is a List2 of cross-references:
// marker at 34, count at 38, then an 8-byte list header, so the refs start at 50.
const (
	featurePropListOffset = 34
	list2Marker           = 0x0002
	list2Owner            = 0x3000
	maxListElements       = 4096
	// refIndexMask strips SecNodeRef's high flag bit; the rest is the node's ordinal.
	refIndexMask = 0x7FFFFFFF
)

// dcNode is one node of the DC segment. A reference names a node by its 1-BASED ORDINAL in walk
// order (InventorLoader assigns node.index = nodeCounter), not by any field inside the node — the
// distinction matters: keying on the u32 at +22 resolves nothing.
type dcNode struct {
	typ     uint32
	payload []byte
}

// dcNodes returns every node of the DC segment in walk order, so a reference r resolves to
// nodes[r-1].
func dcNodes(d *Document) []dcNode {
	var out []dcNode
	d.walkSegment("PmDCSegment", func(typ uint32, pay []byte) bool {
		out = append(out, dcNode{typ, pay})
		return true
	})
	return out
}

// nodeAt resolves a 1-based reference; ok=false when it points outside the segment.
func nodeAt(nodes []dcNode, ref int) (dcNode, bool) {
	if ref < 1 || ref > len(nodes) {
		return dcNode{}, false
	}
	return nodes[ref-1], true
}

// featureProperties returns a feature node's property references (1-based node ordinals) in order.
func featureProperties(pay []byte) ([]int, bool) {
	o := featurePropListOffset
	if o+8 > len(pay) ||
		binary.LittleEndian.Uint16(pay[o:]) != list2Marker ||
		binary.LittleEndian.Uint16(pay[o+2:]) != list2Owner {
		return nil, false
	}
	cnt := int(binary.LittleEndian.Uint32(pay[o+4:]))
	if cnt <= 0 || cnt > 256 {
		return nil, false
	}
	ds := o + 16 // count (4) then the list's 8-byte header
	if ds+cnt*4 > len(pay) {
		return nil, false
	}
	out := make([]int, cnt)
	for i := range out {
		// the high bit is a ref flag, not part of the index (InventorLoader SecNodeRef.mask)
		out[i] = int(binary.LittleEndian.Uint32(pay[ds+i*4:]) & refIndexMask)
	}
	return out, true
}

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

// ExtrudeDepths returns the depth of every extrude feature, in feature order, each read from that
// feature's own referenced distance parameter.
func ExtrudeDepths(d *Document) []float64 {
	nodes := dcNodes(d)
	var out []float64
	for _, n := range nodes {
		if n.typ != featureNodeType {
			continue
		}
		if v, ok := extrudeDepth(nodes, n.payload); ok {
			out = append(out, v)
		}
	}
	return out
}

// The profile side of the feature graph: an extrude names a BoundaryPatch (property 1), and a
// FaceBound node points BACK at that patch while naming the Sketch2D it bounds. So the profile
// binding is patch -> faceBound -> sketch, which is what replaces "extrude i consumes sketch i" —
// a guess that is provably wrong on real parts (BigChunkyPlate's 13th extrude uses its 15th
// sketch). Grounded in InventorLoader: Read_22947391 "BoundaryPatch", Read_424EB7D7 "FaceBound",
// Read_F9884C43 "FaceBoundOuter", ReadHeaderFaceBound.
const (
	boundaryPatchNodeType  = 0x22947391
	faceBoundNodeType      = 0x424EB7D7
	faceBoundOuterNodeType = 0xF9884C43
)

// propProfile is the extrude's BoundaryPatch property (InventorLoader Create_FxExtrude_New's
// `patch = getProperty(properties, 0x01)`).
const propProfile = 1

// faceBoundHeaderEnd is where a FaceBound's three reference lists begin; its `sketch` and `proxy`
// refs follow them and a u32 pair.
const faceBoundHeaderEnd = 30

// ExtrudeProfiles returns, for each extrude in feature order, the index of the sketch it actually
// consumes (indices into GraphSketches). An extrude whose profile can't be resolved yields -1, so
// callers skip it rather than build on a guessed sketch.
func ExtrudeProfiles(d *Document) []int {
	nodes := dcNodes(d)
	_, sketchIndex := sketchOrdinals(nodes)
	patchToSketch := boundPatchSketches(nodes, sketchIndex)
	var out []int
	for _, n := range nodes {
		if n.typ != featureNodeType {
			continue
		}
		if _, ok := extrudeDepth(nodes, n.payload); !ok {
			continue // not an extrude; keep this list aligned with DecodeExtrudes
		}
		out = append(out, profileOf(n.payload, patchToSketch))
	}
	return out
}

// profileOf maps one extrude to its sketch index via its BoundaryPatch property.
func profileOf(pay []byte, patchToSketch map[int]int) int {
	props, ok := featureProperties(pay)
	if !ok || len(props) <= propProfile {
		return -1
	}
	if i, ok := patchToSketch[props[propProfile]]; ok {
		return i
	}
	return -1
}

// boundPatchSketches maps each BoundaryPatch ordinal to the index of the sketch that bounds it,
// by walking the FaceBound nodes that name the patch as their proxy.
func boundPatchSketches(nodes []dcNode, sketchIndex map[int]int) map[int]int {
	out := map[int]int{}
	for _, n := range nodes {
		if n.typ != faceBoundNodeType && n.typ != faceBoundOuterNodeType {
			continue
		}
		sketchRef, patchRef, ok := faceBoundRefs(n.payload)
		if !ok {
			continue
		}
		if i, ok := sketchIndex[sketchRef]; ok {
			out[patchRef] = i
		}
	}
	return out
}

// faceBoundRefs reads a FaceBound's `sketch` and `proxy` references, which follow its three
// variable-length reference lists and a u32 pair.
func faceBoundRefs(pay []byte) (sketchRef, patchRef int, ok bool) {
	i := faceBoundHeaderEnd
	for k := 0; k < 3; k++ {
		_, next, ok := refList2(pay, i)
		if !ok {
			return 0, 0, false
		}
		i = next
	}
	i += 8 // the u32 pair before the refs
	if i+8 > len(pay) {
		return 0, 0, false
	}
	sketchRef = int(binary.LittleEndian.Uint32(pay[i:]) & refIndexMask)
	patchRef = int(binary.LittleEndian.Uint32(pay[i+4:]) & refIndexMask)
	return sketchRef, patchRef, true
}
