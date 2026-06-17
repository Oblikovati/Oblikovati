// SPDX-License-Identifier: GPL-2.0-only

package dwg

// writevport.go encodes the two large symbol-table records: the *Active VPORT (the model
// viewport) and the STANDARD DIMSTYLE. Both field layouts mirror LibreDWG's decode of a real
// R2000 file field-for-field (the bit types matter: BD vs RD, B vs BB, the 4-bit VIEWMODE),
// using neutral default values for a fresh drawing.

// writeVport encodes the *Active VPORT record with a unit-square view and the standard grid
// and snap defaults. The handle-stream tail is the vport control owner, a null xdic, and the
// null named/base UCS handles.
//
//nolint:funlen // one VPORT field per line in the fixed R2000 order; length is the layout.
func writeVport(h *graphHandles) []byte {
	b := newObjectBody(h.vportActive, int(TypeVport))
	writeRecordCommon(b, h.vportControl, "*Active")
	b.data.WriteBD(1)                     // VIEWSIZE
	b.data.WriteBD(1)                     // view width
	b.data.Write2RD([2]float64{0, 0})     // VIEWCTR
	b.data.Write3BD([3]float64{0, 0, 0})  // view target
	b.data.Write3BD([3]float64{0, 0, 1})  // VIEWDIR
	b.data.WriteBD(0)                     // VIEWTWIST
	b.data.WriteBD(50)                    // LENSLENGTH
	b.data.WriteBD(0)                     // FRONTZ
	b.data.WriteBD(0)                     // BACKZ
	b.data.WriteBits(0, 4)                // VIEWMODE (4 bits)
	b.data.WriteRC(0)                     // render mode
	b.data.Write2RD([2]float64{0, 0})     // lower-left
	b.data.Write2RD([2]float64{1, 1})     // upper-right
	b.data.WriteBit(0)                    // UCSFOLLOW
	b.data.WriteBS(1000)                  // circle zoom
	b.data.WriteBit(1)                    // FASTZOOM
	b.data.WriteBits(3, 2)                // UCSICON (2 bits)
	b.data.WriteBit(0)                    // GRIDMODE
	b.data.Write2RD([2]float64{10, 10})   // GRIDUNIT
	b.data.WriteBit(0)                    // SNAPMODE
	b.data.WriteBit(0)                    // SNAPSTYLE
	b.data.WriteBS(0)                     // SNAPISOPAIR
	b.data.WriteBD(0)                     // SNAPANG
	b.data.Write2RD([2]float64{0, 0})     // SNAPBASE
	b.data.Write2RD([2]float64{10, 10})   // SNAPUNIT
	b.data.WriteBit(0)                    // ucs at origin
	b.data.WriteBit(1)                    // UCSVP
	b.data.Write3BD([3]float64{0, 0, 0})  // ucs origin
	b.data.Write3BD([3]float64{1, 0, 0})  // ucs x-dir
	b.data.Write3BD([3]float64{0, 1, 0})  // ucs y-dir
	b.data.WriteBD(0)                     // ucs elevation
	b.data.WriteBS(0)                     // UCSORTHOVIEW
	b.handles.WriteHandle(hardPtrCode, 0) // named UCS (null)
	b.handles.WriteHandle(hardPtrCode, 0) // base UCS (null)
	return frameObject(b)
}

// writeDimstyle encodes the STANDARD dimension style with AutoCAD's default DIMxxx values.
// The trailing handle stream is the style reference (DIMTXSTY → STANDARD text style) and the
// four null arrowhead-block handles (DIMLDRBLK/DIMBLK/DIMBLK1/DIMBLK2).
//
//nolint:funlen // one DIMxxx variable per line in the fixed R2000 order; length is the layout.
func writeDimstyle(h *graphHandles) []byte {
	b := newObjectBody(h.dimstyleStandard, int(TypeDimstyle))
	writeRecordCommon(b, h.dimstyleControl, "Standard")
	writeTextEmpty(b.data)                                                 // DIMPOST
	writeTextEmpty(b.data)                                                 // DIMAPOST
	for _, v := range []float64{1, 0.18, 0.0625, 0.38, 0.18, 0, 0, 0, 0} { // DIMSCALE..DIMTM
		b.data.WriteBD(v)
	}
	for i := 0; i < 6; i++ { // DIMTOL,DIMLIM,DIMTIH,DIMTOH,DIMSE1,DIMSE2
		b.data.WriteBit(0)
	}
	b.data.WriteBS(0)                                                 // DIMTAD
	b.data.WriteBS(0)                                                 // DIMZIN
	b.data.WriteBS(0)                                                 // DIMAZIN
	for _, v := range []float64{0.18, 0.09, 0, 25.4, 1, 0, 1, 0.09} { // DIMTXT..DIMGAP
		b.data.WriteBD(v)
	}
	b.data.WriteBD(0)                                           // DIMALTRND
	b.data.WriteBit(0)                                          // DIMALT
	b.data.WriteBS(2)                                           // DIMALTD
	b.data.WriteBit(1)                                          // DIMTOFL
	b.data.WriteBit(0)                                          // DIMSAH
	b.data.WriteBit(0)                                          // DIMTIX
	b.data.WriteBit(0)                                          // DIMSOXD
	b.data.WriteBS(0)                                           // DIMCLRD
	b.data.WriteBS(0)                                           // DIMCLRE
	b.data.WriteBS(0)                                           // DIMCLRT
	for _, v := range []int{0, 4, 4, 2, 3, 0, 0, 2, 46, 0, 0} { // DIMADEC..DIMJUST
		b.data.WriteBS(v)
	}
	b.data.WriteBit(0)                    // DIMSD1
	b.data.WriteBit(0)                    // DIMSD2
	for _, v := range []int{0, 8, 0, 0} { // DIMTOLJ,DIMTZIN,DIMALTZ,DIMALTTZ
		b.data.WriteBS(v)
	}
	b.data.WriteBit(0)                                  // DIMUPT
	b.data.WriteBS(3)                                   // DIMATFIT
	b.data.WriteBS(-2)                                  // DIMLWD (default)
	b.data.WriteBS(-2)                                  // DIMLWE (default)
	b.data.WriteBit(0)                                  // unknown trailing flag the oracle reads as 0
	b.handles.WriteHandle(hardPtrCode, h.styleStandard) // DIMTXSTY
	for i := 0; i < 4; i++ {
		b.handles.WriteHandle(hardPtrCode, 0) // DIMLDRBLK, DIMBLK, DIMBLK1, DIMBLK2 (null)
	}
	return frameObject(b)
}
