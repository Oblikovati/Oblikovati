// SPDX-License-Identifier: GPL-2.0-only

package dwg

import "fmt"

// seekEntityGeometry returns a data-stream reader positioned at an entity's
// type-specific geometry, having consumed the object header and the common entity
// data (ODA §dwg_decode_entity + common_entity_data.spec). Only data-stream fields
// are read; handle-stream fields (owner/layer/linetype/reactors) live in a
// separate region keyed off the object's bitsize and never advance this cursor, so
// they are simply not read here.
//
// Supports the two target generations: R2000 (flat) and R2018 (paged, == R2010+).
func seekEntityGeometry(data []byte, ref ObjectRef, version Version) (*BitReader, error) {
	r := NewBitReaderAt(data, int(ref.Offset)*8)
	r.ReadMS() // object size
	if version >= R2010 {
		r.ReadUMC() // handle-stream size (R2010+), not part of size
		r.ReadBOT()
	} else {
		r.ReadBS()
	}
	if version >= R2000 && version <= R2007 {
		r.ReadRL() // bitsize (until the handle stream), computed from the header on R2010+
	}
	r.ReadHandle() // the object's own handle
	skipEED(r)
	skipCommonEntityData(r, version)
	if err := r.Err(); err != nil {
		return nil, fmt.Errorf("dwg: entity handle %d common data: %w", ref.Handle, err)
	}
	return r, nil
}

// skipEED consumes the extended entity data block: BS-sized records each carrying
// an app-id handle and that many raw bytes, terminated by a zero size.
func skipEED(r *BitReader) {
	for r.Err() == nil {
		size := r.ReadBS()
		if size == 0 {
			return
		}
		r.ReadHandle() // app id
		for i := 0; i < size; i++ {
			r.ReadRC()
		}
	}
}

// skipCommonEntityData reads the data-stream portion of the common entity data so
// the cursor lands on the entity geometry. Field order and version gating follow
// common_entity_data.spec; handle and string-stream fields are intentionally
// absent (they do not advance the data stream).
func skipCommonEntityData(r *BitReader, version Version) {
	if r.ReadBit() == 1 { // preview_exists
		var sz uint64
		if version >= R2010 {
			sz = r.ReadBLL()
		} else {
			sz = uint64(r.ReadRL())
		}
		for i := uint64(0); i < sz; i++ {
			r.ReadRC()
		}
	}
	r.ReadBits(2) // entmode (BB)
	r.ReadBL()    // num_reactors
	if version >= R2004 {
		r.ReadBit() // is_xdic_missing
	}
	if version <= R2000 { // nolinks: gated R13..R2002
		r.ReadBit()
	}
	if version >= R2013 {
		r.ReadBit() // has_ds_data
	}
	skipEntityColor(r, version)
	r.ReadBD() // ltype_scale
	if version >= R2000 {
		r.ReadBits(2) // ltype_flags
		r.ReadBits(2) // plotstyle_flags
	}
	if version >= R2007 {
		r.ReadBits(2) // material_flags
		r.ReadRC()    // shadow_flags
	}
	if version >= R2010 {
		r.ReadBit() // has_full_visualstyle
		r.ReadBit() // has_face_visualstyle
		r.ReadBit() // has_edge_visualstyle
	}
	r.ReadBS() // invisible
	if version >= R2000 {
		r.ReadRC() // linewt
	}
}

// skipEntityColor consumes the entity colour: a CMC BitShort index before R2004,
// or the R2004+ ENC encoding (a raw BitShort whose high byte flags an optional
// BitLong alpha and an optional BitLong RGB; the colour handle and book names live
// in the handle/string streams).
func skipEntityColor(r *BitReader, version Version) {
	if version < R2004 {
		r.ReadBS() // CMC color index
		return
	}
	raw := r.ReadBS()
	flags := raw >> 8
	if flags&0x20 != 0 {
		r.ReadBL() // alpha
	}
	// 0x40 (colour by handle) and 0x80 (explicit RGB) are mutually exclusive, and
	// 0x40 takes precedence: when set, the colour is a handle in the handle stream
	// and NO RGB long is read from the data stream. Reading it unconditionally on
	// 0x80 (flag 0xC0 = both) desynced these entities.
	if flags&0x40 != 0 {
		return // colour handle lives in the handle stream
	}
	if flags&0x80 != 0 {
		r.ReadBL() // explicit RGB
	}
}
