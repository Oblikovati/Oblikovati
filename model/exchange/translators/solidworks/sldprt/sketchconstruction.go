// SPDX-License-Identifier: GPL-2.0-only

package sldprt

import (
	"bytes"
	"encoding/binary"
)

// constructionAnchor precedes each sketch entity's flag word in the trailing entity-record table:
// the four bytes 80 bf 00 00 (the tail of the entity's canonical -1.0f frame component plus two pad
// bytes), immediately followed by the u32 entity flag. Bit constructionBit of that flag marks
// construction (reference) geometry. Verified on generated known parts across lines, arcs, and
// circles: a two-entity sketch whose second entity is toggled construction yields entity flags
// [0x04, 0x0c]; all-normal parts yield only 0x04.
var constructionAnchor = []byte{0x80, 0xbf, 0x00, 0x00}

// entityFlagNormal / entityFlagConstruction are the two observed entity-record flag values; the only
// difference is constructionBit. Restricting to these two rejects the many coincidental
// constructionAnchor matches elsewhere in the stream (which carry a zero or unrelated following u32).
const (
	entityFlagNormal       = 0x04
	entityFlagConstruction = 0x0c
	constructionBit        = 0x08
)

// entityConstruction returns, in entity-creation order, whether each sketch entity is construction
// (reference) geometry. SolidWorks serialises one flag record per entity in a table that follows all
// of the sketch's cached point coordinates; each record's flag word (0x04 normal, 0x0c construction)
// sits right after constructionAnchor. The records that appear before the last cached point are
// per-point handles (always 0x04) and are skipped, so the result has exactly one entry per curve
// entity in draw order. Proven to match entity count and construction count on every generated known
// part (lines, arcs, circles; dimensioned and plain). The returned order is raw creation order and
// mixes entity kinds; callers map it onto typed geometry only where that order is recoverable.
func entityConstruction(region []byte) []bool {
	last := lastPointOffset(region)
	var flags []bool
	for i := 0; i+len(constructionAnchor)+4 <= len(region); {
		m := bytes.Index(region[i:], constructionAnchor)
		if m < 0 {
			break
		}
		at := i + m
		i = at + 1
		if at <= last {
			continue // a per-point handle record, not an entity record
		}
		flag := binary.LittleEndian.Uint32(region[at+len(constructionAnchor):])
		if flag != entityFlagNormal && flag != entityFlagConstruction {
			continue
		}
		flags = append(flags, flag&constructionBit != 0)
	}
	return flags
}

// lastPointOffset returns the byte offset of the last cached sketch coordinate in the region, or -1
// if none. It is the boundary between the per-point handle records and the trailing entity-record
// table: sketch geometry (points) is serialised first, the entity table after, and any dimension
// sub-features after that (they cache no in-range coordinate before the table).
func lastPointOffset(region []byte) int {
	last := -1
	for i := 0; ; {
		m := bytes.Index(region[i:], pointMarker)
		if m < 0 {
			break
		}
		at := i + m
		if _, ok := readPoint(region, at+len(pointMarker)); ok {
			last = at
		}
		i = at + 1
	}
	return last
}
