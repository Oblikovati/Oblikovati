// SPDX-License-Identifier: GPL-2.0-only

package dwg

// This file frames a single R2000 object on disk. Every object — entity or table/dictionary
// object — shares the same outer layout: an MS byte size, then a bit stream of
// [type][bitsize RL][own-handle][EED][object body][handle stream], byte-aligned, then a
// 16-bit object CRC. The data-stream/handle-stream split at `bitsize` and the CRC seed
// (0xC0C1) are validated against LibreDWG's dwgread, which reads the result with no
// mismatch. See ODA §2.13 / the per-object prescriptions in §20.4.

// Handle reference codes (ODA §2.13). Codes 2–5 store the handle as an absolute value and
// also convey the ownership relation; the writer always uses absolute references (the
// relative forms 6/8/A/C are a size optimisation the reader supports but we do not need).
const (
	hardOwnerCode = 0x3 // owner needs the owned object; it cannot exist alone
	softPtrCode   = 0x4 // referencing object neither owns nor requires the target
	hardPtrCode   = 0x5 // referencing object requires the target, owned elsewhere
	softOwnerCode = 0x2 // owner does not need the owned object
	ownHandleCode = 0x0 // an object's own handle (no relation)
)

// objectBody is one object's two bit streams before framing: the data stream (everything
// after the type and bitsize: own handle, EED, common header, type-specific fields) and the
// handle stream (the trailing handle references). frameObject joins them.
type objectBody struct {
	handle  uint64
	typ     int
	data    *BitWriter // own-handle, EED, common header, type-specific data-stream fields
	handles *BitWriter // trailing handle-stream references
}

// newObjectBody starts an object's body with the common prefix every object shares: the
// own handle (code 0) and an empty EED block (size 0). Callers append the type-specific
// data-stream fields to .data and the handle references to .handles.
func newObjectBody(handle uint64, typ int) *objectBody {
	data := NewBitWriter()
	data.WriteHandle(ownHandleCode, handle) // the object's own handle
	data.WriteBS(0)                         // EED size 0 (no extended data)
	return &objectBody{handle: handle, typ: typ, data: data, handles: NewBitWriter()}
}

// frameObject assembles a finished object record: MS size, the bit stream, byte alignment,
// and the trailing CRC. bitsize is the bit offset (from just after the MS size) at which the
// handle stream begins — type + the RL field itself + the data stream — so a reader can split
// the two streams. The CRC covers the MS size field plus the payload (ODA), seeded 0xC0C1.
func frameObject(b *objectBody) []byte {
	bitsize := bsBits(b.typ) + 32 + b.data.Position() // type + the RL + data stream

	w := NewBitWriter()
	w.WriteBS(b.typ)
	w.WriteRL(uint32(bitsize))
	w.Append(b.data)
	w.Append(b.handles)
	w.AlignToByte()
	payload := w.Bytes()

	out := NewBitWriter()
	out.WriteMS(len(payload)) // object size: payload bytes only (size field and CRC excluded)
	for _, by := range payload {
		out.WriteRC(by)
	}
	out.WriteRS(crc16(0xC0C1, out.Bytes()))
	return out.Bytes()
}

// writeName writes a pre-R2007 text value (TV): a BitShort character count then the raw
// bytes, with NO terminating NUL — the form AutoCAD reads back (the ODA SHAPEFILE example
// stores "STANDARD" as length 8 + 8 bytes). Matches readTV's inverse.
func writeName(w *BitWriter, s string) {
	w.WriteBS(len(s))
	for i := 0; i < len(s); i++ {
		w.WriteRC(s[i])
	}
}

// writeTextEmpty writes an empty text value (length 0, no bytes) for unset
// description/path strings on table records.
func writeTextEmpty(w *BitWriter) { writeName(w, "") }
