// SPDX-License-Identifier: GPL-2.0-only

package dwg

// writeclasses.go emits the R2000 AcDb:Classes section (ODA §10.1): a begin sentinel, the
// class-data byte size, the class entries (each maps a dynamic type code ≥ 500 to a DXF
// name), a CRC, and an end sentinel. Only the non-fixed object types the writer actually
// emits need entries; a drawing of purely fixed types (LINE..SPLINE, the symbol tables)
// needs none, so the section is valid but empty.

// classBeginSentinel / classEndSentinel bracket the class-data area (ODA §10.1).
var (
	classBeginSentinel = []byte{
		0x8D, 0xA1, 0xC4, 0xB8, 0xC4, 0xA9, 0xF8, 0xC5,
		0xC0, 0xDC, 0xF4, 0x5F, 0xE7, 0xCF, 0xB6, 0x8A,
	}
	classEndSentinel = []byte{
		0x72, 0x5E, 0x3B, 0x47, 0x3B, 0x56, 0x07, 0x3A,
		0x3F, 0x23, 0x0B, 0xA0, 0x18, 0x30, 0x49, 0x75,
	}
)

// classDef is one class entry: the dynamic type number (≥ 500), the DXF record name, the
// owning application, and whether instances are objects (true) or entities (false).
type classDef struct {
	num      int
	dxfName  string
	cppName  string
	isObject bool
}

// itemClassEntity / itemClassObject are the ODA item-class-id values distinguishing classes
// that produce entities from those that produce objects.
const (
	itemClassEntity = 0x1F2
	itemClassObject = 0x1F3
)

// encodeClasses builds the AcDb:Classes section bytes for the given class definitions (which
// may be empty). The CRC covers the size field plus the class data, matching the header
// section's framing.
//
//nolint:funlen // sequential section framing: sentinels, class data, size, CRC.
func encodeClasses(defs []classDef) []byte {
	body := NewBitWriter()
	for _, d := range defs {
		body.WriteBS(d.num)
		body.WriteBS(0) // version / proxy flags
		writeName(body, "ObjectDBX Classes")
		writeName(body, d.cppName)
		writeName(body, d.dxfName)
		body.WriteBit(0) // wasazombie
		if d.isObject {
			body.WriteBS(itemClassObject)
		} else {
			body.WriteBS(itemClassEntity)
		}
	}
	body.AlignToByte()
	classData := body.Bytes()

	sized := NewBitWriter()
	sized.WriteRL(uint32(len(classData)))
	for _, by := range classData {
		sized.WriteRC(by)
	}
	sizedBytes := sized.Bytes()

	out := NewBitWriter()
	for _, by := range classBeginSentinel {
		out.WriteRC(by)
	}
	for _, by := range sizedBytes {
		out.WriteRC(by)
	}
	out.WriteRS(crc16(0xC0C1, sizedBytes))
	for _, by := range classEndSentinel {
		out.WriteRC(by)
	}
	return out.Bytes()
}
