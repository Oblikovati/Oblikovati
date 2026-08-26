// SPDX-License-Identifier: GPL-2.0-only

package dwg

// writeheadervars.go emits the R2000 drawing-header-variables section: the VARIABLE_BEGIN
// sentinel, an RL byte size, the variable list (the exact inverse of readHeaderVarsR2000 in
// headervars.go, extended through the trailing GUID/space/linetype handles the reader stops
// short of), a CRC, and the VARIABLE_END sentinel. Numeric variables use neutral defaults; the
// handle references are wired to the synthesised object graph so AutoCAD resolves every system
// pointer. Values that must be specific (INSUNITS, the control/space/linetype handles) are set
// from the caller's graph; the rest mirror a fresh AutoCAD drawing.

// vendSentinel closes the header-variable block (ODA DWG_SENTINEL_VARIABLE_END).
var vendSentinel = []byte{
	0x30, 0x84, 0xE0, 0xDC, 0x02, 0x21, 0xC7, 0x56,
	0xA0, 0x83, 0x97, 0x47, 0xB1, 0x92, 0xCC, 0xA0,
}

// nullGUID is a syntactically valid all-zero GUID for FINGERPRINTGUID/VERSIONGUID.
const nullGUID = "{00000000-0000-0000-0000-000000000000}"

// encodeHeaderVars builds the complete header-variables section bytes for the given graph and
// $INSUNITS code. handseed is the next free handle (must exceed every object handle).
func encodeHeaderVars(h *graphHandles, insunits int, handseed uint64) []byte {
	v := NewBitWriter()
	writeHeaderVarBody(v, h, insunits, handseed)
	v.AlignToByte()
	vardata := v.Bytes()

	out := NewBitWriter()
	for _, by := range vbeginSentinel {
		out.WriteRC(by)
	}
	crcStart := NewBitWriter()
	crcStart.WriteRL(uint32(len(vardata)))
	for _, by := range vardata {
		crcStart.WriteRC(by)
	}
	sized := crcStart.Bytes()
	for _, by := range sized {
		out.WriteRC(by)
	}
	out.WriteRS(crc16(0xC0C1, sized)) // CRC over the size field + variable data
	for _, by := range vendSentinel {
		out.WriteRC(by)
	}
	return out.Bytes()
}

// writeHeaderVarBody writes the R2000 variable list in spec order, the inverse of
// readHeaderVarsR2000. It is one long fixed sequence; the sub-writers group it for clarity.
func writeHeaderVarBody(w *BitWriter, h *graphHandles, insunits int, handseed uint64) {
	writeHeaderUnitsModes(w)
	writeHeaderSizesAndCurrent(w, h, handseed)
	writeHeaderUcsAndExtents(w)
	writeHeaderDimVars(w)
	writeHeaderControlHandles(w, h)
	w.WriteBL(0x2a1d)   // FLAGS
	w.WriteBS(insunits) // INSUNITS
	writeHeaderTail(w, h)
}

// writeHeaderUnitsModes writes the unit ratios/names, the two unknown longs, the VX handle,
// the run of boolean mode flags, and the unit/count BitShorts (DIMASO..TEXTQLTY).
//
//nolint:funlen // sequential header-variable writes in spec order; length is the field layout.
func writeHeaderUnitsModes(w *BitWriter) {
	for range 4 {
		w.WriteBD(1) // unit1..4 ratio
	}
	for range 4 {
		writeTextEmpty(w) // unit1..4 name
	}
	w.WriteBL(0)                  // unknown_8
	w.WriteBL(0)                  // unknown_9
	w.WriteHandle(hardPtrCode, 0) // VX_TABLE_RECORD (null)
	for range 20 {                // DIMASO..PELLIPSE mode bits (see readModeFlags, R2000)
		w.WriteBit(0)
	}
	w.WriteBS(0)  // PROXYGRAPHICS
	w.WriteBS(0)  // TREEDEPTH
	w.WriteBS(2)  // LUNITS
	w.WriteBS(4)  // LUPREC
	w.WriteBS(0)  // AUNITS
	w.WriteBS(0)  // AUPREC
	w.WriteBS(0)  // ATTMODE
	w.WriteBS(0)  // PDMODE
	for range 5 { // USERI1..5
		w.WriteBS(0)
	}
	for range 14 { // SPLINESEGS..TEXTQLTY
		w.WriteBS(0)
	}
}

// writeHeaderSizesAndCurrent writes the BD size defaults, the MENU string, the four timers,
// CECOLOR, then the current-setting handles (HANDSEED, CLAYER, TEXTSTYLE, CELTYPE, DIMSTYLE,
// CMLSTYLE).
//
//nolint:funlen // sequential header size/handle writes in spec order; length is the layout.
func writeHeaderSizesAndCurrent(w *BitWriter, h *graphHandles, handseed uint64) {
	w.WriteBD(1) // LTSCALE
	w.WriteBD(1) // TEXTSIZE
	w.WriteBD(0) // TRACEWID
	w.WriteBD(1) // SKETCHINC
	w.WriteBD(0) // FILLETRAD
	w.WriteBD(0) // THICKNESS
	w.WriteBD(0) // ANGBASE
	w.WriteBD(0) // PDSIZE
	w.WriteBD(0) // PLINEWID
	for range 5 {
		w.WriteBD(0) // USERR1..5
	}
	for range 4 {
		w.WriteBD(0) // CHAMFERA..D
	}
	w.WriteBD(0.5)    // FACETRES
	w.WriteBD(1)      // CMLSCALE
	w.WriteBD(1)      // CELTSCALE
	writeName(w, ".") // MENU
	for range 4 {
		w.WriteBL(0) // TDCREATE, TDUPDATE, TDINDWG, TDUSRTIMER (each TIMEBLL = 2 BL)
		w.WriteBL(0)
	}
	w.WriteBS(colorByLayer)                        // CECOLOR (index)
	w.WriteHandle(ownHandleCode, handseed)         // HANDSEED (DATAHANDLE, code 0)
	w.WriteHandle(hardPtrCode, h.layer0)           // CLAYER
	w.WriteHandle(hardPtrCode, h.styleStandard)    // TEXTSTYLE
	w.WriteHandle(hardPtrCode, h.ltypeByLayer)     // CELTYPE
	w.WriteHandle(hardPtrCode, h.dimstyleStandard) // DIMSTYLE
	w.WriteHandle(hardPtrCode, h.mlineStandard)    // CMLSTYLE
}

// writeHeaderUcsAndExtents writes the paper- and model-space UCS, base, extents and limits
// blocks (all neutral/origin defaults) plus their UCS-name handles and the inline DIMPOST/
// DIMAPOST strings. Mirrors readUcsDimAndInsunits' R2000 path.
//
//nolint:funlen // sequential point/handle writes in the fixed R2000 order; length is the layout.
func writeHeaderUcsAndExtents(w *BitWriter) {
	w.WriteBD(1)                  // PSVPSCALE
	writePoints(w, 3)             // PINSBASE, PEXTMIN, PEXTMAX
	w.Write2RD([2]float64{})      // PLIMMIN
	w.Write2RD([2]float64{})      // PLIMMAX
	w.WriteBD(0)                  // PELEVATION
	writePoints(w, 3)             // PUCSORG, PUCSXDIR, PUCSYDIR
	w.WriteHandle(hardPtrCode, 0) // PUCSNAME
	w.WriteHandle(hardPtrCode, 0) // PUCSORTHOREF
	w.WriteBS(0)                  // PUCSORTHOVIEW
	w.WriteHandle(hardPtrCode, 0) // PUCSBASE
	writePoints(w, 6)             // PUCSORGTOP..BACK
	writePoints(w, 3)             // INSBASE, EXTMIN, EXTMAX
	w.Write2RD([2]float64{})      // LIMMIN
	w.Write2RD([2]float64{})      // LIMMAX
	w.WriteBD(0)                  // ELEVATION
	writePoints(w, 3)             // UCSORG, UCSXDIR, UCSYDIR
	w.WriteHandle(hardPtrCode, 0) // UCSNAME
	w.WriteHandle(hardPtrCode, 0) // UCSORTHOREF
	w.WriteBS(0)                  // UCSORTHOVIEW
	w.WriteHandle(hardPtrCode, 0) // UCSBASE
	writePoints(w, 6)             // UCSORGTOP..BACK
	writeTextEmpty(w)             // DIMPOST
	writeTextEmpty(w)             // DIMAPOST
}

// writePoints writes n origin 3D points (BitDouble triples).
func writePoints(w *BitWriter, n int) {
	for range n {
		w.Write3BD([3]float64{})
	}
}

// writeHeaderDimVars writes the R2000 dimension-variable block (DIMSCALE..DIMLWE) with
// AutoCAD's default values, mirroring readDimVars' R2000 path. The five DIM block handles
// are null.
//
//nolint:funlen // one dimension variable per line in the fixed R2000 order; length is the layout.
func writeHeaderDimVars(w *BitWriter) {
	w.WriteBD(1)                                                        // DIMSCALE
	for _, v := range []float64{0.18, 0.0625, 0.38, 0.18, 0, 0, 0, 0} { // DIMASZ..DIMTM
		w.WriteBD(v)
	}
	for range 6 { // DIMTOL,DIMLIM,DIMTIH,DIMTOH,DIMSE1,DIMSE2
		w.WriteBit(0)
	}
	w.WriteBS(0)                                                      // DIMTAD
	w.WriteBS(0)                                                      // DIMZIN
	w.WriteBS(0)                                                      // DIMAZIN
	for _, v := range []float64{0.18, 0.09, 0, 25.4, 1, 0, 1, 0.09} { // DIMTXT..DIMGAP
		w.WriteBD(v)
	}
	w.WriteBD(0)                                                // DIMALTRND
	w.WriteBit(0)                                               // DIMALT
	w.WriteBS(2)                                                // DIMALTD
	w.WriteBit(1)                                               // DIMTOFL
	w.WriteBit(0)                                               // DIMSAH
	w.WriteBit(0)                                               // DIMTIX
	w.WriteBit(0)                                               // DIMSOXD
	w.WriteBS(0)                                                // DIMCLRD
	w.WriteBS(0)                                                // DIMCLRE
	w.WriteBS(0)                                                // DIMCLRT
	for _, v := range []int{0, 4, 4, 2, 3, 0, 0, 2, 46, 0, 0} { // DIMADEC..DIMJUST
		w.WriteBS(v)
	}
	w.WriteBit(0)                         // DIMSD1
	w.WriteBit(0)                         // DIMSD2
	for _, v := range []int{0, 8, 0, 0} { // DIMTOLJ,DIMTZIN,DIMALTZ,DIMALTTZ
		w.WriteBS(v)
	}
	w.WriteBit(0) // DIMUPT
	w.WriteBS(3)  // DIMATFIT
	for range 5 {
		w.WriteHandle(hardPtrCode, 0) // DIMTXSTY,DIMLDRBLK,DIMBLK,DIMBLK1,DIMBLK2 (null)
	}
	w.WriteBS(-2) // DIMLWD
	w.WriteBS(-2) // DIMLWE
}

// writeHeaderControlHandles writes the nine symbol-table control-object handles, the null
// viewport-entity-header control, the ACAD_GROUP/ACAD_MLINESTYLE/named-object dictionaries,
// the text-stack settings, the two inline strings, and the layout/plotsettings/plotstyle
// dictionaries.
func writeHeaderControlHandles(w *BitWriter, h *graphHandles) {
	for _, c := range []uint64{
		h.blockControl, h.layerControl, h.styleControl, h.ltypeControl,
		h.viewControl, h.ucsControl, h.vportControl, h.appidControl, h.dimstyleControl,
	} {
		w.WriteHandle(hardOwnerCode, c)
	}
	w.WriteHandle(hardOwnerCode, 0)                // VIEWPORT ENTITY HEADER control (null)
	w.WriteHandle(hardPtrCode, h.groupDict)        // ACAD_GROUP
	w.WriteHandle(hardPtrCode, h.mlineDict)        // ACAD_MLINESTYLE
	w.WriteHandle(hardOwnerCode, h.nod)            // NAMED OBJECTS dictionary
	w.WriteBS(1)                                   // TSTACKALIGN
	w.WriteBS(70)                                  // TSTACKSIZE
	writeTextEmpty(w)                              // HYPERLINKBASE
	writeTextEmpty(w)                              // STYLESHEET
	w.WriteHandle(hardPtrCode, h.layoutDict)       // LAYOUTS
	w.WriteHandle(hardPtrCode, h.plotSettingsDict) // PLOTSETTINGS
	w.WriteHandle(hardPtrCode, h.plotStyleDict)    // PLOTSTYLES
}

// writeHeaderTail writes the variables after INSUNITS: the plot-style name type, the two
// GUIDs, then the paper/model space block records and the three system linetype handles.
func writeHeaderTail(w *BitWriter, h *graphHandles) {
	w.WriteBS(0)                                  // CEPSNTYPE (0 → no CPSNID handle)
	writeName(w, nullGUID)                        // FINGERPRINTGUID
	writeName(w, nullGUID)                        // VERSIONGUID
	w.WriteHandle(hardPtrCode, h.paperHdr)        // *PAPER_SPACE block record
	w.WriteHandle(hardPtrCode, h.modelHdr)        // *MODEL_SPACE block record
	w.WriteHandle(hardPtrCode, h.ltypeByLayer)    // LTYPE BYLAYER
	w.WriteHandle(hardPtrCode, h.ltypeByBlock)    // LTYPE BYBLOCK
	w.WriteHandle(hardPtrCode, h.ltypeContinuous) // LTYPE CONTINUOUS
}
