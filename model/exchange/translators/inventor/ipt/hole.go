// SPDX-License-Identifier: GPL-2.0-only

package ipt

import (
	"encoding/binary"
	"strings"
	"unicode/utf16"
)

// Hole feature decoding from PmDCSegment. A HoleFeature is a generic Fx feature node
// (featureNodeType) whose FIRST property is a 0x43CD7C11 "hole kind" node — that node's enum value
// is the HoleType (0 drilled / 1 countersink / 2 counterbore), and the bore dimensions are the
// feature's OWN referenced parameters, at stable property indices (verified on fixtures
// 19/20/25/26/27 and the CapstainNut corpus part):
//
//	prop[1] diameter   prop[2] depth   prop[3] counterDiameter
//	prop[4] counterDepth (counterbore)   prop[5] counterAngle (countersink, radians)
//	prop[9] extent enum (0x92637D29): value 5 = Through All, else blind at prop[2]
//
// This replaced a v1 that read the ordered model-parameter list [thickness, diameter, preset, …]:
// that order held only for the generated fixtures. On a real part with more parameters it picked the
// wrong ones — CapstainNut's bore came out Ø0.55 (the extrude thickness) with depth π/4 (a chamfer
// angle), building the nut with no bore (1.62x). The node's own refs are order-independent.

// holeFeatureType is the "hole kind" node an Fx hole names as its first property; its enum value is
// the HoleType. (InventorLoader Read_43CD7C11.)
const holeFeatureType = 0x43CD7C11

// Hole feature property indices within the Fx feature node.
const (
	holePropKind            = 0 // the 0x43CD7C11 node; its enum value is the HoleType
	holePropDiameter        = 1
	holePropDepth           = 2
	holePropCounterDiameter = 3
	holePropCounterDepth    = 4 // counterbore
	holePropCounterAngle    = 5 // countersink, radians
	holePropExtent          = 9 // 0x92637D29 PartFeatureExtentEnum
)

// holeThroughExtent is the extent enum's "All" value — the bore runs through the whole body, so its
// blind depth is a stale leftover (the same 5 = kExtentAll an extrude uses).
const holeThroughExtent = 5

// HoleType is the hole geometry style, matching Inventor's HoleTypeEnum value.
type HoleType int

const (
	DrilledHole     HoleType = 0
	CountersinkHole HoleType = 1
	CounterboreHole HoleType = 2
)

// Hole is a decoded hole: its type, bore diameter (cm), a through-all or blind-depth extent,
// and — for a counterbore/countersink — the counter diameter plus the counterbore depth or
// the countersink included angle (radians). Placement is handled by the translator (drilled
// on the base feature's top face); off-centre placement and tapped holes are future work.
type Hole struct {
	Type            HoleType
	Diameter        float64
	Depth           float64
	ThroughAll      bool
	CounterDiameter float64 // counterbore / countersink
	CounterDepth    float64 // counterbore
	CounterAngle    float64 // countersink, radians
	Tapped          bool    // a threaded (tapped) hole — the bore is the tap-drill cylinder
	Designation     string  // thread callout (e.g. "M6x1"), tapped holes only
}

// DecodeHole reports the part's hole, if present, reading its bore from the HoleFeature node's own
// parameter references (see the layout above). v1: single hole — the first hole feature. Multi-hole
// and placement are future work.
//
//	if h, ok := DecodeHole(d); ok { addHole(def, h, cx, cy, thickness) }
func DecodeHole(d *Document) (Hole, bool) {
	nodes := dcNodes(d)
	for _, n := range nodes {
		if n.typ != featureNodeType {
			continue
		}
		props, ok := featureProperties(n.payload)
		if !ok || len(props) <= holePropExtent {
			continue
		}
		kind, ok := nodeAt(nodes, props[holePropKind])
		if !ok || kind.typ != holeFeatureType {
			continue
		}
		h := holeFromNode(nodes, props, kind)
		if h.Diameter <= 0 {
			return Hole{}, false // a hole node with no bore is malformed, not a hole to build
		}
		if seg, ok := d.Segment("PmDCSegment"); ok {
			if des, ok := decodeTap(seg); ok {
				h.Tapped, h.Designation = true, des
			}
		}
		return h, true
	}
	return Hole{}, false
}

// holeFromNode reads a hole's dimensions from its feature node's own property references. The kind
// node's enum value gives the HoleType; the bore/depth/counter values are referenced parameters.
func holeFromNode(nodes []dcNode, props []int, kind dcNode) Hole {
	h := Hole{
		Diameter: featureParameter(nodes, props, holePropDiameter),
		Depth:    featureParameter(nodes, props, holePropDepth),
	}
	if v, ok := enumValueOf(kind); ok {
		h.Type = HoleType(v)
	}
	if v, ok := featureEnum(nodes, props, holePropExtent); ok {
		h.ThroughAll = v == holeThroughExtent
	}
	switch h.Type {
	case CounterboreHole:
		h.CounterDiameter = featureParameter(nodes, props, holePropCounterDiameter)
		h.CounterDepth = featureParameter(nodes, props, holePropCounterDepth)
	case CountersinkHole:
		h.CounterDiameter = featureParameter(nodes, props, holePropCounterDiameter)
		h.CounterAngle = featureParameter(nodes, props, holePropCounterAngle)
	}
	return h
}

// threadTypeKeywords mark a thread-type string (e.g. "ANSI Metric M Profile", "ANSI Unified
// Screw Threads", an NPT pipe thread). A plain drilled hole carries no such string.
var threadTypeKeywords = []string{"Profile", "Thread", "Screw", "Pipe"}

// decodeTap reports a hole's thread designation when it is tapped. The tap info is stored as
// Len32Text16 records [designation][threadType][…][class]; the designation is the record
// ending exactly where the threadType record (identified by a keyword) begins.
func decodeTap(seg []byte) (string, bool) {
	for i := 0; i+4 <= len(seg); i++ {
		s, ok := readLen32Text16(seg, i)
		if !ok || !containsAny(s, threadTypeKeywords) {
			continue
		}
		if des, ok := recordEndingAt(seg, i); ok {
			return des, true
		}
	}
	return "", false
}

// readLen32Text16 reads a u32-length-prefixed UTF-16LE string of printable ASCII at off,
// returning ok=false when the bytes are not such a record.
func readLen32Text16(seg []byte, off int) (string, bool) {
	n := int(binary.LittleEndian.Uint32(seg[off:]))
	if n < 1 || n > 64 || off+4+2*n > len(seg) {
		return "", false
	}
	u := make([]uint16, n)
	for k := 0; k < n; k++ {
		c := binary.LittleEndian.Uint16(seg[off+4+2*k:])
		if c < 0x20 || c > 0x7e {
			return "", false
		}
		u[k] = c
	}
	return string(utf16.Decode(u)), true
}

// recordEndingAt returns the Len32Text16 string whose record ends exactly at end (its length
// prefix sits 4+2n bytes before end), scanning candidate lengths.
func recordEndingAt(seg []byte, end int) (string, bool) {
	for n := 1; n <= 64; n++ {
		start := end - 4 - 2*n
		if start < 0 {
			break
		}
		if int(binary.LittleEndian.Uint32(seg[start:])) == n {
			if s, ok := readLen32Text16(seg, start); ok && len([]rune(s)) == n {
				return s, true
			}
		}
	}
	return "", false
}

func containsAny(s string, subs []string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
