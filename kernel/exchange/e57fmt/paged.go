// SPDX-License-Identifier: GPL-2.0-only

package e57fmt

import "fmt"

// pagedFile reads the logical byte stream out of an E57 file's checksummed pages. An E57 file is a
// run of fixed-size pages; the last 4 bytes of every page are a CRC, so payload occupies only the
// first pageSize-4 bytes of each page. Offsets recorded in the header/XML are PHYSICAL (they index
// the raw bytes including CRCs); readLogical walks from a physical offset and stitches the payload
// spans together, hopping over each page's trailing CRC. The CRC itself is not verified — a
// truncated read is the only failure mode that matters for decoding points.
type pagedFile struct {
	data     []byte
	pageSize uint64
}

func newPagedFile(data []byte, pageSize uint64) *pagedFile {
	return &pagedFile{data: data, pageSize: pageSize}
}

// payloadPerPage is the number of usable bytes per page (the page minus its 4-byte CRC).
func (p *pagedFile) payloadPerPage() uint64 { return p.pageSize - 4 }

// readLogical returns n logical bytes starting at physical offset phys, skipping each page's CRC,
// and the physical offset of the byte that follows — so a caller reading consecutive records
// threads that offset back in as a cursor (the logical length of a record is NOT its physical span
// when a page boundary, and its CRC, falls inside it).
func (p *pagedFile) readLogical(phys uint64, n uint64) ([]byte, uint64, error) {
	out := make([]byte, 0, n)
	pos := phys
	for uint64(len(out)) < n {
		pageStart := (pos / p.pageSize) * p.pageSize
		payloadEnd := pageStart + p.payloadPerPage()
		take := minU64(payloadEnd-pos, n-uint64(len(out)))
		if pos+take > uint64(len(p.data)) {
			return nil, 0, fmt.Errorf("e57fmt: truncated reading %d logical bytes at physical %d (file is %d bytes)", n, phys, len(p.data))
		}
		out = append(out, p.data[pos:pos+take]...)
		pos += take
		if pos >= payloadEnd {
			pos = pageStart + p.pageSize // hop over the page CRC to the next page's payload
		}
	}
	return out, pos, nil
}

func minU64(a, b uint64) uint64 {
	if a < b {
		return a
	}
	return b
}
