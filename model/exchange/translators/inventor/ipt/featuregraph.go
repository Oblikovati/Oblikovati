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
//
// This file is the GRAPH DECODING: the raw node list (dcNodes), the pre-2023 content-header
// normalisation, reference resolution (nodeAt) and the property ref-list (featureProperties). The
// per-node value readers live in featuregraph_payload.go; the profile/axis dependency queries in
// featuregraph_deps.go.

import (
	"encoding/binary"
)

const (
	featureNodeType   = 0x90874D91 // InventorLoader Read_90874D91 — the generic "Fx" feature
	parameterNodeType = 0x90874D26 // Read_90874D26 — a named parameter with its value
	directionAxisType = 0xCE52DF40 // an extrude's direction; absent on revolve/other features
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
// nodes[r-1]. Pre-2023 saves lack a 4-byte content-header gap this decoder (calibrated to Inventor
// 2027) expects, so their entity fields sit 4 bytes earlier; such a segment is normalised once to
// the 2027 layout (see olderContentHeader / padContentHeader) so every offset-34/38/42 decoder
// reads it unchanged. The result is memoised because the whole feature/sketch decode re-walks it.
func dcNodes(d *Document) []dcNode {
	if d.dcCached {
		return d.dcCache
	}
	var out []dcNode
	d.walkSegment("PmDCSegment", func(typ uint32, pay []byte) bool {
		out = append(out, dcNode{typ, pay})
		return true
	})
	if olderContentHeader(out) {
		for i := range out {
			out[i].payload = padContentHeader(out[i].payload, out[i].typ)
		}
	}
	d.dcCache, d.dcCached = out, true
	return out
}

// contentHeaderGapFor returns where a pre-2023 node of the given type is missing the 4-byte
// content-header gap 2027 adds. It is NODE-TYPE-DEPENDENT: a type's fields before the gap keep their
// offset while everything after shifts +4, and types differ in how much header precedes the gap.
//   - Most types (sketch entity, feature, FaceBound, FxAttrStyles) carry the gap at 22: a
//     SketchPoint's sentinel sits at 26 on 2027 / 22 on older, so flags@30→34, owner@34→38,
//     coords@38→42, a feature's property marker@30→34, a FaceBound's refs@26→30 all realign.
//   - A Loop keeps its operation at 10 (unshifted) and carries the gap at 14, so its edge list
//     realigns 14→18.
//   - A SketchEntityRef carries the gap at 18, so its associative id realigns 18→22 (and its
//     later point/sketch refs 30/42/50/51→34/46/54/55).
//
// Inserting a uniform gap breaks this: at 22 the Loop/EntityRef fields below 22 never move (no
// region resolves); lower than 14 shifts a Loop's operation off its correct offset.
func contentHeaderGapFor(typ uint32) int {
	switch typ {
	case loopNodeType, loop3DNodeType:
		return 14
	case entityRefNodeType:
		return 18
	default:
		return 22
	}
}

// padContentHeader inserts the missing 4-byte content-header gap at this node type's position so a
// pre-2023 node reads at the 2027 offsets. See contentHeaderGapFor.
func padContentHeader(pay []byte, typ uint32) []byte {
	at := contentHeaderGapFor(typ)
	if len(pay) < at {
		return pay
	}
	out := make([]byte, len(pay)+4)
	copy(out, pay[:at])
	copy(out[at+4:], pay[at:])
	return out
}

// olderContentHeader reports whether the DC segment uses the pre-2023 layout, detected from the
// property/point List2 marker (0x0002,0x3000) that a feature node carries: at offset 34 on a 2027
// save, at 30 on an older one. Falls back to a line's edge-list marker (42 vs 38) when the part has
// no feature nodes; defaults to 2027 (no normalisation) when neither is present, so a file this
// heuristic can't classify is left exactly as today.
func olderContentHeader(nodes []dcNode) bool {
	newer, older := 0, 0
	for _, n := range nodes {
		switch n.typ {
		case featureNodeType:
			newer, older = tallyMarker(n.payload, featurePropListOffset, newer, older)
		case line2DNodeType:
			newer, older = tallyMarker(n.payload, edgePointsListOffset, newer, older)
		}
	}
	return older > newer
}

// tallyMarker increments newer/older by whether the List2 marker sits at the 2027 offset off or the
// 4-byte-earlier pre-2023 offset off-4.
func tallyMarker(pay []byte, off, newer, older int) (int, int) {
	if hasList2Marker(pay, off) {
		newer++
	} else if hasList2Marker(pay, off-4) {
		older++
	}
	return newer, older
}

// hasList2Marker reports whether a List2 header (marker + owner) begins at off.
func hasList2Marker(pay []byte, off int) bool {
	return off >= 0 && off+4 <= len(pay) &&
		binary.LittleEndian.Uint16(pay[off:]) == list2Marker &&
		binary.LittleEndian.Uint16(pay[off+2:]) == list2Owner
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
