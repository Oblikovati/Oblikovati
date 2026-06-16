// SPDX-License-Identifier: GPL-2.0-only

package dwg

import (
	"bytes"
	"fmt"
)

// HeaderVars holds the drawing-header variables relevant to import — chiefly the units
// and scale metadata used to bring DWG coordinates into the document's unit system, plus
// the extents/limits and current settings that contextualise them. It is decoded from the
// "AcDb:Header" section (ODA header_variables.spec) by reading the variable list in order;
// values not needed by the importer are still consumed (in sequence) so the cursor reaches
// the wanted ones, but only the meaningful fields are retained here.
//
// Units: INSUNITS is the authoritative unit code (see MetersPerUnit); the unit*Ratio
// fields are the header's legacy conversion ratios.
type HeaderVars struct {
	RequiredVersions uint64

	Unit1Ratio, Unit2Ratio, Unit3Ratio, Unit4Ratio float64

	// Display/precision settings.
	LUnits, LUPrec, AUnits, AUPrec int

	// Scale and size defaults (drawing units).
	LTScale, TextSize, TraceWidth, FilletRad, Thickness, AngBase float64
	DimScale                                                     float64

	// Model-space extents and limits (drawing units), useful for sanity/extent checks.
	InsBase, ExtMin, ExtMax [3]float64
	LimMin, LimMax          [2]float64
	Elevation               float64

	// Units / scale metadata (the import-critical fields).
	INSUNITS int // unit code, 0 = unitless; see MetersPerUnit
}

// metersPerInsunit maps the ODA $INSUNITS code to metres per drawing unit. Code 0 is
// unitless (no intrinsic scale); unsupported/astronomical codes are omitted (ok=false),
// leaving the importer to fall back to the document's unit.
var metersPerInsunit = map[int]float64{
	1: 0.0254, 2: 0.3048, 3: 1609.344, 4: 0.001, 5: 0.01, 6: 1, 7: 1000,
	8: 0.0254e-6, 9: 0.0254e-3, 10: 0.9144, 11: 1e-10, 12: 1e-9, 13: 1e-6,
	14: 0.1, 15: 10, 16: 100, 17: 1e9,
	21: 0.30480061, 22: 0.0254000508, 23: 0.91440183, 24: 1609.347219,
}

// MetersPerUnit returns the length, in metres, of one drawing unit for the given
// $INSUNITS code, and whether the code carries a known unit. Unitless (0) and unknown
// codes return ok=false.
//
//	m, ok := dwg.MetersPerUnit(4) // 0.001, true (millimetres)
func MetersPerUnit(insunits int) (float64, bool) {
	m, ok := metersPerInsunit[insunits]
	return m, ok
}

// vbeginSentinel marks the start of the header-variable block (ODA
// DWG_SENTINEL_VARIABLE_BEGIN); the RL byte-size follows it, then the variables.
var vbeginSentinel = []byte{
	0xCF, 0x7B, 0x1F, 0x23, 0xFD, 0xDE, 0x38, 0xA9,
	0x5F, 0x7C, 0x68, 0xB8, 0x4E, 0x6D, 0x33, 0x5F,
}

// ParseHeaderVars decodes the header variables from the "AcDb:Header" section bytes. For
// R2007+ the section has three interleaved streams (data, handles, strings); the importer
// needs only the numeric data-stream fields, so handle and string variables are noted but
// not read here (they live in the other streams and do not advance the data cursor). R2000
// keeps everything in one stream and is handled by readHeaderVarsR2000.
func ParseHeaderVars(section []byte, version Version) (*HeaderVars, error) {
	at := bytes.Index(section, vbeginSentinel)
	if at < 0 {
		return nil, fmt.Errorf("dwg: header variable-begin sentinel not found (section %d bytes)", len(section))
	}
	r := NewBitReaderAt(section, (at+len(vbeginSentinel))*8)
	r.ReadRL() // size of the variable block in bytes (not needed; we read by field)
	hv := &HeaderVars{}
	if version >= R2007 {
		readHeaderVarsR2010(r, hv, version)
	} else {
		readHeaderVarsR2000(r, hv, version)
	}
	if err := r.Err(); err != nil {
		return nil, fmt.Errorf("dwg: header variables: %w", err)
	}
	return hv, nil
}

// readHeaderVarsR2010 reads the data-stream header variables for R2007+ (validated on
// R2018) in ODA spec order. Handle-stream (FIELD_HANDLE) and string-stream (FIELD_TV)
// variables are intentionally skipped: in this generation they are stored in separate
// streams and so do not appear in the data stream being read here.
func readHeaderVarsR2010(r *BitReader, hv *HeaderVars, version Version) {
	if version >= R2013 {
		hv.RequiredVersions = r.ReadBLL()
	}
	hv.Unit1Ratio, hv.Unit2Ratio = r.ReadBD(), r.ReadBD()
	hv.Unit3Ratio, hv.Unit4Ratio = r.ReadBD(), r.ReadBD()
	// unit*_name are TV (string stream) — skipped.
	r.ReadBL() // unknown_8 (BLx)
	r.ReadBL() // unknown_9
	readModeFlags(r, version)
	readUnitsAndCounts(r, hv, version)
	readSizesAndTimes(r, hv, version)
	hv.INSUNITS = readUcsDimAndInsunits(r, hv, version)
}

// readModeFlags consumes the run of boolean mode variables (DIMASO..PELLIPSE).
func readModeFlags(r *BitReader, version Version) {
	for i := 0; i < 2; i++ { // DIMASO, DIMSHO
		r.ReadBit()
	}
	for i := 0; i < 7; i++ { // PLINEGEN, ORTHOMODE, REGENMODE, FILLMODE, QTEXTMODE, PSLTSCALE, LIMCHECK
		r.ReadBit()
	}
	if version >= R2004 {
		r.ReadBit() // unknown_11
	}
	for i := 0; i < 4; i++ { // USRTIMER, SKPOLY, ANGDIR, SPLFRAME
		r.ReadBit()
	}
	for i := 0; i < 2; i++ { // MIRRTEXT, WORLDVIEW
		r.ReadBit()
	}
	for i := 0; i < 3; i++ { // TILEMODE, PLIMCHECK, VISRETAIN
		r.ReadBit()
	}
	r.ReadBit() // DISPSILH
	r.ReadBit() // PELLIPSE
}

// readUnitsAndCounts consumes PROXYGRAPHICS..TEXTQLTY, capturing the linear/angular unit
// settings, and the integer count variables in between.
func readUnitsAndCounts(r *BitReader, hv *HeaderVars, version Version) {
	r.ReadBS() // PROXYGRAPHICS
	r.ReadBS() // TREEDEPTH
	hv.LUnits, hv.LUPrec = r.ReadBS(), r.ReadBS()
	hv.AUnits, hv.AUPrec = r.ReadBS(), r.ReadBS()
	r.ReadBS() // ATTMODE
	r.ReadBS() // PDMODE
	if version >= R2004 {
		r.ReadBL() // unknown_12
		r.ReadBL() // unknown_13
		r.ReadBL() // unknown_14
	}
	for i := 0; i < 5; i++ { // USERI1..USERI5
		r.ReadBS()
	}
	for i := 0; i < 14; i++ { // SPLINESEGS..TEXTQLTY
		r.ReadBS()
	}
}

// readSizesAndTimes consumes the real-valued size defaults (LTSCALE..CELTSCALE), the
// creation/update/usage timers, and CECOLOR, capturing the import-relevant scales.
//
//nolint:funlen // sequential header-variable reads in spec order; length is the field layout.
func readSizesAndTimes(r *BitReader, hv *HeaderVars, version Version) {
	hv.LTScale = r.ReadBD()
	hv.TextSize = r.ReadBD()
	hv.TraceWidth = r.ReadBD()
	r.ReadBD() // SKETCHINC
	hv.FilletRad = r.ReadBD()
	hv.Thickness = r.ReadBD()
	hv.AngBase = r.ReadBD()
	r.ReadBD()               // PDSIZE
	r.ReadBD()               // PLINEWID
	for i := 0; i < 5; i++ { // USERR1..USERR5
		r.ReadBD()
	}
	for i := 0; i < 4; i++ { // CHAMFERA..CHAMFERD
		r.ReadBD()
	}
	r.ReadBD()     // FACETRES
	r.ReadBD()     // CMLSCALE
	r.ReadBD()     // CELTSCALE
	readTimeBLL(r) // TDUCREATE
	readTimeBLL(r) // TDUUPDATE
	if version >= R2004 {
		r.ReadBL() // unknown_15
		r.ReadBL() // unknown_16
		r.ReadBL() // unknown_17
	}
	readTimeBLL(r) // TDINDWG
	readTimeBLL(r) // TDUSRTIMER
	readCMC(r)     // CECOLOR
	r.ReadHandle() // HANDSEED is a DATAHANDLE — stored inline in the data stream (unlike the
	// following current layer/style/linetype/material/dimstyle/mlstyle handles, which live in
	// the handle stream and are skipped here).
}

// readUcsDimAndInsunits consumes the paper/model UCS and extents, the dimension-variable
// block, and finally FLAGS, returning the INSUNITS value that immediately follows. The
// model-space extents/limits are captured along the way.
//
//nolint:funlen // sequential header-variable reads in spec order; length is the field layout.
func readUcsDimAndInsunits(r *BitReader, hv *HeaderVars, version Version) int {
	if version >= R2000 {
		r.ReadBD() // PSVPSCALE
	}
	// Paper-space UCS / extents (PINSBASE..PUCSYDIR, then PUCSNAME handle skipped).
	r.Read3BD() // PINSBASE
	r.Read3BD() // PEXTMIN
	r.Read3BD() // PEXTMAX
	r.Read2RD() // PLIMMIN
	r.Read2RD() // PLIMMAX
	r.ReadBD()  // PELEVATION
	r.Read3BD() // PUCSORG
	r.Read3BD() // PUCSXDIR
	r.Read3BD() // PUCSYDIR
	if version >= R2000 {
		r.ReadBS()     // PUCSORTHOVIEW (PUCSORTHOREF handle skipped before, PUCSBASE handle skipped after)
		read3BDn(r, 6) // PUCSORGTOP..PUCSORGBACK
	}
	// Model-space base/extents/limits — the import-relevant geometry context.
	hv.InsBase = r.Read3BD()
	hv.ExtMin = r.Read3BD()
	hv.ExtMax = r.Read3BD()
	hv.LimMin = r.Read2RD()
	hv.LimMax = r.Read2RD()
	hv.Elevation = r.ReadBD()
	r.Read3BD() // UCSORG
	r.Read3BD() // UCSXDIR
	r.Read3BD() // UCSYDIR
	if version >= R2000 {
		r.ReadBS()     // UCSORTHOVIEW
		read3BDn(r, 6) // UCSORGTOP..UCSORGBACK
	}
	readDimVars(r, hv, version)
	r.ReadBL() // FLAGS (BLx)
	return r.ReadBS()
}

// readDimVars consumes the dimension-style variable block (DIMSCALE onward) up to FLAGS,
// capturing DIMSCALE. The pre-R2000 layout (a different bool/BS ordering) is gated out
// here because the supported generations are R2000+.
//
//nolint:funlen,gocyclo // sequential version-gated dimension-variable reads; length/branches are the format.
func readDimVars(r *BitReader, hv *HeaderVars, version Version) {
	hv.DimScale = r.ReadBD()
	for i := 0; i < 8; i++ { // DIMASZ, DIMEXO, DIMDLI, DIMEXE, DIMRND, DIMDLE, DIMTP, DIMTM
		r.ReadBD()
	}
	if version >= R2007 {
		r.ReadBD() // DIMFXL
		r.ReadBD() // DIMJOGANG
		r.ReadBS() // DIMTFILL
		readCMC(r) // DIMTFILLCLR
	}
	if version >= R2000 {
		for i := 0; i < 6; i++ { // DIMTOL, DIMLIM, DIMTIH, DIMTOH, DIMSE1, DIMSE2
			r.ReadBit()
		}
		r.ReadBS() // DIMTAD
		r.ReadBS() // DIMZIN
		r.ReadBS() // DIMAZIN
	}
	if version >= R2007 {
		r.ReadBS() // DIMARCSYM
	}
	for i := 0; i < 8; i++ { // DIMTXT, DIMCEN, DIMTSZ, DIMALTF, DIMLFAC, DIMTVP, DIMTFAC, DIMGAP
		r.ReadBD()
	}
	if version >= R2000 {
		r.ReadBD()  // DIMALTRND
		r.ReadBit() // DIMALT
		r.ReadBS()  // DIMALTD
		r.ReadBit() // DIMTOFL
		r.ReadBit() // DIMSAH
		r.ReadBit() // DIMTIX
		r.ReadBit() // DIMSOXD
	}
	readCMC(r) // DIMCLRD
	readCMC(r) // DIMCLRE
	readCMC(r) // DIMCLRT
	if version >= R2000 {
		for i := 0; i < 11; i++ { // DIMADEC, DIMDEC, DIMTDEC, DIMALTU, DIMALTTD, DIMAUNIT, DIMFRAC, DIMLUNIT, DIMDSEP, DIMTMOVE, DIMJUST
			r.ReadBS()
		}
		r.ReadBit()              // DIMSD1
		r.ReadBit()              // DIMSD2
		for i := 0; i < 4; i++ { // DIMTOLJ, DIMTZIN, DIMALTZ, DIMALTTZ
			r.ReadBS()
		}
		r.ReadBit() // DIMUPT
		r.ReadBS()  // DIMATFIT
	}
	if version >= R2007 {
		r.ReadBit() // DIMFXLON
	}
	if version >= R2010 {
		r.ReadBit() // DIMTXTDIRECTION
		r.ReadBD()  // DIMALTMZF
		r.ReadBD()  // DIMMZF
	}
	// DIMTXSTY..DIMLTEX2 are handles — skipped.
	if version >= R2000 {
		r.ReadBS() // DIMLWD
		r.ReadBS() // DIMLWE
	}
	// Control-object and dictionary handles are in the handle stream — skipped.
	if version >= R2000 {
		r.ReadBS() // TSTACKALIGN
		r.ReadBS() // TSTACKSIZE
	}
	// HYPERLINKBASE/STYLESHEET (TV) and the remaining dictionary handles are in the
	// string/handle streams — skipped; the caller reads FLAGS next.
}

// readHeaderVarsR2000 reads the header variables for R2000, where data, handles, and
// strings share one stream (so TV strings and handle references must be consumed inline
// to stay in sync). Implemented separately from the R2007+ path because of that
// interleaving.
//
//nolint:funlen // the R2000 header is one long inline field sequence; length is the format, not logic.
func readHeaderVarsR2000(r *BitReader, hv *HeaderVars, version Version) {
	hv.Unit1Ratio, hv.Unit2Ratio = r.ReadBD(), r.ReadBD()
	hv.Unit3Ratio, hv.Unit4Ratio = r.ReadBD(), r.ReadBD()
	readTV(r)      // unit1_name
	readTV(r)      // unit2_name
	readTV(r)      // unit3_name
	readTV(r)      // unit4_name
	r.ReadBL()     // unknown_8 (BLx)
	r.ReadBL()     // unknown_9
	r.ReadHandle() // VX_TABLE_RECORD (VERSIONS R13..R2000)
	readModeFlags(r, version)
	readUnitsAndCounts(r, hv, version)

	// Sizes (same BD run as later versions) then MENU (TV, inline pre-R2007).
	hv.LTScale = r.ReadBD()
	hv.TextSize = r.ReadBD()
	hv.TraceWidth = r.ReadBD()
	r.ReadBD() // SKETCHINC
	hv.FilletRad = r.ReadBD()
	hv.Thickness = r.ReadBD()
	hv.AngBase = r.ReadBD()
	r.ReadBD() // PDSIZE
	r.ReadBD() // PLINEWID
	read3BDn(r, 0)
	for i := 0; i < 5; i++ { // USERR1..USERR5
		r.ReadBD()
	}
	for i := 0; i < 4; i++ { // CHAMFERA..CHAMFERD
		r.ReadBD()
	}
	r.ReadBD() // FACETRES
	r.ReadBD() // CMLSCALE
	r.ReadBD() // CELTSCALE
	readTV(r)  // MENU (PRE R2007)
	readTimeBLL(r)
	readTimeBLL(r)
	readTimeBLL(r)
	readTimeBLL(r)
	r.ReadBS()     // CECOLOR (CMC pre-R2004 = index only)
	r.ReadHandle() // HANDSEED
	r.ReadHandle() // CLAYER
	r.ReadHandle() // TEXTSTYLE
	r.ReadHandle() // CELTYPE
	r.ReadHandle() // DIMSTYLE
	r.ReadHandle() // CMLSTYLE
	r.ReadBD()     // PSVPSCALE (SINCE R2000b)
	read3BDn(r, 3) // PINSBASE, PEXTMIN, PEXTMAX
	r.Read2RD()    // PLIMMIN
	r.Read2RD()    // PLIMMAX
	r.ReadBD()     // PELEVATION
	read3BDn(r, 3) // PUCSORG, PUCSXDIR, PUCSYDIR
	r.ReadHandle() // PUCSNAME
	r.ReadHandle() // PUCSORTHOREF (SINCE R2000b)
	r.ReadBS()     // PUCSORTHOVIEW
	r.ReadHandle() // PUCSBASE
	read3BDn(r, 6) // PUCSORGTOP..BACK
	hv.InsBase = r.Read3BD()
	hv.ExtMin = r.Read3BD()
	hv.ExtMax = r.Read3BD()
	hv.LimMin = r.Read2RD()
	hv.LimMax = r.Read2RD()
	hv.Elevation = r.ReadBD()
	read3BDn(r, 3) // UCSORG, UCSXDIR, UCSYDIR
	r.ReadHandle() // UCSNAME
	r.ReadHandle() // UCSORTHOREF
	r.ReadBS()     // UCSORTHOVIEW
	r.ReadHandle() // UCSBASE
	read3BDn(r, 6) // UCSORGTOP..BACK
	readTV(r)      // DIMPOST (PRE R2007, inline)
	readTV(r)      // DIMAPOST

	hv.DimScale = r.ReadBD()
	for i := 0; i < 8; i++ { // DIMASZ..DIMTM
		r.ReadBD()
	}
	for i := 0; i < 6; i++ { // DIMTOL, DIMLIM, DIMTIH, DIMTOH, DIMSE1, DIMSE2
		r.ReadBit()
	}
	r.ReadBS()               // DIMTAD
	r.ReadBS()               // DIMZIN
	r.ReadBS()               // DIMAZIN
	for i := 0; i < 8; i++ { // DIMTXT..DIMGAP
		r.ReadBD()
	}
	r.ReadBD()                // DIMALTRND
	r.ReadBit()               // DIMALT
	r.ReadBS()                // DIMALTD
	r.ReadBit()               // DIMTOFL
	r.ReadBit()               // DIMSAH
	r.ReadBit()               // DIMTIX
	r.ReadBit()               // DIMSOXD
	r.ReadBS()                // DIMCLRD (index only)
	r.ReadBS()                // DIMCLRE
	r.ReadBS()                // DIMCLRT
	for i := 0; i < 11; i++ { // DIMADEC..DIMJUST
		r.ReadBS()
	}
	r.ReadBit()              // DIMSD1
	r.ReadBit()              // DIMSD2
	for i := 0; i < 4; i++ { // DIMTOLJ, DIMTZIN, DIMALTZ, DIMALTTZ
		r.ReadBS()
	}
	r.ReadBit()              // DIMUPT
	r.ReadBS()               // DIMATFIT
	r.ReadHandle()           // DIMTXSTY
	r.ReadHandle()           // DIMLDRBLK
	r.ReadHandle()           // DIMBLK
	r.ReadHandle()           // DIMBLK1
	r.ReadHandle()           // DIMBLK2
	r.ReadBS()               // DIMLWD
	r.ReadBS()               // DIMLWE
	for i := 0; i < 9; i++ { // BLOCK..DIMSTYLE control objects
		r.ReadHandle()
	}
	r.ReadHandle() // VX_CONTROL_OBJECT (VERSIONS R13..R2000)
	r.ReadHandle() // DICTIONARY_ACAD_GROUP
	r.ReadHandle() // DICTIONARY_ACAD_MLINESTYLE
	r.ReadHandle() // DICTIONARY_NAMED_OBJECT
	r.ReadBS()     // TSTACKALIGN
	r.ReadBS()     // TSTACKSIZE
	readTV(r)      // HYPERLINKBASE
	readTV(r)      // STYLESHEET
	r.ReadHandle() // DICTIONARY_LAYOUT
	r.ReadHandle() // DICTIONARY_PLOTSETTINGS
	r.ReadHandle() // DICTIONARY_PLOTSTYLENAME
	r.ReadBL()     // FLAGS
	hv.INSUNITS = r.ReadBS()
}

// readTV consumes a pre-R2007 text variable: a BitShort length then that many bytes.
func readTV(r *BitReader) {
	n := r.ReadBS()
	for i := 0; i < n; i++ {
		r.ReadRC()
	}
}

// readTimeBLL consumes a TIMEBLL (two BitLongs: julian day and milliseconds).
func readTimeBLL(r *BitReader) { r.ReadBL(); r.ReadBL() }

// readCMC consumes an R2004+ entity colour value's data-stream part: a BitShort index, a
// BitLong RGB, and an RC flag (its optional name/book strings live in the string stream).
func readCMC(r *BitReader) { r.ReadBS(); r.ReadBL(); r.ReadRC() }

// read3BDn consumes n consecutive 3D points.
func read3BDn(r *BitReader, n int) {
	for i := 0; i < n; i++ {
		r.Read3BD()
	}
}
