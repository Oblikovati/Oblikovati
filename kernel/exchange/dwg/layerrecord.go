// SPDX-License-Identifier: GPL-2.0-only

package dwg

import (
	"errors"
	"fmt"
)

// errLayerVersionUnsupported marks a LAYER record this decoder deliberately does not read.
var errLayerVersionUnsupported = errors.New("dwg: layer records are decoded for R2000 only")

// LAYER table records (#2015). An entity that states no colour of its own is BYLAYER — the
// overwhelming majority in a real drawing — so keeping an imported drawing's appearance means
// reading the layer it inherits from.
//
// Records are keyed by HANDLE, not by name. The name lives in the R2007+ string stream, which
// this decoder does not read, and resolution never needs it: an entity references its layer by
// handle.
//
// ONLY R2000 IS DECODED, and that limit is measured rather than assumed. Against the seven-file
// corpus the R2000 layout reads 109 layers with zero failures and a textbook colour distribution
// (16 red, 19 yellow, 12 green, 14 white — the standard palette a real layer table uses).
//
// The R2007+ layout is NOT understood. Sweeping the plausible field alignments over the six
// R2018 files, every one of them leaves layers with colour 0 — invalid for a layer — and no
// alignment is uniformly best: the strongest overall (one extra bit before the flag: 14 bad
// across 252 layers) is WORSE than the plain reading on testfile-6 (5 bad versus 0). An
// off-by-a-constant would be uniformly better everywhere, so the record has a field this model
// does not account for. Guessing would produce plausible-but-wrong colours, which is worse than
// inheriting the default: the drawing would look subtly recoloured with nothing to flag it.

// layerRecord is one LAYER table record's formatting.
type layerRecord struct {
	color      int // AutoCAD Color Index
	lineWeight int // hundredths of a millimetre, or dwgLineWeightByLayer
}

// decodeLayerRecord reads one LAYER object's formatting from the object-data stream.
//
// Layout after the shared preamble (ODA layer.spec): the common object data, then the entry name
// (inline before R2007, in the string stream after), a 64-bit flag, an xref index and dependency
// bit, the packed values short — whose bits 5–8 are the line weight — and the colour.
func decodeLayerRecord(r *BitReader, data []byte, ref ObjectRef, version Version) (layerRecord, error) {
	if version >= R2007 {
		return layerRecord{}, errLayerVersionUnsupported
	}
	if err := seekObjectData(r, data, ref, version); err != nil {
		return layerRecord{}, err
	}
	r.ReadBL() // numReactors
	if version >= R2004 {
		r.ReadBit() // xdicmissing
	}
	skipTV(r)   // entry name, inline before the string stream existed
	r.ReadBit() // 64-bit flag
	r.ReadBS()  // xrefindex + 1
	r.ReadBit() // xdep
	values := r.ReadBS()
	color := int(uint16(r.ReadBS()) & 0xFF)
	if err := r.Err(); err != nil {
		return layerRecord{}, fmt.Errorf("dwg: layer handle %d: %w", ref.Handle, err)
	}
	return layerRecord{color: color, lineWeight: layerValuesLineWeight(values)}, nil
}

// layerValuesLineWeight extracts the line weight packed into bits 5–8 of the LAYER values short.
func layerValuesLineWeight(values int) int {
	return dwgLineWeightValue(byte((values >> 5) & 0x1F))
}

// seekObjectData positions r at a non-entity object's data, after the preamble every object
// shares with an entity (size, type, handle, extended data) but before the object-specific
// fields. It is seekEntity's counterpart for table records, which carry no common ENTITY data.
func seekObjectData(r *BitReader, data []byte, ref ObjectRef, version Version) error {
	if ref.Offset < 0 || ref.Offset >= int64(len(data)) {
		return fmt.Errorf("dwg: object handle %d offset %d out of bounds (len %d)", ref.Handle, ref.Offset, len(data))
	}
	r.Reset(data, int(ref.Offset)*8)
	r.ReadMS()
	if version >= R2010 {
		r.ReadUMC() // handle-stream size
	}
	if version >= R2010 {
		r.ReadBOT()
	} else {
		r.ReadBS()
	}
	if version >= R2000 && version < R2010 {
		r.ReadRL() // bitsize
	}
	r.ReadHandle() // the object's own handle
	skipEED(r)
	return r.Err()
}

// skipTV consumes a pre-R2007 text value: a short length followed by that many bytes. The value
// itself is not needed — layers are keyed by handle — so it is skipped rather than decoded.
func skipTV(r *BitReader) {
	n := r.ReadBS()
	for i := 0; i < n && r.Err() == nil; i++ {
		r.ReadRC()
	}
}
