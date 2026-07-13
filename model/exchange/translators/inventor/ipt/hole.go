// SPDX-License-Identifier: GPL-2.0-only

package ipt

import (
	"encoding/binary"
	"strings"
	"unicode/utf16"
)

// Hole feature decoding from PmDCSegment. Inventor's HoleFeature stores many preset
// dimensions (clearance/tap defaults) as model parameters; the hole TYPE is a kind-3 0x0C20
// enum node (like the boolean operation's kind-5 node), and the extent (through vs blind) is
// inferred geometrically. v1 reads the dimensions from the ordered model parameters:
// [thickness, diameter, preset, counterDiameter, counterDepth, counterAngle, throughDefault].

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

// DecodeHole reports the part's hole, if present. v1: single hole — the ordered model
// parameters are [thickness, diameter, preset, counterDiameter, counterDepth, counterAngle,
// throughDefault]; the type is the kind-3 enum node; the bore is Through All when the last
// parameter reaches the material thickness. Multi-hole, placement, and tap decode are future
// work.
func DecodeHole(seg []byte) (Hole, bool) {
	if !containsUTF16(seg, "Hole1") {
		return Hole{}, false
	}
	mp := modelParamValues(seg)
	if len(mp) < 2 {
		return Hole{}, false
	}
	thickness, diameter, depth := mp[0], mp[1], mp[len(mp)-1]
	if diameter <= 0 {
		return Hole{}, false
	}
	h := Hole{Diameter: diameter, Depth: depth, ThroughAll: depth >= thickness}
	if kinds := enumNodeValues(seg, 3); len(kinds) > 0 {
		h.Type = HoleType(kinds[0])
	}
	switch h.Type {
	case CounterboreHole:
		if len(mp) >= 5 {
			h.CounterDiameter, h.CounterDepth = mp[3], mp[4]
		}
	case CountersinkHole:
		if len(mp) >= 6 {
			h.CounterDiameter, h.CounterAngle = mp[3], mp[5]
		}
	}
	if des, ok := decodeTap(seg); ok {
		h.Tapped, h.Designation = true, des
	}
	return h, true
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
