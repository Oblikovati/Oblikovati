// SPDX-License-Identifier: GPL-2.0-only

package topomap

import (
	"fmt"

	"oblikovati/kernel/exchange/step/geommap"
	"oblikovati/kernel/topo"
)

// vertex returns the shared kernel Vertex for a VERTEX_POINT id, building it once so
// every edge that meets there references the same vertex.
func (a *assembler) vertex(vertexID int) (*topo.Vertex, error) {
	if v, ok := a.verts[vertexID]; ok {
		return v, nil
	}
	v, err := a.buildVertex(vertexID)
	if err != nil {
		return nil, err
	}
	a.verts[vertexID] = v
	return v, nil
}

// buildVertex maps VERTEX_POINT(name, point) to a kernel vertex at the scaled point.
func (a *assembler) buildVertex(vertexID int) (*topo.Vertex, error) {
	ent, err := a.g.Lookup(vertexID)
	if err != nil {
		return nil, err
	}
	if ent.Keyword != "VERTEX_POINT" {
		return nil, fmt.Errorf("topomap: #%d is %s, want VERTEX_POINT", vertexID, ent.Keyword)
	}
	pointID, err := refParam(ent.Params, 1)
	if err != nil {
		return nil, err
	}
	p, err := geommap.CartesianPoint(a.g, pointID, a.scale)
	if err != nil {
		return nil, err
	}
	lineage := topo.NewLineage(topo.Tok(a.feat, "vertex", a.nextV))
	a.nextV++
	return a.builder.AddVertex(p, lineage), nil
}
