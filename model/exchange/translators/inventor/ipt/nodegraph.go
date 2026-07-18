// SPDX-License-Identifier: GPL-2.0-only

package ipt

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"io"

	"github.com/klauspost/compress/zstd"
)

// Node-graph decoding of an RSeStorage segment. A segment's decompressed B-stream is a
// sequence of typed blocks; the block sizes and the block-type table live in the paired
// M-stream (itself carrying an inner-compressed metadata region). This ports the framing
// from InventorLoader (importerReader.py / importerSegment.py): each block is
// [u32 header (low byte → type table index)] [payload] [u32 length] [trailer].
//
// It exists to reach data the targeted byte-pattern decoders can't — chiefly an assembly's
// occurrence placement transforms, which sit in typed nodes (AmRxSegment 232792BC).

// segMeta is a segment's block layout parsed from its M-stream.
type segMeta struct {
	sizes []int    // payload length of each block
	flags []bool   // whether each block is present in the B-stream data
	types []uint32 // block-type table: index → 4-byte type id (UUID time_low)
}

// parseSegmentMeta parses the M-stream metadata: the block-size table and the block-type
// table, skipping the intervening sections. Returns false if the stream is malformed.
func parseSegmentMeta(mRaw []byte) (*segMeta, bool) {
	c := newCursor(mRaw)
	c.text8() // "RSe Meta Stream Version 8"
	ver := c.u16()
	c.skip(16) // arr1 u16[8]
	c.text16() // segment name
	c.skip(16) // segID UUID
	c.skip(12) // arr2 u32[3]
	c.text8()  // dat1 (ver>=7)
	c.text8()  // dat2
	c.u8()     // bTrue
	if c.bad {
		return nil, false
	}
	inner := inflateStream(mRaw[c.i:])
	if inner == nil {
		return nil, false
	}
	ic := newCursor(inner)
	ic.skip(14) // arr3 u16[7]
	// BlocksSize: cnt × u32 (high bit = flags, low 31 = length)
	n := int(ic.u32())
	m := &segMeta{}
	for k := 0; k < n; k++ {
		v := ic.u32()
		m.sizes = append(m.sizes, int(v&0x7FFFFFFF))
		m.flags = append(m.flags, v>>31 != 0)
	}
	ic.u32() // section byte size
	// Section2 (entry size depends on the M-stream version) then Section3 (28 bytes each).
	n2 := int(ic.u32())
	ic.skip(section2EntrySize(ver) * n2)
	ic.u32()
	n3 := int(ic.u32())
	ic.skip(28 * n3)
	ic.u32()
	// BlocksType: cnt × (UUID(16) + 12); the type id is the UUID's leading u32.
	nT := int(ic.u32())
	for k := 0; k < nT; k++ {
		if !ic.has(28) {
			return nil, false
		}
		m.types = append(m.types, binary.LittleEndian.Uint32(inner[ic.i:]))
		ic.skip(28)
	}
	if ic.bad {
		return nil, false
	}
	return m, true
}

// section2EntrySize is the byte width of one RSeMetaData Section2 record for the given
// M-stream version (v3: UUID+u32+u16+u16[5]; v4: u32+u32+u16+u16[5]; else: u32+u32+u16).
func section2EntrySize(ver uint16) int {
	switch ver {
	case 3:
		return 32
	case 4:
		return 20
	default:
		return 10
	}
}

// walkNodes iterates the segment's present blocks in order, calling fn(typeID, payload).
// Stops early if fn returns false or the framing desyncs. version selects the trailer form
// (>2014 carries a per-block property trailer).
func walkNodes(payload []byte, meta *segMeta, version int, fn func(typ uint32, payload []byte) bool) {
	i := 0
	for k := 0; k < len(meta.sizes); k++ {
		if !meta.flags[k] {
			continue
		}
		if i+4 > len(payload) {
			return
		}
		idx := int(payload[i] & 0xFF)
		var typ uint32
		if idx < len(meta.types) {
			typ = meta.types[idx]
		}
		start := i + 4
		end := start + meta.sizes[k]
		if end+4 > len(payload) {
			return
		}
		if !fn(typ, payload[start:end]) {
			return
		}
		i = end
		if l := int(binary.LittleEndian.Uint32(payload[i:])); l != meta.sizes[k] {
			return // framing desync — the trailing length must echo the block size
		}
		i += 4
		next, ok := readTrailer(payload, i, version)
		if !ok {
			return
		}
		i = next
	}
}

// readTrailer skips a block's trailing metadata (a property list, versions > 2014). Returns
// the offset past it, or ok=false if it can't be parsed. Port of ReadTrailer.
func readTrailer(data []byte, off, version int) (int, bool) {
	c := &cursor{b: data, i: off}
	if version <= 2014 {
		return c.i, true
	}
	if c.u8() == 0 { // no trailing properties (the common case)
		return c.i, !c.bad
	}
	n := c.u32()
	if n&0x80000000 != 0 {
		return c.i, !c.bad
	}
	for j := 0; j < int(n); j++ {
		c.text8() // property key
		switch c.u32() {
		case 0b0001:
			c.skip(3)
		case 0b0011, 0b0111:
			c.skip(4)
		case 0b1000, 0b1010:
			c.skip(6)
		case 0b1011:
			c.skip(10)
		case 0b1110:
			c.u16()
			c.skip(int(c.u32()))
		default:
			return c.i, false
		}
	}
	c.skip(4) // CheckList: u16[2] list marker
	if cnt := int(c.u32()); cnt > 0 {
		c.skip(8) // arr32 u32[2]
		for j := 0; j < cnt; j++ {
			c.text8()
			c.u32()
		}
	}
	return c.i, !c.bad
}

// Matrix4 is a row-major 4×4 transform; the zero value is not identity — decodeTransform3D
// seeds identity before overwriting the stored cells.
type Matrix4 [16]float64

// identityMatrix4 is the 4×4 identity.
func identityMatrix4() Matrix4 {
	return Matrix4{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1}
}

// decodeTransform3D reads Inventor's sparse row-major 4×4 transform at off: an optional
// 0x00000203 marker, two u16 bitmasks (d1 = value bits, d2 = mask bits), then an explicit
// float64 for each cell that is neither masked (d2) nor a unit (d1). Identity cells store
// no float. Port of InventorLoader Transformation3D.read. Translation is cells 3/7/11.
func decodeTransform3D(data []byte, off int) (Matrix4, bool) {
	m := identityMatrix4()
	c := &cursor{b: data, i: off}
	if c.has(4) && binary.LittleEndian.Uint32(data[c.i:]) == 0x00000203 {
		c.skip(4)
	}
	d1 := c.u16()
	d2 := c.u16()
	for j := 0; j < 16; j++ {
		b := uint16(1) << uint(j)
		switch {
		case d2&b != 0:
			if d1&b == 0 {
				m[j] = 0
			} else {
				m[j] = -1
			}
		case d1&b != 0:
			m[j] = 1
		default:
			v := c.f64()
			if absf(v) < 1e-6 {
				v = 0
			}
			m[j] = v
		}
	}
	return m, !c.bad
}

// inflateStream inflates a Zstd (Inventor 2027+) or zlib (older) compressed region.
func inflateStream(b []byte) []byte {
	if len(b) >= 4 && [4]byte{b[0], b[1], b[2], b[3]} == zstdMagic {
		dec, err := zstd.NewReader(nil)
		if err != nil {
			return nil
		}
		defer dec.Close()
		out, err := dec.DecodeAll(b, nil)
		if err != nil {
			return nil
		}
		return out
	}
	zr, err := zlib.NewReader(bytes.NewReader(b))
	if err != nil {
		return nil
	}
	out, err := io.ReadAll(zr)
	if err != nil {
		return nil
	}
	return out
}
