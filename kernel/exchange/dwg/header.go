// SPDX-License-Identifier: GPL-2.0-only

package dwg

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// R2000 section-locator record IDs. The fixed-position locator table at the head
// of an R13–R2000 file points at each top-level section by this id.
const (
	secHeaderVars  = 0 // drawing header variables
	secClasses     = 1 // class definitions for custom objects
	secObjectMap   = 2 // handle → file-offset directory ("object map")
	secObjFreeSpec = 3 // object free space (usually empty)
	secTemplate    = 4 // template / second-header region
	secAuxHeader   = 5 // auxiliary header
)

// sentinelHeaderEndR2000 terminates the R2000 locator table; verifying it catches
// a mis-parsed table early rather than as garbage further in. Only the first 15
// bytes are a stable signature — the 16th byte is not fixed across files (0x00 in
// the corpus), so framing checks compare sentinelSignatureLen bytes.
var sentinelHeaderEndR2000 = [16]byte{
	0x95, 0xA0, 0x4E, 0x28, 0x99, 0x82, 0x1A, 0xE5,
	0x5E, 0x41, 0xE0, 0x5F, 0x9D, 0x3A, 0x4D, 0x00,
}

const sentinelSignatureLen = 15

// SectionLocator is one entry of the R2000 locator table: where a top-level
// section lives in the file and how big it is.
type SectionLocator struct {
	ID      byte
	Address int64
	Size    int64
}

// FileHeader is the version-independent summary of a DWG file's top-level layout.
// For R2000 it carries the locator table; the paged R2004+ equivalent is filled by
// the container layer (see header_r2004.go).
type FileHeader struct {
	Version            Version
	MaintenanceVersion byte
	PreviewAddress     int64
	Codepage           int
	Sections           []SectionLocator // R2000 flat locator table
	r2004              *r2004Container  // R2004+ paging layer (nil for R2000)
}

// LogicalSection returns the assembled, decompressed bytes of a named logical
// section (R2004+ only), e.g. "AcDb:Handles", "AcDb:Classes", "AcDb:AcDbObjects".
// The bytes are reconstructed on demand from the section's data pages.
//
// Example:
//
//	handles, err := h.LogicalSection(data, "AcDb:Handles")
func (h *FileHeader) LogicalSection(data []byte, name string) ([]byte, error) {
	if h.r2004 == nil {
		return nil, fmt.Errorf("dwg: LogicalSection %q requested on a non-paged (R2000) file", name)
	}
	for _, d := range h.r2004.sections {
		if d.name == name {
			return assembleSection(data, h.r2004.pages, d)
		}
	}
	return nil, fmt.Errorf("dwg: logical section %q not found", name)
}

// ObjectMapBytes returns the raw object-map (handle directory) bytes, abstracting
// the version: the AcDb:Handles logical section for R2004+, or the flat object-map
// section for R2000.
func (h *FileHeader) ObjectMapBytes(data []byte) ([]byte, error) {
	if h.Version.Paged() {
		return h.LogicalSection(data, "AcDb:Handles")
	}
	return h.SectionBytes(data, secObjectMap)
}

// ObjectData returns the stream that object-map offsets index into: the assembled
// AcDb:AcDbObjects section for R2004+, or the whole file for R2000.
func (h *FileHeader) ObjectData(data []byte) ([]byte, error) {
	if h.Version.Paged() {
		return h.LogicalSection(data, "AcDb:AcDbObjects")
	}
	return data, nil
}

// SectionNames lists the logical section names present in a paged file, for
// diagnostics and tests.
func (h *FileHeader) SectionNames() []string {
	if h.r2004 == nil {
		return nil
	}
	names := make([]string, len(h.r2004.sections))
	for i, d := range h.r2004.sections {
		names[i] = d.name
	}
	return names
}

// Locator returns the locator record with the given id, or false if the file does
// not carry that section.
func (h *FileHeader) Locator(id byte) (SectionLocator, bool) {
	for _, s := range h.Sections {
		if s.ID == id {
			return s, true
		}
	}
	return SectionLocator{}, false
}

// SectionBytes returns the raw on-disk bytes of section id from data. For R2000
// these sections are uncompressed, so the slice is the section as-is.
func (h *FileHeader) SectionBytes(data []byte, id byte) ([]byte, error) {
	loc, ok := h.Locator(id)
	if !ok {
		return nil, fmt.Errorf("dwg: section id %d not present in locator table", id)
	}
	end := loc.Address + loc.Size
	if loc.Address < 0 || end > int64(len(data)) {
		return nil, fmt.Errorf("dwg: section id %d range [%d,%d) out of file bounds (len %d)", id, loc.Address, end, len(data))
	}
	return data[loc.Address:end], nil
}

// parseHeaderR2000 decodes the fixed-layout R13–R2000 file header: scalar fields
// at known byte offsets, then a locator table of (id, address, size) records, a
// CRC over the table, and the closing sentinel.
func parseHeaderR2000(data []byte, version Version) (*FileHeader, error) {
	if len(data) < 0x16 {
		return nil, fmt.Errorf("dwg: R2000 header truncated: have %d bytes, need >= 22", len(data))
	}
	h := &FileHeader{
		Version:            version,
		MaintenanceVersion: data[0x0B],
		PreviewAddress:     int64(binary.LittleEndian.Uint32(data[0x0D:])),
		Codepage:           int(binary.LittleEndian.Uint16(data[0x13:])),
	}
	count := int(binary.LittleEndian.Uint32(data[0x15:]))
	if err := readLocatorTable(data, count, h); err != nil {
		return nil, err
	}
	return h, nil
}

// readLocatorTable reads count (id, address, size) records starting at 0x19,
// verifies the trailing CRC and sentinel, and stores the records on h.
func readLocatorTable(data []byte, count int, h *FileHeader) error {
	const recStart, recSize = 0x19, 9
	if count < 0 || count > 64 {
		return fmt.Errorf("dwg: implausible R2000 section count %d (want 0..64)", count)
	}
	end := recStart + count*recSize
	if end+2+16 > len(data) {
		return fmt.Errorf("dwg: R2000 locator table of %d records overruns file (need %d bytes, have %d)", count, end+18, len(data))
	}
	h.Sections = make([]SectionLocator, count)
	for i := 0; i < count; i++ {
		off := recStart + i*recSize
		h.Sections[i] = SectionLocator{
			ID:      data[off],
			Address: int64(binary.LittleEndian.Uint32(data[off+1:])),
			Size:    int64(binary.LittleEndian.Uint32(data[off+5:])),
		}
	}
	return verifyLocatorFraming(data, end)
}

// verifyLocatorFraming checks the CRC and the closing 16-byte sentinel. The CRC
// runs from byte 0 over the whole header up to the stored value, seeded with the
// ODA header constant 0xC0C1 (verified against real R2000 files); a mismatch
// means we mis-located the table.
func verifyLocatorFraming(data []byte, end int) error {
	want := binary.LittleEndian.Uint16(data[end:])
	if got := crc16(0xC0C1, data[0:end]); got != want {
		return fmt.Errorf("dwg: R2000 locator CRC mismatch: computed %#04x, stored %#04x", got, want)
	}
	var sentinel [16]byte
	copy(sentinel[:], data[end+2:end+2+16])
	if !bytes.Equal(sentinel[:sentinelSignatureLen], sentinelHeaderEndR2000[:sentinelSignatureLen]) {
		return fmt.Errorf("dwg: R2000 header sentinel mismatch: got % X, want signature % X", sentinel, sentinelHeaderEndR2000[:sentinelSignatureLen])
	}
	return nil
}
