// SPDX-License-Identifier: GPL-2.0-only

package dwg

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// r2004FileIDString marks the start of the decrypted R2004+ file header; finding
// it confirms the LCG decryption ran correctly.
var r2004FileIDString = []byte("AcFssFcAJMB\x00")

// r2004BaseOffset is added to in-file addresses stored in the R2004 header and
// page map: the first 0x100 bytes are the meta-data/header region, and page
// addresses are measured from the end of it.
const r2004BaseOffset = 0x100

// pageMapMagic / sectionMapMagic are the system-page type tags (ODA §4.3/§4.4).
const (
	pageMapMagic    = 0x41630e3b
	sectionMapMagic = 0x4163003b
	dataPageMagic   = 0x4163043b
)

// r2004Header is the decrypted top-of-file header for the paged container. It
// locates the two system maps (page map and section map) that, between them, say
// where every logical section's data lives.
type r2004Header struct {
	pageMapAddress int64 // file offset of the page-map page (already + base)
	pageMapID      int
	sectionMapID   int
}

// decryptR2004Header recovers the 0x6C-byte header stored "encrypted" at 0x80. The
// cipher is a XOR against the high byte of the classic LCG (rseed = rseed*0x343fd
// + 0x269ec3); the AcFssFcAJMB marker validates the result.
func decryptR2004Header(data []byte) ([]byte, error) {
	const start, size = 0x80, 0x6c
	if len(data) < start+size {
		return nil, fmt.Errorf("dwg: R2004 header truncated: have %d bytes, need >= %d", len(data), start+size)
	}
	dec := make([]byte, size)
	var seed uint32 = 1
	for i := 0; i < size; i++ {
		seed = seed*0x343fd + 0x269ec3
		dec[i] = data[start+i] ^ byte(seed>>0x10)
	}
	if !bytes.HasPrefix(dec, r2004FileIDString) {
		return nil, fmt.Errorf("dwg: R2004 header decryption failed: marker = %q, want %q", dec[:12], r2004FileIDString)
	}
	return dec, nil
}

// parseR2004HeaderFields reads the system-map locators out of the decrypted
// header. Offsets are from the ODA layout and were confirmed against the corpus
// (page map id at 0x50, its 64-bit address at 0x54, section map id at 0x5c).
func parseR2004HeaderFields(dec []byte) *r2004Header {
	return &r2004Header{
		pageMapID:      int(binary.LittleEndian.Uint32(dec[0x50:])),
		pageMapAddress: int64(binary.LittleEndian.Uint64(dec[0x54:])) + r2004BaseOffset,
		sectionMapID:   int(binary.LittleEndian.Uint32(dec[0x5c:])),
	}
}

// systemPageHeader is the 20-byte header prefixing each (uncompressed-on-disk)
// system/data page: a type tag, the decompressed and compressed sizes, the
// compression flag, and a header CRC.
type systemPageHeader struct {
	pageType   uint32
	decompSize int
	compSize   int
	compType   uint32
}

const systemPageHeaderLen = 20

// readSystemPageHeader parses the 20-byte page header at off.
func readSystemPageHeader(data []byte, off int64) (systemPageHeader, error) {
	if off < 0 || off+systemPageHeaderLen > int64(len(data)) {
		return systemPageHeader{}, fmt.Errorf("dwg: system page header at %#x out of bounds (file len %d)", off, len(data))
	}
	b := data[off:]
	return systemPageHeader{
		pageType:   binary.LittleEndian.Uint32(b),
		decompSize: int(binary.LittleEndian.Uint32(b[4:])),
		compSize:   int(binary.LittleEndian.Uint32(b[8:])),
		compType:   binary.LittleEndian.Uint32(b[12:]),
	}, nil
}

// pageEntry locates one physical page in the file by its (signed) page number.
// Negative numbers denote gaps/free space, which still occupy file space and so
// advance the running offset.
type pageEntry struct {
	number int32
	size   int64
	offset int64
}

// parsePageMap decompresses the page-map page and walks its (number, size) entries
// in physical order, accumulating the file offset of each from r2004BaseOffset.
// The result maps positive page numbers to their location.
func parsePageMap(data []byte, h *r2004Header) (map[int32]pageEntry, error) {
	raw, err := readSystemSection(data, h.pageMapAddress, pageMapMagic)
	if err != nil {
		return nil, fmt.Errorf("dwg: page map: %w", err)
	}
	pages := make(map[int32]pageEntry)
	offset := int64(r2004BaseOffset)
	for i := 0; i+8 <= len(raw); i += 8 {
		num := int32(binary.LittleEndian.Uint32(raw[i:]))
		size := int64(binary.LittleEndian.Uint32(raw[i+4:]))
		if num > 0 {
			pages[num] = pageEntry{number: num, size: size, offset: offset}
		}
		offset += size
	}
	if len(pages) == 0 {
		return nil, fmt.Errorf("dwg: page map yielded no positive pages from %d bytes", len(raw))
	}
	return pages, nil
}

// readSystemSection reads the page at off, checks its type tag, and decompresses
// its body when the compression flag is set (compType 2), otherwise returns it
// verbatim.
func readSystemSection(data []byte, off int64, wantType uint32) ([]byte, error) {
	hdr, err := readSystemPageHeader(data, off)
	if err != nil {
		return nil, err
	}
	if hdr.pageType != wantType {
		return nil, fmt.Errorf("dwg: page at %#x type %#x, want %#x", off, hdr.pageType, wantType)
	}
	bodyStart := off + systemPageHeaderLen
	if bodyStart+int64(hdr.compSize) > int64(len(data)) {
		return nil, fmt.Errorf("dwg: page body at %#x size %d overruns file (len %d)", bodyStart, hdr.compSize, len(data))
	}
	body := data[bodyStart : bodyStart+int64(hdr.compSize)]
	if hdr.compType == 2 {
		return decompressR2004(body, hdr.decompSize)
	}
	return body, nil
}

// parseHeaderR2004 decodes the R2004+ paged container. Implemented incrementally;
// the encrypted header, page map and section map are landing in M1, after which
// this returns a fully-populated FileHeader.
func parseHeaderR2004(data []byte, version Version) (*FileHeader, error) {
	dec, err := decryptR2004Header(data)
	if err != nil {
		return nil, err
	}
	h := parseR2004HeaderFields(dec)
	if _, err := parsePageMap(data, h); err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("dwg: %s section-map decode not yet implemented (page map OK)", version)
}
