// SPDX-License-Identifier: GPL-2.0-only

package dwg

import "fmt"

// ObjectRef locates one object in the file by its handle. Offset is a byte offset
// into the object-data stream: the whole file for R13–R2000, or the assembled
// AcDb:AcDbObjects logical section for R2004+ (ODA §23).
type ObjectRef struct {
	Handle uint64
	Offset int64
}

// objectMapMaxSection bounds a single object-map section. ODA §23 caps sections
// near 2032 bytes; the size field counts itself plus the pairs and is observed
// just above 2032, so 2040 (the reference decoder's guard) bounds a mis-aligned
// stream.
const objectMapMaxSection = 2040

// parseObjectMap decodes the handle→offset directory. The map is a run of
// big-endian-sized sections; within each, (handle, location) pairs are stored as
// deltas from running totals — handle deltas unsigned and increasing, location
// deltas signed. A section whose size is 2 (CRC only) terminates the map.
//
// Example:
//
//	refs, err := parseObjectMap(handlesSection) // refs[i].Offset into AcDbObjects
//
//nolint:funlen // section + (handle,location) delta-decode loop; length is the wire format.
func parseObjectMap(data []byte) ([]ObjectRef, error) {
	r := NewBitReader(data)
	// Pre-size from the map length: each (handle,location) pair is a UMC+MC delta of at
	// least ~3 bytes, so len/4 is a close lower bound on the final count and spares the
	// slice the repeated doubling-and-copy that dominated decode allocation (157 MB → ~4 MB).
	refs := make([]ObjectRef, 0, len(data)/4)
	for r.Err() == nil {
		size := int(r.ReadRC())<<8 | int(r.ReadRC()) // section size, big-endian
		if size == 2 {
			break // final section: just the CRC
		}
		if size < 2 || size > objectMapMaxSection {
			return nil, fmt.Errorf("dwg object map: implausible section size %d at byte %d", size, r.Position()/8)
		}
		// The handle/offset accumulators reset to 0 at the start of every section:
		// each section is an independent delta chain (the first pair carries the
		// absolute handle and offset). The size field counts itself, so the pair
		// bytes run to startpos+size and the 2-byte CRC follows.
		var lastHandle uint64
		var lastLoc int64
		end := r.Position()/8 + size - 2
		for r.Position()/8 < end && r.Err() == nil {
			lastHandle += r.ReadUMC()
			lastLoc += int64(r.ReadMC())
			refs = append(refs, ObjectRef{Handle: lastHandle, Offset: lastLoc})
		}
		r.ReadRC() // skip the 2-byte section CRC
		r.ReadRC()
	}
	if err := r.Err(); err != nil {
		return nil, fmt.Errorf("dwg object map: %w", err)
	}
	if len(refs) == 0 {
		return nil, fmt.Errorf("dwg object map: no objects decoded from %d bytes", len(data))
	}
	return refs, nil
}
