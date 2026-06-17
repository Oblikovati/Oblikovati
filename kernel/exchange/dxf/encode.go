// SPDX-License-Identifier: GPL-2.0-only

package dxf

import (
	"strconv"
	"strings"

	"oblikovati.org/kernel/exchange/drawing"
)

// Encode writes a drawing's model-space entities as an ASCII DXF file of the given version
// (R2000 or R2018). It emits the standard section set AutoCAD expects — HEADER, TABLES,
// BLOCKS, ENTITIES, OBJECTS — so the file opens without repair, not just the bare ENTITIES.
// The geometry encoders are the inverse of the decoders, so a value survives
// Decode→…→Encode→Decode unchanged.
//
//	data, err := dxf.Encode(dr, dxf.R2000)
func Encode(dr *drawing.Drawing, version Version) ([]byte, error) {
	h := newEncHandles()
	for range dr.Entities {
		h.alloc() // reserve an entity handle per entity (assigned in order on write)
	}
	w := &tagWriter{}
	writeHeader(w, version, dr.Units, h.next+1)
	writeTables(w, h)
	writeBlocks(w, h)
	writeEntitiesSection(w, dr.Entities, h)
	writeObjects(w, h)
	w.tag(0, "EOF")
	return []byte(w.string()), nil
}

// writeHeader emits the HEADER section with the variables a reader needs: the version, the
// next-free handle, and the unit code. Other variables default in the reader.
func writeHeader(w *tagWriter, version Version, insunits int, handseed uint64) {
	w.tag(0, "SECTION")
	w.tag(2, "HEADER")
	w.variable(9, "$ACADVER", 1, version.ACADVer())
	w.tag(9, "$HANDSEED")
	w.handle(5, handseed)
	w.variable(9, "$INSUNITS", 70, strconv.Itoa(insunits))
	w.tag(0, "ENDSEC")
}

// writeEntitiesSection emits the ENTITIES section: every model-space entity, owned by the
// *Model_Space block record. Entity handles are drawn from the reserved range in order.
func writeEntitiesSection(w *tagWriter, entities []drawing.Entity, h *encHandles) {
	w.tag(0, "SECTION")
	w.tag(2, "ENTITIES")
	handle := h.entityBase
	for _, e := range entities {
		encodeEntity(w, e, handle, h.modelSpaceBR)
		handle++
	}
	w.tag(0, "ENDSEC")
}

// tagWriter accumulates DXF group-code/value lines.
type tagWriter struct {
	b strings.Builder
}

// tag writes one group-code/value pair (code line, then value line).
func (w *tagWriter) tag(code int, value string) {
	w.b.WriteString(strconv.Itoa(code))
	w.b.WriteByte('\n')
	w.b.WriteString(value)
	w.b.WriteByte('\n')
}

// integer writes an integer-valued group code.
func (w *tagWriter) integer(code, v int) { w.tag(code, strconv.Itoa(v)) }

// real writes a real-valued group code, always with a decimal point so it reads back as a
// float (and matches the DXF convention "10.0" rather than "10").
func (w *tagWriter) real(code int, v float64) { w.tag(code, formatFloat(v)) }

// handle writes a hex handle (uppercase, no leading zeros — the AutoCAD convention).
func (w *tagWriter) handle(code int, h uint64) {
	w.tag(code, strings.ToUpper(strconv.FormatUint(h, 16)))
}

// coord writes a 3D coordinate as the X/Y/Z group-code triple cx/cx+10/cx+20.
func (w *tagWriter) coord(cx int, v [3]float64) {
	w.real(cx, v[0])
	w.real(cx+10, v[1])
	w.real(cx+20, v[2])
}

// variable writes a HEADER variable: its code-9 name then its single value pair.
func (w *tagWriter) variable(nameCode int, name string, valCode int, value string) {
	w.tag(nameCode, name)
	w.tag(valCode, value)
}

func (w *tagWriter) string() string { return w.b.String() }

// formatFloat renders a float for DXF: shortest round-trippable form, but always carrying a
// decimal point (or exponent) so integers read back as reals.
func formatFloat(v float64) string {
	s := strconv.FormatFloat(v, 'g', -1, 64)
	if !strings.ContainsAny(s, ".eE") {
		s += ".0"
	}
	return s
}
