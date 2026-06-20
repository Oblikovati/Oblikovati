// SPDX-License-Identifier: GPL-2.0-only

package e57fmt

import (
	"encoding/binary"
	"fmt"
)

// headerSize is the fixed E57 file header: an 8-byte signature followed by seven little-endian
// fields (two uint32 versions, then five uint64 offsets/lengths) — 48 bytes total (ASTM E2807).
const headerSize = 48

// signature is the magic the file must start with.
var signature = []byte("ASTM-E57")

// fileHeader is the parsed E57 file header. The XML descriptor lives at xmlPhysicalOffset for
// xmlLogicalLength logical bytes; pageSize is the checksummed-page size (the last 4 bytes of each
// page are a CRC, so only pageSize-4 bytes per page carry payload).
type fileHeader struct {
	versionMajor      uint32
	versionMinor      uint32
	filePhysicalLen   uint64
	xmlPhysicalOffset uint64
	xmlLogicalLength  uint64
	pageSize          uint64
}

// parseHeader reads and validates the 48-byte header, erroring on a non-E57 input or an
// implausible page size (the de-pager divides by pageSize-4, so it must exceed 4).
func parseHeader(data []byte) (fileHeader, error) {
	if len(data) < headerSize {
		return fileHeader{}, fmt.Errorf("e57fmt: file is %d bytes, shorter than the %d-byte header", len(data), headerSize)
	}
	if string(data[:8]) != string(signature) {
		return fileHeader{}, fmt.Errorf("e57fmt: not an E57 file (signature %q, want %q)", data[:8], signature)
	}
	h := fileHeader{
		versionMajor:      binary.LittleEndian.Uint32(data[8:]),
		versionMinor:      binary.LittleEndian.Uint32(data[12:]),
		filePhysicalLen:   binary.LittleEndian.Uint64(data[16:]),
		xmlPhysicalOffset: binary.LittleEndian.Uint64(data[24:]),
		xmlLogicalLength:  binary.LittleEndian.Uint64(data[32:]),
		pageSize:          binary.LittleEndian.Uint64(data[40:]),
	}
	if h.pageSize <= 4 {
		return fileHeader{}, fmt.Errorf("e57fmt: implausible page size %d (must exceed the 4-byte page CRC)", h.pageSize)
	}
	return h, nil
}
