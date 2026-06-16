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
	b := newR2000Builder()
	return b.build(d.Entities)
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

// encodeEntityObject lays out one R2000 object: the MS byte size, then the data stream
// (BS type, RL bitsize, the object's own handle, empty EED, common entity data, geometry),
// then the handle stream and the object CRC. bitsize — the bit offset from just after the
// MS size to the handle stream — is computed from the pre-encoded body so it can be
// written before the body. Validated against LibreDWG's dwgread, which reads the resulting
// entities with no CRC mismatch.
func encodeEntityObject(handle uint64, e Entity) ([]byte, error) {
	typ, ok := objectTypeBS(e)
	if !ok {
		return nil, fmt.Errorf("dwg: cannot write entity type %s", e.EntityType().Name())
	}
	body := NewBitWriter()
	body.WriteHandle(0, handle) // the object's own handle
	body.WriteBS(0)             // EED size 0 (none)
	writeCommonEntityData(body)
	writeGeometry(body, e)

	bitsize := bsBits(typ) + 32 + body.Position() // type + the RL itself + body

	w := NewBitWriter()
	w.WriteBS(typ)
	w.WriteRL(uint32(bitsize))
	w.Append(body)
	// Handle stream (starts exactly at bitsize, bit-packed): the common-entity handles a
	// reader expects for a model-space entity with nolinks set — a null xdic and a null
	// layer reference. Without these a conforming reader runs past the object decoding the
	// handles. They are null (code 5, value 0) until a layer table is emitted.
	w.WriteHandle(5, 0) // xdicobjhandle (null)
	w.WriteHandle(5, 0) // layer (null)
	w.AlignToByte()
	payload := w.Bytes()

	out := NewBitWriter()
	out.WriteMS(len(payload)) // object size: the data bytes only (size field and CRC excluded)
	for _, by := range payload {
		out.WriteRC(by)
	}
	// Object CRC: over the object so far — the MS size field plus the payload — seeded
	// 0xC0C1 and stored little-endian, matching what conforming readers verify (ODA).
	out.WriteRS(crc16(0xC0C1, out.Bytes()))
	return out.Bytes(), nil
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

// writeCommonEntityData emits the R2000 common-entity-data data-stream fields for a
// model-space entity with no reactors, links, or special colour — the inverse of
// readCommonEntityData on the R2000 path.
func writeCommonEntityData(w *BitWriter) {
	w.WriteBit(0)     // preview_exists = no
	w.WriteBits(2, 2) // entmode = 2 (model space)
	w.WriteBL(0)      // num_reactors
	w.WriteBit(1)     // nolinks = 1 (no prev/next entity handles)
	w.WriteBS(0)      // colour (CMC index, pre-R2004)
	w.WriteBD(1.0)    // ltype_scale
	w.WriteBits(0, 2) // ltype_flags = ByLayer
	w.WriteBits(0, 2) // plotstyle_flags = ByLayer
	w.WriteBS(0)      // invisible
	w.WriteRC(0)      // linewt
}

// writeGeometry dispatches to the per-type geometry encoder (exact inverse of decodeEntity).
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

// r2000Builder assembles the flat R2000 file: a header + section locator, then the object
// records, then the handle→offset object map, with the locator pointing at the map.
type r2000Builder struct {
	refs []ObjectRef
}

func newR2000Builder() *r2000Builder { return &r2000Builder{} }

// sentinelHeaderEnd is the 16-byte sentinel that closes the R2000 header section (the
// reader checks the first sentinelSignatureLen bytes).
var sentinelHeaderEnd = [16]byte{
	0x95, 0xA0, 0x4E, 0x28, 0x99, 0x82, 0x1A, 0xE5,
	0x5E, 0x41, 0xE0, 0x5F, 0x9D, 0x3A, 0x4D, 0x00,
}

// build lays out the whole file: a fixed header area is reserved, objects are written
// after it (recording each handle→offset), then the object map, and finally the header +
// locator are filled in pointing at the map.
func (b *r2000Builder) build(entities []Entity) ([]byte, error) {
	const headerReserve = 0x100 // header + locator table fit comfortably in 256 bytes
	out := make([]byte, headerReserve)

	for i, e := range entities {
		if _, ok := objectTypeBS(e); !ok {
			continue
		}
		handle := uint64(i + 1)
		obj, err := encodeEntityObject(handle, e)
		if err != nil {
			return nil, err
		}
		b.refs = append(b.refs, ObjectRef{Handle: handle, Offset: int64(len(out))})
		out = append(out, obj...)
	}

	mapOffset := len(out)
	out = append(out, b.encodeObjectMap()...)
	mapSize := len(out) - mapOffset

	header := b.encodeHeader(int64(mapOffset), int64(mapSize))
	if len(header) > headerReserve {
		return nil, fmt.Errorf("dwg: R2000 header %d bytes exceeds reserve %d", len(header), headerReserve)
	}
	copy(out, header)
	return out, nil
}

// encodeObjectMap writes the handle→offset directory: one section of (handle delta,
// location delta) pairs followed by the terminating size-2 section.
func (b *r2000Builder) encodeObjectMap() []byte {
	pairs := NewBitWriter()
	var lastHandle uint64
	var lastLoc int64
	for _, ref := range b.refs {
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

// encodeHeader writes the fixed R2000 file header and the section locator table, pointing
// section id secObjectMap at the object map. The header-variables and classes sections are
// declared empty (address/size 0); Decode treats an absent header as unitless.
func (b *r2000Builder) encodeHeader(mapOffset, mapSize int64) []byte {
	buf := make([]byte, 0x19)
	copy(buf, "AC1015")
	buf[0x0C] = 0x00
	binary.LittleEndian.PutUint16(buf[0x13:], 30) // codepage ANSI_1252
	records := []SectionLocator{
		{ID: secHeaderVars, Address: 0, Size: 0},
		{ID: secClasses, Address: 0, Size: 0},
		{ID: secObjectMap, Address: mapOffset, Size: mapSize},
	}
	binary.LittleEndian.PutUint32(buf[0x15:], uint32(len(records)))
	for _, r := range records {
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
