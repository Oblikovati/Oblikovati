// SPDX-License-Identifier: GPL-2.0-only

package topomap

import (
	"fmt"

	"oblikovati.org/kernel/exchange/step/part21"
	"oblikovati.org/kernel/topo"
)

// boundLoop is one face bound before outer/inner is finalized: the oriented edge uses and
// whether the STEP keyword (FACE_OUTER_BOUND) declared it the outer loop. A degenerate bound
// is a VERTEX_LOOP — a single point at a parametric singularity (a sphere pole, a cone apex)
// that carries no edges and so imposes no boundary; it is dropped from the face's loops.
type boundLoop struct {
	outer      bool
	degenerate bool
	uses       []topo.Use
}

// buildBound maps a FACE_OUTER_BOUND/FACE_BOUND to a boundLoop. The bound's orientation
// flag flips the whole loop traversal when false; it composes with each oriented-edge
// orientation. FACE_OUTER_BOUND declares the outer loop; FACE_BOUND a (possibly outer —
// see ensureOuterLoop) bound. The outer/inner decision is finalized by the caller.
func (a *assembler) buildBound(boundID int) (boundLoop, error) {
	ent, err := a.g.Lookup(boundID)
	if err != nil {
		return boundLoop{}, err
	}
	outer := ent.Keyword == "FACE_OUTER_BOUND"
	if !outer && ent.Keyword != "FACE_BOUND" {
		return boundLoop{}, fmt.Errorf("topomap: #%d is %s, want FACE_(OUTER_)BOUND", boundID, ent.Keyword)
	}
	loopID, err := refParam(ent.Params, 1)
	if err != nil {
		return boundLoop{}, err
	}
	uses, degenerate, err := a.boundUses(ent, loopID)
	if err != nil {
		return boundLoop{}, err
	}
	return boundLoop{outer: outer, degenerate: degenerate, uses: uses}, nil
}

// boundUses resolves a bound's oriented edge uses, applying the bound's orientation flag. A
// VERTEX_LOOP (a singular point — a sphere pole / cone apex) carries no edges and is reported
// as degenerate so the caller drops it from the face's loops.
func (a *assembler) boundUses(ent *part21.RawEntity, loopID int) (uses []topo.Use, degenerate bool, err error) {
	loopEnt, err := a.g.Lookup(loopID)
	if err != nil {
		return nil, false, err
	}
	if loopEnt.Keyword == "VERTEX_LOOP" {
		return nil, true, nil
	}
	flip, err := ent.Params[2].AsBool()
	if err != nil {
		return nil, false, err
	}
	uses, err = a.buildLoopUses(loopID, !flip)
	return uses, false, err
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
