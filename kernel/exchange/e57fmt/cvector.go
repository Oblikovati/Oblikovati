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

// decodeFieldValues reads one field's records out of each data packet of the points
// CompressedVector and returns one float64 per record (already scaled). It walks only the
// requested field's bytestream per packet, so unrelated channels (intensity, colour) are skipped.
func (d *Document) decodeFieldValues(fieldIndex int) ([]float64, error) {
	hdr, _, err := d.paged.readLogical(d.points.fileOffset, cvSectionHeaderSize)
	if err != nil {
		return nil, err
	}
	if hdr[0] != cvSectionType {
		return nil, fmt.Errorf("e57fmt: points section id is %d, want %d (CompressedVector)", hdr[0], cvSectionType)
	}
	pos := binary.LittleEndian.Uint64(hdr[16:]) // dataPhysicalOffset of the first data packet
	out := make([]float64, 0, d.points.recordCount)
	for uint64(len(out)) < d.points.recordCount {
		vals, next, err := d.readPacketField(pos, fieldIndex)
		if err != nil {
			return nil, err
		}
		remaining := d.points.recordCount - uint64(len(out))
		if uint64(len(vals)) > remaining {
			vals = vals[:remaining]
		}
		out = append(out, vals...)
		pos = next
	}
	return out, nil
}

// readPacketField reads the data packet at physical offset pos and decodes the bytestream of the
// requested field, returning its values and the offset just past the packet.
func (d *Document) readPacketField(pos uint64, fieldIndex int) ([]float64, uint64, error) {
	head, afterHead, err := d.paged.readLogical(pos, 6)
	if err != nil {
		return nil, 0, err
	}
	if head[0] != dataPacketType {
		return nil, 0, fmt.Errorf("e57fmt: expected a data packet (type %d) at %d, got type %d", dataPacketType, pos, head[0])
	}
	packetLen := uint64(binary.LittleEndian.Uint16(head[2:])) + 1
	bsCount := int(binary.LittleEndian.Uint16(head[4:]))
	body, next, err := d.paged.readLogical(afterHead, packetLen-6)
	if err != nil {
		return nil, 0, err
	}
	buf, err := bytestream(body, bsCount, fieldIndex)
	if err != nil {
		return nil, 0, err
	}
	vals := decodeBytestream(buf, d.points.fields[fieldIndex])
	return vals, next, nil
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
