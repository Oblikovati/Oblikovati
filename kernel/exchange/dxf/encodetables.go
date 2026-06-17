// SPDX-License-Identifier: GPL-2.0-only

package dxf

// This file writes the boilerplate sections AutoCAD expects around the geometry: the symbol
// TABLES, the model/paper-space BLOCKS, and the named-object dictionary in OBJECTS. The
// records are the standard minimum (the "0" layer, "Continuous"/"ByLayer"/"ByBlock" line
// types, the "Standard" text/dim styles, the "ACAD" app id, the *Model_Space/*Paper_Space
// blocks) wired together by the handles allocated in handles.go. Validated against the
// dxf2dwg → dwgread oracle.

// writeTables emits the TABLES section with the nine standard symbol tables.
func writeTables(w *tagWriter, h *encHandles) {
	w.tag(0, "SECTION")
	w.tag(2, "TABLES")
	writeVportTable(w, h)
	writeLtypeTable(w, h)
	writeLayerTable(w, h)
	writeStyleTable(w, h)
	writeEmptyTable(w, "VIEW", h.viewTable)
	writeEmptyTable(w, "UCS", h.ucsTable)
	writeAppidTable(w, h)
	writeDimstyleTable(w, h)
	writeBlockRecordTable(w, h)
	w.tag(0, "ENDSEC")
}

// tableHead writes a symbol-table header: the TABLE marker, name, handle, owner (the
// drawing, handle 0), the AcDbSymbolTable subclass, and the record count.
func tableHead(w *tagWriter, name string, handle uint64, count int) {
	w.tag(0, "TABLE")
	w.tag(2, name)
	w.handle(5, handle)
	w.handle(330, 0)
	w.tag(100, "AcDbSymbolTable")
	w.integer(70, count)
}

// recordHead writes a table record's common preamble: the record marker, handle, owner (the
// table), the AcDbSymbolTableRecord subclass and the record-type subclass.
func recordHead(w *tagWriter, typ string, handle, owner uint64, subclass string) {
	w.tag(0, typ)
	w.handle(5, handle)
	w.handle(330, owner)
	w.tag(100, "AcDbSymbolTableRecord")
	w.tag(100, subclass)
}

// writeEmptyTable writes a table header with no records (VIEW, UCS).
func writeEmptyTable(w *tagWriter, name string, handle uint64) {
	tableHead(w, name, handle, 0)
	w.tag(0, "ENDTAB")
}

// writeVportTable writes the VPORT table with the single *Active viewport.
//
//nolint:funlen // sequential VPORT field writes; the field list is the format.
func writeVportTable(w *tagWriter, h *encHandles) {
	tableHead(w, "VPORT", h.vportTable, 1)
	recordHead(w, "VPORT", h.vportActive, h.vportTable, "AcDbViewportTableRecord")
	w.tag(2, "*Active")
	w.integer(70, 0)
	w.real(10, 0)
	w.real(20, 0) // lower-left corner
	w.real(11, 1)
	w.real(21, 1) // upper-right corner
	w.real(12, 0)
	w.real(22, 0) // view centre
	w.real(13, 0)
	w.real(23, 0) // snap base
	w.real(14, 0.5)
	w.real(24, 0.5) // snap spacing
	w.real(15, 0.5)
	w.real(25, 0.5)                  // grid spacing
	w.coord(16, [3]float64{0, 0, 1}) // view direction
	w.coord(17, [3]float64{0, 0, 0}) // view target
	w.real(40, 1)                    // view height
	w.real(41, 1)                    // aspect ratio
	w.real(42, 50)                   // lens length
	w.real(43, 0)                    // front clip
	w.real(44, 0)                    // back clip
	w.real(50, 0)                    // snap angle
	w.real(51, 0)                    // view twist
	w.integer(71, 0)                 // view mode
	w.integer(72, 1000)              // circle zoom
	w.integer(73, 1)                 // fast zoom
	w.integer(74, 3)                 // UCSICON
	w.integer(75, 0)                 // snap on
	w.integer(76, 1)                 // grid on
	w.integer(77, 0)                 // snap style
	w.integer(78, 0)                 // snap isopair
	w.tag(0, "ENDTAB")
}

// writeLtypeTable writes the LTYPE table with the three standard line types.
func writeLtypeTable(w *tagWriter, h *encHandles) {
	tableHead(w, "LTYPE", h.ltypeTable, 3)
	writeLtype(w, h.ltypeByBlock, h.ltypeTable, "ByBlock", "")
	writeLtype(w, h.ltypeByLayer, h.ltypeTable, "ByLayer", "")
	writeLtype(w, h.ltypeContinuous, h.ltypeTable, "Continuous", "Solid line")
	w.tag(0, "ENDTAB")
}

// writeLtype writes one LTYPE record (a non-patterned line type: zero dashes).
func writeLtype(w *tagWriter, handle, owner uint64, name, desc string) {
	recordHead(w, "LTYPE", handle, owner, "AcDbLinetypeTableRecord")
	w.tag(2, name)
	w.integer(70, 0)
	w.tag(3, desc)
	w.integer(72, 65) // alignment 'A'
	w.integer(73, 0)  // dash count
	w.real(40, 0)     // pattern length
}

// writeLayerTable writes the LAYER table with the default "0" layer.
func writeLayerTable(w *tagWriter, h *encHandles) {
	tableHead(w, "LAYER", h.layerTable, 1)
	recordHead(w, "LAYER", h.layer0, h.layerTable, "AcDbLayerTableRecord")
	w.tag(2, "0")
	w.integer(70, 0)
	w.integer(62, 7) // colour white
	w.tag(6, "Continuous")
	w.integer(370, -3) // lineweight by default
	w.tag(0, "ENDTAB")
}

// writeStyleTable writes the STYLE table with the "Standard" text style.
func writeStyleTable(w *tagWriter, h *encHandles) {
	tableHead(w, "STYLE", h.styleTable, 1)
	recordHead(w, "STYLE", h.styleStandard, h.styleTable, "AcDbTextStyleTableRecord")
	w.tag(2, "Standard")
	w.integer(70, 0)
	w.real(40, 0) // fixed height
	w.real(41, 1) // width factor
	w.real(50, 0) // oblique angle
	w.integer(71, 0)
	w.real(42, 0.2) // last height
	w.tag(3, "txt")
	w.tag(4, "")
	w.tag(0, "ENDTAB")
}

// writeAppidTable writes the APPID table with the "ACAD" registered application.
func writeAppidTable(w *tagWriter, h *encHandles) {
	tableHead(w, "APPID", h.appidTable, 1)
	recordHead(w, "APPID", h.appidACAD, h.appidTable, "AcDbRegAppTableRecord")
	w.tag(2, "ACAD")
	w.integer(70, 0)
	w.tag(0, "ENDTAB")
}

// writeDimstyleTable writes the DIMSTYLE table with the "Standard" dimension style. The
// DIMSTYLE table is special: its header carries the AcDbDimStyleTable subclass and its
// records use handle code 105 (not 5).
func writeDimstyleTable(w *tagWriter, h *encHandles) {
	tableHead(w, "DIMSTYLE", h.dimstyleTable, 1)
	w.tag(100, "AcDbDimStyleTable")
	w.integer(71, 0)
	w.tag(0, "DIMSTYLE")
	w.handle(105, h.dimstyleStd)
	w.handle(330, h.dimstyleTable)
	w.tag(100, "AcDbSymbolTableRecord")
	w.tag(100, "AcDbDimStyleTableRecord")
	w.tag(2, "Standard")
	w.integer(70, 0)
	w.tag(0, "ENDTAB")
}

// writeBlockRecordTable writes the BLOCK_RECORD table with the *Model_Space and
// *Paper_Space records that own the model- and paper-space entities.
func writeBlockRecordTable(w *tagWriter, h *encHandles) {
	tableHead(w, "BLOCK_RECORD", h.blockRecordTable, 2)
	writeBlockRecord(w, h.modelSpaceBR, h.blockRecordTable, "*Model_Space")
	writeBlockRecord(w, h.paperSpaceBR, h.blockRecordTable, "*Paper_Space")
	w.tag(0, "ENDTAB")
}

// writeBlockRecord writes one BLOCK_RECORD record.
func writeBlockRecord(w *tagWriter, handle, owner uint64, name string) {
	recordHead(w, "BLOCK_RECORD", handle, owner, "AcDbBlockTableRecord")
	w.tag(2, name)
}

// writeBlocks emits the BLOCKS section with the empty *Model_Space and *Paper_Space block
// definitions every drawing carries.
func writeBlocks(w *tagWriter, h *encHandles) {
	w.tag(0, "SECTION")
	w.tag(2, "BLOCKS")
	writeBlock(w, h.modelBlock, h.modelSpaceBR, "*Model_Space", false)
	writeEndblk(w, h.modelEndblk, h.modelSpaceBR)
	writeBlock(w, h.paperBlock, h.paperSpaceBR, "*Paper_Space", true)
	writeEndblk(w, h.paperEndblk, h.paperSpaceBR)
	w.tag(0, "ENDSEC")
}

// writeBlock writes a BLOCK begin marker for a block definition.
func writeBlock(w *tagWriter, handle, owner uint64, name string, paper bool) {
	w.tag(0, "BLOCK")
	w.handle(5, handle)
	w.handle(330, owner)
	w.tag(100, "AcDbEntity")
	if paper {
		w.integer(67, 1) // paper-space flag
	}
	w.tag(8, "0")
	w.tag(100, "AcDbBlockBegin")
	w.tag(2, name)
	w.integer(70, 0)
	w.coord(10, [3]float64{0, 0, 0})
	w.tag(3, name)
	w.tag(1, "")
}

// writeEndblk writes a block's ENDBLK end marker.
func writeEndblk(w *tagWriter, handle, owner uint64) {
	w.tag(0, "ENDBLK")
	w.handle(5, handle)
	w.handle(330, owner)
	w.tag(100, "AcDbEntity")
	w.tag(100, "AcDbBlockEnd")
}

// writeObjects emits the OBJECTS section with the named-object root dictionary.
func writeObjects(w *tagWriter, h *encHandles) {
	w.tag(0, "SECTION")
	w.tag(2, "OBJECTS")
	w.tag(0, "DICTIONARY")
	w.handle(5, h.rootDict)
	w.handle(330, 0)
	w.tag(100, "AcDbDictionary")
	w.integer(281, 1)
	w.tag(0, "ENDSEC")
}
