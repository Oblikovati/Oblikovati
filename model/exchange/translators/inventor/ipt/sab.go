// SPDX-License-Identifier: GPL-2.0-only

package ipt

import (
	"encoding/binary"
	"math"
	"strings"
)

// SAB (Standard ACIS Binary) chunk tags. Inventor's PmBRepSegment embeds a
// ShapeManager body in this tagged binary form. Reference: ACIS SAB grammar.
const (
	tagChar     = 2
	tagShort    = 3
	tagLong     = 4
	tagFloat    = 5
	tagDouble   = 6
	tagStrU8    = 7
	tagStrU16   = 8
	tagStrU32A  = 9
	tagTrue     = 10
	tagFalse    = 11
	tagRef      = 12 // entity reference: int32 record index (-1 = null)
	tagIdent    = 13 // base class name (u8 len + chars)
	tagSubIdent = 14 // leaf subtype name (u8 len + chars)
	tagOpen     = 15
	tagClose    = 16
	tagEnd      = 17 // record terminator
	tagStrU32B  = 18
	tagPosition = 19 // 3 doubles (a point)
	tagVector   = 20 // 3 doubles (a direction, may be unnormalized)
	tagEnum     = 21
	tagVector2D = 22
	tagInt64    = 23
)

// Chunk is one tagged field within a SAB record. Only the field matching Tag is set.
type Chunk struct {
	Tag byte
	Ref int32      // tagRef
	F   float64    // tagDouble / tagFloat
	Vec [3]float64 // tagPosition / tagVector
	Str string     // tagIdent / tagSubIdent / string tags
}

// Record is one ACIS entity: a base class ("surface"), optional leaf subtype
// ("plane" -> Name "plane-surface"), and its ordered field chunks.
type Record struct {
	Name    string
	Base    string
	Subtype string
	Chunks  []Chunk
}

// Refs returns the entity-reference targets of this record, in field order.
func (r Record) Refs() []int32 {
	var out []int32
	for _, c := range r.Chunks {
		if c.Tag == tagRef {
			out = append(out, c.Ref)
		}
	}
	return out
}

// Positions / Vectors / Doubles return the typed field values in field order.
func (r Record) Positions() [][3]float64 { return r.vecsOf(tagPosition) }
func (r Record) Vectors() [][3]float64   { return r.vecsOf(tagVector) }

func (r Record) vecsOf(tag byte) [][3]float64 {
	var out [][3]float64
	for _, c := range r.Chunks {
		if c.Tag == tag {
			out = append(out, c.Vec)
		}
	}
	return out
}

func (r Record) Doubles() []float64 {
	var out []float64
	for _, c := range r.Chunks {
		if c.Tag == tagDouble {
			out = append(out, c.F)
		}
	}
	return out
}

// ParseSAB tokenizes a decompressed PmBRepSegment into ACIS records, indexed by
// appearance order (the target space of entity references).
func ParseSAB(seg []byte) []Record {
	start := firstIdent(seg)
	if start < 0 {
		return nil
	}
	var recs []Record
	var cur *Record
	pendingSub := ""
	i := start
	for i < len(seg)-1 {
		switch seg[i] {
		case tagSubIdent:
			s, ni := readStr(seg, i+1)
			pendingSub, i = s, ni
			continue
		case tagIdent:
			s, ni := readStr(seg, i+1)
			name := s
			if pendingSub != "" {
				name = pendingSub + "-" + s
			}
			recs = append(recs, Record{Name: name, Base: s, Subtype: pendingSub})
			cur, pendingSub = &recs[len(recs)-1], ""
			i = ni
			continue
		case tagEnd:
			cur, i = nil, i+1
			continue
		}
		if cur == nil {
			i++
			continue
		}
		ch, ni, ok := readChunk(seg, i)
		if !ok {
			cur, pendingSub, i = nil, "", ni
			continue
		}
		cur.Chunks = append(cur.Chunks, ch)
		i = ni
	}
	return recs
}

// readChunk decodes one tagged value; ok=false on an unknown tag (caller resyncs).
func readChunk(d []byte, i int) (Chunk, int, bool) {
	tag := d[i]
	i++
	switch tag {
	case tagDouble:
		return Chunk{Tag: tag, F: f64(d, i)}, i + 8, true
	case tagFloat:
		if i+4 > len(d) {
			return Chunk{}, i, false
		}
		return Chunk{Tag: tag, F: float64(math.Float32frombits(binary.LittleEndian.Uint32(d[i:])))}, i + 4, true
	case tagLong, tagEnum:
		return Chunk{Tag: tag, Ref: i32(d, i)}, i + 4, true
	case tagRef:
		return Chunk{Tag: tag, Ref: i32(d, i)}, i + 4, true
	case tagShort:
		return Chunk{Tag: tag}, i + 2, true
	case tagChar:
		return Chunk{Tag: tag}, i + 1, true
	case tagInt64:
		return Chunk{Tag: tag}, i + 8, true
	case tagPosition, tagVector:
		return Chunk{Tag: tag, Vec: [3]float64{f64(d, i), f64(d, i+8), f64(d, i+16)}}, i + 24, true
	case tagVector2D:
		return Chunk{Tag: tag, Vec: [3]float64{f64(d, i), f64(d, i+8)}}, i + 16, true
	case tagTrue, tagFalse:
		return Chunk{Tag: tag}, i, true
	case tagStrU8, tagIdent, tagSubIdent:
		s, ni := readStr(d, i)
		return Chunk{Tag: tag, Str: s}, ni, true
	case tagOpen, tagClose:
		return Chunk{Tag: tag}, i, true
	}
	return Chunk{}, i, false
}

// firstIdent finds the first plausible entity record (tagIdent + short known name).
func firstIdent(d []byte) int {
	known := map[string]bool{
		"body": true, "lump": true, "shell": true, "face": true, "loop": true,
		"coedge": true, "edge": true, "vertex": true, "point": true, "transform": true,
	}
	for i := 0; i+3 < len(d); i++ {
		if d[i] != tagIdent {
			continue
		}
		n := int(d[i+1])
		if n < 3 || n > 20 || i+2+n > len(d) {
			continue
		}
		name := string(d[i+2 : i+2+n])
		base := name
		if k := strings.IndexByte(base, '-'); k >= 0 {
			base = base[:k]
		}
		if known[base] {
			return i
		}
	}
	return -1
}

func readStr(d []byte, i int) (string, int) {
	if i >= len(d) {
		return "", i
	}
	n := int(d[i])
	i++
	if i+n > len(d) {
		return "", i
	}
	return string(d[i : i+n]), i + n
}

func f64(d []byte, i int) float64 {
	if i+8 > len(d) {
		return 0
	}
	return math.Float64frombits(binary.LittleEndian.Uint64(d[i:]))
}

func i32(d []byte, i int) int32 {
	if i+4 > len(d) {
		return 0
	}
	return int32(binary.LittleEndian.Uint32(d[i:]))
}
