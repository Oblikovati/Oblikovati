// SPDX-License-Identifier: GPL-2.0-only

package topomap

import (
	"oblikovati/kernel/exchange/step/part21"
	"oblikovati/kernel/topo"
)

// emitFace emits an ADVANCED_FACE for f: its surface, its FACE_(OUTER_)BOUND loops,
// and its same_sense flag (.F. for a reversed face, whose material side opposes the
// surface normal — the dual of topo.AddReversedFace).
func (d *disassembler) emitFace(f *topo.Face) (int, error) {
	surfID, err := d.emit.SurfaceToStep(f.Geometry())
	if err != nil {
		return 0, err
	}
	bounds, err := d.emitBounds(f)
	if err != nil {
		return 0, err
	}
	w := d.emit.Writer()
	return w.Add("ADVANCED_FACE", part21.QuoteString(""), refList(bounds),
		part21.Ref(surfID), part21.FormatBool(!f.Reversed())), nil
}

// emitBounds emits each loop of a face as a FACE_OUTER_BOUND (outer) or FACE_BOUND
// (inner), returning their entity ids.
func (d *disassembler) emitBounds(f *topo.Face) ([]int, error) {
	var ids []int
	for _, loop := range f.Loops() {
		id, err := d.emitBound(loop)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// emitBound emits one loop as an EDGE_LOOP wrapped in a FACE_(OUTER_)BOUND with
// orientation .T. (the oriented edges already carry the per-use direction).
func (d *disassembler) emitBound(loop *topo.Loop) (int, error) {
	orientedIDs, err := d.emitOrientedEdges(loop)
	if err != nil {
		return 0, err
	}
	w := d.emit.Writer()
	edgeLoop := w.Add("EDGE_LOOP", part21.QuoteString(""), refList(orientedIDs))
	keyword := "FACE_BOUND"
	if loop.IsOuter() {
		keyword = "FACE_OUTER_BOUND"
	}
	return w.Add(keyword, part21.QuoteString(""), part21.Ref(edgeLoop), part21.FormatBool(true)), nil
}

// emitOrientedEdges emits an ORIENTED_EDGE per edge use, with orientation .F. when
// the use is reversed.
func (d *disassembler) emitOrientedEdges(loop *topo.Loop) ([]int, error) {
	var ids []int
	for _, use := range loop.EdgeUses() {
		edgeID, err := d.edge(use.Edge())
		if err != nil {
			return nil, err
		}
		oe := d.emit.Writer().Add("ORIENTED_EDGE", part21.QuoteString(""), "*", "*",
			part21.Ref(edgeID), part21.FormatBool(!use.Reversed()))
		ids = append(ids, oe)
	}
	return ids, nil
}
