// SPDX-License-Identifier: GPL-2.0-only

package dwg

import (
	"encoding/binary"
	"fmt"
)

// sectionDescriptor is one logical section's entry in the R2004 section map: its
// name (e.g. "AcDb:Handles"), total decompressed size, whether its pages are
// LZ77-compressed, and the ordered list of data pages that make it up.
type sectionDescriptor struct {
	name       string
	size       int64
	maxDecomp  int // per-page decompressed window (normally 0x7400)
	compressed bool
	pages      []sectionPageRef
}

// defaultMaxDecomp is the usual per-page decompressed window when a descriptor's
// field is missing/implausible (ODA §4.5).
const defaultMaxDecomp = 0x7400

// sectionPageRef points at one page of a logical section: which physical page
// (index into the page map), its compressed size, and where it lands in the
// assembled (decompressed) section.
type sectionPageRef struct {
	number      int32
	dataSize    int64
	startOffset int64
}

// sectionInfoHeaderLen is the fixed Section Info header (NumDescriptions + four
// constant longs) that precedes the descriptors (ODA §4.5).
const sectionInfoHeaderLen = 20

// descriptorFixedLen is the fixed part of each descriptor before its page list:
// size(8)+pageCount(4)+maxDecomp(4)+unknown(4)+compressed(4)+id(4)+encrypted(4)
// +name(64) (ODA §4.5).
const descriptorFixedLen = 0x60

// parseSectionMap decodes the decompressed Section Info into descriptors. raw is
// the assembled section-map bytes (see [readSystemSection]).
func parseSectionMap(raw []byte) ([]sectionDescriptor, error) {
	if len(raw) < sectionInfoHeaderLen {
		return nil, fmt.Errorf("dwg: section map too small: %d bytes", len(raw))
	}
	count := int(binary.LittleEndian.Uint32(raw))
	if count < 0 || count > 1024 {
		return nil, fmt.Errorf("dwg: implausible section count %d", count)
	}
	descs := make([]sectionDescriptor, 0, count)
	off := sectionInfoHeaderLen
	for i := range count {
		d, next, err := readDescriptor(raw, off)
		if err != nil {
			return nil, fmt.Errorf("dwg: section descriptor %d: %w", i, err)
		}
		descs = append(descs, d)
		off = next
	}
	return descs, nil
}

// readDescriptor parses one descriptor (fixed fields + page list) at off and
// returns it plus the offset of the next descriptor.
func readDescriptor(raw []byte, off int) (sectionDescriptor, int, error) {
	if off+descriptorFixedLen > len(raw) {
		return sectionDescriptor{}, 0, fmt.Errorf("descriptor header at %d overruns map (len %d)", off, len(raw))
	}
	pageCount := int(binary.LittleEndian.Uint32(raw[off+8:]))
	if pageCount < 0 || off+descriptorFixedLen+pageCount*16 > len(raw) {
		return sectionDescriptor{}, 0, fmt.Errorf("descriptor page count %d overruns map", pageCount)
	}
	maxDecomp := int(binary.LittleEndian.Uint32(raw[off+0x0C:]))
	if maxDecomp <= 0 {
		maxDecomp = defaultMaxDecomp
	}
	d := sectionDescriptor{
		size:       int64(binary.LittleEndian.Uint64(raw[off:])),
		maxDecomp:  maxDecomp,
		compressed: binary.LittleEndian.Uint32(raw[off+0x14:]) == 2,
		name:       cStringFromBytes(raw[off+0x20 : off+0x60]),
		pages:      make([]sectionPageRef, pageCount),
	}
	p := off + descriptorFixedLen
	for j := range pageCount {
		base := p + j*16
		d.pages[j] = sectionPageRef{
			number:      int32(binary.LittleEndian.Uint32(raw[base:])),
			dataSize:    int64(binary.LittleEndian.Uint32(raw[base+4:])),
			startOffset: int64(binary.LittleEndian.Uint64(raw[base+8:])),
		}
	}
	return d, p + pageCount*16, nil
}

// dataPageHeaderLen is the encrypted data-page header size (ODA §4.6).
const dataPageHeaderLen = 32

// dataPageHeader is the decrypted header of a data section page (ODA §4.6).
type dataPageHeader struct {
	pageType   uint32
	compSize   int
	decompSize int
}

// decryptDataPageHeader reads and XOR-decrypts the 32-byte data-page header at
// offset. The key is keyed on the page's own file offset (secMask = 0x4164536b ^
// offset) applied across the eight header longs (ODA §4.6).
func decryptDataPageHeader(data []byte, offset int64) (dataPageHeader, error) {
	if offset < 0 || offset+dataPageHeaderLen > int64(len(data)) {
		return dataPageHeader{}, fmt.Errorf("dwg: data page header at %#x out of bounds", offset)
	}
	mask := uint32(0x4164536b) ^ uint32(offset)
	var h [8]uint32
	for i := range h {
		h[i] = binary.LittleEndian.Uint32(data[offset+int64(i*4):]) ^ mask
	}
	return dataPageHeader{pageType: h[0], compSize: int(h[2]), decompSize: int(h[3])}, nil
}

// assembleSection reconstructs a logical section's decompressed bytes by placing
// each of its data pages at its start offset and expanding any unwritten gaps as
// zero pages (ODA §4.5: zero-only pages are omitted from the file). The result is
// exactly desc.size bytes.
func assembleSection(data []byte, pageMap map[int32]pageEntry, desc sectionDescriptor) ([]byte, error) {
	out := make([]byte, desc.size)
	// One reusable decompression window for the whole section. Each page is expanded into it
	// and copied straight into out before the next page overwrites it, so a single allocation
	// replaces one maxDecomp-byte buffer per page (#1549: the dominant decode allocation).
	var scratch []byte
	if desc.compressed {
		scratch = make([]byte, desc.maxDecomp)
	}
	for _, ref := range desc.pages {
		page, ok := pageMap[ref.number]
		if !ok {
			return nil, fmt.Errorf("dwg: section %q references missing page %d", desc.name, ref.number)
		}
		body, err := readDataPage(data, page.offset, desc.compressed, scratch)
		if err != nil {
			return nil, fmt.Errorf("dwg: section %q page %d: %w", desc.name, ref.number, err)
		}
		if ref.startOffset < 0 || ref.startOffset > int64(len(out)) {
			return nil, fmt.Errorf("dwg: section %q page %d start offset %d outside section size %d", desc.name, ref.number, ref.startOffset, desc.size)
		}
		// A page's decompression window (maxDecomp) overshoots its logical bytes
		// into discardable tail padding, so copy only what fits before the next
		// page's start / the section end.
		copy(out[ref.startOffset:], body)
	}
	return out, nil
}

// readDataPage decrypts a data page's header at offset and returns its body. When the section
// is compressed the page is expanded into scratch (whose length is the maxDecomp window) and a
// scratch sub-slice is returned — the caller must copy it out before the next call reuses
// scratch. The returned window may include tail padding that assembleSection clips. An
// uncompressed page returns a sub-slice of data directly (scratch is unused).
func readDataPage(data []byte, offset int64, compressed bool, scratch []byte) ([]byte, error) {
	hdr, err := decryptDataPageHeader(data, offset)
	if err != nil {
		return nil, err
	}
	if hdr.pageType != dataPageMagic {
		return nil, fmt.Errorf("data page type %#x, want %#x", hdr.pageType, dataPageMagic)
	}
	bodyStart := offset + dataPageHeaderLen
	if bodyStart+int64(hdr.compSize) > int64(len(data)) {
		return nil, fmt.Errorf("data page body size %d overruns file", hdr.compSize)
	}
	body := data[bodyStart : bodyStart+int64(hdr.compSize)]
	if compressed {
		n, derr := decompressR2004Into(scratch, body)
		if derr != nil {
			return nil, derr
		}
		return scratch[:n], nil
	}
	return body, nil
}

// cStringFromBytes trims a fixed-width field at its first NUL.
func cStringFromBytes(b []byte) string {
	for i, c := range b {
		if c == 0 {
			return string(b[:i])
		}
	}
	return string(b)
}
