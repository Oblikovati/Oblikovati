// SPDX-License-Identifier: GPL-2.0-only

package dwg

// writelayouts.go encodes the named-object-dictionary chain AutoCAD expects under the NOD:
// the ACAD_GROUP/ACAD_MLINESTYLE/ACAD_PLOTSETTINGS/ACAD_PLOTSTYLENAME/ACAD_LAYOUT
// dictionaries and the objects they own — the STANDARD multiline style, the "Normal"
// plot-style placeholder, and the Model/Layout1 paper layouts (each embedding a default
// PLOTSETTINGS block). These are the non-fixed (class-resolved) object types, so the writer
// also declares their classes (writeclasses.go). Field layouts mirror the oracle's R2000
// decode.

// Dynamic class numbers for the non-fixed objects the chain emits. They index the class list
// emitted in this same order (class 0 → 500), so writeclasses.go must list them identically.
const (
	classDictWDflt   = 500 // ACDBDICTIONARYWDFLT (the plot-style-name dictionary)
	classPlaceholder = 501 // ACDBPLACEHOLDER (a plot-style entry)
	classLayout      = 502 // LAYOUT
)

// chainClasses are the class definitions for the dictionary-chain objects, in the order their
// type numbers assume (classDictWDflt first → 500).
func chainClasses() []classDef {
	return []classDef{
		{num: classDictWDflt, dxfName: "ACDBDICTIONARYWDFLT", cppName: "AcDbDictionaryWithDefault", isObject: true},
		{num: classPlaceholder, dxfName: "ACDBPLACEHOLDER", cppName: "AcDbPlaceHolder", isObject: true},
		{num: classLayout, dxfName: "LAYOUT", cppName: "AcDbLayout", isObject: true},
	}
}

// writeSubDictionary frames a DICTIONARY owned by another dictionary (e.g. ACAD_MLINESTYLE
// under the NOD). It is writeDictionary with a non-root owner.
func writeSubDictionary(handle, owner uint64, entries []dictEntry) []byte {
	return writeDictionary(handle, owner, entries)
}

// writeDictionaryWDflt frames an ACDBDICTIONARYWDFLT (the plot-style-name dictionary): a
// DICTIONARY plus a trailing default-entry hard pointer. Used for ACAD_PLOTSTYLENAME, whose
// default is the "Normal" placeholder.
func writeDictionaryWDflt(handle, owner uint64, entries []dictEntry, defaultID uint64) []byte {
	b := newObjectBody(handle, classDictWDflt)
	b.data.WriteBL(0)            // numreactors
	b.data.WriteBL(len(entries)) // numitems
	b.data.WriteBS(1)            // cloning flag
	b.data.WriteRC(0)            // hard-owner flag
	for _, e := range entries {
		writeName(b.data, e.name)
	}
	b.handles.WriteHandle(softPtrCode, owner)
	b.handles.WriteHandle(hardOwnerCode, 0) // xdic
	for _, e := range entries {
		b.handles.WriteHandle(softOwnerCode, e.handle)
	}
	b.handles.WriteHandle(hardPtrCode, defaultID) // default entry
	return frameObject(b)
}

// writePlaceholder frames an ACDBPLACEHOLDER ("Normal" plot style): a common object header
// only, owned by the plot-style-name dictionary.
func writePlaceholder(handle, owner uint64) []byte {
	b := newObjectBody(handle, classPlaceholder)
	b.data.WriteBL(0) // numreactors
	b.handles.WriteHandle(softPtrCode, owner)
	b.handles.WriteHandle(hardOwnerCode, 0) // xdic
	return frameObject(b)
}

// writeMlinestyle frames the STANDARD MLINESTYLE: two element lines at ±0.5 offset, ByLayer
// colour, the default linetype index. MLINESTYLE is a fixed type (0x49).
func writeMlinestyle(handle, owner uint64) []byte {
	b := newObjectBody(handle, 0x49)
	b.data.WriteBL(0)                  // numreactors
	writeName(b.data, "Standard")      // name
	writeTextEmpty(b.data)             // description
	b.data.WriteBS(0)                  // flags
	b.data.WriteBS(colorByLayer)       // fill colour
	b.data.WriteBD(1.5707963267948966) // start angle (90°)
	b.data.WriteBD(1.5707963267948966) // end angle
	b.data.WriteRC(2)                  // num lines
	for _, off := range []float64{0.5, -0.5} {
		b.data.WriteBD(off)          // element offset
		b.data.WriteBS(colorByLayer) // element colour
		b.data.WriteBS(32767)        // element linetype index (BYLAYER sentinel)
	}
	b.handles.WriteHandle(softPtrCode, owner)
	b.handles.WriteHandle(hardOwnerCode, 0) // xdic
	return frameObject(b)
}

// layoutRefs carries the cross-references a LAYOUT needs: its owning dictionary and the block
// record (paper or model space) it controls.
type layoutRefs struct {
	owner, blockHeader uint64
	name               string
	tabOrder, flags    int
}

// writeLayout frames a LAYOUT object: a default PLOTSETTINGS block followed by the layout's
// name, tab order, flags and UCS/extent defaults, then the handle stream (owner dictionary,
// the controlled block record, and null viewport/UCS handles). Field order and bit types
// mirror the oracle's R2000 decode.
//
//nolint:funlen // one PLOTSETTINGS/LAYOUT field per line in the fixed R2000 order.
func writeLayout(handle uint64, r layoutRefs) []byte {
	b := newObjectBody(handle, classLayout)
	b.data.WriteBL(0) // numreactors
	writePlotSettings(b.data)
	writeName(b.data, r.name)            // layout name
	b.data.WriteBS(r.tabOrder)           // tab order
	b.data.WriteBS(r.flags)              // layout flags
	b.data.Write3BD([3]float64{})        // INSBASE
	b.data.Write2RD([2]float64{0, 0})    // LIMMIN
	b.data.Write2RD([2]float64{12, 9})   // LIMMAX
	b.data.Write3BD([3]float64{})        // UCSORG
	b.data.Write3BD([3]float64{1, 0, 0}) // UCSXDIR
	b.data.Write3BD([3]float64{0, 1, 0}) // UCSYDIR
	b.data.WriteBD(0)                    // ucs elevation
	b.data.WriteBS(0)                    // UCSORTHOVIEW
	b.data.Write3BD([3]float64{})        // EXTMIN
	b.data.Write3BD([3]float64{})        // EXTMAX

	b.handles.WriteHandle(softPtrCode, r.owner)
	b.handles.WriteHandle(hardOwnerCode, 0)           // xdic
	b.handles.WriteHandle(softPtrCode, r.blockHeader) // associated block record
	b.handles.WriteHandle(softPtrCode, 0)             // active viewport (null)
	b.handles.WriteHandle(hardPtrCode, 0)             // base UCS (null)
	b.handles.WriteHandle(hardPtrCode, 0)             // named UCS (null)
	return frameObject(b)
}

// writePlotSettings writes the embedded PLOTSETTINGS block with the default "none_device"
// page setup (ODA §20.4 PLOTSETTINGS; matches the oracle's field order/types).
//
//nolint:funlen // one PLOTSETTINGS field per line in the fixed R2000 order; length is the layout.
func writePlotSettings(w *BitWriter) {
	writeTextEmpty(w)                            // page setup / printer config file
	writeName(w, "none_device")                  // plot device name
	w.WriteBS(0x2eb0)                            // plot layout flags
	w.WriteBD(6.35)                              // left margin
	w.WriteBD(19.05)                             // bottom margin
	w.WriteBD(6.35)                              // right margin
	w.WriteBD(19.05)                             // top margin
	w.WriteBD(215.9)                             // paper width (mm)
	w.WriteBD(279.4)                             // paper height (mm)
	writeName(w, "ANSI_A_(8.50_x_11.00_Inches)") // canonical media name
	w.WriteBD(0)                                 // plot origin x
	w.WriteBD(0)                                 // plot origin y
	w.WriteBS(0)                                 // plot paper units
	w.WriteBS(0)                                 // plot rotation
	w.WriteBS(0)                                 // plot type
	w.WriteBD(0)                                 // plot window LL x
	w.WriteBD(0)                                 // plot window LL y
	w.WriteBD(0)                                 // plot window UR x
	w.WriteBD(0)                                 // plot window UR y
	writeTextEmpty(w)                            // plot view name
	w.WriteBD(1)                                 // custom-scale numerator
	w.WriteBD(1)                                 // custom-scale denominator
	writeTextEmpty(w)                            // current style sheet
	w.WriteBS(0)                                 // standard scale type
	w.WriteBD(1)                                 // scale factor
	w.WriteBD(0)                                 // paper image origin x
	w.WriteBD(0)                                 // paper image origin y
}
