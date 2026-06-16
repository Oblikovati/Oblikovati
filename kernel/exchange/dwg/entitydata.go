// SPDX-License-Identifier: GPL-2.0-only

package dwg

import "fmt"

// commonEntity holds the common-entity-data flags that decide which references the
// entity's handle stream carries (ODA common_entity_data.spec). They are captured
// during the data-stream pass so the handle stream can be walked in the right order
// (see handlestream.go) to reach the owner block (entmode 0) and an INSERT's
// block_header handle.
type commonEntity struct {
	entmode        int  // 0 = owned by a block (owner handle present), 1 = paper space, 2 = model space
	numReactors    int  // count of reactor handles in the handle stream
	xdicMissing    bool // R2004+: no xdicobjhandle in the stream when true (pre-R2004 always present)
	noLinks        bool // <=R2000: prev/next entity handles present when false
	colorByHandle  bool // colour given as a handle (ENC flag 0x40) → a leading colour handle
	ltypeFlags     int
	plotstyleFlags int
	materialFlags  int  // R2007+
	shadowHandle   bool // R2007+: shadow_flags == 3 → a shadow handle
	hasFullVS      bool // R2010+ visual-style handles
	hasFaceVS      bool
	hasEdgeVS      bool
}

// entityCursor is the result of seeking an entity: a reader positioned at its
// type-specific geometry (data stream), the common-entity flags, the entity's own
// handle (the reference for relative handle resolution), and the absolute bit offset
// where its handle stream begins.
type entityCursor struct {
	geom        *BitReader
	common      commonEntity
	ownHandle   uint64
	handleStart int
}

// seekEntity positions a reader at an entity's geometry, capturing the common-entity
// flags and the start of its handle stream. It supersedes the previous
// seekEntityGeometry, which discarded both (ADR: blocks/INSERT need the handle stream
// and entmode-based space filtering). Supports R2000 (flat) and R2010+ (paged).
func seekEntity(data []byte, ref ObjectRef, version Version) (*entityCursor, error) {
	r := NewBitReaderAt(data, int(ref.Offset)*8)
	size := r.ReadMS() // object size, in bytes, measured from the address below
	var hssize uint64
	if version >= R2010 {
		hssize = r.ReadUMC() // handle-stream size in bits, not counted in size
	}
	addrBit := r.Position() // obj.address: the reference for both size and bitsize
	if version >= R2010 {
		r.ReadBOT()
	} else {
		r.ReadBS()
	}
	// bitsize is the bit offset (from addrBit) where the data stream ends and the
	// handle stream begins. R2010+ derives it from the sizes; R2000–R2007 stores it.
	var bitsize int
	if version >= R2010 {
		bitsize = size*8 - int(hssize)
	} else if version >= R2000 {
		bitsize = int(r.ReadRL())
	}
	own := r.ReadHandle() // the object's own handle
	skipEED(r)
	ce := readCommonEntityData(r, version)
	if err := r.Err(); err != nil {
		return nil, fmt.Errorf("dwg: entity handle %d common data: %w", ref.Handle, err)
	}
	return &entityCursor{geom: r, common: ce, ownHandle: own.Value, handleStart: addrBit + bitsize}, nil
}

// skipEED consumes the extended entity data block: BS-sized records each carrying an
// app-id handle and that many raw bytes, terminated by a zero size.
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

// readCommonEntityData reads the data-stream portion of the common entity data so the
// cursor lands on the entity geometry, capturing the flags that drive handle-stream
// parsing. Field order and version gating follow common_entity_data.spec; handle and
// string-stream fields are absent here (they do not advance the data stream).
func readCommonEntityData(r *BitReader, version Version) commonEntity {
	var ce commonEntity
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
	ce.entmode = int(r.ReadBits(2))
	ce.numReactors = r.ReadBL()
	if version >= R2004 {
		ce.xdicMissing = r.ReadBit() == 1
	}
	if version <= R2000 { // nolinks: gated R13..R2002
		ce.noLinks = r.ReadBit() == 1
	}
	if version >= R2013 {
		r.ReadBit() // has_ds_data
	}
	ce.colorByHandle = readEntityColor(r, version)
	r.ReadBD() // ltype_scale
	if version >= R2000 {
		ce.ltypeFlags = int(r.ReadBits(2))
		ce.plotstyleFlags = int(r.ReadBits(2))
	}
	if version >= R2007 {
		ce.materialFlags = int(r.ReadBits(2))
		ce.shadowHandle = r.ReadRC() == 3
	}
	if version >= R2010 {
		ce.hasFullVS = r.ReadBit() == 1
		ce.hasFaceVS = r.ReadBit() == 1
		ce.hasEdgeVS = r.ReadBit() == 1
	}
	r.ReadBS() // invisible
	if version >= R2000 {
		r.ReadRC() // linewt
	}
	return ce
}

// readEntityColor consumes the entity colour and reports whether the colour is given
// as a handle (ENC flag 0x40), which adds a leading handle to the handle stream. A CMC
// BitShort index before R2004 carries no handle.
func readEntityColor(r *BitReader, version Version) (byHandle bool) {
	if version < R2004 {
		r.ReadBS() // CMC color index
		return false
	}
	raw := r.ReadBS()
	flags := raw >> 8
	if flags&0x20 != 0 {
		r.ReadBL() // alpha
	}
	// 0x40 (colour by handle) and 0x80 (explicit RGB) are mutually exclusive, 0x40
	// wins: when set, the colour is a handle in the handle stream and NO RGB long is
	// read here (#commit colour-ENC-desync fix).
	if flags&0x40 != 0 {
		return true
	}
	if flags&0x80 != 0 {
		r.ReadBL() // explicit RGB
	}
	return false
}
