// SPDX-License-Identifier: GPL-2.0-only

package dwg

import "fmt"

// rsBlockSize is the codeword length of the RS(255,239) scheme R2004+ uses for its
// system pages (page map / section map): 239 data bytes + 16 parity bytes.
const rsBlockSize = 255

// reedSolomonDeinterleave recovers the logical bytes of an R2004+ system page.
// On disk the page is stored as blockCount interleaved RS(255,239) codewords; the
// original stream is reconstructed by taking the first dataSize bytes of each
// 255-byte block and interleaving them (dst[i*blockCount+j] = block j, byte i).
//
// This is the happy-path recovery: it strips the 16 parity bytes per block but
// does NOT perform Galois-field error correction. Valid, uncorrupted files (the
// only ones we round-trip) carry no errors to correct; a corrupt block would
// surface later as a failed CRC or a malformed structure rather than silently.
//
// Example:
//
//	raw := reedSolomonDeinterleave(pageBytes, pageCount, 0xEF)
func reedSolomonDeinterleave(src []byte, blockCount, dataSize int) ([]byte, error) {
	if blockCount <= 0 || dataSize <= 0 || dataSize > rsBlockSize {
		return nil, fmt.Errorf("dwg reed-solomon: invalid shape blockCount=%d dataSize=%d (want >0 and dataSize<=%d)", blockCount, dataSize, rsBlockSize)
	}
	need := blockCount * rsBlockSize
	if len(src) < need {
		return nil, fmt.Errorf("dwg reed-solomon: short input %d bytes, need %d (%d blocks * %d)", len(src), need, blockCount, rsBlockSize)
	}
	dst := make([]byte, blockCount*dataSize)
	for j := 0; j < blockCount; j++ {
		block := src[j*rsBlockSize:]
		for i := 0; i < dataSize; i++ {
			dst[i*blockCount+j] = block[i]
		}
	}
	return dst, nil
}
