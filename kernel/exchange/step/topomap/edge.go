// SPDX-License-Identifier: GPL-2.0-only

package topomap

import (
	"fmt"

	"oblikovati/kernel/exchange/step/part21"
	"oblikovati/kernel/topo"
)

// orientedEdge maps ORIENTED_EDGE(name, *, *, edge_element, orientation) to a kernel
// use. The kernel Edge always runs start→end; an ORIENTED_EDGE with orientation=false
// traverses it end→start, i.e. a reversed use. boundFlip toggles each use again so the
// whole loop can be reversed (the bound's orientation), composing the sense triple.
func (a *assembler) orientedEdge(orientedID int, boundFlip bool) (topo.Use, error) {
	ent, err := a.g.Lookup(orientedID)
	if err != nil {
		return topo.Use{}, err
	}
	if ent.Keyword != "ORIENTED_EDGE" {
		return topo.Use{}, fmt.Errorf("topomap: #%d is %s, want ORIENTED_EDGE", orientedID, ent.Keyword)
	}
	edgeID, err := refParam(ent.Params, 3)
	if err != nil {
		return topo.Use{}, err
	}
	orientation, err := ent.Params[4].AsBool()
	if err != nil {
		return topo.Use{}, err
	}
	edge, err := a.edge(edgeID)
	if err != nil {
		return topo.Use{}, err
	}
	reversed := !orientation
	if boundFlip {
		reversed = !reversed
	}
	return topo.Use{Edge: edge, Reversed: reversed}, nil
}

// edge returns the shared kernel Edge for an EDGE_CURVE id, building it once. Sharing
// is what lets two adjacent faces reference the same edge (the closure prerequisite).
func (a *assembler) edge(edgeCurveID int) (*topo.Edge, error) {
	if e, ok := a.edges[edgeCurveID]; ok {
		return e, nil
	}
	e, err := a.buildEdge(edgeCurveID)
	if err != nil {
		return nil, err
	}
	a.edges[edgeCurveID] = e
	return e, nil
}

// edgeCurveRefs holds the resolved parts of an EDGE_CURVE statement.
type edgeCurveRefs struct {
	startVertexID int
	endVertexID   int
	curveID       int
	sameSense     bool
}

// readEdgeCurve parses EDGE_CURVE(name, start_vertex, end_vertex, curve, same_sense).
func (a *assembler) readEdgeCurve(edgeCurveID int) (edgeCurveRefs, error) {
	ent, err := a.g.Lookup(edgeCurveID)
	if err != nil {
		return edgeCurveRefs{}, err
	}
	if ent.Keyword != "EDGE_CURVE" {
		return edgeCurveRefs{}, fmt.Errorf("topomap: #%d is %s, want EDGE_CURVE", edgeCurveID, ent.Keyword)
	}
	return parseEdgeCurve(ent)
}

// parseEdgeCurve extracts the five EDGE_CURVE fields.
func parseEdgeCurve(ent *part21.RawEntity) (edgeCurveRefs, error) {
	if len(ent.Params) < 5 {
		return edgeCurveRefs{}, fmt.Errorf("topomap: EDGE_CURVE #%d wants 5 params, got %d", ent.ID, len(ent.Params))
	}
	var r edgeCurveRefs
	var err error
	if r.startVertexID, err = ent.Params[1].AsRef(); err != nil {
		return r, err
	}
	if r.endVertexID, err = ent.Params[2].AsRef(); err != nil {
		return r, err
	}
	if r.curveID, err = ent.Params[3].AsRef(); err != nil {
		return r, err
	}
	r.sameSense, err = ent.Params[4].AsBool()
	return r, err
}
