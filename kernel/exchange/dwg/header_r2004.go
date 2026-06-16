// SPDX-License-Identifier: GPL-2.0-only

package dwg

import "fmt"

// parseHeaderR2004 will decode the R2004+ paged container: the magic-sequence
// "encrypted" file header at offset 0x80, then the Reed-Solomon system pages
// (page map + section map) that index the LZ77-compressed data pages.
//
// It is implemented incrementally; until the LZ77 + page-map layer lands it
// reports an explicit unsupported error so callers fail loudly rather than on
// garbage. Tracked by the M1 container tasks.
func parseHeaderR2004(data []byte, version Version) (*FileHeader, error) {
	return nil, fmt.Errorf("dwg: %s paged container not yet implemented (R2004+ LZ77/page-map layer pending)", version)
}
