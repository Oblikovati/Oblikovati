// SPDX-License-Identifier: GPL-2.0-only

package dwg

import "hash/crc32"

// crc16Table is the reflected CRC-16/ARC table (polynomial 0xA001), generated at
// init. The ODA spec calls this checksum "crc8" even though it is 16-bit; it
// guards the sentinel-bounded sections of R13–R2000 files.
var crc16Table = func() [256]uint16 {
	var t [256]uint16
	for i := range t {
		crc := uint16(i)
		for b := 0; b < 8; b++ {
			if crc&1 != 0 {
				crc = crc>>1 ^ 0xA001
			} else {
				crc >>= 1
			}
		}
		t[i] = crc
	}
	return t
}()

// crc16 computes the ODA section checksum over data, continuing from seed. DWG
// seeds each section with a section-specific constant, so seed is a parameter
// rather than a fixed initial value.
//
// Example:
//
//	sum := crc16(0xC0C1, sectionBytes) // compare against the trailing stored CRC
func crc16(seed uint16, data []byte) uint16 {
	dx := seed
	for _, b := range data {
		al := b ^ byte(dx&0xFF)
		dx = dx>>8&0xFF ^ crc16Table[al]
	}
	return dx
}

// crc32 computes the standard CRC-32 (IEEE/reflected) used to protect R2004+ data
// pages, continuing from seed. With seed 0 it equals the ordinary CRC-32 of data;
// DWG threads a running seed across a page's regions, so seed is exposed.
//
// Example:
//
//	sum := crc32(0, pageBytes)
func crc32sum(seed uint32, data []byte) uint32 {
	return crc32.Update(seed, crc32.IEEETable, data)
}
