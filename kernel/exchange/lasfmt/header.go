// SPDX-License-Identifier: GPL-2.0-only

package lasfmt

import (
	"encoding/binary"
	"fmt"
	"math"
)

// minHeaderSize is the smallest public header block (LAS 1.0–1.2 is 227 bytes); we only read up to
// the scale/offset doubles (which end at byte 227), so any conformant header is large enough.
const minHeaderSize = 227

// las14PointCountOffset is where LAS 1.4 keeps the authoritative 64-bit point count (the legacy
// 32-bit count at legacyPointCountOffset stays 0 for the big or extended-format files).
const las14PointCountOffset = 247

// legacyPointCountOffset is the LAS 1.0–1.3 (and 1.4 legacy) 32-bit point count.
const legacyPointCountOffset = 107

// signature is the 4-byte magic every LAS file starts with.
var signature = []byte("LASF")

// lasHeader is the subset of the public header block a point-cloud import needs: where the point
// records start, how many there are and how long each is, and the scale/offset that turn the
// stored integer XYZ into real coordinates (ASPRS LAS R15, §2.2).
type lasHeader struct {
	versionMajor    uint8
	versionMinor    uint8
	headerSize      uint16
	pointDataOffset uint32
	vlrCount        uint32
	pointFormat     uint8
	recordLength    uint16
	pointCount      uint64
	scale           [3]float64
	offset          [3]float64
}

// parseHeader reads and validates the public header block, erroring on a non-LAS input, a too-short
// header, or an out-of-range point data record format (0–10 are defined).
func parseHeader(data []byte) (lasHeader, error) {
	if len(data) < minHeaderSize {
		return lasHeader{}, fmt.Errorf("lasfmt: file is %d bytes, shorter than the %d-byte header", len(data), minHeaderSize)
	}
	if string(data[:4]) != string(signature) {
		return lasHeader{}, fmt.Errorf("lasfmt: not a LAS file (signature %q, want %q)", data[:4], signature)
	}
	h := lasHeader{
		versionMajor:    data[24],
		versionMinor:    data[25],
		headerSize:      binary.LittleEndian.Uint16(data[94:]),
		pointDataOffset: binary.LittleEndian.Uint32(data[96:]),
		vlrCount:        binary.LittleEndian.Uint32(data[100:]),
		pointFormat:     data[104] & 0x3f, // mask the compression bits a LAZ writer may set
		recordLength:    binary.LittleEndian.Uint16(data[105:]),
	}
	if h.pointFormat > 10 {
		return lasHeader{}, fmt.Errorf("lasfmt: point data record format %d is out of range (0–10)", h.pointFormat)
	}
	h.pointCount = pointCount(data)
	h.scale = readVec3(data, 131)
	h.offset = readVec3(data, 155)
	return h, nil
}

// pointCount returns the authoritative record count: LAS 1.4's 64-bit field when it is present and
// the legacy 32-bit field is zero, otherwise the legacy field.
func pointCount(data []byte) uint64 {
	legacy := uint64(binary.LittleEndian.Uint32(data[legacyPointCountOffset:]))
	is14 := data[24] > 1 || (data[24] == 1 && data[25] >= 4)
	if is14 && legacy == 0 && len(data) >= las14PointCountOffset+8 {
		return binary.LittleEndian.Uint64(data[las14PointCountOffset:])
	}
	return legacy
}

// readVec3 reads three consecutive little-endian float64 starting at off.
func readVec3(data []byte, off int) [3]float64 {
	var v [3]float64
	for i := 0; i < 3; i++ {
		v[i] = math.Float64frombits(binary.LittleEndian.Uint64(data[off+i*8:]))
	}
	return v
}
