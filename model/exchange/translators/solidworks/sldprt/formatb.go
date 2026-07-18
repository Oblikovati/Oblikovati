// SPDX-License-Identifier: GPL-2.0-only

package sldprt

import (
	"bytes"
	"compress/flate"
	"encoding/binary"
	"errors"
	"io"
)

// Format B is the SolidWorks 2026 native container (non-OLE): a log-structured record store
// that holds the SAME logical streams as the CFBF format ("Contents/Config-0", …). Each record is
//
//	[marker 14 00 06 00 08 00][flags:u32][hash:u32][compSize:u32][uncompSize:u32][nameLen:u32]
//	[name: nameLen bytes, NIBBLE-SWAPPED][payload: compSize bytes, raw DEFLATE]
//
// so once the name is un-swapped and the payload raw-inflated, every stream is byte-identical to
// the CFBF format's — the two containers share one downstream decoder (the MFC CArchive object
// graph in "Contents/Config-0"). Reverse-engineered from SolidWorks 2026 (build/version 19000)
// using generated known parts; validated by Config-0 inflating to the `moPart_c` CArchive header.
var recordMarker = []byte{0x14, 0x00, 0x06, 0x00, 0x08, 0x00}

const (
	formatBVersionTag = 4  // data[4:8] little-endian == 4 tags the SolidWorks-2026 container
	recordHeaderLen   = 26 // marker(6) + flags(4) + hash(4) + compSize(4) + uncompSize(4) + nameLen(4)
	maxStreamNameLen  = 64
)

// isFormatB reports whether data is the SolidWorks-2026 non-OLE container (not a compound file).
// The tag at data[4:8] is stored big-endian ("00 00 00 04") — the one big-endian field in an
// otherwise little-endian container, so it doubles as a format discriminator.
func isFormatB(data []byte) bool {
	return len(data) >= 8 && binary.BigEndian.Uint32(data[4:]) == formatBVersionTag
}

// nibbleSwap swaps the high and low nibble of every byte. SolidWorks stores stream NAMES this way
// in format B (raw "34 f6 e6 47 …" un-swaps to "Contents…"); the binary header fields are stored
// straight, so only the name bytes are swapped back.
func nibbleSwap(b []byte) []byte {
	out := make([]byte, len(b))
	for i, x := range b {
		out[i] = (x&0x0f)<<4 | (x>>4)&0x0f
	}
	return out
}

// parseFormatB walks the record log and returns each stream's decoded (raw-inflated) bytes. Records
// are read sequentially — after a record its payload length is exact, so the scan resumes past the
// payload and never mistakes marker-like bytes inside a payload for a record. When a stream name
// recurs (a later save appended a new version), the last record wins (log-structured current state).
func parseFormatB(data []byte) (map[string][]byte, error) {
	streams := map[string][]byte{}
	pos := firstMarker(data, 0)
	for pos >= 0 && pos+recordHeaderLen <= len(data) {
		rec, next, ok := readRecord(data, pos)
		if !ok {
			pos = firstMarker(data, pos+1)
			continue
		}
		streams[rec.name] = rec.payload
		pos = firstMarker(data, next)
	}
	if len(streams) == 0 {
		return nil, errors.New("sldprt: no format-B records found")
	}
	return streams, nil
}

type record struct {
	name    string
	payload []byte
}

// readRecord parses one record at pos and returns it plus the offset just past its payload. ok is
// false when the header does not validate (a coincidental marker match), so the caller can skip it.
func readRecord(data []byte, pos int) (record, int, bool) {
	compSize := binary.LittleEndian.Uint32(data[pos+14:])
	uncompSize := binary.LittleEndian.Uint32(data[pos+18:])
	nameLen := int(binary.LittleEndian.Uint32(data[pos+22:]))
	nameStart := pos + recordHeaderLen
	payloadStart := nameStart + nameLen
	if nameLen == 0 || nameLen > maxStreamNameLen || payloadStart+int(compSize) > len(data) {
		return record{}, 0, false
	}
	name := nibbleSwap(data[nameStart:payloadStart])
	if !isPrintableName(name) {
		return record{}, 0, false
	}
	payload, err := inflateStream(data[payloadStart:payloadStart+int(compSize)], compSize, uncompSize)
	if err != nil {
		return record{}, 0, false
	}
	return record{name: string(name), payload: payload}, payloadStart + int(compSize), true
}

// inflateStream raw-inflates a record payload, or returns it verbatim when it is stored
// uncompressed (compSize == uncompSize). A length mismatch means this was not a real record.
func inflateStream(payload []byte, compSize, uncompSize uint32) ([]byte, error) {
	if compSize == uncompSize {
		return payload, nil
	}
	r := flate.NewReader(bytes.NewReader(payload))
	defer r.Close()
	out, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	if uint32(len(out)) != uncompSize {
		return nil, errors.New("sldprt: inflated size mismatch")
	}
	return out, nil
}

func firstMarker(data []byte, from int) int {
	if from < 0 {
		from = 0
	}
	i := bytes.Index(data[from:], recordMarker)
	if i < 0 {
		return -1
	}
	return from + i
}

func isPrintableName(b []byte) bool {
	for _, c := range b {
		if c < 0x20 || c > 0x7e {
			return false
		}
	}
	return len(b) > 0
}
