// SPDX-License-Identifier: GPL-2.0-only

package dwg

import "fmt"

// ObjectHeader is the decoded preamble of one object: its handle and offset (from
// the object map), its byte size, and its type. It is the entry point for the
// per-type geometry decoders that follow.
type ObjectHeader struct {
	Handle uint64
	Offset int64
	Size   int
	Type   ObjectType
}

// decodeObjectHeader reads an object's preamble at its offset in the object-data
// stream (ODA §2.13): an MS byte size, then on R2010+ a UMC handle-stream size
// (not counted in the size), then the type — a BOT on R2010+ or a plain BitShort
// before. data is the AcDb:AcDbObjects section (R2004+) or the whole file (R2000).
func decodeObjectHeader(data []byte, ref ObjectRef, version Version) (ObjectHeader, error) {
	if ref.Offset < 0 || ref.Offset >= int64(len(data)) {
		return ObjectHeader{}, fmt.Errorf("dwg: object handle %d offset %d out of bounds (len %d)", ref.Handle, ref.Offset, len(data))
	}
	// A stack-local reader (vs NewBitReaderAt's heap *BitReader): this runs once per
	// object in the map — 100k+ on large drawings — and the reader does not escape.
	var r BitReader
	r.Reset(data, int(ref.Offset)*8)
	size := r.ReadMS()
	if version >= R2010 {
		r.ReadUMC() // handle-stream size, not part of the object size
	}
	var typ int
	if version >= R2010 {
		typ = r.ReadBOT()
	} else {
		typ = r.ReadBS()
	}
	if err := r.Err(); err != nil {
		return ObjectHeader{}, fmt.Errorf("dwg: object handle %d header: %w", ref.Handle, err)
	}
	return ObjectHeader{Handle: ref.Handle, Offset: ref.Offset, Size: size, Type: ObjectType(typ)}, nil
}

// WalkObjectHeaders decodes the preamble of every object the map references. An
// individual malformed object is skipped (its error collected) so one bad record
// does not abort the whole drawing; the headers and the slice of errors are both
// returned.
func WalkObjectHeaders(data []byte, refs []ObjectRef, version Version) ([]ObjectHeader, []error) {
	headers := make([]ObjectHeader, 0, len(refs))
	var errs []error
	for _, ref := range refs {
		h, err := decodeObjectHeader(data, ref, version)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		headers = append(headers, h)
	}
	return headers, errs
}
