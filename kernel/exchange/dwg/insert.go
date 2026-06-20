// SPDX-License-Identifier: GPL-2.0-only

package dwg

import "fmt"

// decodeInsert reads an INSERT entity: its placement (insertion point, scale, rotation)
// from the data stream and the referenced block's handle from the handle stream
// (ODA INSERT spec, R2000+). Pre-2004 attribute handles are not needed and left unread.
func decodeInsert(hr *BitReader, data []byte, cur *entityCursor, handle uint64, version Version) (in *Insert, owner uint64, err error) {
	r := cur.geom
	ins := r.Read3BD()
	scale := readInsertScale(r)
	rot := r.ReadBD()
	r.Read3BD() // extrusion (3DPOINT, unused; sketch import is planar)
	hasAttribs := r.ReadBit() == 1
	if version >= R2004 && hasAttribs {
		r.ReadBL() // num_owned
	}
	if e := r.Err(); e != nil {
		return nil, 0, fmt.Errorf("dwg: INSERT handle %d data: %w", handle, e)
	}
	owner, _ = commonEntityHandles(hr, data, cur, version)
	block := readResolvedHandle(hr, cur.ownHandle) // block_header is the first entity-specific handle
	if e := hr.Err(); e != nil {
		return nil, 0, fmt.Errorf("dwg: INSERT handle %d block ref: %w", handle, e)
	}
	return &Insert{Handle: handle, BlockHeader: block, Insertion: ins, Scale: scale, Rotation: rot}, owner, nil
}

// readInsertScale decodes the data-stream scale per the 2-bit scale_flag (R2000+):
// 3 = unit; 1 = x is 1 with y,z as deltas; 2 = uniform x; 0 = x then y,z as deltas.
func readInsertScale(r *BitReader) [3]float64 {
	switch r.ReadBits(2) {
	case 3:
		return [3]float64{1, 1, 1}
	case 1:
		return [3]float64{1, r.ReadDD(1), r.ReadDD(1)}
	case 2:
		x := r.ReadRD()
		return [3]float64{x, x, x}
	default: // 0
		x := r.ReadRD()
		return [3]float64{x, r.ReadDD(x), r.ReadDD(x)}
	}
}
