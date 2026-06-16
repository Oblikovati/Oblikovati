// SPDX-License-Identifier: GPL-2.0-only

package dwg

import (
	"math"
	"testing"
)

// buildInsertObject hand-encodes a single R2000 INSERT object: the data stream (type,
// bitsize, own handle, EED, common entity data, insert placement) and the handle stream
// (common handles then the block_header reference). It lets the INSERT decode + handle
// stream be exercised without the real .dwg corpus.
func buildInsertObject(ownHandle, blockHandle uint64, ins [3]float64, rot float64) []byte {
	body := NewBitWriter()
	body.WriteHandle(0, ownHandle)     // own handle
	body.WriteBS(0)                    // EED size 0
	writeCommonEntityData(body)        // model-space entity (entmode 2), nolinks
	body.Write3BD(ins)                 // ins_pt
	body.WriteBits(3, 2)               // scale_flag 3 -> (1,1,1)
	body.WriteBD(rot)                  // rotation
	body.Write3BD([3]float64{0, 0, 1}) // extrusion (3BD)
	body.WriteBit(0)                   // has_attribs = 0

	w := NewBitWriter()
	w.WriteBS(int(TypeInsert))
	w.WriteRL(uint32(bsBits(int(TypeInsert)) + 32 + body.Position()))
	w.Append(body)
	w.WriteHandle(3, 0)           // xdicobjhandle (R2000 always present; null)
	w.WriteHandle(5, 0)           // layer (null)
	w.WriteHandle(5, blockHandle) // block_header -- the referenced block
	w.AlignToByte()
	payload := w.Bytes()

	out := NewBitWriter()
	out.WriteMS(len(payload))
	for _, b := range payload {
		out.WriteRC(b)
	}
	out.WriteRS(crc16(0xC0C1, out.Bytes()))
	return out.Bytes()
}

// TestDecodeInsertSynthetic decodes a hand-built INSERT object and checks the placement and
// the resolved block reference — covering decodeInsert and the common-entity handle-stream
// walk in CI (no corpus needed).
func TestDecodeInsertSynthetic(t *testing.T) {
	data := buildInsertObject(0x10, 0x42, [3]float64{5, 6, 0}, 0.75)
	cur, err := seekEntity(data, ObjectRef{Handle: 0x10, Offset: 0}, R2000)
	if err != nil {
		t.Fatalf("seekEntity: %v", err)
	}
	in, _, err := decodeInsert(data, cur, 0x10, R2000)
	if err != nil {
		t.Fatalf("decodeInsert: %v", err)
	}
	if in.BlockHeader != 0x42 {
		t.Errorf("block ref = %#x, want 0x42", in.BlockHeader)
	}
	if math.Abs(in.Insertion[0]-5) > 1e-9 || math.Abs(in.Insertion[1]-6) > 1e-9 {
		t.Errorf("insertion = %v, want (5,6,0)", in.Insertion)
	}
	if math.Abs(in.Rotation-0.75) > 1e-9 || in.Scale != [3]float64{1, 1, 1} {
		t.Errorf("rotation/scale = %g %v", in.Rotation, in.Scale)
	}
}

// TestEntityIdentity covers the EntityHandle/EntityType accessors on every entity type.
func TestEntityIdentity(t *testing.T) {
	cases := []struct {
		e Entity
		t ObjectType
	}{
		{&Line{Handle: 1}, TypeLine}, {&Circle{Handle: 2}, TypeCircle},
		{&Arc{Handle: 3}, TypeArc}, {&Point{Handle: 4}, TypePoint},
		{&Ellipse{Handle: 5}, TypeEllipse}, {&LwPolyline{Handle: 6}, TypeLwpolyline},
		{&Spline{Handle: 7}, TypeSpline}, {&Insert{Handle: 8}, TypeInsert},
	}
	for i, c := range cases {
		if c.e.EntityType() != c.t {
			t.Errorf("case %d type = %v, want %v", i, c.e.EntityType(), c.t)
		}
		if c.e.EntityHandle() != uint64(i+1) {
			t.Errorf("case %d handle = %d, want %d", i, c.e.EntityHandle(), i+1)
		}
	}
}

// TestTransformEntityAllTypes covers the per-type affine transform for the curve types not
// already exercised (ellipse, polyline, spline) under a translate+scale.
func TestTransformEntityAllTypes(t *testing.T) {
	m := affine{{2, 0, 0, 1}, {0, 2, 0, 1}, {0, 0, 2, 0}} // scale 2, translate (1,1)
	el := transformEntity(&Ellipse{Center: [3]float64{1, 0, 0}, MajorAxis: [3]float64{1, 0, 0}, AxisRatio: 0.5}, m).(*Ellipse)
	if el.Center != [3]float64{3, 1, 0} || el.MajorAxis != [3]float64{2, 0, 0} {
		t.Errorf("ellipse transform: c=%v major=%v", el.Center, el.MajorAxis)
	}
	pl := transformEntity(&LwPolyline{Points: [][2]float64{{0, 0}, {1, 1}}, Bulges: []float64{0.2, 0}}, m).(*LwPolyline)
	if pl.Points[1] != [2]float64{3, 3} || pl.Bulges[0] != 0.2 {
		t.Errorf("polyline transform: %v bulges %v", pl.Points, pl.Bulges)
	}
	sp := transformEntity(&Spline{ControlPoints: [][3]float64{{0, 0, 0}, {1, 1, 0}}, FitPoints: [][3]float64{{2, 2, 0}}}, m).(*Spline)
	if sp.ControlPoints[1] != [3]float64{3, 3, 0} || sp.FitPoints[0] != [3]float64{5, 5, 0} {
		t.Errorf("spline transform: ctrl=%v fit=%v", sp.ControlPoints, sp.FitPoints)
	}
}

// TestReadInsertScaleFlags covers all four INSERT scale_flag forms.
func TestReadInsertScaleFlags(t *testing.T) {
	cases := []struct {
		build func(*BitWriter)
		want  [3]float64
	}{
		{func(w *BitWriter) { w.WriteBits(3, 2) }, [3]float64{1, 1, 1}},                                           // 3: unit
		{func(w *BitWriter) { w.WriteBits(1, 2); w.WriteDD(2); w.WriteDD(3) }, [3]float64{1, 2, 3}},               // 1: x=1, y,z DD
		{func(w *BitWriter) { w.WriteBits(2, 2); w.WriteRD(4) }, [3]float64{4, 4, 4}},                             // 2: uniform
		{func(w *BitWriter) { w.WriteBits(0, 2); w.WriteRD(2); w.WriteDD(5); w.WriteDD(6) }, [3]float64{2, 5, 6}}, // 0: x RD, y,z DD
	}
	for i, c := range cases {
		w := NewBitWriter()
		c.build(w)
		got := readInsertScale(NewBitReader(w.Bytes()))
		if got != c.want {
			t.Errorf("scale_flag case %d = %v, want %v", i, got, c.want)
		}
	}
}

// TestReadEntityColorBranches covers the colour encodings: a pre-R2004 CMC index and the
// R2004+ ENC flag variants (RGB, by-handle, alpha).
func TestReadEntityColorBranches(t *testing.T) {
	// Pre-R2004: a plain BitShort index, never by-handle.
	w := NewBitWriter()
	w.WriteBS(7)
	if readEntityColor(NewBitReader(w.Bytes()), R2000) {
		t.Error("pre-R2004 colour reported by-handle")
	}
	// R2004+ ENC: the high byte of the BitShort carries the flags.
	cases := []struct {
		raw      int
		extra    func(*BitWriter)
		byHandle bool
	}{
		{0x0001, nil, false}, // plain index
		{0x8000, func(w *BitWriter) { w.WriteBL(0xFF0000) }, false}, // 0x80: explicit RGB
		{0x4000, nil, true}, // 0x40: colour by handle
		{0x2000, func(w *BitWriter) { w.WriteBL(1) }, false}, // 0x20: alpha
	}
	for i, c := range cases {
		w := NewBitWriter()
		w.WriteBS(c.raw)
		if c.extra != nil {
			c.extra(w)
		}
		if got := readEntityColor(NewBitReader(w.Bytes()), R2018); got != c.byHandle {
			t.Errorf("ENC case %d byHandle = %v, want %v", i, got, c.byHandle)
		}
	}
}

// TestReadCommonEntityDataModernVersions runs the common-entity-data reader on the modern
// version branches (R2007/R2018) over a zero buffer so the version-gated reads execute.
func TestReadCommonEntityDataModernVersions(t *testing.T) {
	for _, v := range []Version{R2007, R2018} {
		ce := readCommonEntityData(NewBitReader(make([]byte, 64)), v)
		if ce.entmode != 0 {
			t.Errorf("version %d: zero buffer entmode = %d, want 0", v, ce.entmode)
		}
	}
}
