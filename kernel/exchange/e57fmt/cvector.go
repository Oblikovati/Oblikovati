// SPDX-License-Identifier: GPL-2.0-only

package e57fmt

import (
	"encoding/binary"
	"fmt"
	"math"
)

// cvSectionHeaderSize is the CompressedVector binary-section header: a 1-byte section id, 7
// reserved bytes, then three uint64 (section length and the data/index physical offsets).
const cvSectionHeaderSize = 32

// cvSectionType is the section-id byte marking a CompressedVector binary section.
const cvSectionType = 1

// dataPacketType is the packet-type byte of a data packet (vs. an index/empty packet).
const dataPacketType = 1

// maxBitPackedBits caps the per-value width the bit reader fills without overflowing its 64-bit
// accumulator; every E57 cartesian scan field fits well under it (32 bits).
const maxBitPackedBits = 56

// decodeFields reads the requested prototype fields out of the points CompressedVector in a single
// pass over its data packets, returning one already-scaled float64 column per field keyed by field
// index (each column has recordCount values). Decoding every wanted channel (XYZ plus colour and
// intensity, #645) in one walk avoids re-reading the whole section once per field, which matters on
// multi-hundred-MB scans. A constant field (an Integer/ScaledInteger whose declared min==max carries
// no packed bytes) is materialised as recordCount copies of its constant value rather than decoded.
func (d *Document) decodeFields(indices []int) (map[int][]float64, error) {
	out, variable := d.partitionConstantFields(indices)
	if len(variable) == 0 {
		return out, nil
	}
	pos, err := d.firstDataPacketOffset()
	if err != nil {
		return nil, err
	}
	ref := variable[0]
	for uint64(len(out[ref])) < d.points.recordCount {
		vals, next, err := d.readPacketFields(pos, variable)
		if err != nil {
			return nil, err
		}
		remaining := d.points.recordCount - uint64(len(out[ref]))
		appendPacketColumns(out, vals, variable, remaining)
		pos = next
	}
	return out, nil
}

// partitionConstantFields splits indices into columns already materialised (constant fields carry no
// packed bytes, so they are filled with their single value up front) and the still-to-decode variable
// field indices, seeding each variable column with recordCount capacity.
func (d *Document) partitionConstantFields(indices []int) (map[int][]float64, []int) {
	out := make(map[int][]float64, len(indices))
	variable := make([]int, 0, len(indices))
	for _, i := range indices {
		if isConstantField(d.points.fields[i]) {
			out[i] = constantColumn(d.points.fields[i], d.points.recordCount)
			continue
		}
		variable = append(variable, i)
		out[i] = make([]float64, 0, d.points.recordCount)
	}
	return out, variable
}

// firstDataPacketOffset reads and validates the CompressedVector section header and returns the
// physical offset of its first data packet.
func (d *Document) firstDataPacketOffset() (uint64, error) {
	hdr, _, err := d.paged.readLogical(d.points.fileOffset, cvSectionHeaderSize)
	if err != nil {
		return 0, err
	}
	if hdr[0] != cvSectionType {
		return 0, fmt.Errorf("e57fmt: points section id is %d, want %d (CompressedVector)", hdr[0], cvSectionType)
	}
	return binary.LittleEndian.Uint64(hdr[16:]), nil // dataPhysicalOffset of the first data packet
}

// appendPacketColumns appends each variable field's freshly-decoded packet values onto its output
// column, capping the packet at remaining so the final packet never overshoots recordCount.
func appendPacketColumns(out, vals map[int][]float64, variable []int, remaining uint64) {
	for _, i := range variable {
		v := vals[i]
		if uint64(len(v)) > remaining {
			v = v[:remaining]
		}
		out[i] = append(out[i], v...)
	}
}

// readPacketFields reads the data packet at physical offset pos and decodes the bytestreams of the
// requested fields, returning their values and the offset just past the packet.
func (d *Document) readPacketFields(pos uint64, indices []int) (map[int][]float64, uint64, error) {
	body, bsCount, next, err := d.readDataPacketBody(pos)
	if err != nil {
		return nil, 0, err
	}
	vals, err := d.decodePacketColumns(body, bsCount, indices)
	if err != nil {
		return nil, 0, err
	}
	return vals, next, nil
}

// readDataPacketBody reads and validates the data packet at pos, returning its bytestream body, the
// bytestream count, and the physical offset just past the packet.
func (d *Document) readDataPacketBody(pos uint64) (body []byte, bsCount int, next uint64, err error) {
	head, afterHead, err := d.paged.readLogical(pos, 6)
	if err != nil {
		return nil, 0, 0, err
	}
	if head[0] != dataPacketType {
		return nil, 0, 0, fmt.Errorf("e57fmt: expected a data packet (type %d) at %d, got type %d", dataPacketType, pos, head[0])
	}
	packetLen := uint64(binary.LittleEndian.Uint16(head[2:])) + 1
	bsCount = int(binary.LittleEndian.Uint16(head[4:]))
	body, next, err = d.paged.readLogical(afterHead, packetLen-6)
	if err != nil {
		return nil, 0, 0, err
	}
	return body, bsCount, next, nil
}

// decodePacketColumns decodes each requested field's bytestream from the packet body. Each field in a
// packet encodes the same record count, but a bit-packed field's trailing byte padding can yield one
// extra value, so every column is truncated to the shortest to stay row-aligned.
func (d *Document) decodePacketColumns(body []byte, bsCount int, indices []int) (map[int][]float64, error) {
	vals := make(map[int][]float64, len(indices))
	rows := -1
	for _, fi := range indices {
		buf, err := bytestream(body, bsCount, fi)
		if err != nil {
			return nil, err
		}
		v := decodeBytestream(buf, d.points.fields[fi])
		vals[fi] = v
		if rows < 0 || len(v) < rows {
			rows = len(v)
		}
	}
	for fi := range vals {
		vals[fi] = vals[fi][:rows]
	}
	return vals, nil
}

// bytestream slices the requested field's buffer out of a data packet body, which begins with one
// uint16 length per bytestream followed by the buffers back to back, in field order.
func bytestream(body []byte, bsCount, fieldIndex int) ([]byte, error) {
	if fieldIndex >= bsCount {
		return nil, fmt.Errorf("e57fmt: field index %d out of range for %d bytestreams", fieldIndex, bsCount)
	}
	if len(body) < 2*bsCount {
		return nil, fmt.Errorf("e57fmt: packet body %d bytes too small for %d bytestream lengths", len(body), bsCount)
	}
	off := 2 * bsCount
	var start, length int
	for i := 0; i < bsCount; i++ {
		l := int(binary.LittleEndian.Uint16(body[2*i:]))
		if i == fieldIndex {
			start, length = off, l
		}
		off += l
	}
	if start+length > len(body) {
		return nil, fmt.Errorf("e57fmt: bytestream %d [%d:%d] exceeds packet body %d", fieldIndex, start, start+length, len(body))
	}
	return body[start : start+length], nil
}

// isConstantField reports whether a bit-packed field carries no bytes because its declared range is
// empty (min==max). Such a field is stored implicitly as its single value; a Float field is always
// materialised in the stream, so it is never constant here.
func isConstantField(f protoField) bool {
	return f.kind != kindFloat && f.max <= f.min
}

// constantColumn materialises a constant (empty-range) field as n copies of its value, applying the
// same value = min*scale + offset rule the bit-packed decoder uses.
func constantColumn(f protoField, n uint64) []float64 {
	v := float64(f.min)*f.scale + f.offset
	out := make([]float64, n)
	for i := range out {
		out[i] = v
	}
	return out
}

// decodeBytestream turns one field's packet buffer into scaled float64 values per its kind.
func decodeBytestream(buf []byte, f protoField) []float64 {
	if f.kind == kindFloat {
		return decodeFloats(buf, f.doublePrec)
	}
	return decodeScaledInts(buf, f)
}

// decodeFloats reads consecutive byte-aligned IEEE-754 values (single or double precision).
func decodeFloats(buf []byte, double bool) []float64 {
	if double {
		out := make([]float64, len(buf)/8)
		for i := range out {
			out[i] = math.Float64frombits(binary.LittleEndian.Uint64(buf[i*8:]))
		}
		return out
	}
	out := make([]float64, len(buf)/4)
	for i := range out {
		out[i] = float64(math.Float32frombits(binary.LittleEndian.Uint32(buf[i*4:])))
	}
	return out
}

// decodeScaledInts bit-unpacks the field's integers (LSB-first) and maps each through the E57
// rule value = (raw + minimum)*scale + offset (scale=1/offset=0 for a plain Integer).
func decodeScaledInts(buf []byte, f protoField) []float64 {
	bits := bitsToEncode(uint64(f.max - f.min))
	if bits == 0 || bits > maxBitPackedBits {
		return nil // a constant field (range 0) carries no bytes; oversized widths are unsupported
	}
	count := (len(buf) * 8) / int(bits)
	out := make([]float64, 0, count)
	r := bitReader{buf: buf}
	for i := 0; i < count; i++ {
		raw := int64(r.read(bits)) + f.min
		out = append(out, float64(raw)*f.scale+f.offset)
	}
	return out
}

// bitsToEncode is the number of bits needed to represent every value in [0, span].
func bitsToEncode(span uint64) uint {
	bits := uint(0)
	for bits < 64 && (uint64(1)<<bits) <= span {
		bits++
	}
	return bits
}
