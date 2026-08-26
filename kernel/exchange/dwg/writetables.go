// SPDX-License-Identifier: GPL-2.0-only

package dwg

// writetables.go encodes the standard symbol-table records every R2000 drawing carries:
// layer "0", the three system linetypes, the STANDARD text and dimension styles, the ACAD
// application id, and the *Active viewport. Field order, types and the handle-stream tails
// are taken from LibreDWG's decode of a real R2000 file (the validation oracle) and the ODA
// §20.4 prescriptions. Each record's common preamble (name, flags, owner, xdic) is written
// by writeRecordCommon; the type-specific fields follow.

// writeRecordCommon writes the shared symbol-table-record preamble: numreactors and the
// entry name in the data stream, then the handle stream's owning control object, a null
// xdic, and a null external-reference block handle. The oracle reads that external-reference
// handle on every table record (even non-xref ones), so it is always emitted here.
func writeRecordCommon(b *objectBody, control uint64, name string) {
	b.data.WriteBL(0)       // numreactors
	writeName(b.data, name) // entry name (TV)
	b.data.WriteBit(1)      // 64-flag: entry is referenced (the 0x40 bit of the 70 group)
	b.data.WriteBS(0)       // xrefindex+1 → -1 (not from an xref)
	b.data.WriteBit(0)      // Xdep: not xref-dependent
	b.handles.WriteHandle(softPtrCode, control)
	b.handles.WriteHandle(hardOwnerCode, 0) // xdicobjhandle (null)
	b.handles.WriteHandle(hardPtrCode, 0)   // external reference block (null)
}

// writeLayer encodes LAYER "0": the on/plottable flag word with default lineweight, the
// layer colour (7, white), then the plotstyle and linetype handles. The flag word 0x3F0 is
// the value the oracle decodes for a normal on, plottable, default-lineweight layer.
func writeLayer(h *graphHandles) []byte {
	b := newObjectBody(h.layer0, int(TypeLayer))
	writeRecordCommon(b, h.layerControl, "0")
	b.data.WriteBS(0x3F0)                                 // values: plot flag (0x10) | lineweight 0x1F (mask 0x3E0)
	b.data.WriteBS(7)                                     // colour index (white)
	b.handles.WriteHandle(hardPtrCode, h.placeholder)     // plotstyle → "Normal" placeholder
	b.handles.WriteHandle(hardPtrCode, h.ltypeContinuous) // linetype
	return frameObject(b)
}

// writeStyle encodes the STANDARD text style (SHAPEFILE record): a plain (non-vertical,
// non-shape) font with unit width and the default "txt" font file.
func writeStyle(h *graphHandles) []byte {
	b := newObjectBody(h.styleStandard, int(TypeStyle))
	writeRecordCommon(b, h.styleControl, "Standard")
	b.data.WriteBit(0)       // vertical
	b.data.WriteBit(0)       // shape file
	b.data.WriteBD(0)        // fixed height
	b.data.WriteBD(1)        // width factor
	b.data.WriteBD(0)        // oblique angle
	b.data.WriteRC(0)        // generation flags
	b.data.WriteBD(0)        // last height
	writeName(b.data, "txt") // font name
	writeTextEmpty(b.data)   // big-font name
	return frameObject(b)
}

// TypeStyle/TypeLtype/TypeVport/TypeAppid/TypeDimstyle are the fixed R2000 type codes for
// the symbol-table records (ODA §20.3); only the ones the writer emits are named.
const (
	TypeStyle    ObjectType = 0x35
	TypeLtype    ObjectType = 0x39
	TypeVport    ObjectType = 0x41
	TypeAppid    ObjectType = 0x43
	TypeDimstyle ObjectType = 0x45
)

// writeLinetype encodes a simple (dashless) LTYPE record. R2000 LTYPE records always carry
// the 256-byte string area (confirmed by the oracle: a continuous linetype decodes to a
// 280-byte object), so 256 zero bytes are emitted after the dash count.
func writeLinetype(handle, control uint64, name string) []byte {
	b := newObjectBody(handle, int(TypeLtype))
	writeRecordCommon(b, control, name)
	writeTextEmpty(b.data) // description
	b.data.WriteBD(0)      // pattern length
	b.data.WriteRC('A')    // alignment, always 'A'
	b.data.WriteRC(0)      // numdashes
	for range 256 {
		b.data.WriteRC(0) // string area
	}
	return frameObject(b)
}

// writeAppid encodes the ACAD application id: the standard preamble plus a single trailing
// unknown byte (DXF 71) the oracle always reads as 0.
func writeAppid(h *graphHandles) []byte {
	b := newObjectBody(h.appidAcad, int(TypeAppid))
	writeRecordCommon(b, h.appidControl, "ACAD")
	b.data.WriteRC(0) // unknown (DXF 71)
	return frameObject(b)
}
