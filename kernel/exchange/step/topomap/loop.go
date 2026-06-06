// SPDX-License-Identifier: GPL-2.0-only

package topomap

import (
	"fmt"

	"oblikovati/kernel/exchange/step/part21"
	"oblikovati/kernel/topo"
)

// buildBound maps a FACE_OUTER_BOUND/FACE_BOUND to a LoopSpec. The bound's
// orientation flag flips the whole loop traversal when false; it composes with each
// oriented-edge orientation. FACE_OUTER_BOUND is the outer loop; FACE_BOUND an inner
// (hole) loop.
func (a *assembler) buildBound(boundID int) (topo.LoopSpec, error) {
	ent, err := a.g.Lookup(boundID)
	if err != nil {
		return topo.LoopSpec{}, err
	}
	outer := ent.Keyword == "FACE_OUTER_BOUND"
	if !outer && ent.Keyword != "FACE_BOUND" {
		return topo.LoopSpec{}, fmt.Errorf("topomap: #%d is %s, want FACE_(OUTER_)BOUND", boundID, ent.Keyword)
	}
	loopID, err := refParam(ent.Params, 1)
	if err != nil {
		return topo.LoopSpec{}, err
	}
	flip, err := ent.Params[2].AsBool()
	if err != nil {
		return topo.LoopSpec{}, err
	}
	uses, err := a.buildLoopUses(loopID, !flip)
	if err != nil {
		return topo.LoopSpec{}, err
	}
	return loopSpec(outer, uses), nil
}

// loopSpec wraps oriented uses as an outer or inner loop spec.
func loopSpec(outer bool, uses []topo.Use) topo.LoopSpec {
	if outer {
		return topo.OuterLoop(uses...)
	}
	return topo.InnerLoop(uses...)
}

// buildLoopUses maps an EDGE_LOOP's ORIENTED_EDGE list to kernel uses. boundFlip is
// applied to every use (it is the bound's orientation already negated by the caller
// so that boundFlip=true means "reverse this loop"); it composes with each
// oriented-edge orientation.
func (a *assembler) buildLoopUses(loopID int, boundFlip bool) ([]topo.Use, error) {
	ent, err := a.g.Lookup(loopID)
	if err != nil {
		return nil, err
	}
	if ent.Keyword != "EDGE_LOOP" {
		return nil, fmt.Errorf("topomap: #%d is %s, want EDGE_LOOP", loopID, ent.Keyword)
	}
	orientedRefs, err := refListValues(ent.Params[1])
	if err != nil {
		return nil, err
	}
	return a.orientedUses(orientedRefs, boundFlip)
}

// orientedUses converts ORIENTED_EDGE refs to kernel uses, reversing the list order
// when the bound is flipped (so the loop winds the other way).
func (a *assembler) orientedUses(orientedRefs []int, boundFlip bool) ([]topo.Use, error) {
	uses := make([]topo.Use, len(orientedRefs))
	for i, ref := range orientedRefs {
		u, err := a.orientedEdge(ref, boundFlip)
		if err != nil {
			return nil, err
		}
		uses[i] = u
	}
	if boundFlip {
		reverseUses(uses)
	}
	return uses, nil
}

// reverseUses reverses a slice of uses in place (loop traversal order flip).
func reverseUses(uses []topo.Use) {
	for i, j := 0, len(uses)-1; i < j; i, j = i+1, j-1 {
		uses[i], uses[j] = uses[j], uses[i]
	}
}

// refParam returns the entity id referenced by parameter i.
func refParam(params []part21.Value, i int) (int, error) {
	if i >= len(params) {
		return 0, fmt.Errorf("topomap: missing reference parameter %d (have %d)", i, len(params))
	}
	return params[i].AsRef()
}
