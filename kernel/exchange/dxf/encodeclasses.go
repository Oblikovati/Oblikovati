// SPDX-License-Identifier: GPL-2.0-only

package dxf

// stdClass is one entry of the CLASSES section: a class the drawing may instantiate, named
// by its DXF record name and C++ class name. Class numbers are assigned by position (≥500)
// when the file is read, so listing the standard classes keeps the class table valid.
type stdClass struct {
	dxfName  string
	cppName  string
	isEntity bool
}

// standardClasses are the classes AutoCAD always registers. We declare them with zero
// instances (the named-object dictionary entries that would use them are a follow-up); their
// presence is what a complete file carries, and it gives the class table the non-empty,
// ≥500-numbered range R2018 readers expect.
var standardClasses = []stdClass{
	{"ACDBDICTIONARYWDFLT", "AcDbDictionaryWithDefault", false},
	{"ACDBPLACEHOLDER", "AcDbPlaceHolder", false},
	{"LAYOUT", "AcDbLayout", false},
}

// writeClasses emits the CLASSES section. R2000 readers tolerate its absence, but the R2018
// class table must be non-empty (an empty one leaves the max class number at the invalid
// 499), so the section is written for R2018.
func writeClasses(w *tagWriter, version Version) {
	if version != R2018 {
		return
	}
	w.tag(0, "SECTION")
	w.tag(2, "CLASSES")
	for _, c := range standardClasses {
		writeClass(w, c)
	}
	w.tag(0, "ENDSEC")
}

// writeClass writes one CLASS record.
func writeClass(w *tagWriter, c stdClass) {
	w.tag(0, "CLASS")
	w.tag(1, c.dxfName)
	w.tag(2, c.cppName)
	w.tag(3, "ObjectDBX Classes")
	w.integer(90, 0) // class flags
	w.integer(91, 0) // instance count
	w.integer(280, 0)
	w.integer(281, boolFlag(c.isEntity))
}

// boolFlag renders a bool as the 0/1 a DXF integer flag uses.
func boolFlag(b bool) int {
	if b {
		return 1
	}
	return 0
}
