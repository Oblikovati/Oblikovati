// SPDX-License-Identifier: GPL-2.0-only

// Package ipt decodes the payload of an Autodesk Inventor .ipt file: its named
// segments (parameters/sketches in PmDCSegment, the ACIS B-rep in PmBRepSegment),
// which live in paired B<uid>/M<uid> streams inside the RSeStorage of the CFBF
// container. Segment payloads are Zstandard-compressed (Inventor 2027+).
package ipt

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
	"strings"

	"github.com/klauspost/compress/zstd"
	"oblikovati.org/model/exchange/translators/olecf"
)

const (
	rseStorage   = "RSeStorage"
	bStreamStart = 18 // B-stream body = UUID(16) + u16(2) + compressed payload
)

// zstdMagic marks a Zstandard frame; older Inventor versions used zlib (unhandled here).
var zstdMagic = [4]byte{0x28, 0xB5, 0x2F, 0xFD}

// Document is a decoded .ipt: segments keyed by their human name. metaRaw keeps each
// segment's raw M-stream so the node-graph layer can parse its block metadata on demand.
type Document struct {
	segments map[string][]byte
	metaRaw  map[string][]byte
	// dcCache memoises the walked (and, for a pre-2023 save, layout-normalised) PmDCSegment nodes;
	// the feature and sketch decoders each re-walk it, and the normalisation must run once. See
	// dcNodes.
	dcCache  []dcNode
	dcCached bool
}

// Open parses the CFBF container and decompresses every B/M segment pair.
func Open(data []byte) (*Document, error) {
	f, err := olecf.Open(data)
	if err != nil {
		return nil, err
	}
	dec, err := zstd.NewReader(nil)
	if err != nil {
		return nil, err
	}
	defer dec.Close()
	d := &Document{segments: map[string][]byte{}, metaRaw: map[string][]byte{}}
	for name, pr := range collectPairs(f) {
		payload, err := decompress(dec, pr.b)
		if err != nil {
			continue // skip an unreadable segment rather than fail the whole doc
		}
		d.segments[name] = payload
		d.metaRaw[name] = pr.m
	}
	if len(d.segments) == 0 {
		return nil, fmt.Errorf("ipt: no readable segments (not an Inventor part?)")
	}
	return d, nil
}

// walkSegment walks the typed node blocks of a named segment, calling fn(typeID, payload)
// for each. It parses the segment's M-stream block metadata; a no-op if either the segment
// or its metadata is missing/unreadable.
func (d *Document) walkSegment(name string, fn func(typ uint32, payload []byte) bool) {
	payload, ok := d.segments[name]
	if !ok {
		return
	}
	meta, ok := parseSegmentMeta(d.metaRaw[name])
	if !ok {
		return
	}
	walkNodes(payload, meta, invVersion, fn)
}

// invVersion is the Inventor file generation we target (2027). It selects the >2014 block
// trailer form in the node-graph walk.
const invVersion = 2027

// Segment returns the decompressed bytes of a named segment (e.g. "PmDCSegment").
func (d *Document) Segment(name string) ([]byte, bool) {
	b, ok := d.segments[name]
	return b, ok
}

// SegmentNames lists the decoded segment names.
func (d *Document) SegmentNames() []string {
	out := make([]string, 0, len(d.segments))
	for n := range d.segments {
		out = append(out, n)
	}
	return out
}

// streamPair is a segment's paired M-stream (metadata) and B-stream (compressed data).
type streamPair struct{ m, b []byte }

// collectPairs maps each segment's human name (from its M-stream header) to its paired
// M- and B-streams, matched by shared uid suffix.
func collectPairs(f *olecf.File) map[string]streamPair {
	mByUID, bByUID := map[string][]byte{}, map[string][]byte{}
	for _, path := range f.Streams() {
		parent, leaf, ok := splitTop(path)
		if !ok || parent != rseStorage || len(leaf) < 2 {
			continue
		}
		data, err := f.Read(path)
		if err != nil {
			continue
		}
		switch leaf[0] {
		case 'M':
			mByUID[leaf[1:]] = data
		case 'B':
			bByUID[leaf[1:]] = data
		}
	}
	out := map[string]streamPair{}
	for uid, m := range mByUID {
		b, ok := bByUID[uid]
		if !ok {
			continue
		}
		if name := segmentName(m); name != "" {
			out[name] = streamPair{m: m, b: b}
		}
	}
	return out
}

// segmentName reads the segment's human name from the (uncompressed) M-header:
// text8 tag, u16 ver, u16[8], text16 name, ...
func segmentName(m []byte) string {
	c := newCursor(m)
	c.text8()  // "RSe Meta Stream Version 8"
	c.u16()    // version
	c.skip(16) // u16[8]
	return c.text16()
}

// decompress strips the 18-byte B-stream header and inflates the payload, which is
// Zstd (Inventor 2027+) or zlib/deflate (older Inventor, e.g. 2020–2026 — the format the
// bulk of real-world .ipt files still use).
func decompress(dec *zstd.Decoder, bStream []byte) ([]byte, error) {
	if len(bStream) < bStreamStart+4 {
		return nil, fmt.Errorf("ipt: B-stream too short (%d bytes)", len(bStream))
	}
	payload := bStream[bStreamStart:]
	if [4]byte(payload[:4]) == zstdMagic {
		return dec.DecodeAll(payload, nil)
	}
	if payload[0] == zlibCMF { // older Inventor: zlib-wrapped deflate (0x78 header)
		zr, err := zlib.NewReader(bytes.NewReader(payload))
		if err != nil {
			return nil, err
		}
		defer zr.Close()
		return io.ReadAll(zr)
	}
	return nil, fmt.Errorf("ipt: unexpected compression (magic %x); only Zstd and zlib are supported", payload[:4])
}

// zlibCMF is the zlib header's first byte (deflate, 32K window): 0x78 for all common
// FLG variants (0x78 0x01/0x5E/0x9C/0xDA).
const zlibCMF = 0x78

// splitTop splits "A/B" into ("A","B"); paths with more than two parts are rejected
// so nested storages (RSeEmbeddings, Templates, V1) are excluded from segment pairing.
func splitTop(path string) (parent, leaf string, ok bool) {
	parts := strings.Split(path, "/")
	if len(parts) != 2 {
		return "", "", false
	}
	return parts[0], parts[1], true
}
