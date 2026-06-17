// SPDX-License-Identifier: GPL-2.0-only

package dwg

import (
	"encoding/binary"
	"fmt"
)

// Write encodes a drawing's model-space entities as an R2000 (AC1015) DWG file. R2000 is
// the flat, uncompressed generation, so the container is a section locator over plain
// sections — far simpler to emit correctly than the paged R2004+ format while remaining a
// standard DWG other tools read. The geometry encoders are exact inverses of the
// decoders, so a value survives Decode→...→Write→Decode unchanged.
//
//	data, err := dwg.Write(&dwg.Drawing{Entities: ents, Units: 5})
func Write(d *Drawing) ([]byte, error) {
	if d == nil {
		return nil, fmt.Errorf("dwg: Write nil drawing")
	}
	return buildR2000(d.Entities, d.Units)
}

// objectTypeBS is the BitShort type code each writable entity encodes with.
func objectTypeBS(e Entity) (int, bool) {
	switch e.(type) {
	case *Line:
		return int(TypeLine), true
	case *Circle:
		return int(TypeCircle), true
	case *Arc:
		return int(TypeArc), true
	case *Point:
		return int(TypePoint), true
	case *Ellipse:
		return int(TypeEllipse), true
	case *LwPolyline:
		return int(TypeLwpolyline), true
	case *Spline:
		return int(TypeSpline), true
	default:
		return 0, false
	}
}

// bsBits returns the encoded bit length of a BitShort type code, matching WriteBS.
func bsBits(v int) int {
	switch {
	case v == 0, v == 256:
		return 2
	case v > 0 && v < 256:
		return 2 + 8
	default:
		return 2 + 16
	}
}

// writeGeometry dispatches to the per-type geometry encoder (exact inverse of decodeEntity).
//
//nolint:funlen // one-case-per-type geometry-encode dispatch (inverse of decodeEntity).
func writeGeometry(w *BitWriter, e Entity) {
	switch g := e.(type) {
	case *Line:
		writeLineGeom(w, g)
	case *Circle:
		w.Write3BD(g.Center)
		w.WriteBD(g.Radius)
		w.WriteBT(0)
		w.WriteBE(g.Normal)
	case *Arc:
		w.Write3BD(g.Center)
		w.WriteBD(g.Radius)
		w.WriteBT(0)
		w.WriteBE(g.Normal)
		w.WriteBD(g.StartAngle)
		w.WriteBD(g.EndAngle)
	case *Point:
		w.WriteBD(g.Position[0])
		w.WriteBD(g.Position[1])
		w.WriteBD(g.Position[2])
		w.WriteBT(0)
		w.WriteBE([3]float64{0, 0, 1})
		w.WriteBD(0) // x-angle
	case *Ellipse:
		w.Write3BD(g.Center)
		w.Write3BD(g.MajorAxis)
		w.Write3BD(normalOrZ(g.Normal))
		w.WriteBD(g.AxisRatio)
		w.WriteBD(g.StartAngle)
		w.WriteBD(g.EndAngle)
	case *LwPolyline:
		writeLwPolylineGeom(w, g)
	case *Spline:
		writeSplineGeom(w, g)
	}
}

// normalOrZ defaults a zero normal to +Z (a 2D entity).
func normalOrZ(n [3]float64) [3]float64 {
	if n == [3]float64{} {
		return [3]float64{0, 0, 1}
	}
	return n
}

// writeLineGeom encodes LINE: a z-is-zero flag then x/y (start raw, end as DD), optional
// z, thickness and extrusion.
func writeLineGeom(w *BitWriter, l *Line) {
	zZero := l.Start[2] == 0 && l.End[2] == 0
	if zZero {
		w.WriteBit(1)
	} else {
		w.WriteBit(0)
	}
	w.WriteRD(l.Start[0])
	w.WriteDD(l.End[0])
	w.WriteRD(l.Start[1])
	w.WriteDD(l.End[1])
	if !zZero {
		w.WriteRD(l.Start[2])
		w.WriteDD(l.End[2])
	}
	w.WriteBT(0)
	w.WriteBE([3]float64{0, 0, 1})
}

// writeLwPolylineGeom encodes LWPOLYLINE: a flag word gating the optional fields, the
// counts, the vertices (first full, rest DD deltas), then bulges.
//
//nolint:funlen // sequential flag-gated LWPOLYLINE field writes; length is the format.
func writeLwPolylineGeom(w *BitWriter, p *LwPolyline) {
	flag := 0
	if p.Closed {
		flag |= lwClosed
	}
	hasBulges := false
	for _, b := range p.Bulges {
		if b != 0 {
			hasBulges = true
			break
		}
	}
	if hasBulges {
		flag |= lwHasBulges
	}
	if p.Elevation != 0 {
		flag |= lwElevation
	}
	w.WriteBS(flag)
	if flag&lwElevation != 0 {
		w.WriteBD(p.Elevation)
	}
	w.WriteBL(len(p.Points))
	if hasBulges {
		w.WriteBL(len(p.Bulges))
	}
	if len(p.Points) > 0 {
		w.Write2RD(p.Points[0])
		for i := 1; i < len(p.Points); i++ {
			w.WriteDD(p.Points[i][0])
			w.WriteDD(p.Points[i][1])
		}
	}
	if hasBulges {
		for _, b := range p.Bulges {
			w.WriteBD(b)
		}
	}
}

// writeSplineGeom encodes SPLINE in control-point form. The knot vector is omitted
// (count 0): the importer rebuilds the curve from the control points, so the round trip
// preserves them without carrying knots.
//
//nolint:funlen // sequential SPLINE field writes; length is the format.
func writeSplineGeom(w *BitWriter, s *Spline) {
	if len(s.ControlPoints) >= 2 {
		w.WriteBL(1) // scenario: control points
		degree := s.Degree
		if degree == 0 {
			degree = 3
		}
		w.WriteBL(degree)
		w.WriteBit(boolBit(s.Rational))
		w.WriteBit(boolBit(s.Closed))
		w.WriteBit(0) // periodic
		w.WriteBD(0)  // knot tolerance
		w.WriteBD(0)  // control tolerance
		w.WriteBL(0)  // num knots (omitted; importer ignores)
		w.WriteBL(len(s.ControlPoints))
		w.WriteBit(0) // not weighted
		for _, c := range s.ControlPoints {
			w.Write3BD(c)
		}
		return
	}
	// Fit-point form fallback.
	w.WriteBL(2) // scenario: fit points
	degree := s.Degree
	if degree == 0 {
		degree = 3
	}
	w.WriteBL(degree)
	w.WriteBD(0)             // fit tolerance
	w.Write3BD([3]float64{}) // begin tangent
	w.Write3BD([3]float64{}) // end tangent
	w.WriteBL(len(s.FitPoints))
	for _, p := range s.FitPoints {
		w.Write3BD(p)
	}
}

func boolBit(b bool) uint {
	if b {
		return 1
	}
	return 0
}

// sentinelHeaderEnd is the 16-byte sentinel that closes the R2000 file header (the reader
// checks the first sentinelSignatureLen bytes).
var sentinelHeaderEnd = [16]byte{
	0x95, 0xA0, 0x4E, 0x28, 0x99, 0x82, 0x1A, 0xE5,
	0x5E, 0x41, 0xE0, 0x5F, 0x9D, 0x3A, 0x4D, 0x00,
}

// fileHeaderReserve is the bytes set aside at the start of the file for the version string,
// section-locator table, CRC and sentinel (the three records fit in well under this).
const fileHeaderReserve = 0x80

// buildR2000 lays out a complete R2000 file: the file header, the header-variables section,
// the classes section, every system object plus the model-space entities, and the object
// map. Section addresses are absolute, so the file header is filled in last once every
// section's size is known. The synthesised object graph (writegraph.go and friends) gives the
// drawing the symbol tables, block records and dictionaries AutoCAD requires.
//
//nolint:funlen // sequential file assembly: handles, sections, objects, map, locator table.
func buildR2000(entities []Entity, units int) ([]byte, error) {
	var h graphHandles
	h.allocate()
	clearDictionaryChain(&h) // not yet emitted; header/layer/blocks reference them as null

	objs, refs, err := encodeGraph(&h, entities)
	if err != nil {
		return nil, err
	}
	handseed := refs[len(refs)-1].Handle + 1

	out := make([]byte, fileHeaderReserve)
	hdrAddr := len(out)
	out = append(out, encodeHeaderVars(&h, units, handseed)...)
	classAddr := len(out)
	out = append(out, encodeClasses(nil)...)

	objBase := int64(len(out))
	for i := range refs {
		refs[i].Offset += objBase // offsets were relative to the object block start
	}
	out = append(out, objs...)

	mapAddr := len(out)
	out = append(out, encodeObjectMap(refs)...)

	sections := []SectionLocator{
		{ID: secHeaderVars, Address: int64(hdrAddr), Size: int64(classAddr - hdrAddr)},
		{ID: secClasses, Address: int64(classAddr), Size: objBase - int64(classAddr)},
		{ID: secObjectMap, Address: int64(mapAddr), Size: int64(len(out) - mapAddr)},
	}
	header := encodeFileHeader(sections)
	if len(header) > fileHeaderReserve {
		return nil, fmt.Errorf("dwg: R2000 file header %d bytes exceeds reserve %d", len(header), fileHeaderReserve)
	}
	copy(out, header)
	return out, nil
}

// clearDictionaryChain zeroes the named-object-dictionary-chain handles so the header,
// layer and block records reference them as null pointers until the chain is emitted.
func clearDictionaryChain(h *graphHandles) {
	h.groupDict, h.mlineDict, h.mlineStandard = 0, 0, 0
	h.plotStyleDict, h.placeholder = 0, 0
	h.layoutDict, h.layoutModel, h.layoutPaper, h.plotSettingsDict = 0, 0, 0, 0
}

// encodeObjectMap writes the handle→offset directory: one section of (handle delta,
// location delta) pairs followed by the terminating size-2 section.
//
//nolint:funlen // builds the object-map section and its CRC byte-by-byte; length is the format.
func encodeObjectMap(refs []ObjectRef) []byte {
	pairs := NewBitWriter()
	var lastHandle uint64
	var lastLoc int64
	for _, ref := range refs {
		pairs.WriteUMC(ref.Handle - lastHandle)
		pairs.WriteMC(int(ref.Offset - lastLoc))
		lastHandle, lastLoc = ref.Handle, ref.Offset
	}
	pairBytes := pairs.Bytes()

	section := NewBitWriter()
	size := len(pairBytes) + 2       // size counts itself (2) + the pairs
	section.WriteRC(byte(size >> 8)) // section size, big-endian
	section.WriteRC(byte(size))
	for _, by := range pairBytes {
		section.WriteRC(by)
	}
	// Section CRC: over the size field + pairs, seed 0xC0C1, stored big-endian (RS_BE) —
	// what the handles-section reader checks.
	crc := crc16(0xC0C1, section.Bytes())

	out := NewBitWriter()
	for _, by := range section.Bytes() {
		out.WriteRC(by)
	}
	out.WriteRC(byte(crc >> 8))
	out.WriteRC(byte(crc))
	// Terminating section: size 2 (no pairs), followed by its own CRC — a reader bounded by
	// the section length reads that CRC, so it must be present.
	term := []byte{0, 2}
	tcrc := crc16(0xC0C1, term)
	out.WriteRC(term[0])
	out.WriteRC(term[1])
	out.WriteRC(byte(tcrc >> 8))
	out.WriteRC(byte(tcrc))
	return out.Bytes()
}

// encodeFileHeader writes the fixed R2000 file header and the section-locator table for the
// given sections (header variables, classes, object map). The CRC is seeded 0xC0C1 over the
// header up to the CRC — the same value Decode and dwgread verify for a three-record table.
func encodeFileHeader(sections []SectionLocator) []byte {
	buf := make([]byte, 0x19)
	copy(buf, "AC1015")
	buf[0x0C] = 0x00
	binary.LittleEndian.PutUint16(buf[0x13:], 30) // codepage ANSI_1252
	binary.LittleEndian.PutUint32(buf[0x15:], uint32(len(sections)))
	for _, r := range sections {
		rec := make([]byte, 9)
		rec[0] = r.ID
		binary.LittleEndian.PutUint32(rec[1:], uint32(r.Address))
		binary.LittleEndian.PutUint32(rec[5:], uint32(r.Size))
		buf = append(buf, rec...)
	}
	crc := crc16(0xC0C1, buf)
	buf = binary.LittleEndian.AppendUint16(buf, crc)
	buf = append(buf, sentinelHeaderEnd[:]...)
	return buf
}
