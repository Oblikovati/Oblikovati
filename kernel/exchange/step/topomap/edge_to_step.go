// SPDX-License-Identifier: GPL-2.0-only

package topomap

import (
	"oblikovati/kernel/exchange/step/part21"
	"oblikovati/kernel/topo"
)

// edge returns the shared EDGE_CURVE id for a kernel edge, emitting it (and its
// vertices and curve) once.
func (d *disassembler) edge(e *topo.Edge) (int, error) {
	if id, ok := d.edges[e]; ok {
		return id, nil
	}
	id, err := d.emitEdge(e)
	if err != nil {
		return 0, err
	}
	d.edges[e] = id
	return id, nil
}

// emitEdge emits EDGE_CURVE(start_vertex, end_vertex, curve, same_sense). The
// same_sense flag comes from the curve emitter (an arc swept CW emits .F.).
func (d *disassembler) emitEdge(e *topo.Edge) (int, error) {
	startV := d.vertex(e.StartVertex())
	endV := d.vertex(e.EndVertex())
	curveID, sameSense, err := d.emit.CurveToStep(e.Geometry())
	if err != nil {
		return 0, err
	}
	w := d.emit.Writer()
	return w.Add("EDGE_CURVE", part21.QuoteString(""), part21.Ref(startV), part21.Ref(endV),
		part21.Ref(curveID), part21.FormatBool(sameSense)), nil
}

// vertex returns the shared VERTEX_POINT id for a kernel vertex, emitting it once.
func (d *disassembler) vertex(v *topo.Vertex) int {
	if id, ok := d.verts[v]; ok {
		return id
	}
	pt := d.emit.Point(v.Point())
	id := d.emit.Writer().Add("VERTEX_POINT", part21.QuoteString(""), part21.Ref(pt))
	d.verts[v] = id
	return id
}
